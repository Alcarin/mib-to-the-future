package app

import (
	"testing"

	"mib-to-the-future/backend/mib"
)

// TestNormalizeScalarOID verifica che gli OID scalar vengano completati con l'istanza `.0`.
func TestNormalizeScalarOID(t *testing.T) {
	tempDir := t.TempDir()

	db, err := mib.NewDatabase(tempDir)
	if err != nil {
		t.Fatalf("NewDatabase() error = %v", err)
	}
	t.Cleanup(func() {
		db.Close()
	})

	moduleID, err := db.SaveModule("TEST-MIB", "")
	if err != nil {
		t.Fatalf("SaveModule() error = %v", err)
	}

	if err := db.SaveNode(&mib.Node{
		OID:       "1.3.6.1.2.1.1.5",
		Name:      "sysName",
		Type:      "scalar",
		ParentOID: "1.3.6.1.2.1.1",
	}, moduleID); err != nil {
		t.Fatalf("SaveNode() scalar error = %v", err)
	}

	if err := db.SaveNode(&mib.Node{
		OID:       "1.3.6.1.2.1.2.2.1.2",
		Name:      "ifDescr",
		Type:      "column",
		ParentOID: "1.3.6.1.2.1.2.2.1",
	}, moduleID); err != nil {
		t.Fatalf("SaveNode() column error = %v", err)
	}

	app := &App{mibDB: db}

	if got := app.normalizeScalarOID("1.3.6.1.2.1.1.5"); got != "1.3.6.1.2.1.1.5.0" {
		t.Errorf("normalizeScalarOID() without instance = %s, want 1.3.6.1.2.1.1.5.0", got)
	}

	if got := app.normalizeScalarOID("1.3.6.1.2.1.1.5.0"); got != "1.3.6.1.2.1.1.5.0" {
		t.Errorf("normalizeScalarOID() with instance = %s, want 1.3.6.1.2.1.1.5.0", got)
	}

	if got := app.normalizeScalarOID("1.3.6.1.2.1.2.2.1.2"); got != "1.3.6.1.2.1.2.2.1.2" {
		t.Errorf("normalizeScalarOID() column = %s, want 1.3.6.1.2.1.2.2.1.2", got)
	}
}

func TestGetMIBModuleDetails(t *testing.T) {
	tempDir := t.TempDir()

	db, err := mib.NewDatabase(tempDir)
	if err != nil {
		t.Fatalf("NewDatabase() error = %v", err)
	}
	t.Cleanup(func() {
		db.Close()
	})

	app := &App{mibDB: db}

	moduleID, err := db.SaveModule("TEST-MIB", "")
	if err != nil {
		t.Fatalf("SaveModule() error = %v", err)
	}

	nodes := []*mib.Node{
		{OID: "1.3.6", Name: "dod", Type: "node", Module: "TEST-MIB"},
		{OID: "1.3.6.1", Name: "internet", ParentOID: "1.3.6", Type: "scalar", Module: "TEST-MIB"},
	}

	if err := db.SaveNodes(nodes, moduleID); err != nil {
		t.Fatalf("SaveNodes() error = %v", err)
	}

	stats := mib.ModuleStats{
		NodeCount:   2,
		ScalarCount: 1,
		TableCount:  0,
		ColumnCount: 0,
		TypeCount:   3,
	}
	if err := db.UpdateModuleStats("TEST-MIB", stats); err != nil {
		t.Fatalf("UpdateModuleStats() error = %v", err)
	}

	missing := []string{"DEPENDENCY-MIB"}
	if err := db.UpdateModuleMetadata("TEST-MIB", 4, missing); err != nil {
		t.Fatalf("UpdateModuleMetadata() error = %v", err)
	}

	details, err := app.GetMIBModuleDetails("TEST-MIB")
	if err != nil {
		t.Fatalf("GetMIBModuleDetails() error = %v", err)
	}

	if details.Module != "TEST-MIB" {
		t.Errorf("details.Module = %s, want TEST-MIB", details.Module)
	}
	if details.Stats.NodeCount != 2 || details.Stats.ScalarCount != 1 || details.Stats.TypeCount != 3 {
		t.Errorf("unexpected stats %+v", details.Stats)
	}
	if details.Stats.SkippedNodes != 4 {
		t.Errorf("details.Stats.SkippedNodes = %d, want 4", details.Stats.SkippedNodes)
	}
	if len(details.Tree) != 1 {
		t.Fatalf("expected 1 root node, got %d", len(details.Tree))
	}
	if details.Tree[0].Name != "dod" || len(details.Tree[0].Children) != 1 {
		t.Errorf("unexpected tree structure: %+v", details.Tree)
	}
	if len(details.MissingImports) != 1 || details.MissingImports[0] != "DEPENDENCY-MIB" {
		t.Errorf("details.MissingImports = %v, want %v", details.MissingImports, missing)
	}
}
