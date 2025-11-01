package mib

import (
	"embed"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"

	"github.com/sleepinggenius2/gosmi"
	"github.com/sleepinggenius2/gosmi/types"
)

// Parser gestisce il parsing dei file MIB
type Parser struct {
	db *Database
}

var initOnce sync.Once

//go:embed standard/*
var standardMibsFS embed.FS

// NewParser crea un parser che utilizza il database indicato per la risoluzione dei nodi.
func NewParser(db *Database) *Parser { return &Parser{db: db} }

func ensureGosmiInit(appDataDir string) {
	initOnce.Do(func() {
		gosmi.Init()

		// Percorso dove estrarremo i MIB standard
		embeddedMibsPath := filepath.Join(appDataDir, "mibs", "standard")

		// Estrai i MIB standard se non esistono
		if err := extractEmbeddedMibs(embeddedMibsPath); err != nil {
			fmt.Printf("ERROR: Failed to extract standard MIBs: %v\n", err)
			// Continuiamo comunque, magari i file ci sono già
		}

		// Aggiungi directory MIB standard e di sistema al search path
		standardPaths := []string{
			embeddedMibsPath,       // La nostra cartella di MIB estratti
			"/usr/share/snmp/mibs", // Percorso comune su Linux
		}

		for _, path := range standardPaths {
			gosmi.AppendPath(path)
		}
		fmt.Printf("Gosmi initialized. Standard MIBs path: %s\n", embeddedMibsPath)
	})
}

// extractEmbeddedMibs estrae i MIB da embed.FS alla destinazione specificata
func extractEmbeddedMibs(destPath string) error {
	// Controlla se la directory esiste già e non è vuota
	if _, err := os.Stat(destPath); !os.IsNotExist(err) {
		// Directory esiste, assumiamo sia a posto.
		// Per una logica più robusta, si potrebbe controllare la versione dell'app
		// e sovrascrivere se necessario.
		return nil
	}

	fmt.Printf("Extracting standard MIBs to %s...\n", destPath)

	// Itera sui file nel FS embeddato
	return fs.WalkDir(standardMibsFS, "standard", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil // Salta le directory
		}

		data, err := standardMibsFS.ReadFile(path)
		if err != nil {
			return fmt.Errorf("failed to read embedded file %s: %w", path, err)
		}

		// Crea il percorso di destinazione
		destFilePath := filepath.Join(destPath, strings.TrimPrefix(path, "standard/"))
		if err := os.MkdirAll(filepath.Dir(destFilePath), 0755); err != nil {
			return fmt.Errorf("failed to create directory for %s: %w", destFilePath, err)
		}

		// Scrivi il file
		return os.WriteFile(destFilePath, data, 0644)
	})
}

// LoadMIBFile carica e parsifica un file MIB partendo dal path locale.
// Ricava il nome modulo dal filename e lo carica tramite gosmi.
func (p *Parser) LoadMIBFile(filePath string, appDataDir string) (string, error) {
	ensureGosmiInit(appDataDir)

	// Aggiungi la directory del file alla search path (per risolvere le dipendenze).
	dir := filepath.Dir(filePath)
	gosmi.AppendPath(dir)

	// Nome modulo = nome file senza estensione (IF-MIB, SNMPv2-MIB, ecc.)
	base := filepath.Base(filePath)
	modName := strings.TrimSuffix(base, filepath.Ext(base))
	if modName == "" {
		return "", fmt.Errorf("impossibile ricavare il nome modulo da %q", filePath)
	}

	// Carica modulo per NOME (LoadModule accetta un nome o un percorso in path noti).
	loadedName, err := gosmi.LoadModule(modName)
	if err != nil {
		return "", fmt.Errorf("failed to load MIB module %q: %v", modName, err)
	}

	// Carica anche le dipendenze standard comuni se non già caricate
	standardMIBs := []string{"RFC1155-SMI", "RFC-1212", "SNMPv2-SMI", "SNMPv2-TC", "SNMPv2-CONF"}
	for _, stdMib := range standardMIBs {
		if _, err := gosmi.GetModule(stdMib); err != nil {
			// Non ancora caricato, prova a caricarlo
			if loadedMib, err := gosmi.LoadModule(stdMib); err != nil {
				fmt.Printf("Warning: Could not load standard MIB %s: %v\n", stdMib, err)
			} else {
				fmt.Printf("Loaded dependency: %s\n", loadedMib)
			}
		}
	}

	// Recupera l'oggetto modulo per verificare che sia stato caricato
	gosmiModule, err := gosmi.GetModule(loadedName)
	if err != nil {
		return "", fmt.Errorf("failed to get module object %q: %v", loadedName, err)
	}

	// Salva modulo nel DB
	moduleID, err := p.db.SaveModule(loadedName, filePath)
	if err != nil {
		return "", fmt.Errorf("failed to save module %q: %v", loadedName, err)
	}

	// Determina eventuali dipendenze mancanti dal modulo importato
	missingImportsSet := make(map[string]struct{})
	for _, imp := range gosmiModule.GetImports() {
		dependency := strings.TrimSpace(imp.Module)
		if dependency == "" || strings.EqualFold(dependency, loadedName) {
			continue
		}
		exists, err := p.db.ModuleExists(dependency)
		if err != nil {
			return "", fmt.Errorf("failed to verify dependency %q: %v", dependency, err)
		}
		if !exists {
			missingImportsSet[dependency] = struct{}{}
		}
	}
	missingImports := make([]string, 0, len(missingImportsSet))
	for dep := range missingImportsSet {
		missingImports = append(missingImports, dep)
	}
	sort.Strings(missingImports)

	// Parsifica e salva i nodi di TUTTI i moduli caricati (incluse dipendenze)
	nodes, skippedCount, err := p.parseAllLoadedModules()
	if err != nil {
		return "", fmt.Errorf("failed to parse modules: %v", err)
	}

	// Conta nodi con OID vuoti (dipendenze mancanti)
	emptyOidCount := 0
	for _, mod := range gosmi.GetLoadedModules() {
		for _, node := range mod.GetNodes() {
			if node.RenderNumeric() == "" && node.Name != "" {
				emptyOidCount++
			}
		}
	}

	if emptyOidCount > 0 {
		fmt.Printf("⚠️  WARNING: %d nodes have unresolved OIDs (missing dependencies were skipped).\n", skippedCount)
		fmt.Printf("   Load the required MIB modules first to resolve all OIDs.\n")
	}

	fmt.Printf("Parsed %d nodes successfully (skipped %d with unresolved OIDs)\n", len(nodes), skippedCount)

	if err := p.db.SaveNodes(nodes, moduleID); err != nil {
		return "", fmt.Errorf("failed to save nodes for module %q: %v", loadedName, err)
	}

	// Calcola statistiche per modulo e aggiorna il database
	statsByModule := make(map[string]ModuleStats)
	statsByModule[loadedName] = ModuleStats{}

	for _, node := range nodes {
		moduleName := node.Module
		if moduleName == "" {
			moduleName = loadedName
		}
		stats := statsByModule[moduleName]
		stats.NodeCount++
		switch node.Type {
		case "scalar":
			stats.ScalarCount++
		case "table":
			stats.TableCount++
		case "column":
			stats.ColumnCount++
		}
		statsByModule[moduleName] = stats
	}

	for _, module := range gosmi.GetLoadedModules() {
		stats := statsByModule[module.Name]
		stats.TypeCount = len(module.GetTypes())
		statsByModule[module.Name] = stats
	}

	for moduleName, stats := range statsByModule {
		if err := p.db.UpdateModuleStats(moduleName, stats); err != nil {
			return "", fmt.Errorf("failed to update stats for module %q: %v", moduleName, err)
		}
	}

	if err := p.db.UpdateModuleMetadata(loadedName, skippedCount, missingImports); err != nil {
		return "", fmt.Errorf("failed to update metadata for module %q: %v", loadedName, err)
	}

	return loadedName, nil
}

// parseAllLoadedModules parsifica TUTTI i nodi da tutti i moduli caricati
func (p *Parser) parseAllLoadedModules() (nodes []*Node, skippedCount int, err error) {
	var allNodes []*Node
	processedNodes := make(map[string]bool) // Mappa per evitare duplicati

	modules := gosmi.GetLoadedModules()
	fmt.Printf("Parsing all %d loaded modules...\n", len(modules))

	for _, module := range modules {
		smiNodes := module.GetNodes()
		for _, smiNode := range smiNodes {
			if smiNode.Name == "" {
				continue
			}

			oidStr := smiNode.RenderNumeric()
			if oidStr == "" || oidStr == "0" || oidStr == "0.0" || oidStr == "2" {
				if oidStr == "" {
					skippedCount++
				}
				continue
			}

			if !processedNodes[oidStr] {
				if mibNode := p.convertNode(smiNode); mibNode != nil {
					allNodes = append(allNodes, mibNode)
					processedNodes[oidStr] = true
				}
			}
		}
	}
	return allNodes, skippedCount, nil
}

// convertNode converte un gosmi.SmiNode nel nostro Node
func (p *Parser) convertNode(smiNode gosmi.SmiNode) *Node {
	if smiNode.Name == "" {
		return nil
	}

	module := smiNode.GetModule()
	moduleName := module.Name

	// OID numerico completo
	oidNum := smiNode.RenderNumeric()
	if oidNum == "" {
		fmt.Printf("    WARNING: Node %s has empty OID (raw: %v, len: %d)\n", smiNode.Name, smiNode.Oid, smiNode.OidLen)
		return nil
	}

	// Calcola parent OID (rimuove l'ultima parte)
	parentOID := ""
	if idx := strings.LastIndex(oidNum, "."); idx > 0 {
		parentOID = oidNum[:idx]
	}

	// Determina il tipo di nodo
	nodeType := getNodeType(smiNode)

	// Per nodi root comuni, usa parent OID specifici per raggruppare meglio
	// Esempio: tutti i nodi sotto mib-2 (1.3.6.1.2.1) avranno parent = "1.3.6.1.2.1"
	if strings.HasPrefix(oidNum, "1.3.6.1.2.1.") && strings.Count(oidNum, ".") == 5 {
		// Nodi diretti sotto mib-2 (system, interfaces, etc.)
		parentOID = "1.3.6.1.2.1"
	} else if strings.HasPrefix(oidNum, "1.3.6.1.4.1.") && strings.Count(oidNum, ".") == 5 {
		// Nodi diretti sotto enterprises
		parentOID = "1.3.6.1.4.1"
	}

	return &Node{
		OID:         oidNum,
		Name:        smiNode.Name,
		ParentOID:   parentOID,
		Type:        nodeType,
		Syntax:      getSyntax(smiNode),
		Access:      getAccess(smiNode),
		Status:      getStatus(smiNode),
		Description: cleanDescription(smiNode.Description),
		Module:      moduleName,
	}
}

// getNodeType determina il tipo di nodo
func getNodeType(smiNode gosmi.SmiNode) string {
	switch smiNode.Kind {
	case types.NodeNode:
		return "node"
	case types.NodeScalar:
		return "scalar"
	case types.NodeTable:
		return "table"
	case types.NodeRow:
		return "row"
	case types.NodeColumn:
		return "column"
	case types.NodeNotification:
		return "notification"
	case types.NodeGroup:
		return "group"
	case types.NodeCompliance:
		return "compliance"
	default:
		return "unknown"
	}
}

// getSyntax ottiene la sintassi del nodo (tipo + ranges + enum)
func getSyntax(smiNode gosmi.SmiNode) string {
	t := smiNode.Type
	if t == nil {
		return ""
	}

	// Proviamo a usare il nome del tipo se disponibile, altrimenti il rendering
	syntax := t.Name
	if syntax == "" {
		syntax = t.String()
	}

	// Intervalli
	if len(t.Ranges) > 0 {
		var rs []string
		for _, r := range t.Ranges {
			rs = append(rs, fmt.Sprintf("%d..%d", r.MinValue, r.MaxValue))
		}
		syntax += fmt.Sprintf(" (%s)", strings.Join(rs, " | "))
	}

	// Enum (Values)
	if t.Enum != nil && len(t.Enum.Values) > 0 {
		var parts []string
		for _, nn := range t.Enum.Values {
			parts = append(parts, fmt.Sprintf("%s(%d)", nn.Name, nn.Value))
		}
		syntax += fmt.Sprintf(" {%s}", strings.Join(parts, ", "))
	}

	return syntax
}

// getAccess ottiene il livello di accesso
func getAccess(smiNode gosmi.SmiNode) string {
	switch smiNode.Access {
	case types.AccessNotAccessible:
		return "not-accessible"
	case types.AccessNotify:
		return "accessible-for-notify"
	case types.AccessReadOnly:
		return "read-only"
	case types.AccessReadWrite:
		return "read-write"
	default:
		return ""
	}
}

// getStatus ottiene lo status del nodo
func getStatus(smiNode gosmi.SmiNode) string {
	switch smiNode.Status {
	case types.StatusCurrent:
		return "current"
	case types.StatusDeprecated:
		return "deprecated"
	case types.StatusObsolete:
		return "obsolete"
	case types.StatusMandatory:
		return "mandatory"
	case types.StatusOptional:
		return "optional"
	default:
		return ""
	}
}

// cleanDescription pulisce la descrizione rimuovendo whitespace eccessivo
func cleanDescription(desc string) string {
	desc = strings.TrimSpace(desc)
	if desc == "" {
		return ""
	}
	lines := strings.Split(desc, "\n")
	cleaned := make([]string, 0, len(lines))
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" {
			cleaned = append(cleaned, line)
		}
	}
	return strings.Join(cleaned, "\n")
}

// LoadStandardMIBs carica i MIB standard comuni passando i **nomi** modulo.
// Aggiunge anche la cartella ai path di gosmi, così le dipendenze vengono risolte.
func (p *Parser) LoadStandardMIBs(appDataDir string, mibsDir string) error {
	ensureGosmiInit(appDataDir)
	if mibsDir != "" {
		gosmi.AppendPath(mibsDir)
	}

	standardMIBs := []string{
		"SNMPv2-SMI",
		"SNMPv2-TC",
		"SNMPv2-CONF",
		"SNMPv2-MIB",
		"IF-MIB",
		"IP-MIB",
		"TCP-MIB",
		"UDP-MIB",
	}

	for _, name := range standardMIBs {
		if _, err := gosmi.LoadModule(name); err != nil {
			fmt.Printf("Warning: could not load module %s: %v\n", name, err)
			continue
		}
	}
	return nil
}
