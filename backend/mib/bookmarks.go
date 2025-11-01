package mib

import (
	"database/sql"
	"fmt"
	"strings"
	"time"
)

// BookmarkFolderKeyPrefix è il prefisso utilizzato per identificare le cartelle nei nodi synthetic.
const BookmarkFolderKeyPrefix = "bookmark-folder:"

// BookmarkFolder rappresenta una cartella di bookmark con eventuali figli.
type BookmarkFolder struct {
	ID        int64             `json:"id"`
	Name      string            `json:"name"`
	ParentID  *int64            `json:"parentId,omitempty"`
	CreatedAt time.Time         `json:"createdAt"`
	Children  []*BookmarkFolder `json:"children,omitempty"`
	Bookmarks []*BookmarkEntry  `json:"bookmarks,omitempty"`
}

// BookmarkEntry rappresenta un singolo bookmark associato a una cartella.
type BookmarkEntry struct {
	OID       string    `json:"oid"`
	FolderID  *int64    `json:"folderId,omitempty"`
	CreatedAt time.Time `json:"createdAt"`
}

// AddBookmark crea o aggiorna un bookmark, assegnandolo a una cartella opzionale.
func (d *Database) AddBookmark(oid string, folderID *int64) error {
	if d == nil || d.db == nil {
		return fmt.Errorf("database not initialized")
	}
	trimmed := strings.TrimSpace(oid)
	if trimmed == "" {
		return fmt.Errorf("oid is required")
	}

	if folderID != nil {
		if err := d.ensureFolderExists(*folderID); err != nil {
			return err
		}
	}

	var parent interface{}
	if folderID != nil {
		parent = *folderID
	} else {
		parent = nil
	}

	_, err := d.db.Exec(`
		INSERT INTO bookmarks (oid, folder_id)
		VALUES (?, ?)
		ON CONFLICT(oid) DO UPDATE SET folder_id = excluded.folder_id
	`, trimmed, parent)
	if err != nil {
		return fmt.Errorf("failed to upsert bookmark: %w", err)
	}

	return nil
}

// MoveBookmark sposta un bookmark esistente in una nuova cartella (o nella root).
func (d *Database) MoveBookmark(oid string, folderID *int64) error {
	return d.AddBookmark(oid, folderID)
}

// RemoveBookmark elimina un bookmark a partire dal suo OID.
func (d *Database) RemoveBookmark(oid string) error {
	if d == nil || d.db == nil {
		return fmt.Errorf("database not initialized")
	}
	trimmed := strings.TrimSpace(oid)
	if trimmed == "" {
		return fmt.Errorf("oid is required")
	}

	if _, err := d.db.Exec("DELETE FROM bookmarks WHERE oid = ?", trimmed); err != nil {
		return fmt.Errorf("failed to remove bookmark: %w", err)
	}
	return nil
}

// CreateBookmarkFolder crea una nuova cartella per i bookmark.
func (d *Database) CreateBookmarkFolder(name string, parentID *int64) (*BookmarkFolder, error) {
	if d == nil || d.db == nil {
		return nil, fmt.Errorf("database not initialized")
	}

	trimmed := strings.TrimSpace(name)
	if trimmed == "" {
		return nil, fmt.Errorf("folder name is required")
	}

	var parent interface{}
	if parentID != nil {
		if err := d.ensureFolderExists(*parentID); err != nil {
			return nil, err
		}
		parent = *parentID
	}

	if err := d.ensureFolderNameUnique(trimmed, parentID); err != nil {
		return nil, err
	}

	result, err := d.db.Exec(`INSERT INTO bookmark_folders (name, parent_folder_id) VALUES (?, ?)`, trimmed, parent)
	if err != nil {
		return nil, fmt.Errorf("failed to create bookmark folder: %w", err)
	}

	folderID, err := result.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("failed to resolve new folder id: %w", err)
	}

	folder := &BookmarkFolder{
		ID:   folderID,
		Name: trimmed,
	}
	if parentID != nil {
		folder.ParentID = parentID
	}

	if err := d.db.QueryRow(`SELECT created_at FROM bookmark_folders WHERE id = ?`, folderID).Scan(&folder.CreatedAt); err != nil {
		return nil, fmt.Errorf("failed to fetch folder metadata: %w", err)
	}

	return folder, nil
}

// RenameBookmarkFolder aggiorna il nome di una cartella esistente.
func (d *Database) RenameBookmarkFolder(id int64, newName string) error {
	if d == nil || d.db == nil {
		return fmt.Errorf("database not initialized")
	}
	if id <= 0 {
		return fmt.Errorf("folder id is required")
	}

	trimmed := strings.TrimSpace(newName)
	if trimmed == "" {
		return fmt.Errorf("folder name is required")
	}

	var parent sql.NullInt64
	err := d.db.QueryRow(`SELECT parent_folder_id FROM bookmark_folders WHERE id = ?`, id).Scan(&parent)
	if err == sql.ErrNoRows {
		return fmt.Errorf("bookmark folder %d not found", id)
	}
	if err != nil {
		return fmt.Errorf("failed to load folder metadata: %w", err)
	}

	var parentPtr *int64
	if parent.Valid {
		parentPtr = &parent.Int64
	}

	if err := d.ensureFolderNameUnique(trimmed, parentPtr, id); err != nil {
		return err
	}

	result, err := d.db.Exec(`UPDATE bookmark_folders SET name = ? WHERE id = ?`, trimmed, id)
	if err != nil {
		return fmt.Errorf("failed to rename bookmark folder: %w", err)
	}
	affected, _ := result.RowsAffected()
	if affected == 0 {
		return fmt.Errorf("bookmark folder %d not found", id)
	}
	return nil
}

// DeleteBookmarkFolder elimina una cartella (e i suoi contenuti). L'eliminazione è cascata.
func (d *Database) DeleteBookmarkFolder(id int64) error {
	if d == nil || d.db == nil {
		return fmt.Errorf("database not initialized")
	}
	if id <= 0 {
		return fmt.Errorf("folder id is required")
	}

	result, err := d.db.Exec(`DELETE FROM bookmark_folders WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("failed to delete bookmark folder: %w", err)
	}
	affected, _ := result.RowsAffected()
	if affected == 0 {
		return fmt.Errorf("bookmark folder %d not found", id)
	}
	return nil
}

// MoveBookmarkFolder assegna un nuovo parent a una cartella esistente.
func (d *Database) MoveBookmarkFolder(id int64, parentID *int64) error {
	if d == nil || d.db == nil {
		return fmt.Errorf("database not initialized")
	}
	if id <= 0 {
		return fmt.Errorf("folder id is required")
	}

	var current sql.NullInt64
	err := d.db.QueryRow(`SELECT parent_folder_id FROM bookmark_folders WHERE id = ?`, id).Scan(&current)
	if err == sql.ErrNoRows {
		return fmt.Errorf("bookmark folder %d not found", id)
	}
	if err != nil {
		return fmt.Errorf("failed to load folder metadata: %w", err)
	}

	if parentID != nil {
		if err := d.ensureFolderExists(*parentID); err != nil {
			return err
		}
		if *parentID == id {
			return fmt.Errorf("a folder cannot be its own parent")
		}
		if err := d.ensureNotDescendant(id, *parentID); err != nil {
			return err
		}
	}

	target := sql.NullInt64{}
	if parentID != nil {
		target = sql.NullInt64{Int64: *parentID, Valid: true}
	}

	if current.Valid == target.Valid && (!current.Valid || current.Int64 == target.Int64) {
		return nil
	}

	var value interface{}
	if parentID != nil {
		value = *parentID
	} else {
		value = nil
	}

	if _, err := d.db.Exec(`UPDATE bookmark_folders SET parent_folder_id = ? WHERE id = ?`, value, id); err != nil {
		return fmt.Errorf("failed to move bookmark folder: %w", err)
	}
	return nil
}

// GetBookmarkHierarchy ricostruisce l'albero delle cartelle e dei bookmark.
func (d *Database) GetBookmarkHierarchy() (*BookmarkFolder, error) {
	if d == nil || d.db == nil {
		return nil, fmt.Errorf("database not initialized")
	}

	type folderRecord struct {
		folder   *BookmarkFolder
		parentID int64
	}

	root := &BookmarkFolder{
		ID:        0,
		Name:      "Bookmarks",
		CreatedAt: time.Now(),
	}
	folderMap := map[int64]*BookmarkFolder{
		0: root,
	}

	rows, err := d.db.Query(`
		SELECT id, name, parent_folder_id, created_at
		FROM bookmark_folders
		ORDER BY created_at ASC, id ASC
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to query bookmark folders: %w", err)
	}
	defer rows.Close()

	var records []folderRecord

	for rows.Next() {
		var (
			id      int64
			name    string
			parent  sql.NullInt64
			created time.Time
		)
		if scanErr := rows.Scan(&id, &name, &parent, &created); scanErr != nil {
			return nil, fmt.Errorf("failed to scan bookmark folder: %w", scanErr)
		}

		folder := &BookmarkFolder{
			ID:        id,
			Name:      name,
			CreatedAt: created,
		}
		parentID := int64(0)
		if parent.Valid {
			parentID = parent.Int64
			folder.ParentID = &parentID
		}

		folderMap[id] = folder
		records = append(records, folderRecord{
			folder:   folder,
			parentID: parentID,
		})
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to iterate bookmark folders: %w", err)
	}

	for _, rec := range records {
		parent := folderMap[rec.parentID]
		if parent == nil {
			parent = root
		}
		parent.Children = append(parent.Children, rec.folder)
	}

	bookmarkRows, err := d.db.Query(`
		SELECT oid, folder_id, created_at
		FROM bookmarks
		ORDER BY created_at DESC, oid ASC
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to query bookmarks: %w", err)
	}
	defer bookmarkRows.Close()

	for bookmarkRows.Next() {
		var (
			oid      string
			folderID sql.NullInt64
			created  time.Time
		)
		if scanErr := bookmarkRows.Scan(&oid, &folderID, &created); scanErr != nil {
			return nil, fmt.Errorf("failed to scan bookmark: %w", scanErr)
		}

		entry := &BookmarkEntry{
			OID:       oid,
			CreatedAt: created,
		}
		parentID := int64(0)
		if folderID.Valid {
			parentID = folderID.Int64
			entry.FolderID = &parentID
		}

		parentFolder, ok := folderMap[parentID]
		if !ok {
			parentFolder = root
		}
		parentFolder.Bookmarks = append(parentFolder.Bookmarks, entry)
	}

	if err := bookmarkRows.Err(); err != nil {
		return nil, fmt.Errorf("failed to iterate bookmarks: %w", err)
	}

	return root, nil
}

// ensureFolderExists verifica che una cartella esista.
func (d *Database) ensureFolderExists(id int64) error {
	var exists int
	if err := d.db.QueryRow(`SELECT COUNT(1) FROM bookmark_folders WHERE id = ?`, id).Scan(&exists); err != nil {
		return fmt.Errorf("failed to validate bookmark folder %d: %w", id, err)
	}
	if exists == 0 {
		return fmt.Errorf("bookmark folder %d not found", id)
	}
	return nil
}

// ensureFolderNameUnique garantisce che non esistano duplicati nello stesso parent.
func (d *Database) ensureFolderNameUnique(name string, parentID *int64, exclude ...int64) error {
	query := `SELECT COUNT(1) FROM bookmark_folders WHERE name = ?`
	args := []interface{}{name}

	if parentID != nil {
		query += ` AND parent_folder_id = ?`
		args = append(args, *parentID)
	} else {
		query += ` AND parent_folder_id IS NULL`
	}

	if len(exclude) > 0 {
		query += ` AND id != ?`
		args = append(args, exclude[0])
	}

	var count int
	if err := d.db.QueryRow(query, args...).Scan(&count); err != nil {
		return fmt.Errorf("failed to check folder name uniqueness: %w", err)
	}
	if count > 0 {
		return fmt.Errorf("a folder named %q already exists in the selected location", name)
	}
	return nil
}

// ensureNotDescendant impedisce di creare cicli spostando cartelle sotto i propri discendenti.
func (d *Database) ensureNotDescendant(folderID, candidateParent int64) error {
	var count int
	err := d.db.QueryRow(`
		WITH RECURSIVE subtree(id) AS (
			SELECT ?
			UNION ALL
			SELECT bf.id FROM bookmark_folders bf
			INNER JOIN subtree s ON bf.parent_folder_id = s.id
		)
		SELECT COUNT(1) FROM subtree WHERE id = ?
	`, folderID, candidateParent).Scan(&count)
	if err != nil {
		return fmt.Errorf("failed to validate folder hierarchy: %w", err)
	}
	if count > 0 {
		return fmt.Errorf("cannot move a folder inside its own subtree")
	}
	return nil
}
