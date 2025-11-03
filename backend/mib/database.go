package mib

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	_ "modernc.org/sqlite"
)

// Node rappresenta un nodo MIB
type Node struct {
	ID          int64   `json:"id"`
	OID         string  `json:"oid"`
	Name        string  `json:"name"`
	ParentOID   string  `json:"parentOid"`
	Type        string  `json:"type"`   // node, scalar, table, column
	Syntax      string  `json:"syntax"` // INTEGER, OCTET STRING, etc.
	Access      string  `json:"access"` // read-only, read-write, etc.
	Status      string  `json:"status"` // current, deprecated, obsolete
	Description string  `json:"description"`
	Module      string  `json:"module"` // Nome modulo MIB (es. SNMPv2-MIB)
	Children    []*Node `json:"children,omitempty"`
}

// ModuleStats rappresenta conteggi aggregati per un modulo MIB.
type ModuleStats struct {
	NodeCount    int `json:"nodeCount"`
	ScalarCount  int `json:"scalarCount"`
	TableCount   int `json:"tableCount"`
	ColumnCount  int `json:"columnCount"`
	TypeCount    int `json:"typeCount"`
	SkippedNodes int `json:"skippedNodes"`
	MissingCount int `json:"missingCount"`
}

// ModuleSummary descrive i metadati principali di un modulo salvato nel database.
type ModuleSummary struct {
	Name           string   `json:"name"`
	FilePath       string   `json:"filePath"`
	NodeCount      int      `json:"nodeCount"`
	ScalarCount    int      `json:"scalarCount"`
	TableCount     int      `json:"tableCount"`
	ColumnCount    int      `json:"columnCount"`
	TypeCount      int      `json:"typeCount"`
	SkippedNodes   int      `json:"skippedNodes"`
	MissingImports []string `json:"missingImports"`
}

func decodeMissingImports(raw string) []string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}

	var imports []string
	if err := json.Unmarshal([]byte(raw), &imports); err == nil {
		return imports
	}

	parts := strings.Split(raw, ",")
	for _, part := range parts {
		value := strings.TrimSpace(part)
		if value != "" {
			imports = append(imports, value)
		}
	}
	return imports
}

func encodeMissingImports(values []string) string {
	if len(values) == 0 {
		return ""
	}
	normalized := make([]string, 0, len(values))
	for _, v := range values {
		v = strings.TrimSpace(v)
		if v != "" {
			normalized = append(normalized, v)
		}
	}
	if len(normalized) == 0 {
		return ""
	}
	data, err := json.Marshal(normalized)
	if err != nil {
		return strings.Join(normalized, ",")
	}
	return string(data)
}

// Database gestisce lo storage SQLite dei MIB
type Database struct {
	db   *sql.DB
	path string
}

// NewDatabase crea una nuova istanza del database MIB
func NewDatabase(dataDir string) (*Database, error) {
	// Crea directory se non esiste
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create data directory %q: %w", dataDir, err)
	}

	dbPath := filepath.Join(dataDir, "mibs.db")

	// Apri database senza parametri extra nel percorso: su Windows caratteri come `?`
	// rendono il nome file invalido. Abilitiamo le foreign key con un PRAGMA esplicito.
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database %q: %w", dbPath, err)
	}

	if _, err := db.Exec("PRAGMA foreign_keys = ON"); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to enable foreign keys for %q: %w", dbPath, err)
	}

	mibDB := &Database{
		db:   db,
		path: dbPath,
	}

	// Inizializza schema
	if err := mibDB.initSchema(); err != nil {
		db.Close()
		return nil, err
	}

	return mibDB, nil
}

// initSchema crea le tabelle se non esistono
func (d *Database) initSchema() error {
	schema := `
	CREATE TABLE IF NOT EXISTS mib_modules (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT UNIQUE NOT NULL,
		file_path TEXT,
		loaded_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		node_count INTEGER NOT NULL DEFAULT 0,
		scalar_count INTEGER NOT NULL DEFAULT 0,
		table_count INTEGER NOT NULL DEFAULT 0,
		column_count INTEGER NOT NULL DEFAULT 0,
		type_count INTEGER NOT NULL DEFAULT 0,
		skipped_nodes INTEGER NOT NULL DEFAULT 0,
		missing_imports TEXT NOT NULL DEFAULT ''
	);

	CREATE TABLE IF NOT EXISTS mib_nodes (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		oid TEXT UNIQUE NOT NULL,
		name TEXT NOT NULL,
		parent_oid TEXT,
		type TEXT,
		syntax TEXT,
		access TEXT,
		status TEXT,
		description TEXT,
		module_id INTEGER,
		FOREIGN KEY (module_id) REFERENCES mib_modules(id) ON DELETE CASCADE
	);

	CREATE INDEX IF NOT EXISTS idx_oid ON mib_nodes(oid);
	CREATE INDEX IF NOT EXISTS idx_name ON mib_nodes(name);
	CREATE INDEX IF NOT EXISTS idx_parent_oid ON mib_nodes(parent_oid);
	CREATE INDEX IF NOT EXISTS idx_module_id ON mib_nodes(module_id);

	-- Tabella per metadata e configurazioni
	CREATE TABLE IF NOT EXISTS app_metadata (
		key TEXT PRIMARY KEY,
		value TEXT
	);

	-- Tabella per la persistenza degli host SNMP configurati
	CREATE TABLE IF NOT EXISTS host_configs (
		address TEXT PRIMARY KEY,
		port INTEGER NOT NULL DEFAULT 161,
		community TEXT NOT NULL DEFAULT 'public',
		write_community TEXT NOT NULL DEFAULT 'public',
		version TEXT NOT NULL DEFAULT 'v2c',
		last_used_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
		created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
		context_name TEXT NOT NULL DEFAULT '',
		security_level TEXT NOT NULL DEFAULT '',
		security_username TEXT NOT NULL DEFAULT '',
		auth_protocol TEXT NOT NULL DEFAULT '',
		auth_password TEXT NOT NULL DEFAULT '',
		priv_protocol TEXT NOT NULL DEFAULT '',
		priv_password TEXT NOT NULL DEFAULT ''
	);

	CREATE INDEX IF NOT EXISTS idx_host_last_used ON host_configs(last_used_at DESC);
	`

	_, err := d.db.Exec(schema)
	if err != nil {
		return fmt.Errorf("failed to create schema for %q: %w", d.path, err)
	}

	if err := d.ensureModuleExtendedSchema(); err != nil {
		return err
	}

	if err := d.ensureBookmarkSchema(); err != nil {
		return err
	}

	return nil
}

// ensureModuleExtendedSchema aggiunge le colonne di metadati ai moduli se mancanti.
func (d *Database) ensureModuleExtendedSchema() error {
	if d == nil || d.db == nil {
		return fmt.Errorf("database not initialized")
	}

	alterStatements := []struct {
		query string
		err   string
	}{
		{
			query: `ALTER TABLE mib_modules ADD COLUMN node_count INTEGER NOT NULL DEFAULT 0`,
			err:   "failed to add node_count column to mib_modules",
		},
		{
			query: `ALTER TABLE mib_modules ADD COLUMN scalar_count INTEGER NOT NULL DEFAULT 0`,
			err:   "failed to add scalar_count column to mib_modules",
		},
		{
			query: `ALTER TABLE mib_modules ADD COLUMN table_count INTEGER NOT NULL DEFAULT 0`,
			err:   "failed to add table_count column to mib_modules",
		},
		{
			query: `ALTER TABLE mib_modules ADD COLUMN column_count INTEGER NOT NULL DEFAULT 0`,
			err:   "failed to add column_count column to mib_modules",
		},
		{
			query: `ALTER TABLE mib_modules ADD COLUMN type_count INTEGER NOT NULL DEFAULT 0`,
			err:   "failed to add type_count column to mib_modules",
		},
		{
			query: `ALTER TABLE mib_modules ADD COLUMN skipped_nodes INTEGER NOT NULL DEFAULT 0`,
			err:   "failed to add skipped_nodes column to mib_modules",
		},
		{
			query: `ALTER TABLE mib_modules ADD COLUMN missing_imports TEXT NOT NULL DEFAULT ''`,
			err:   "failed to add missing_imports column to mib_modules",
		},
	}

	for _, stmt := range alterStatements {
		if _, err := d.db.Exec(stmt.query); err != nil {
			if !strings.Contains(strings.ToLower(err.Error()), "duplicate column name") {
				return fmt.Errorf("%s: %w", stmt.err, err)
			}
		}
	}

	return nil
}

// ensureBookmarkSchema crea o aggiorna lo schema relativo ai bookmark.
func (d *Database) ensureBookmarkSchema() error {
	if d == nil || d.db == nil {
		return fmt.Errorf("database not initialized")
	}

	statements := []struct {
		query string
		err   string
	}{
		{
			query: `CREATE TABLE IF NOT EXISTS bookmark_folders (
				id INTEGER PRIMARY KEY AUTOINCREMENT,
				name TEXT NOT NULL,
				parent_folder_id INTEGER,
				created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
				FOREIGN KEY (parent_folder_id) REFERENCES bookmark_folders(id) ON DELETE CASCADE
			)`,
			err: "failed to ensure bookmark_folders table",
		},
		{
			query: `CREATE INDEX IF NOT EXISTS idx_bookmark_folders_parent ON bookmark_folders(parent_folder_id)`,
			err:   "failed to ensure bookmark_folders parent index",
		},
		{
			query: `CREATE TABLE IF NOT EXISTS bookmarks (
				oid TEXT PRIMARY KEY,
				folder_id INTEGER REFERENCES bookmark_folders(id) ON DELETE CASCADE,
				created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
			)`,
			err: "failed to ensure bookmarks table",
		},
	}

	for _, stmt := range statements {
		if _, execErr := d.db.Exec(stmt.query); execErr != nil {
			return fmt.Errorf("%s: %w", stmt.err, execErr)
		}
	}

	if _, err := d.db.Exec(`ALTER TABLE bookmarks ADD COLUMN folder_id INTEGER REFERENCES bookmark_folders(id) ON DELETE CASCADE`); err != nil {
		if !strings.Contains(strings.ToLower(err.Error()), "duplicate column name") {
			return fmt.Errorf("failed to add folder_id column to bookmarks: %w", err)
		}
	}

	if _, err := d.db.Exec(`CREATE INDEX IF NOT EXISTS idx_bookmarks_folder ON bookmarks(folder_id)`); err != nil {
		return fmt.Errorf("failed to ensure bookmarks folder index: %w", err)
	}

	return nil
}

// EnsureHostConfigSchema verifica che la tabella host_configs disponga delle colonne richieste per SNMPv3.
func (d *Database) EnsureHostConfigSchema() error {
	if d == nil || d.db == nil {
		return fmt.Errorf("database not initialized")
	}

	columns := []struct {
		name string
		def  string
	}{
		{"write_community", "TEXT NOT NULL DEFAULT 'public'"},
		{"context_name", "TEXT NOT NULL DEFAULT ''"},
		{"security_level", "TEXT NOT NULL DEFAULT ''"},
		{"security_username", "TEXT NOT NULL DEFAULT ''"},
		{"auth_protocol", "TEXT NOT NULL DEFAULT ''"},
		{"auth_password", "TEXT NOT NULL DEFAULT ''"},
		{"priv_protocol", "TEXT NOT NULL DEFAULT ''"},
		{"priv_password", "TEXT NOT NULL DEFAULT ''"},
	}

	for _, col := range columns {
		query := fmt.Sprintf("ALTER TABLE host_configs ADD COLUMN %s %s", col.name, col.def)
		if _, err := d.db.Exec(query); err != nil {
			if !strings.Contains(strings.ToLower(err.Error()), "duplicate column name") {
				return fmt.Errorf("failed to add column %s: %w", col.name, err)
			}
		}
	}

	if _, err := d.db.Exec("UPDATE host_configs SET write_community = community"); err != nil {
		return fmt.Errorf("failed to backfill write community column: %w", err)
	}

	return nil
}

// IsNew controlla se il database è stato appena creato (controllando se ci sono moduli).
func (d *Database) IsNew() (bool, error) {
	var count int
	err := d.db.QueryRow("SELECT COUNT(*) FROM mib_modules").Scan(&count)
	if err != nil {
		return false, fmt.Errorf("failed to count modules: %v", err)
	}
	return count == 0, nil
}

// Close chiude la connessione al database
func (d *Database) Close() error {
	return d.db.Close()
}

// SaveModule salva informazioni sul modulo MIB
func (d *Database) SaveModule(name, filePath string) (int64, error) {
	_, err := d.db.Exec(
		"INSERT INTO mib_modules (name, file_path) VALUES (?, ?) ON CONFLICT(name) DO UPDATE SET file_path = excluded.file_path",
		name, filePath,
	)
	if err != nil {
		return 0, err
	}

	return d.GetModuleID(name)
}

// GetModuleID recupera l'ID del modulo
func (d *Database) GetModuleID(name string) (int64, error) {
	var id int64
	err := d.db.QueryRow("SELECT id FROM mib_modules WHERE name = ?", name).Scan(&id)
	return id, err
}

// ModuleExists verifica se un modulo è presente nel database.
func (d *Database) ModuleExists(name string) (bool, error) {
	var exists bool
	err := d.db.QueryRow("SELECT EXISTS(SELECT 1 FROM mib_modules WHERE name = ?)", name).Scan(&exists)
	if err != nil {
		return false, err
	}
	return exists, nil
}

// SaveNode salva un nodo MIB nel database
func (d *Database) SaveNode(node *Node, moduleID int64) error {
	parentOID := sql.NullString{}
	if node.ParentOID != "" {
		parentOID.String = node.ParentOID
		parentOID.Valid = true
	}

	_, err := d.db.Exec(`
		INSERT INTO mib_nodes (oid, name, parent_oid, type, syntax, access, status, description, module_id)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(oid) DO UPDATE SET
			name = excluded.name,
			parent_oid = excluded.parent_oid,
			type = excluded.type,
			syntax = excluded.syntax,
			access = excluded.access,
			status = excluded.status,
			description = excluded.description,
			module_id = excluded.module_id
	`, node.OID, node.Name, parentOID, node.Type, node.Syntax, node.Access, node.Status, node.Description, moduleID)

	return err
}

// SaveNodes salva multipli nodi in una transazione
func (d *Database) SaveNodes(nodes []*Node, moduleID int64) error {
	tx, err := d.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	stmt, err := tx.Prepare(`
		INSERT INTO mib_nodes (oid, name, parent_oid, type, syntax, access, status, description, module_id)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(oid) DO UPDATE SET
			name = CASE WHEN excluded.name <> '' THEN excluded.name ELSE name END,
			parent_oid = CASE WHEN excluded.parent_oid <> '' THEN excluded.parent_oid ELSE parent_oid END,
			type = CASE WHEN excluded.type <> '' THEN excluded.type ELSE type END,
			syntax = CASE WHEN excluded.syntax <> '' THEN excluded.syntax ELSE syntax END,
			access = CASE WHEN excluded.access <> '' THEN excluded.access ELSE access END,
			status = CASE WHEN excluded.status <> '' THEN excluded.status ELSE status END,
			description = CASE WHEN excluded.description <> '' THEN excluded.description ELSE description END,
			module_id = excluded.module_id
	`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	moduleCache := make(map[string]int64)

	for _, node := range nodes {
		parentOID := sql.NullString{}
		if node.ParentOID != "" {
			parentOID.String = node.ParentOID
			parentOID.Valid = true
		}

		targetModuleID := moduleID
		if node.Module != "" {
			if cachedID, ok := moduleCache[node.Module]; ok {
				targetModuleID = cachedID
			} else {
				id, lookupErr := d.GetModuleID(node.Module)
				if lookupErr != nil {
					newID, createErr := d.SaveModule(node.Module, "")
					if createErr != nil {
						id = moduleID
					} else {
						id = newID
					}
				}
				if id != 0 {
					moduleCache[node.Module] = id
					targetModuleID = id
				}
			}
		}

		_, err = stmt.Exec(
			node.OID, node.Name, parentOID, node.Type,
			node.Syntax, node.Access, node.Status, node.Description, targetModuleID,
		)
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}

// GetNode recupera un nodo per OID
func (d *Database) GetNode(oid string) (*Node, error) {
	if oid == "" {
		return nil, fmt.Errorf("oid is empty")
	}

	variants := []string{}
	seen := make(map[string]struct{})

	addVariant := func(value string) {
		value = strings.TrimSpace(value)
		if value == "" {
			return
		}
		if _, ok := seen[value]; ok {
			return
		}
		variants = append(variants, value)
		seen[value] = struct{}{}
	}

	addVariant(oid)

	trimmed := strings.TrimPrefix(oid, ".")
	addVariant(trimmed)
	if trimmed != "" {
		addVariant("." + trimmed)
	}

	if strings.HasPrefix(oid, ".") {
		addVariant(strings.TrimPrefix(oid, "."))
	}

	baseVariants := make([]string, len(variants))
	copy(baseVariants, variants)
	for _, value := range baseVariants {
		if value == "" {
			continue
		}
		if strings.HasSuffix(value, ".0") {
			base := strings.TrimSuffix(value, ".0")
			addVariant(base)
			addVariant(strings.TrimPrefix(base, "."))
			trimmedBase := strings.TrimPrefix(base, ".")
			if trimmedBase != "" {
				addVariant("." + trimmedBase)
			}
		}
	}

	var lastErr error

	for _, candidate := range variants {
		node := &Node{}
		var parentOID, syntax, access, status, description, moduleName sql.NullString

		err := d.db.QueryRow(`
		SELECT n.id, n.oid, n.name, n.parent_oid, n.type, n.syntax, n.access, n.status, n.description, m.name
		FROM mib_nodes n
		LEFT JOIN mib_modules m ON n.module_id = m.id
		WHERE n.oid = ?
	`, candidate).Scan(
			&node.ID, &node.OID, &node.Name, &parentOID, &node.Type,
			&syntax, &access, &status, &description, &moduleName,
		)

		if err != nil {
			lastErr = err
			continue
		}

		if parentOID.Valid {
			node.ParentOID = parentOID.String
		}
		if syntax.Valid {
			node.Syntax = syntax.String
		}
		if access.Valid {
			node.Access = access.String
		}
		if status.Valid {
			node.Status = status.String
		}
		if description.Valid {
			node.Description = description.String
		}
		if moduleName.Valid {
			node.Module = moduleName.String
		}

		return node, nil
	}

	if lastErr != nil {
		return nil, lastErr
	}

	return nil, sql.ErrNoRows
}

// GetNodeByName recupera un nodo per nome
func (d *Database) GetNodeByName(name string) (*Node, error) {
	node := &Node{}
	var parentOID, syntax, access, status, description, moduleName sql.NullString

	err := d.db.QueryRow(`
		SELECT n.id, n.oid, n.name, n.parent_oid, n.type, n.syntax, n.access, n.status, n.description, m.name
		FROM mib_nodes n
		LEFT JOIN mib_modules m ON n.module_id = m.id
		WHERE n.name = ? LIMIT 1
	`, name).Scan(
		&node.ID, &node.OID, &node.Name, &parentOID, &node.Type,
		&syntax, &access, &status, &description, &moduleName,
	)

	if err != nil {
		return nil, err
	}

	if parentOID.Valid {
		node.ParentOID = parentOID.String
	}
	if syntax.Valid {
		node.Syntax = syntax.String
	}
	if access.Valid {
		node.Access = access.String
	}
	if status.Valid {
		node.Status = status.String
	}
	if description.Valid {
		node.Description = description.String
	}
	if moduleName.Valid {
		node.Module = moduleName.String
	}

	return node, nil
}

// GetNodeAncestors restituisce il nodo richiesto e tutti i suoi antenati fino alla radice.
func (d *Database) GetNodeAncestors(oid string) ([]*Node, error) {
	if oid == "" {
		return nil, fmt.Errorf("oid is empty")
	}

	node, err := d.GetNode(oid)
	if err != nil {
		return nil, err
	}

	var ancestors []*Node
	visited := make(map[string]struct{})
	current := node

	for current != nil {
		canonical := strings.TrimPrefix(current.OID, ".")
		if canonical == "" {
			canonical = current.OID
		}
		if _, seen := visited[canonical]; seen {
			break
		}
		visited[canonical] = struct{}{}
		ancestors = append(ancestors, current)

		parentOID := strings.TrimPrefix(current.ParentOID, ".")
		if parentOID == "" {
			break
		}

		parent, err := d.GetNode(parentOID)
		if err != nil {
			break
		}
		current = parent
	}

	return ancestors, nil
}

// GetChildren recupera i figli di un nodo
func (d *Database) GetChildren(parentOID string) ([]*Node, error) {
	rows, err := d.db.Query(`
		SELECT n.id, n.oid, n.name, n.parent_oid, n.type, n.syntax, n.access, n.status, n.description, m.name
		FROM mib_nodes n
		LEFT JOIN mib_modules m ON n.module_id = m.id
		WHERE n.parent_oid = ?
		ORDER BY oid
	`, parentOID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var nodes []*Node
	for rows.Next() {
		node := &Node{}
		var parentOID, syntax, access, status, description, moduleName sql.NullString

		err := rows.Scan(
			&node.ID, &node.OID, &node.Name, &parentOID, &node.Type,
			&syntax, &access, &status, &description, &moduleName,
		)
		if err != nil {
			return nil, err
		}

		if parentOID.Valid {
			node.ParentOID = parentOID.String
		}
		if syntax.Valid {
			node.Syntax = syntax.String
		}
		if access.Valid {
			node.Access = access.String
		}
		if status.Valid {
			node.Status = status.String
		}
		if description.Valid {
			node.Description = description.String
		}
		if moduleName.Valid {
			node.Module = moduleName.String
		}

		nodes = append(nodes, node)
	}

	return nodes, rows.Err()
}

// GetTree costruisce l'albero MIB completo
func (d *Database) GetTree() ([]*Node, error) {
	// Prendi tutti i nodi
	allNodes, err := d.getAllNodes()
	if err != nil {
		return nil, err
	}

	// Crea mappa per accesso veloce
	nodesMap := make(map[string]*Node)
	for _, node := range allNodes {
		nodesMap[node.OID] = node
		node.Children = []*Node{} // Inizializza children
	}

	// Costruisci gerarchia - solo nodi con nome
	var roots []*Node
	for _, node := range allNodes {
		if node.ParentOID == "" {
			roots = append(roots, node)
		} else {
			if parent, exists := nodesMap[node.ParentOID]; exists {
				parent.Children = append(parent.Children, node)
			} else {
				// Parent non esiste, aggiungi come root
				roots = append(roots, node)
			}
		}
	}

	sortTreeNodes(roots)

	return roots, nil
}

// sortTreeNodes ordina ricorsivamente i nodi in base all'OID usando un confronto numerico.
func sortTreeNodes(nodes []*Node) {
	if len(nodes) == 0 {
		return
	}

	sort.Slice(nodes, func(i, j int) bool {
		return CompareOIDs(nodes[i].OID, nodes[j].OID) < 0
	})

	for _, node := range nodes {
		if len(node.Children) > 0 {
			sortTreeNodes(node.Children)
		}
	}
}

// getAllNodes recupera tutti i nodi dal database
func (d *Database) getAllNodes() ([]*Node, error) {
	rows, err := d.db.Query(`
		SELECT n.id, n.oid, n.name, n.parent_oid, n.type, n.syntax, n.access, n.status, n.description, m.name
		FROM mib_nodes n
		LEFT JOIN mib_modules m ON n.module_id = m.id
		ORDER BY oid
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var nodes []*Node
	for rows.Next() {
		node := &Node{}
		var parentOID, syntax, access, status, description, moduleName sql.NullString

		err := rows.Scan(
			&node.ID, &node.OID, &node.Name, &parentOID, &node.Type,
			&syntax, &access, &status, &description, &moduleName,
		)
		if err != nil {
			return nil, err
		}

		if parentOID.Valid {
			node.ParentOID = parentOID.String
		}
		if syntax.Valid {
			node.Syntax = syntax.String
		}
		if access.Valid {
			node.Access = access.String
		}
		if status.Valid {
			node.Status = status.String
		}
		if description.Valid {
			node.Description = description.String
		}
		if moduleName.Valid {
			node.Module = moduleName.String
		}

		nodes = append(nodes, node)
	}

	return nodes, rows.Err()
}

// getRootNodes recupera i nodi senza parent
func (d *Database) getRootNodes() ([]*Node, error) {
	rows, err := d.db.Query(`
		SELECT id, oid, name, type, syntax, access, status, description
		FROM mib_nodes WHERE parent_oid IS NULL
		ORDER BY oid
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var nodes []*Node
	for rows.Next() {
		node := &Node{}
		var syntax, access, status, description sql.NullString
		err := rows.Scan(
			&node.ID, &node.OID, &node.Name, &node.Type,
			&syntax, &access, &status, &description,
		)
		if err != nil {
			return nil, err
		}
		if syntax.Valid {
			node.Syntax = syntax.String
		}
		if access.Valid {
			node.Access = access.String
		}
		if status.Valid {
			node.Status = status.String
		}
		if description.Valid {
			node.Description = description.String
		}
		nodes = append(nodes, node)
	}

	return nodes, rows.Err()
}

// SearchNodes cerca nodi per nome o OID
func (d *Database) SearchNodes(query string) ([]*Node, error) {
	rows, err := d.db.Query(`
		SELECT n.id, n.oid, n.name, n.parent_oid, n.type, n.syntax, n.access, n.status, n.description, m.name
		FROM mib_nodes n
		LEFT JOIN mib_modules m ON n.module_id = m.id
		WHERE name LIKE ? OR oid LIKE ?
		ORDER BY oid
		LIMIT 100
	`, "%"+query+"%", "%"+query+"%")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var nodes []*Node
	for rows.Next() {
		node := &Node{}
		var parentOID, syntax, access, status, description, moduleName sql.NullString

		err := rows.Scan(
			&node.ID, &node.OID, &node.Name, &parentOID, &node.Type,
			&syntax, &access, &status, &description, &moduleName,
		)
		if err != nil {
			return nil, err
		}

		if parentOID.Valid {
			node.ParentOID = parentOID.String
		}
		if syntax.Valid {
			node.Syntax = syntax.String
		}
		if access.Valid {
			node.Access = access.String
		}
		if status.Valid {
			node.Status = status.String
		}
		if description.Valid {
			node.Description = description.String
		}
		if moduleName.Valid {
			node.Module = moduleName.String
		}

		nodes = append(nodes, node)
	}

	return nodes, rows.Err()
}

// ListModules elenca tutti i moduli MIB caricati con le relative statistiche.
func (d *Database) ListModules() ([]ModuleSummary, error) {
	rows, err := d.db.Query(`
		SELECT name, file_path, node_count, scalar_count, table_count, column_count, type_count, skipped_nodes, missing_imports
		FROM mib_modules
		ORDER BY name
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var modules []ModuleSummary
	for rows.Next() {
		var summary ModuleSummary
		var missingRaw string
		if err := rows.Scan(
			&summary.Name,
			&summary.FilePath,
			&summary.NodeCount,
			&summary.ScalarCount,
			&summary.TableCount,
			&summary.ColumnCount,
			&summary.TypeCount,
			&summary.SkippedNodes,
			&missingRaw,
		); err != nil {
			return nil, err
		}
		summary.MissingImports = decodeMissingImports(missingRaw)
		modules = append(modules, summary)
	}

	return modules, rows.Err()
}

// UpdateModuleMetadata aggiorna le informazioni sulle dipendenze mancanti di un modulo.
func (d *Database) UpdateModuleMetadata(name string, skippedNodes int, missingImports []string) error {
	if _, err := d.db.Exec(
		`UPDATE mib_modules SET skipped_nodes = ?, missing_imports = ? WHERE name = ?`,
		skippedNodes,
		encodeMissingImports(missingImports),
		name,
	); err != nil {
		return fmt.Errorf("failed to update module metadata for %s: %w", name, err)
	}
	return nil
}

// UpdateModuleStats salva le statistiche calcolate per un modulo.
func (d *Database) UpdateModuleStats(name string, stats ModuleStats) error {
	_, err := d.db.Exec(
		`UPDATE mib_modules SET 
			node_count = ?, 
			scalar_count = ?, 
			table_count = ?, 
			column_count = ?, 
			type_count = ?
		WHERE name = ?`,
		stats.NodeCount,
		stats.ScalarCount,
		stats.TableCount,
		stats.ColumnCount,
		stats.TypeCount,
		name,
	)
	if err != nil {
		return fmt.Errorf("failed to update stats for module %s: %w", name, err)
	}
	return nil
}

// GetModuleSummary recupera i metadati di un singolo modulo.
func (d *Database) GetModuleSummary(name string) (*ModuleSummary, error) {
	row := d.db.QueryRow(`
		SELECT name, file_path, node_count, scalar_count, table_count, column_count, type_count, skipped_nodes, missing_imports
		FROM mib_modules
		WHERE name = ?
	`, name)

	var summary ModuleSummary
	var missingRaw string
	if err := row.Scan(
		&summary.Name,
		&summary.FilePath,
		&summary.NodeCount,
		&summary.ScalarCount,
		&summary.TableCount,
		&summary.ColumnCount,
		&summary.TypeCount,
		&summary.SkippedNodes,
		&missingRaw,
	); err != nil {
		return nil, err
	}
	summary.MissingImports = decodeMissingImports(missingRaw)

	return &summary, nil
}

// GetModuleTree restituisce l'albero dei nodi appartenenti a un modulo specifico.
func (d *Database) GetModuleTree(name string) ([]*Node, error) {
	rows, err := d.db.Query(`
		SELECT n.id, n.oid, n.name, n.parent_oid, n.type, n.syntax, n.access, n.status, n.description, m.name
		FROM mib_nodes n
		INNER JOIN mib_modules m ON n.module_id = m.id
		WHERE m.name = ?
		ORDER BY n.oid
	`, name)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var nodes []*Node
	for rows.Next() {
		node := &Node{}
		var parentOID, syntax, access, status, description, moduleName sql.NullString
		if err := rows.Scan(
			&node.ID, &node.OID, &node.Name, &parentOID, &node.Type,
			&syntax, &access, &status, &description, &moduleName,
		); err != nil {
			return nil, err
		}
		if parentOID.Valid {
			node.ParentOID = parentOID.String
		}
		if syntax.Valid {
			node.Syntax = syntax.String
		}
		if access.Valid {
			node.Access = access.String
		}
		if status.Valid {
			node.Status = status.String
		}
		if description.Valid {
			node.Description = description.String
		}
		if moduleName.Valid {
			node.Module = moduleName.String
		}
		node.Children = []*Node{}
		nodes = append(nodes, node)
	}

	nodeMap := make(map[string]*Node, len(nodes))
	for _, node := range nodes {
		nodeMap[node.OID] = node
		canonical := strings.TrimPrefix(node.OID, ".")
		if canonical != node.OID {
			nodeMap[canonical] = node
		} else {
			nodeMap["."+node.OID] = node
		}
	}

	var roots []*Node
	for _, node := range nodes {
		parentOID := strings.TrimSpace(node.ParentOID)
		if parentOID == "" {
			roots = append(roots, node)
			continue
		}
		parent, hasParent := nodeMap[parentOID]
		if !hasParent {
			alt := strings.TrimPrefix(parentOID, ".")
			parent, hasParent = nodeMap[alt]
			if !hasParent {
				parent, hasParent = nodeMap["."+alt]
			}
		}
		if hasParent {
			parent.Children = append(parent.Children, node)
		} else {
			roots = append(roots, node)
		}
	}

	return roots, rows.Err()
}

// DeleteModule elimina un modulo e tutti i suoi nodi
func (d *Database) DeleteModule(name string) error {
	_, err := d.db.Exec("DELETE FROM mib_modules WHERE name = ?", name)
	return err
}

// ExportTree esporta l'albero MIB in JSON
func (d *Database) ExportTree() (string, error) {
	tree, err := d.GetTree()
	if err != nil {
		return "", err
	}

	data, err := json.MarshalIndent(tree, "", "  ")
	if err != nil {
		return "", err
	}

	return string(data), nil
}

// GetStats ritorna statistiche sul database
func (d *Database) GetStats() (map[string]int, error) {
	stats := make(map[string]int)

	var modulesCount int
	// Conta moduli
	err := d.db.QueryRow("SELECT COUNT(*) FROM mib_modules").Scan(&modulesCount)
	if err != nil {
		return nil, err
	}
	stats["modules"] = modulesCount

	var totalNodesCount int
	// Conta nodi totali
	err = d.db.QueryRow("SELECT COUNT(*) FROM mib_nodes").Scan(&totalNodesCount)
	if err != nil {
		return nil, err
	}
	stats["total_nodes"] = totalNodesCount

	// Conta per tipo
	rows, err := d.db.Query("SELECT type, COUNT(*) FROM mib_nodes GROUP BY type")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var nodeType string
		var count int
		if err := rows.Scan(&nodeType, &count); err != nil {
			return nil, err
		}
		stats[nodeType] = count
	}

	return stats, nil
}

// GetBookmarks recupera tutti gli OID dei bookmark
func (d *Database) GetBookmarks() ([]string, error) {
	rows, err := d.db.Query("SELECT oid FROM bookmarks ORDER BY created_at DESC")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var oids []string
	for rows.Next() {
		var oid string
		if err := rows.Scan(&oid); err != nil {
			return nil, err
		}
		oids = append(oids, oid)
	}

	return oids, rows.Err()
}
