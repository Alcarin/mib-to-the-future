package mib

import (
	"database/sql"
	"testing"
)

func TestBookmarkFolderCRUD(t *testing.T) {
	db := newTestDB(t)

	rootFolder, err := db.CreateBookmarkFolder("Network", nil)
	if err != nil {
		t.Fatalf("CreateBookmarkFolder root error: %v", err)
	}
	if rootFolder.ID == 0 {
		t.Fatalf("expected folder id > 0")
	}
	if rootFolder.ParentID != nil {
		t.Fatalf("expected root folder to have nil parent")
	}

	childFolder, err := db.CreateBookmarkFolder("Interfaces", &rootFolder.ID)
	if err != nil {
		t.Fatalf("CreateBookmarkFolder child error: %v", err)
	}
	if childFolder.ParentID == nil || *childFolder.ParentID != rootFolder.ID {
		t.Fatalf("expected child parent id to be %d, got %v", rootFolder.ID, childFolder.ParentID)
	}

	if err := db.RenameBookmarkFolder(childFolder.ID, "NICs"); err != nil {
		t.Fatalf("RenameBookmarkFolder error: %v", err)
	}

	if err := db.MoveBookmarkFolder(childFolder.ID, nil); err != nil {
		t.Fatalf("MoveBookmarkFolder to root error: %v", err)
	}

	var parent sql.NullInt64
	if err := db.db.QueryRow(`SELECT parent_folder_id FROM bookmark_folders WHERE id = ?`, childFolder.ID).Scan(&parent); err != nil {
		t.Fatalf("failed to query bookmark folder after move: %v", err)
	}
	if parent.Valid {
		t.Fatalf("expected folder to be moved to root, got parent %v", parent.Int64)
	}

	if err := db.DeleteBookmarkFolder(childFolder.ID); err != nil {
		t.Fatalf("DeleteBookmarkFolder error: %v", err)
	}

	var remaining int
	if err := db.db.QueryRow(`SELECT COUNT(1) FROM bookmark_folders`).Scan(&remaining); err != nil {
		t.Fatalf("failed counting bookmark folders: %v", err)
	}
	if remaining != 1 {
		t.Fatalf("expected only root folder to remain, got %d", remaining)
	}
}

func TestMoveBookmarkFolderPreventsCycles(t *testing.T) {
	db := newTestDB(t)

	parent, err := db.CreateBookmarkFolder("Parent", nil)
	if err != nil {
		t.Fatalf("CreateBookmarkFolder parent error: %v", err)
	}
	child, err := db.CreateBookmarkFolder("Child", &parent.ID)
	if err != nil {
		t.Fatalf("CreateBookmarkFolder child error: %v", err)
	}

	if err := db.MoveBookmarkFolder(parent.ID, &child.ID); err == nil {
		t.Fatalf("expected MoveBookmarkFolder to fail when moving parent under child")
	}
}

func TestAddBookmarkWithFolder(t *testing.T) {
	db := newTestDB(t)

	if _, err := db.CreateBookmarkFolder("Root Folder", nil); err != nil {
		t.Fatalf("CreateBookmarkFolder error: %v", err)
	}
	sub, err := db.CreateBookmarkFolder("Sub Folder", nil)
	if err != nil {
		t.Fatalf("CreateBookmarkFolder sub error: %v", err)
	}

	if err := db.AddBookmark("1.3.6.1", nil); err != nil {
		t.Fatalf("AddBookmark root error: %v", err)
	}

	if err := db.AddBookmark("1.3.6.1.2", &sub.ID); err != nil {
		t.Fatalf("AddBookmark child error: %v", err)
	}

	var (
		rootFolder  sql.NullInt64
		childFolder sql.NullInt64
	)

	if err := db.db.QueryRow(`SELECT folder_id FROM bookmarks WHERE oid = ?`, "1.3.6.1").Scan(&rootFolder); err != nil {
		t.Fatalf("failed to load root bookmark: %v", err)
	}
	if rootFolder.Valid {
		t.Fatalf("expected root bookmark to have NULL folder, got %v", rootFolder.Int64)
	}

	if err := db.db.QueryRow(`SELECT folder_id FROM bookmarks WHERE oid = ?`, "1.3.6.1.2").Scan(&childFolder); err != nil {
		t.Fatalf("failed to load child bookmark: %v", err)
	}
	if !childFolder.Valid || childFolder.Int64 != sub.ID {
		t.Fatalf("expected child bookmark folder id %d, got %v", sub.ID, childFolder)
	}
}

func TestGetBookmarkHierarchy(t *testing.T) {
	db := newTestDB(t)

	parent, err := db.CreateBookmarkFolder("Parent", nil)
	if err != nil {
		t.Fatalf("CreateBookmarkFolder parent error: %v", err)
	}
	child, err := db.CreateBookmarkFolder("Child", &parent.ID)
	if err != nil {
		t.Fatalf("CreateBookmarkFolder child error: %v", err)
	}

	if err := db.AddBookmark("1.3", nil); err != nil {
		t.Fatalf("AddBookmark root error: %v", err)
	}
	if err := db.AddBookmark("1.3.6", &parent.ID); err != nil {
		t.Fatalf("AddBookmark parent error: %v", err)
	}
	if err := db.AddBookmark("1.3.6.1", &child.ID); err != nil {
		t.Fatalf("AddBookmark child error: %v", err)
	}

	root, err := db.GetBookmarkHierarchy()
	if err != nil {
		t.Fatalf("GetBookmarkHierarchy error: %v", err)
	}

	if root == nil {
		t.Fatalf("expected hierarchy root not to be nil")
	}

	if len(root.Bookmarks) != 1 {
		t.Fatalf("expected 1 root bookmark, got %d", len(root.Bookmarks))
	}

	if len(root.Children) != 1 {
		t.Fatalf("expected 1 top-level folder, got %d", len(root.Children))
	}

	top := root.Children[0]
	if top.Name != "Parent" {
		t.Fatalf("expected top folder to be Parent, got %s", top.Name)
	}
	if len(top.Bookmarks) != 1 || top.Bookmarks[0].OID != "1.3.6" {
		t.Fatalf("expected bookmark 1.3.6 in parent folder")
	}
	if len(top.Children) != 1 {
		t.Fatalf("expected parent to have one child folder")
	}
	if top.Children[0].Name != "Child" {
		t.Fatalf("expected child folder name Child, got %s", top.Children[0].Name)
	}
	if len(top.Children[0].Bookmarks) != 1 || top.Children[0].Bookmarks[0].OID != "1.3.6.1" {
		t.Fatalf("expected bookmark 1.3.6.1 in child folder")
	}
}
