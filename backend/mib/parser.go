package mib

import (
	"bufio"
	"bytes"
	"embed"
	"fmt"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strings"
	"sync"

	"github.com/sleepinggenius2/gosmi"
	"github.com/sleepinggenius2/gosmi/types"
)

// Parser gestisce il parsing dei file MIB
type Parser struct {
	db      *Database
	debug   bool
	logger  *log.Logger
}

var (
	initOnce sync.Once
	initErr  error
)

//go:embed standard/*
var standardMibsFS embed.FS

// NewParser crea un parser che utilizza il database indicato per la risoluzione dei nodi.
func NewParser(db *Database) *Parser {
	return &Parser{
		db:     db,
		debug:  true, // Abilita debug di default
		logger: log.New(os.Stderr, "[MIB-PARSER] ", log.LstdFlags|log.Lshortfile),
	}
}

// SetDebug abilita o disabilita il logging dettagliato
func (p *Parser) SetDebug(enabled bool) {
	p.debug = enabled
}

func (p *Parser) debugLog(format string, args ...interface{}) {
	if p.debug && p.logger != nil {
		p.logger.Printf(format, args...)
	}
}

func (p *Parser) errorLog(format string, args ...interface{}) {
	if p.logger != nil {
		p.logger.Printf("ERROR: "+format, args...)
	}
}

func (p *Parser) warnLog(format string, args ...interface{}) {
	if p.logger != nil {
		p.logger.Printf("WARNING: "+format, args...)
	}
}

func ensureGosmiInit(appDataDir string) error {
	initOnce.Do(func() {
		log.Printf("[MIB-PARSER] Initializing gosmi library...")
		gosmi.Init()

		// Percorso dove estrarremo i MIB standard
		embeddedMibsPath := filepath.Join(appDataDir, "mibs", "standard")
		log.Printf("[MIB-PARSER] Standard MIBs will be extracted to: %s", embeddedMibsPath)

		// Estrai i MIB standard se non esistono
		if err := extractEmbeddedMibs(embeddedMibsPath); err != nil {
			initErr = fmt.Errorf("failed to extract standard MIBs: %w", err)
			log.Printf("[MIB-PARSER] ERROR: %v", initErr)
			return
		}

		// Aggiungi directory MIB standard e di sistema al search path (cross-platform)
		standardPaths := getPlatformMIBPaths(embeddedMibsPath)

		log.Printf("[MIB-PARSER] Adding %d MIB search paths:", len(standardPaths))
		for i, path := range standardPaths {
			if stat, err := os.Stat(path); err == nil && stat.IsDir() {
				gosmi.AppendPath(path)
				log.Printf("[MIB-PARSER]   [%d] %s (exists)", i+1, path)
			} else {
				log.Printf("[MIB-PARSER]   [%d] %s (skipped: %v)", i+1, path, err)
			}
		}

		log.Printf("[MIB-PARSER] Gosmi initialized successfully")
	})
	return initErr
}

// getPlatformMIBPaths restituisce i percorsi di ricerca MIB specifici per la piattaforma
func getPlatformMIBPaths(embeddedMibsPath string) []string {
	paths := []string{embeddedMibsPath}

	switch runtime.GOOS {
	case "linux":
		paths = append(paths,
			"/usr/share/snmp/mibs",
			"/usr/share/mibs",
			"/var/lib/mibs",
		)
	case "darwin": // macOS
		paths = append(paths,
			"/usr/local/share/snmp/mibs",
			"/opt/homebrew/share/snmp/mibs",
			"/usr/share/snmp/mibs",
		)
	case "windows":
		// Windows: cerca in %ProgramFiles% e %ProgramData%
		if programFiles := os.Getenv("ProgramFiles"); programFiles != "" {
			paths = append(paths, filepath.Join(programFiles, "snmp", "mibs"))
		}
		if programData := os.Getenv("ProgramData"); programData != "" {
			paths = append(paths, filepath.Join(programData, "snmp", "mibs"))
		}
	}

	return paths
}

// PreloadStandardMIBs precarica i MIB standard comuni per evitare errori di dipendenze mancanti.
// Questa funzione dovrebbe essere chiamata all'avvio dell'applicazione dopo ensureGosmiInit.
func (p *Parser) PreloadStandardMIBs(appDataDir string) error {
	p.debugLog("=== PreloadStandardMIBs START ===")

	// Assicurati che gosmi sia inizializzato
	if err := ensureGosmiInit(appDataDir); err != nil {
		return fmt.Errorf("failed to initialize gosmi: %w", err)
	}

	// Lista dei MIB standard da precaricare (in ordine di dipendenza)
	standardMIBs := []string{
		// SMIv1 base
		"RFC1155-SMI",    // Structure of Management Information
		"RFC-1212",       // Concise MIB Definitions (OBJECT-TYPE macro)
		"RFC-1215",       // TRAP-TYPE macro
		"RFC1213-MIB",    // MIB-II

		// SMIv2 base
		"SNMPv2-SMI",     // Structure of Management Information for SNMPv2
		"SNMPv2-TC",      // Textual Conventions for SNMPv2
		"SNMPv2-CONF",    // Conformance Statements for SNMPv2
		"SNMPv2-MIB",     // MIB for SNMPv2

		// Common dependencies
		"IANAifType-MIB", // IANA-maintained interface types
		"IF-MIB",         // Interfaces MIB
		"IP-MIB",         // IP MIB
		"TCP-MIB",        // TCP MIB
		"UDP-MIB",        // UDP MIB

		// Network services
		"INET-ADDRESS-MIB",      // Internet address textual conventions
		"TRANSPORT-ADDRESS-MIB", // Transport address textual conventions

		// SNMP framework (SNMPv3)
		"SNMP-FRAMEWORK-MIB",
		"SNMP-TARGET-MIB",
		"SNMP-NOTIFICATION-MIB",
		"SNMP-COMMUNITY-MIB",
	}

	loadedCount := 0
	failedCount := 0
	alreadyLoadedCount := 0

	p.debugLog("Attempting to preload %d standard MIB modules...", len(standardMIBs))

	for _, mibName := range standardMIBs {
		// Controlla se già caricato
		if _, err := gosmi.GetModule(mibName); err == nil {
			p.debugLog("  [%d/%d] %s - already loaded", loadedCount+alreadyLoadedCount+failedCount+1, len(standardMIBs), mibName)
			alreadyLoadedCount++
			continue
		}

		// Prova a caricare
		p.debugLog("  [%d/%d] Loading %s...", loadedCount+alreadyLoadedCount+failedCount+1, len(standardMIBs), mibName)
		loaded, err := gosmi.LoadModule(mibName)
		if err != nil {
			p.warnLog("  [%d/%d] %s - FAILED: %v", loadedCount+alreadyLoadedCount+failedCount+1, len(standardMIBs), mibName, err)
			failedCount++
			continue
		}

		if loaded == "" {
			p.warnLog("  [%d/%d] %s - FAILED: LoadModule returned empty name", loadedCount+alreadyLoadedCount+failedCount+1, len(standardMIBs), mibName)
			failedCount++
			continue
		}

		p.debugLog("  [%d/%d] %s - OK", loadedCount+alreadyLoadedCount+failedCount+1, len(standardMIBs), mibName)
		loadedCount++
	}

	p.debugLog("=== PreloadStandardMIBs COMPLETE ===")
	p.debugLog("Summary: %d loaded, %d already loaded, %d failed", loadedCount, alreadyLoadedCount, failedCount)

	if loadedCount+alreadyLoadedCount == 0 {
		return fmt.Errorf("failed to preload any standard MIBs (0/%d successful)", len(standardMIBs))
	}

	// Salva i moduli precaricati nel database per renderli visibili nell'UI
	p.debugLog("Saving preloaded modules to database...")
	savedCount := 0
	embeddedMibsPath := filepath.Join(appDataDir, "mibs", "standard")

	for _, mibName := range standardMIBs {
		// Controlla se il modulo è stato caricato in gosmi
		module, err := gosmi.GetModule(mibName)
		if err != nil {
			continue // Salta i moduli che non sono stati caricati
		}

		// Controlla se è già nel database
		exists, err := p.db.ModuleExists(module.Name)
		if err != nil {
			p.warnLog("Failed to check if module %s exists in DB: %v", module.Name, err)
			continue
		}
		if exists {
			p.debugLog("  Module %s already in database", module.Name)
			continue
		}

		// Costruisci il path del file MIB standard
		// Prova diverse estensioni comuni
		var filePath string
		possibleExtensions := []string{".txt", ".mib", ".my", ""}
		for _, ext := range possibleExtensions {
			testPath := filepath.Join(embeddedMibsPath, module.Name+ext)
			if _, err := os.Stat(testPath); err == nil {
				filePath = testPath
				break
			}
		}

		if filePath == "" {
			p.warnLog("Could not find file for module %s in %s", module.Name, embeddedMibsPath)
			continue
		}

		// Salva il modulo nel database
		moduleID, err := p.db.SaveModule(module.Name, filePath)
		if err != nil {
			p.warnLog("Failed to save module %s to database: %v", module.Name, err)
			continue
		}

		// Parsifica e salva i nodi solo di questo modulo specifico
		nodes, skippedCount := p.parseModuleNodes(module)

		if len(nodes) > 0 {
			if err := p.db.SaveNodes(nodes, moduleID); err != nil {
				p.warnLog("Failed to save nodes for module %s: %v", module.Name, err)
				continue
			}
		}

		// Aggiorna metadati
		if err := p.db.UpdateModuleMetadata(module.Name, skippedCount, nil); err != nil {
			p.warnLog("Failed to update metadata for module %s: %v", module.Name, err)
		}

		p.debugLog("  Saved module %s to database (%d nodes, %d skipped)", module.Name, len(nodes), skippedCount)
		savedCount++
	}

	log.Printf("[MIB-PARSER] Preloaded %d standard MIBs (%d already loaded, %d failed, %d saved to DB)",
		loadedCount, alreadyLoadedCount, failedCount, savedCount)

	return nil
}

// validateMIBFile verifica che il file MIB sia valido e leggibile
func (p *Parser) validateMIBFile(filePath string) error {
	// Controlla che il path non sia vuoto
	if filePath == "" {
		return fmt.Errorf("file path is empty")
	}

	// Controlla che il file esista
	stat, err := os.Stat(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("file does not exist: %s", filePath)
		}
		return fmt.Errorf("cannot stat file: %w", err)
	}

	// Controlla che sia un file regolare
	if stat.IsDir() {
		return fmt.Errorf("path is a directory, not a file: %s", filePath)
	}

	// Controlla dimensione del file (max 10MB per un MIB è ragionevole)
	const maxSize = 10 * 1024 * 1024
	if stat.Size() > maxSize {
		return fmt.Errorf("file too large: %d bytes (max %d)", stat.Size(), maxSize)
	}

	// Controlla che il file sia leggibile
	f, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("cannot open file: %w", err)
	}
	defer f.Close()

	// Leggi i primi 512 byte per validazione base
	buf := make([]byte, 512)
	n, err := f.Read(buf)
	if err != nil && n == 0 {
		return fmt.Errorf("cannot read file: %w", err)
	}

	// Controlla che contenga testo ASCII/UTF-8 valido
	content := string(buf[:n])
	if !strings.Contains(content, "DEFINITIONS") &&
	   !strings.Contains(content, "IMPORTS") &&
	   !strings.Contains(content, "BEGIN") {
		p.warnLog("File may not be a valid MIB file (missing expected keywords)")
	}

	p.debugLog("File validation passed: %s (size: %d bytes)", filePath, stat.Size())
	return nil
}

// extractEmbeddedMibs estrae i MIB da embed.FS alla destinazione specificata
// Sovrascrive sempre i file esistenti per garantire aggiornamenti
func extractEmbeddedMibs(destPath string) error {
	// Rimuovi la directory esistente per forzare aggiornamento
	if stat, err := os.Stat(destPath); err == nil && stat.IsDir() {
		log.Printf("[MIB-PARSER] Removing old standard MIBs directory: %s", destPath)
		if err := os.RemoveAll(destPath); err != nil {
			log.Printf("[MIB-PARSER] WARNING: Failed to remove old MIBs: %v", err)
			// Continua comunque, sovrascriveremo i file
		}
	}

	log.Printf("[MIB-PARSER] Extracting standard MIBs to %s...", destPath)

	// Crea la directory di destinazione
	if err := os.MkdirAll(destPath, 0755); err != nil {
		return fmt.Errorf("failed to create destination directory: %w", err)
	}

	extractedCount := 0

	// Itera sui file nel FS embeddato
	err := fs.WalkDir(standardMibsFS, "standard", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return fmt.Errorf("walk error at %s: %w", path, err)
		}
		if d.IsDir() {
			return nil // Salta le directory
		}

		data, err := standardMibsFS.ReadFile(path)
		if err != nil {
			return fmt.Errorf("failed to read embedded file %s: %w", path, err)
		}

		// Crea il percorso di destinazione
		relPath := strings.TrimPrefix(path, "standard/")
		destFilePath := filepath.Join(destPath, relPath)

		// Crea directory intermedie se necessario
		destDir := filepath.Dir(destFilePath)
		if err := os.MkdirAll(destDir, 0755); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", destDir, err)
		}

		// Scrivi il file
		if err := os.WriteFile(destFilePath, data, 0644); err != nil {
			return fmt.Errorf("failed to write file %s: %w", destFilePath, err)
		}

		extractedCount++
		log.Printf("[MIB-PARSER]   Extracted: %s", relPath)
		return nil
	})

	if err != nil {
		return err
	}

	log.Printf("[MIB-PARSER] Successfully extracted %d standard MIB files", extractedCount)
	return nil
}

// LoadMIBFile carica e parsifica un file MIB partendo dal path locale.
// Ricava il nome modulo dal filename e lo carica tramite gosmi.
func (p *Parser) LoadMIBFile(filePath string, appDataDir string) (string, error) {
	p.debugLog("=== LoadMIBFile START ===")
	p.debugLog("File path: %s", filePath)
	p.debugLog("App data dir: %s", appDataDir)

	// Validazione del file in input
	if err := p.validateMIBFile(filePath); err != nil {
		p.errorLog("File validation failed: %v", err)
		return "", fmt.Errorf("invalid MIB file: %w", err)
	}

	// Inizializza gosmi
	if err := ensureGosmiInit(appDataDir); err != nil {
		p.errorLog("Gosmi initialization failed: %v", err)
		return "", fmt.Errorf("failed to initialize gosmi: %w", err)
	}

	// Aggiungi la directory del file alla search path (per risolvere le dipendenze).
	dir := filepath.Dir(filePath)
	absDir, err := filepath.Abs(dir)
	if err != nil {
		p.warnLog("Cannot get absolute path for %s: %v", dir, err)
		absDir = dir
	}
	p.debugLog("Adding directory to search path: %s", absDir)
	gosmi.AppendPath(absDir)

	// Nome modulo = nome file senza estensione (IF-MIB, SNMPv2-MIB, ecc.)
	base := filepath.Base(filePath)
	modName := strings.TrimSuffix(base, filepath.Ext(base))
	if modName == "" {
		return "", fmt.Errorf("impossibile ricavare il nome modulo da %q", filePath)
	}
	p.debugLog("Module name from filename: %s", modName)

	loadedName, loadErr := p.loadModuleWithFallbacks(modName, filePath, appDataDir)
	if loadErr != nil {
		p.errorLog("Failed to load module: %v", loadErr)
		return "", loadErr
	}
	p.debugLog("Successfully loaded module: %s", loadedName)

	// Carica anche le dipendenze standard comuni se non già caricate
	p.debugLog("Loading standard MIB dependencies...")
	standardMIBs := []string{"RFC1155-SMI", "RFC-1212", "SNMPv2-SMI", "SNMPv2-TC", "SNMPv2-CONF"}
	for _, stdMib := range standardMIBs {
		if _, err := gosmi.GetModule(stdMib); err != nil {
			// Non ancora caricato, prova a caricarlo
			p.debugLog("  Attempting to load %s...", stdMib)
			if loadedMib, err := gosmi.LoadModule(stdMib); err != nil {
				p.warnLog("Could not load standard MIB %s: %v", stdMib, err)
			} else {
				p.debugLog("  Successfully loaded dependency: %s", loadedMib)
			}
		} else {
			p.debugLog("  %s already loaded", stdMib)
		}
	}

	// Recupera l'oggetto modulo per verificare che sia stato caricato
	p.debugLog("Retrieving module object...")
	gosmiModule, err := gosmi.GetModule(loadedName)
	if err != nil {
		p.errorLog("Failed to get module object %q: %v", loadedName, err)
		return "", fmt.Errorf("failed to get module object %q: %v", loadedName, err)
	}
	p.debugLog("Module object retrieved: %s (organization: %s)", gosmiModule.Name, gosmiModule.Organization)

	// Salva modulo nel DB
	p.debugLog("Saving module to database...")
	moduleID, err := p.db.SaveModule(loadedName, filePath)
	if err != nil {
		p.errorLog("Failed to save module %q to database: %v", loadedName, err)
		return "", fmt.Errorf("failed to save module %q: %v", loadedName, err)
	}
	p.debugLog("Module saved with ID: %d", moduleID)

	// Determina eventuali dipendenze mancanti dal modulo importato
	p.debugLog("Checking module dependencies...")
	missingImportsSet := make(map[string]struct{})
	imports := gosmiModule.GetImports()
	p.debugLog("Module has %d imports", len(imports))

	for _, imp := range imports {
		dependency := strings.TrimSpace(imp.Module)
		if dependency == "" || strings.EqualFold(dependency, loadedName) {
			continue
		}
		p.debugLog("  Checking dependency: %s", dependency)
		exists, err := p.db.ModuleExists(dependency)
		if err != nil {
			p.errorLog("Failed to verify dependency %q: %v", dependency, err)
			return "", fmt.Errorf("failed to verify dependency %q: %v", dependency, err)
		}
		if !exists {
			p.warnLog("  Missing dependency: %s", dependency)
			missingImportsSet[dependency] = struct{}{}
		} else {
			p.debugLog("  Dependency %s is available", dependency)
		}
	}
	missingImports := make([]string, 0, len(missingImportsSet))
	for dep := range missingImportsSet {
		missingImports = append(missingImports, dep)
	}
	sort.Strings(missingImports)

	if len(missingImports) > 0 {
		p.warnLog("Module has %d missing dependencies: %v", len(missingImports), missingImports)
	}

	// Parsifica e salva i nodi di TUTTI i moduli caricati (incluse dipendenze)
	p.debugLog("Parsing all loaded modules...")
	nodes, skippedCount, err := p.parseAllLoadedModules()
	if err != nil {
		p.errorLog("Failed to parse modules: %v", err)
		return "", fmt.Errorf("failed to parse modules: %v", err)
	}
	p.debugLog("Parsed %d nodes, skipped %d nodes with unresolved OIDs", len(nodes), skippedCount)

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
		p.warnLog("⚠️  %d nodes have unresolved OIDs (missing dependencies)", skippedCount)
		p.warnLog("   Load the required MIB modules first to resolve all OIDs")
	}

	p.debugLog("Saving %d nodes to database...", len(nodes))
	if err := p.db.SaveNodes(nodes, moduleID); err != nil {
		p.errorLog("Failed to save nodes: %v", err)
		return "", fmt.Errorf("failed to save nodes for module %q: %v", loadedName, err)
	}
	p.debugLog("Nodes saved successfully")

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

	p.debugLog("=== LoadMIBFile SUCCESS ===")
	p.debugLog("Module %s loaded with %d nodes (%d skipped)", loadedName, len(nodes), skippedCount)
	return loadedName, nil
}

// parseModuleNodes parsifica i nodi di un singolo modulo
func (p *Parser) parseModuleNodes(module gosmi.SmiModule) (nodes []*Node, skippedCount int) {
	var moduleNodes []*Node
	processedOIDs := make(map[string]bool)

	smiNodes := module.GetNodes()
	p.debugLog("    Module %s has %d nodes", module.Name, len(smiNodes))

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

		// Evita duplicati
		if processedOIDs[oidStr] {
			continue
		}

		if mibNode := p.convertNode(smiNode); mibNode != nil {
			moduleNodes = append(moduleNodes, mibNode)
			processedOIDs[oidStr] = true
		}
	}

	p.debugLog("    Parsed %d nodes from %s (%d skipped)", len(moduleNodes), module.Name, skippedCount)
	return moduleNodes, skippedCount
}

// parseAllLoadedModules parsifica TUTTI i nodi da tutti i moduli caricati
func (p *Parser) parseAllLoadedModules() (nodes []*Node, skippedCount int, err error) {
	var allNodes []*Node
	processedNodes := make(map[string]bool) // Mappa per evitare duplicati

	modules := gosmi.GetLoadedModules()
	p.debugLog("Parsing all %d loaded modules...", len(modules))

	for _, module := range modules {
		p.debugLog("  Processing module: %s", module.Name)
		smiNodes := module.GetNodes()
		p.debugLog("    Module has %d nodes", len(smiNodes))

		moduleNodeCount := 0
		moduleSkipCount := 0

		for _, smiNode := range smiNodes {
			if smiNode.Name == "" {
				continue
			}

			oidStr := smiNode.RenderNumeric()
			if oidStr == "" || oidStr == "0" || oidStr == "0.0" || oidStr == "2" {
				if oidStr == "" {
					skippedCount++
					moduleSkipCount++
					p.debugLog("      Skipped node %s (empty OID)", smiNode.Name)
				}
				continue
			}

			if !processedNodes[oidStr] {
				if mibNode := p.convertNode(smiNode); mibNode != nil {
					allNodes = append(allNodes, mibNode)
					processedNodes[oidStr] = true
					moduleNodeCount++
				} else {
					p.warnLog("      Failed to convert node %s (OID: %s)", smiNode.Name, oidStr)
				}
			}
		}
		p.debugLog("    Processed %d nodes from %s (%d skipped)", moduleNodeCount, module.Name, moduleSkipCount)
	}
	p.debugLog("Total nodes processed: %d (total skipped: %d)", len(allNodes), skippedCount)
	return allNodes, skippedCount, nil
}

// convertNode converte un gosmi.SmiNode nel nostro Node
func (p *Parser) convertNode(smiNode gosmi.SmiNode) *Node {
	if smiNode.Name == "" {
		return nil
	}

	module := smiNode.GetModule()
	moduleName := module.Name
	if moduleName == "" {
		p.warnLog("Node %s has empty module name", smiNode.Name)
		return nil
	}

	// OID numerico completo
	oidNum := smiNode.RenderNumeric()
	if oidNum == "" {
		p.debugLog("Node %s has empty OID (raw: %v, len: %d)", smiNode.Name, smiNode.Oid, smiNode.OidLen)
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

// extractModuleName legge il file MIB e cerca la dichiarazione del modulo.
func extractModuleName(filePath string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	const maxCapacity = 1024 * 1024 // 1MB, sufficiente per righe lunghe
	buf := make([]byte, 0, 64*1024)
	scanner.Buffer(buf, maxCapacity)

	for scanner.Scan() {
		line := scanner.Text()
		if idx := strings.Index(line, "--"); idx >= 0 {
			line = line[:idx]
		}
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		upperLine := strings.ToUpper(line)
		if strings.Contains(upperLine, "DEFINITIONS") {
			parts := strings.Fields(line)
			if len(parts) > 0 {
				return parts[0], nil
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return "", err
	}
	return "", fmt.Errorf("modulo non trovato in %s", filePath)
}

var (
	// Match "..MAX" or "(digit..MAX" pattern (es: "0..MAX", "1..MAX")
	reDoubleDotMax = regexp.MustCompile(`(\d+)?\.\.\s*MAX\b`)
	reCRLF         = regexp.MustCompile(`\r\n?`)

	// Common MIB syntax errors found in Net-SNMP rfcmibs.diff
	// Fix INTEGER range overflow: 2147483648 > INT32_MAX
	reIntegerOverflow = regexp.MustCompile(`INTEGER\s*\(\s*(\d+)\s*\.\.\s*2147483648\s*\)`)

	// Fix lowercase SIZE keyword (should be uppercase)
	reLowercaseSize = regexp.MustCompile(`\(\s*size\s+\(`)

	// Fix hex integer with leading zero: '07fffffff'h -> '7fffffff'h
	reHexLeadingZero = regexp.MustCompile(`'0([0-9a-fA-F]+)'h`)

	// Fix LAST-UPDATED with too many digits (should be YYYYMMDDHHmmZ, not YYYYMMDDHHmmssZ)
	reLastUpdatedLong = regexp.MustCompile(`LAST-UPDATED\s+"(\d{12})\d{2}(Z)"`)
)

func (p *Parser) loadModuleWithFallbacks(filenameBase string, originalPath string, appDataDir string) (string, error) {
	p.debugLog("=== loadModuleWithFallbacks START ===")
	p.debugLog("Filename base: %s", filenameBase)
	p.debugLog("Original path: %s", originalPath)

	var firstErr error
	var tried []string

	addTried := func(label string, err error) {
		if label != "" && err != nil {
			tried = append(tried, fmt.Sprintf("%s: %v", label, err))
			p.debugLog("  Attempt failed: %s: %v", label, err)
			if firstErr == nil {
				firstErr = err
			}
		}
	}

	tryLoad := func(name string) (string, error) {
		if name == "" {
			return "", fmt.Errorf("nome modulo vuoto")
		}
		p.debugLog("  Trying to load module: %s", name)
		loaded, err := gosmi.LoadModule(name)
		if err != nil {
			return "", err
		}
		if loaded == "" {
			return "", fmt.Errorf("gosmi.LoadModule ha restituito nome vuoto per %q", name)
		}
		p.debugLog("  Successfully loaded: %s", loaded)
		return loaded, nil
	}

	p.debugLog("Step 1: Trying original file candidates...")
	moduleCandidates := orderedUnique()
	if moduleName, err := extractModuleName(originalPath); err == nil && moduleName != "" {
		p.debugLog("  Extracted module name from file: %s", moduleName)
		moduleCandidates.add(moduleName)
	} else if err != nil {
		addTried("extract module name", err)
	}
	moduleCandidates.add(filenameBase)
	if upper := strings.ToUpper(filenameBase); upper != filenameBase {
		moduleCandidates.add(upper)
	}
	if baseWithExt := filepath.Base(originalPath); baseWithExt != "" {
		moduleCandidates.add(baseWithExt)
		if upperExt := strings.ToUpper(baseWithExt); upperExt != baseWithExt {
			moduleCandidates.add(upperExt)
		}
	}

	p.debugLog("Trying %d module name candidates", len(moduleCandidates.values()))
	for _, candidate := range moduleCandidates.values() {
		if loaded, err := tryLoad(candidate); err == nil {
			p.debugLog("=== loadModuleWithFallbacks SUCCESS ===")
			return loaded, nil
		} else {
			addTried(candidate, err)
		}
	}

	p.debugLog("Step 2: Creating sanitized copy and retrying...")
	sanitizedPath, sanitizeErr := p.ensureSanitizedCopy(originalPath, appDataDir)
	if sanitizeErr != nil {
		addTried("sanitize", sanitizeErr)
		p.errorLog("All loading attempts failed. Tried: %s", strings.Join(tried, " | "))
		return "", fmt.Errorf("impossibile caricare il modulo %q: %v (tentativi: %s)", originalPath, firstErr, strings.Join(tried, " | "))
	}

	// Rimuovi temporaneamente la directory originale dal search path per dare priorità alla versione sanificata
	sanitizedDir := filepath.Dir(sanitizedPath)

	// Aggiungi la directory sanificata come prima nel path
	p.debugLog("Prioritizing sanitized directory in search path: %s", sanitizedDir)

	// Purtroppo gosmi non ha un modo per rimuovere path, quindi usiamo un nome univoco
	// per il file sanificato per evitare conflitti con l'originale
	gosmi.AppendPath(sanitizedDir)

	// Prova a caricare il file sanificato usando il path ASSOLUTO invece del nome del modulo
	// Questo forza gosmi a usare il file esatto che vogliamo
	p.debugLog("  Trying to load from absolute sanitized path: %s", sanitizedPath)

	// Crea un symlink o rinomina temporaneamente il file con un nome univoco
	uniqueName := fmt.Sprintf("_sanitized_%s", filepath.Base(sanitizedPath))
	uniquePath := filepath.Join(sanitizedDir, uniqueName)

	// Copia con nome unico per evitare conflitti
	sanitizedData, err := os.ReadFile(sanitizedPath)
	if err == nil {
		if err := os.WriteFile(uniquePath, sanitizedData, 0644); err == nil {
			p.debugLog("  Created unique sanitized copy: %s", uniquePath)
			defer os.Remove(uniquePath) // Pulisci dopo
		}
	}

	sanitizedCandidates := orderedUnique()

	// Prova prima con il nome univoco
	uniqueModName := strings.TrimSuffix(uniqueName, filepath.Ext(uniqueName))
	sanitizedCandidates.add(uniqueModName)

	if moduleName, err := extractModuleName(sanitizedPath); err == nil && moduleName != "" {
		p.debugLog("  Extracted module name from sanitized file: %s", moduleName)
		sanitizedCandidates.add(moduleName)
	} else if err != nil {
		addTried("extract module name (sanitized)", err)
	}
	sanitizedCandidates.add(filenameBase)
	sanitizedCandidates.add(filepath.Base(sanitizedPath))

	p.debugLog("Trying %d sanitized candidates", len(sanitizedCandidates.values()))
	for _, candidate := range sanitizedCandidates.values() {
		if loaded, err := tryLoad(candidate); err == nil {
			p.debugLog("Successfully loaded module %s from sanitized copy: %s", loaded, sanitizedPath)
			p.debugLog("=== loadModuleWithFallbacks SUCCESS ===")
			return loaded, nil
		} else {
			addTried(candidate+" (sanitized)", err)
		}
	}

	if firstErr == nil {
		firstErr = fmt.Errorf("nessun tentativo di caricamento eseguito")
	}

	p.errorLog("All loading attempts failed. Tried: %s", strings.Join(tried, " | "))
	return "", fmt.Errorf("impossibile caricare il modulo %q: %v (tentativi: %s)", originalPath, firstErr, strings.Join(tried, " | "))
}

type orderedUniqueSet struct {
	seen map[string]struct{}
	list []string
}

func orderedUnique() *orderedUniqueSet {
	return &orderedUniqueSet{
		seen: make(map[string]struct{}),
		list: make([]string, 0, 4),
	}
}

func (s *orderedUniqueSet) add(value string) {
	value = strings.TrimSpace(value)
	if value == "" {
		return
	}
	if _, ok := s.seen[value]; ok {
		return
	}
	s.seen[value] = struct{}{}
	s.list = append(s.list, value)
}

func (s *orderedUniqueSet) values() []string {
	return s.list
}

// fixRFC1212Structure corregge la struttura del file RFC1212-MIB
// Il file RFC1212 ha un bug noto: IndexSyntax è definito DOPO il macro END
// invece che prima. Questo causa errori di parsing.
func fixRFC1212Structure(data []byte) []byte {
	content := string(data)

	// Cerca il pattern problematico: END seguito da IndexSyntax
	if !strings.Contains(content, "RFC1212") {
		return data // Non è RFC1212, non modificare
	}

	// Trova la riga con END (con spazi iniziali)
	lines := strings.Split(content, "\n")
	endLineIdx := -1
	for i, line := range lines {
		if strings.TrimSpace(line) == "END" && i > 10 { // Non il primo END
			endLineIdx = i
			break
		}
	}

	if endLineIdx == -1 {
		return data // END non trovato
	}

	// Cerca IndexSyntax dopo END
	indexSyntaxStartLine := -1
	for i := endLineIdx + 1; i < len(lines); i++ {
		if strings.Contains(lines[i], "IndexSyntax ::=") {
			indexSyntaxStartLine = i
			break
		}
	}

	if indexSyntaxStartLine == -1 {
		return data // IndexSyntax non trovato dopo END, va bene così
	}

	// Trova la fine del blocco IndexSyntax (chiusura graffa con indentazione specifica)
	indexSyntaxEndLine := -1
	for i := indexSyntaxStartLine + 1; i < len(lines); i++ {
		if strings.TrimSpace(lines[i]) == "}" {
			indexSyntaxEndLine = i
			break
		}
	}

	if indexSyntaxEndLine == -1 {
		return data // Fine non trovata
	}

	// Estrai il blocco IndexSyntax (inclusa la riga vuota dopo})
	indexSyntaxBlock := lines[indexSyntaxStartLine : indexSyntaxEndLine+1]

	// Ricostruisci: prima parte (fino a END escluso) + IndexSyntax + END + resto (dopo IndexSyntax)
	var newLines []string
	newLines = append(newLines, lines[:endLineIdx]...)           // Prima di END
	newLines = append(newLines, indexSyntaxBlock...)              // IndexSyntax
	newLines = append(newLines, "")                               // Riga vuota
	newLines = append(newLines, lines[endLineIdx])                // END
	newLines = append(newLines, lines[indexSyntaxEndLine+1:]...) // Dopo IndexSyntax

	return []byte(strings.Join(newLines, "\n"))
}

// ensureSanitizedCopy normalizza alcune costruzioni non supportate da libsmi
// creando una copia temporanea nella cartella dati dell'applicazione.
func (p *Parser) ensureSanitizedCopy(originalPath string, appDataDir string) (string, error) {
	p.debugLog("Creating sanitized copy of MIB file...")
	p.debugLog("  Original: %s", originalPath)

	data, err := os.ReadFile(originalPath)
	if err != nil {
		return "", fmt.Errorf("read original MIB: %w", err)
	}
	p.debugLog("  File size: %d bytes", len(data))

	// Normalizza line endings (Windows -> Unix)
	normalized := reCRLF.ReplaceAll(data, []byte("\n"))
	normalizeCount := (len(data) - len(normalized))
	if normalizeCount > 0 {
		p.debugLog("  Normalized %d CRLF sequences to LF", normalizeCount)
	}

	// Fix specifico per RFC1212-MIB che ha IndexSyntax DOPO il macro END
	// Questo è un bug noto nel file RFC1212
	beforeFix := normalized
	normalized = fixRFC1212Structure(normalized)
	if !bytes.Equal(beforeFix, normalized) {
		p.debugLog("  Applied RFC1212 structure fix (moved IndexSyntax before END)")
	}

	// Applica tutte le sanitizzazioni comuni basate su Net-SNMP rfcmibs.diff
	sanitized := normalized
	fixesApplied := 0

	// 1. Fix INTEGER overflow: INTEGER(1..2147483648) -> INTEGER(1..2147483647)
	if matches := reIntegerOverflow.FindAll(sanitized, -1); len(matches) > 0 {
		sanitized = reIntegerOverflow.ReplaceAll(sanitized, []byte("INTEGER ($1..2147483647)"))
		fixesApplied += len(matches)
		p.debugLog("  Fixed %d INTEGER range overflow(s) (2147483648 -> 2147483647)", len(matches))
	}

	// 2. Fix lowercase 'size' -> 'SIZE'
	if matches := reLowercaseSize.FindAll(sanitized, -1); len(matches) > 0 {
		sanitized = reLowercaseSize.ReplaceAll(sanitized, []byte("(SIZE ("))
		fixesApplied += len(matches)
		p.debugLog("  Fixed %d lowercase 'size' keyword(s) -> 'SIZE'", len(matches))
	}

	// 3. Fix hex literals with leading zeros: '07fffffff'h -> '7fffffff'h
	if matches := reHexLeadingZero.FindAll(sanitized, -1); len(matches) > 0 {
		sanitized = reHexLeadingZero.ReplaceAll(sanitized, []byte("'$1'h"))
		fixesApplied += len(matches)
		p.debugLog("  Fixed %d hex literal(s) with leading zero", len(matches))
	}

	// 4. Fix LAST-UPDATED timestamp: "YYYYMMDDHHmmssZ" -> "YYYYMMDDHHmmZ"
	if matches := reLastUpdatedLong.FindAll(sanitized, -1); len(matches) > 0 {
		sanitized = reLastUpdatedLong.ReplaceAll(sanitized, []byte(`LAST-UPDATED "$1$2"`))
		fixesApplied += len(matches)
		p.debugLog("  Fixed %d LAST-UPDATED timestamp(s) (removed seconds)", len(matches))
	}

	// 5. Sostituisci "..MAX" con un valore numerico valido
	// Gestisce sia "..MAX" che "N..MAX" (es: "0..MAX", "1..MAX")
	maxPatternCount := 0
	sanitized = reDoubleDotMax.ReplaceAllFunc(sanitized, func(match []byte) []byte {
		matchStr := string(match)
		maxPatternCount++
		// Estrai il numero iniziale se presente (es: "0" in "0..MAX")
		if idx := strings.Index(matchStr, ".."); idx > 0 {
			prefix := matchStr[:idx]
			return []byte(prefix + "..2147483647")
		}
		// Se non c'è numero, sostituisci solo MAX
		return bytes.Replace(match, []byte("MAX"), []byte("2147483647"), 1)
	})

	if maxPatternCount > 0 {
		fixesApplied += maxPatternCount
		p.debugLog("  Replaced %d '..MAX' pattern(s) with numeric value", maxPatternCount)
	}

	// Log riepilogo
	totalChanges := normalizeCount + fixesApplied
	if totalChanges == 0 {
		p.debugLog("  No sanitization needed (file is clean)")
	} else {
		p.debugLog("  File sanitized: %d total fix(es) applied", fixesApplied)
		if normalizeCount > 0 {
			p.debugLog("    - %d line ending normalization(s)", normalizeCount)
		}
	}

	sanitizedDir := filepath.Join(appDataDir, "mibs", "sanitized")
	if err := os.MkdirAll(sanitizedDir, 0o755); err != nil {
		return "", fmt.Errorf("create sanitized dir: %w", err)
	}

	sanitizedPath := filepath.Join(sanitizedDir, filepath.Base(originalPath))
	if err := os.WriteFile(sanitizedPath, sanitized, 0o644); err != nil {
		return "", fmt.Errorf("write sanitized copy: %w", err)
	}

	p.debugLog("  Sanitized copy saved: %s", sanitizedPath)
	return sanitizedPath, nil
}
