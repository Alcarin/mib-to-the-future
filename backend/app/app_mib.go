package app

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"mib-to-the-future/backend/mib"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

const bookmarkRootKey = "bookmarks"

// BookmarkFolderDTO rappresenta una cartella in formato serializzabile per il frontend.
type BookmarkFolderDTO struct {
	ID        int64     `json:"id"`
	Name      string    `json:"name"`
	Key       string    `json:"key"`
	ParentKey string    `json:"parentKey"`
	CreatedAt time.Time `json:"createdAt"`
}

// ModuleDetails rappresenta le informazioni aggregate per un modulo MIB specifico.
type ModuleDetails struct {
	Module         string          `json:"module"`
	Tree           []*mib.Node     `json:"tree"`
	Stats          mib.ModuleStats `json:"stats"`
	MissingImports []string        `json:"missingImports"`
}

func folderKeyFromID(id int64) string {
	return mib.BookmarkFolderKeyPrefix + strconv.FormatInt(id, 10)
}

func parseFolderKey(key string) (*int64, error) {
	if key == "" || key == bookmarkRootKey {
		return nil, nil
	}
	if !strings.HasPrefix(key, mib.BookmarkFolderKeyPrefix) {
		return nil, fmt.Errorf("invalid folder key: %s", key)
	}
	raw := strings.TrimPrefix(key, mib.BookmarkFolderKeyPrefix)
	value, err := strconv.ParseInt(raw, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid folder key: %s", key)
	}
	return &value, nil
}

// LoadMIBFile apre una finestra di dialogo per permettere all'utente di selezionare uno o più file MIB.
// Ogni file selezionato viene parsificato e caricato nel database MIB.
// Ritorna i nomi dei moduli MIB caricati in caso di successo, o un errore.
func (a *App) LoadMIBFile() ([]string, error) {
	if a.mibDB == nil {
		return nil, a.mibNotInitializedErr()
	}

	// Apri file dialog
	filePaths, err := runtime.OpenMultipleFilesDialog(a.ctx, runtime.OpenDialogOptions{
		Title: "Select MIB File",
		Filters: []runtime.FileFilter{
			{DisplayName: "MIB Files (*.mib, *.txt)", Pattern: "*.mib;*.txt"},
			{DisplayName: "All Files", Pattern: "*.*"},
		},
	})

	if err != nil {
		return nil, err
	}

	if len(filePaths) == 0 {
		return nil, fmt.Errorf("no file selected")
	}

	// Parsifica e carica MIB
	parser := mib.NewParser(a.mibDB)

	configDir, err := os.UserConfigDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get user config dir: %v", err)
	}
	dataDir := filepath.Join(configDir, "MIB to the Future")

	moduleNames := make([]string, 0, len(filePaths))
	for _, filePath := range filePaths {
		moduleName, err := parser.LoadMIBFile(filePath, dataDir)
		if err != nil {
			return nil, fmt.Errorf("failed to load MIB %s: %v", filepath.Base(filePath), err)
		}

		runtime.LogInfo(a.ctx, fmt.Sprintf("Loaded MIB module: %s", moduleName))
		moduleNames = append(moduleNames, moduleName)
	}

	return moduleNames, nil
}

// GetMIBTree recupera e restituisce l'intero albero MIB gerarchico dal database.
// Include un nodo root "Bookmarks" come primo elemento se esistono bookmark salvati.
// Utile per visualizzare l'intera struttura MIB nel frontend.
// Ritorna una slice di nodi radice dell'albero in caso di successo, o un errore.
func (a *App) GetMIBTree() ([]*mib.Node, error) {
	if a.mibDB == nil {
		return nil, a.mibNotInitializedErr()
	}

	tree, err := a.mibDB.GetTree()
	if err != nil {
		return nil, fmt.Errorf("failed to get MIB tree: %v", err)
	}

	// Recupera la struttura gerarchica dei bookmark
	hierarchy, err := a.mibDB.GetBookmarkHierarchy()
	if err != nil {
		runtime.LogError(a.ctx, fmt.Sprintf("Failed to load bookmarks: %v", err))
		hierarchy = nil
	}

	var bookmarkChildren []*mib.Node
	if hierarchy != nil {
		bookmarkChildren = a.buildBookmarkChildren(hierarchy, bookmarkRootKey)
	} else {
		bookmarkChildren = []*mib.Node{}
	}

	bookmarkRoot := &mib.Node{
		OID:      "bookmarks",
		Name:     "Bookmarks",
		Type:     "bookmark-root",
		Children: bookmarkChildren,
	}

	// Inserisci il nodo bookmarks come primo elemento dell'albero
	result := make([]*mib.Node, 0, len(tree)+1)
	result = append(result, bookmarkRoot)
	result = append(result, tree...)

	return result, nil
}

func (a *App) buildBookmarkChildren(folder *mib.BookmarkFolder, parentKey string) []*mib.Node {
	if folder == nil {
		return nil
	}

	nodes := make([]*mib.Node, 0, len(folder.Children)+len(folder.Bookmarks))

	for _, subFolder := range folder.Children {
		folderKey := folderKeyFromID(subFolder.ID)
		child := &mib.Node{
			OID:       folderKey,
			Name:      subFolder.Name,
			ParentOID: parentKey,
			Type:      "bookmark-folder",
		}
		child.Children = a.buildBookmarkChildren(subFolder, folderKey)
		nodes = append(nodes, child)
	}

	for _, entry := range folder.Bookmarks {
		original, err := a.mibDB.GetNode(entry.OID)
		if err != nil {
			runtime.LogWarning(a.ctx, fmt.Sprintf("Bookmark OID %s not found in MIB database", entry.OID))
			continue
		}

		bookmarkType := "bookmark"
		if original.Type != "" && original.Type != "scalar" {
			bookmarkType = "bookmark-" + original.Type
		}

		bookmarkNode := &mib.Node{
			ID:          original.ID,
			OID:         entry.OID,
			Name:        original.Name,
			ParentOID:   parentKey,
			Type:        bookmarkType,
			Syntax:      original.Syntax,
			Access:      original.Access,
			Status:      original.Status,
			Description: original.Description,
			Module:      original.Module,
			Children:    nil,
		}
		nodes = append(nodes, bookmarkNode)
	}

	return nodes
}

// GetMIBNode recupera un singolo nodo MIB dal database usando il suo OID.
// Parametri:
//   - oid: l'Object Identifier del nodo da recuperare.
//
// Ritorna un puntatore al nodo MIB se trovato, altrimenti un errore.
func (a *App) GetMIBNode(oid string) (*mib.Node, error) {
	if a.mibDB == nil {
		return nil, a.mibNotInitializedErr()
	}

	node, err := a.mibDB.GetNode(oid)
	if err != nil {
		return nil, fmt.Errorf("node not found: %v", err)
	}

	return node, nil
}

// SearchMIBNodes cerca nodi nel database MIB che corrispondono a una query.
// La ricerca viene effettuata sia sul nome del nodo che sull'OID.
// Parametri:
//   - query: la stringa di testo da cercare.
//
// Ritorna una slice di nodi MIB che corrispondono alla ricerca, o un errore.
func (a *App) SearchMIBNodes(query string) ([]*mib.Node, error) {
	if a.mibDB == nil {
		return nil, a.mibNotInitializedErr()
	}

	nodes, err := a.mibDB.SearchNodes(query)
	if err != nil {
		return nil, fmt.Errorf("search failed: %v", err)
	}

	return nodes, nil
}

// ListMIBModules restituisce l'elenco dei moduli MIB caricati con le statistiche principali.
func (a *App) ListMIBModules() ([]mib.ModuleSummary, error) {
	if a.mibDB == nil {
		return nil, a.mibNotInitializedErr()
	}

	modules, err := a.mibDB.ListModules()
	if err != nil {
		return nil, fmt.Errorf("failed to list modules: %v", err)
	}

	return modules, nil
}

// DeleteMIBModule rimuove un modulo MIB e tutti i suoi nodi associati dal database.
// Parametri:
//   - moduleName: il nome del modulo MIB da eliminare.
//
// Ritorna un errore se l'operazione fallisce.
func (a *App) DeleteMIBModule(moduleName string) error {
	if a.mibDB == nil {
		return a.mibNotInitializedErr()
	}

	err := a.mibDB.DeleteModule(moduleName)
	if err != nil {
		return fmt.Errorf("failed to delete module: %v", err)
	}

	runtime.LogInfo(a.ctx, fmt.Sprintf("Deleted MIB module: %s", moduleName))

	return nil
}

// GetMIBStats calcola e restituisce statistiche sul database MIB.
// Le statistiche includono il numero totale di moduli, nodi, etc.
// Ritorna una mappa con le statistiche o un errore.
func (a *App) GetMIBStats() (map[string]int, error) {
	if a.mibDB == nil {
		return nil, a.mibNotInitializedErr()
	}

	stats, err := a.mibDB.GetStats()
	if err != nil {
		return nil, fmt.Errorf("failed to get stats: %v", err)
	}

	return stats, nil
}

// GetMIBModuleDetails restituisce l'albero e le statistiche relative a un modulo specifico.
func (a *App) GetMIBModuleDetails(moduleName string) (*ModuleDetails, error) {
	if a.mibDB == nil {
		return nil, a.mibNotInitializedErr()
	}
	moduleName = strings.TrimSpace(moduleName)
	if moduleName == "" {
		return nil, fmt.Errorf("module name is empty")
	}

	summary, err := a.mibDB.GetModuleSummary(moduleName)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve module summary: %v", err)
	}

	tree, err := a.mibDB.GetModuleTree(moduleName)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve module tree: %v", err)
	}

	stats := mib.ModuleStats{
		NodeCount:    summary.NodeCount,
		ScalarCount:  summary.ScalarCount,
		TableCount:   summary.TableCount,
		ColumnCount:  summary.ColumnCount,
		TypeCount:    summary.TypeCount,
		SkippedNodes: summary.SkippedNodes,
		MissingCount: len(summary.MissingImports),
	}

	return &ModuleDetails{
		Module:         summary.Name,
		Tree:           tree,
		Stats:          stats,
		MissingImports: summary.MissingImports,
	}, nil
}

// ExportMIBTree esporta l'intero albero MIB in formato JSON.
// Se l'utente seleziona un percorso, il file JSON viene salvato su disco.
// Ritorna la stringa JSON dell'albero e un errore se il salvataggio fallisce.
func (a *App) ExportMIBTree() (string, error) {
	if a.mibDB == nil {
		return "", a.mibNotInitializedErr()
	}

	jsonData, err := a.mibDB.ExportTree()
	if err != nil {
		return "", fmt.Errorf("failed to export tree: %v", err)
	}

	// Salva in file
	filePath, err := runtime.SaveFileDialog(a.ctx, runtime.SaveDialogOptions{
		Title:           "Export MIB Tree",
		DefaultFilename: "mib-tree.json",
		Filters: []runtime.FileFilter{
			{DisplayName: "JSON Files", Pattern: "*.json"},
		},
	})

	if err != nil || filePath == "" {
		return jsonData, nil // Ritorna comunque i dati
	}

	// Scrivi file
	if err := os.WriteFile(filePath, []byte(jsonData), 0644); err != nil {
		return "", fmt.Errorf("failed to write file: %v", err)
	}

	runtime.LogInfo(a.ctx, fmt.Sprintf("Exported MIB tree to: %s", filePath))

	return jsonData, nil
}

// SaveCSVFile apre un dialogo di salvataggio e scrive su disco il contenuto CSV fornito.
// Restituisce true se il file è stato salvato, false se l'utente annulla l'operazione.
func (a *App) SaveCSVFile(defaultFilename string, csvContent string) (bool, error) {
	filename := strings.TrimSpace(defaultFilename)
	if filename == "" {
		filename = fmt.Sprintf("export-%d.csv", time.Now().Unix())
	}
	if !strings.HasSuffix(strings.ToLower(filename), ".csv") {
		filename += ".csv"
	}

	filePath, err := runtime.SaveFileDialog(a.ctx, runtime.SaveDialogOptions{
		Title:           "Salva CSV",
		DefaultFilename: filename,
		Filters: []runtime.FileFilter{
			{DisplayName: "File CSV", Pattern: "*.csv"},
			{DisplayName: "Tutti i file", Pattern: "*"},
		},
	})
	if err != nil {
		return false, fmt.Errorf("errore durante l'apertura del dialogo di salvataggio: %w", err)
	}
	if filePath == "" {
		return false, nil
	}

	if err := os.WriteFile(filePath, []byte(csvContent), 0644); err != nil {
		return false, fmt.Errorf("impossibile scrivere il file CSV: %w", err)
	}

	runtime.LogInfo(a.ctx, fmt.Sprintf("CSV salvato in: %s", filePath))
	return true, nil
}

// GetMIBNodeByName cerca un nodo MIB nel database usando il suo nome.
// Parametri:
//   - name: il nome del nodo da cercare.
//
// Ritorna un puntatore al nodo MIB se trovato, altrimenti un errore.
func (a *App) GetMIBNodeByName(name string) (*mib.Node, error) {
	if a.mibDB == nil {
		return nil, a.mibNotInitializedErr()
	}

	node, err := a.mibDB.GetNodeByName(name)
	if err != nil {
		return nil, fmt.Errorf("node not found: %v", err)
	}

	return node, nil
}

// GetMIBNodeAncestors restituisce la catena di antenati di un nodo MIB a partire dall'OID fornito.
func (a *App) GetMIBNodeAncestors(oid string) ([]*mib.Node, error) {
	if a.mibDB == nil {
		return nil, a.mibNotInitializedErr()
	}
	if oid == "" {
		return nil, fmt.Errorf("invalid OID")
	}

	nodes, err := a.mibDB.GetNodeAncestors(oid)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve ancestors: %v", err)
	}

	return nodes, nil
}

// ReloadMIBDatabase chiude e ricarica il database MIB dalla sua posizione su disco.
// Funzione utile principalmente per scopi di debug.
// Ritorna un errore se il ricaricamento fallisce.
func (a *App) ReloadMIBDatabase() error {
	if a.mibDB != nil {
		a.mibDB.Close()
		a.mibDB = nil
	}

	configDir, err := os.UserConfigDir()
	if err != nil {
		a.mibInitErr = fmt.Errorf("failed to resolve user config dir: %w", err)
		return a.mibInitErr
	}

	dataDir := filepath.Join(configDir, "MIB to the Future")

	db, err := mib.NewDatabase(dataDir)
	if err != nil {
		a.mibInitErr = fmt.Errorf("failed to reload database from %s: %w", dataDir, err)
		return a.mibInitErr
	}

	a.mibDB = db
	a.mibInitErr = nil

	runtime.LogInfo(a.ctx, fmt.Sprintf("MIB database reloaded from: %s", dataDir))

	return nil
}

// AddBookmark aggiunge un OID alla lista dei bookmark in una cartella facoltativa.
// Parametri:
//   - oid: l'Object Identifier da aggiungere.
//   - folderKey: la cartella di destinazione (usare "bookmarks" per la root).
//
// Ritorna un errore se l'operazione fallisce.
func (a *App) AddBookmark(oid string, folderKey string) error {
	if a.mibDB == nil {
		return a.mibNotInitializedErr()
	}
	trimmedOID := strings.TrimSpace(oid)
	if trimmedOID == "" {
		return fmt.Errorf("OID is required")
	}

	folderID, err := parseFolderKey(strings.TrimSpace(folderKey))
	if err != nil {
		return err
	}

	if err := a.mibDB.AddBookmark(trimmedOID, folderID); err != nil {
		return fmt.Errorf("failed to add bookmark: %w", err)
	}

	target := bookmarkRootKey
	if folderID != nil {
		target = folderKeyFromID(*folderID)
	}

	runtime.LogInfo(a.ctx, fmt.Sprintf("Added bookmark: %s (folder=%s)", trimmedOID, target))
	return nil
}

// MoveBookmark sposta un bookmark esistente in una nuova cartella.
// Parametri:
//   - oid: l'OID del bookmark da spostare.
//   - folderKey: la chiave della cartella di destinazione (usare "bookmarks" per la root).
func (a *App) MoveBookmark(oid string, folderKey string) error {
	if a.mibDB == nil {
		return a.mibNotInitializedErr()
	}
	trimmedOID := strings.TrimSpace(oid)
	if trimmedOID == "" {
		return fmt.Errorf("OID is required")
	}

	folderID, err := parseFolderKey(strings.TrimSpace(folderKey))
	if err != nil {
		return err
	}

	if err := a.mibDB.MoveBookmark(trimmedOID, folderID); err != nil {
		return fmt.Errorf("failed to move bookmark: %w", err)
	}

	target := bookmarkRootKey
	if folderID != nil {
		target = folderKeyFromID(*folderID)
	}

	runtime.LogInfo(a.ctx, fmt.Sprintf("Moved bookmark: %s -> %s", trimmedOID, target))
	return nil
}

// RemoveBookmark rimuove un OID dalla lista dei bookmark.
// Parametri:
//   - oid: l'Object Identifier da rimuovere dai bookmark.
//
// Ritorna un errore se l'operazione fallisce.
func (a *App) RemoveBookmark(oid string) error {
	if a.mibDB == nil {
		return a.mibNotInitializedErr()
	}
	trimmedOID := strings.TrimSpace(oid)
	if trimmedOID == "" {
		return fmt.Errorf("OID is required")
	}

	err := a.mibDB.RemoveBookmark(trimmedOID)
	if err != nil {
		return fmt.Errorf("failed to remove bookmark: %w", err)
	}

	runtime.LogInfo(a.ctx, fmt.Sprintf("Removed bookmark: %s", trimmedOID))
	return nil
}

// CreateBookmarkFolder crea una nuova cartella per i bookmark.
// Parametri:
//   - name: nome della cartella.
//   - parentKey: chiave della cartella padre ("bookmarks" per la root).
func (a *App) CreateBookmarkFolder(name string, parentKey string) (*BookmarkFolderDTO, error) {
	if a.mibDB == nil {
		return nil, a.mibNotInitializedErr()
	}

	parentID, err := parseFolderKey(strings.TrimSpace(parentKey))
	if err != nil {
		return nil, err
	}

	folder, err := a.mibDB.CreateBookmarkFolder(name, parentID)
	if err != nil {
		return nil, err
	}

	parentKeyValue := bookmarkRootKey
	if folder.ParentID != nil {
		parentKeyValue = folderKeyFromID(*folder.ParentID)
	}

	dto := &BookmarkFolderDTO{
		ID:        folder.ID,
		Name:      folder.Name,
		Key:       folderKeyFromID(folder.ID),
		ParentKey: parentKeyValue,
		CreatedAt: folder.CreatedAt,
	}

	runtime.LogInfo(a.ctx, fmt.Sprintf("Created bookmark folder: %s (parent=%s)", folder.Name, parentKeyValue))
	return dto, nil
}

// RenameBookmarkFolder rinomina una cartella esistente.
// Parametri:
//   - folderKey: chiave della cartella da rinominare.
//   - name: nuovo nome.
func (a *App) RenameBookmarkFolder(folderKey string, name string) error {
	if a.mibDB == nil {
		return a.mibNotInitializedErr()
	}

	folderID, err := parseFolderKey(strings.TrimSpace(folderKey))
	if err != nil {
		return err
	}
	if folderID == nil {
		return fmt.Errorf("cannot rename the root bookmarks folder")
	}

	if err := a.mibDB.RenameBookmarkFolder(*folderID, name); err != nil {
		return err
	}

	runtime.LogInfo(a.ctx, fmt.Sprintf("Renamed bookmark folder %s to %s", folderKey, strings.TrimSpace(name)))
	return nil
}

// DeleteBookmarkFolder elimina una cartella di bookmark (con contenuti cascata).
// Parametri:
//   - folderKey: chiave della cartella da eliminare.
func (a *App) DeleteBookmarkFolder(folderKey string) error {
	if a.mibDB == nil {
		return a.mibNotInitializedErr()
	}

	folderID, err := parseFolderKey(strings.TrimSpace(folderKey))
	if err != nil {
		return err
	}
	if folderID == nil {
		return fmt.Errorf("cannot delete the root bookmarks folder")
	}

	if err := a.mibDB.DeleteBookmarkFolder(*folderID); err != nil {
		return err
	}

	runtime.LogInfo(a.ctx, fmt.Sprintf("Deleted bookmark folder %s", folderKey))
	return nil
}

// MoveBookmarkFolder cambia il parent di una cartella.
// Parametri:
//   - folderKey: cartella da spostare.
//   - parentKey: nuovo parent (usare "bookmarks" per la root).
func (a *App) MoveBookmarkFolder(folderKey string, parentKey string) error {
	if a.mibDB == nil {
		return a.mibNotInitializedErr()
	}

	folderID, err := parseFolderKey(strings.TrimSpace(folderKey))
	if err != nil {
		return err
	}
	if folderID == nil {
		return fmt.Errorf("cannot move the root bookmarks folder")
	}

	parentID, err := parseFolderKey(strings.TrimSpace(parentKey))
	if err != nil {
		return err
	}

	if err := a.mibDB.MoveBookmarkFolder(*folderID, parentID); err != nil {
		return err
	}

	target := bookmarkRootKey
	if parentID != nil {
		target = folderKeyFromID(*parentID)
	}
	runtime.LogInfo(a.ctx, fmt.Sprintf("Moved bookmark folder %s -> %s", folderKey, target))
	return nil
}
