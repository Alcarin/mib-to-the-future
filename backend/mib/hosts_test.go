package mib

import (
	"os"
	"path/filepath"
	"testing"
)

func setupTestDB(t *testing.T) *Database {
	t.Helper()
	dbPath := filepath.Join(t.TempDir(), "test.db")
	db, err := NewDatabase(dbPath)
	if err != nil {
		t.Fatalf("Failed to create test database: %v", err)
	}

	// Drop the table if it exists and recreate it with the new schema
	_, err = db.db.Exec(`DROP TABLE IF EXISTS host_configs`)
	if err != nil {
		t.Fatalf("Failed to drop table: %v", err)
	}

	_, err = db.db.Exec(`
	CREATE TABLE host_configs (
		address TEXT PRIMARY KEY,
		port INTEGER,
		community TEXT,
		write_community TEXT,
		version TEXT,
		last_used_at TEXT,
		created_at TEXT DEFAULT CURRENT_TIMESTAMP,
		context_name TEXT,
		security_level TEXT,
		security_username TEXT,
		auth_protocol TEXT,
		auth_password TEXT,
		priv_protocol TEXT,
		priv_password TEXT
	)
	`)
	if err != nil {
		t.Fatalf("Failed to create table: %v", err)
	}

	t.Cleanup(func() {
		db.Close()
		os.Remove(dbPath)
	})

	return db
}

func TestSaveAndListHosts(t *testing.T) {
	db := setupTestDB(t)

	// Test saving a new host
	host1 := HostConfig{
		Address:        "localhost",
		Port:           161,
		Community:      "public",
		WriteCommunity: "public",
		Version:        "v2c",
	}
	_, err := db.SaveHost(host1)
	if err != nil {
		t.Fatalf("SaveHost() insert error = %v", err)
	}

	saved, err := db.GetHost("localhost")
	if err != nil {
		t.Fatalf("GetHost() error = %v", err)
	}
	if saved == nil {
		t.Fatalf("expected host to be saved")
	}
	if saved.WriteCommunity != host1.Community {
		t.Fatalf("expected write community %s, got %s", host1.Community, saved.WriteCommunity)
	}

	// Test updating an existing host
	host2 := HostConfig{
		Address:        "localhost",
		Port:           1161,
		Community:      "private",
		WriteCommunity: "private-write",
		Version:        "v1",
	}
	_, err = db.SaveHost(host2)
	if err != nil {
		t.Fatalf("SaveHost() update error = %v", err)
	}

	// Test listing hosts
	hosts, err := db.ListHosts(0)
	if err != nil {
		t.Fatalf("ListHosts() error = %v", err)
	}

	if len(hosts) != 1 {
		t.Fatalf("expected 1 host, got %d", len(hosts))
	}

	if hosts[0].Address != "localhost" {
		t.Errorf("expected address localhost, got %s", hosts[0].Address)
	}

	if hosts[0].Port != 1161 {
		t.Errorf("expected port 1161, got %d", hosts[0].Port)
	}

	if hosts[0].Community != "private" {
		t.Errorf("expected community private, got %s", hosts[0].Community)
	}

	if hosts[0].WriteCommunity != "private-write" {
		t.Errorf("expected write community private-write, got %s", hosts[0].WriteCommunity)
	}

	if hosts[0].Version != "v1" {
		t.Errorf("expected version v1, got %s", hosts[0].Version)
	}
}
