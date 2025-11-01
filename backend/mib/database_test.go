package mib

import (
	"reflect"
	"testing"
)

// newTestDB crea un database in memoria per i test.
func newTestDB(t *testing.T) *Database {
	t.Helper()

	// Usa un file temporaneo per il db in-memory che viene pulito
	tempDir := t.TempDir()
	db, err := NewDatabase(tempDir)
	if err != nil {
		t.Fatalf("Failed to create test database: %v", err)
	}

	// La Close verrà chiamata automaticamente alla fine del test grazie a t.Cleanup
	t.Cleanup(func() {
		db.Close()
	})

	return db
}

func TestForeignKeysEnabled(t *testing.T) {
	db := newTestDB(t)

	var enabled int
	if err := db.db.QueryRow("PRAGMA foreign_keys").Scan(&enabled); err != nil {
		t.Fatalf("PRAGMA foreign_keys query failed: %v", err)
	}

	if enabled != 1 {
		t.Fatalf("foreign_keys PRAGMA = %d, want 1", enabled)
	}
}

func TestSaveAndGetModule(t *testing.T) {
	db := newTestDB(t)

	moduleName := "TEST-MIB"
	filePath := "/tmp/TEST-MIB.txt"

	id, err := db.SaveModule(moduleName, filePath)
	if err != nil {
		t.Fatalf("SaveModule() error = %v", err)
	}

	if id == 0 {
		t.Errorf("SaveModule() returned id 0")
	}

	retrievedID, err := db.GetModuleID(moduleName)
	if err != nil {
		t.Fatalf("GetModuleID() error = %v", err)
	}

	if retrievedID != id {
		t.Errorf("GetModuleID() = %v, want %v", retrievedID, id)
	}
}

func TestSaveAndGetNode(t *testing.T) {
	db := newTestDB(t)

	moduleID, _ := db.SaveModule("TEST-MIB", "")

	node := &Node{
		OID:         ".1.3.6.1",
		Name:        "iso",
		Type:        "node",
		Description: "Root of the MIB tree",
	}

	if err := db.SaveNode(node, moduleID); err != nil {
		t.Fatalf("SaveNode() error = %v", err)
	}

	retrievedNode, err := db.GetNode(node.OID)
	if err != nil {
		t.Fatalf("GetNode() error = %v", err)
	}

	if retrievedNode.Name != node.Name {
		t.Errorf("GetNode() Name = %v, want %v", retrievedNode.Name, node.Name)
	}
	if retrievedNode.Description != node.Description {
		t.Errorf("GetNode() Description = %v, want %v", retrievedNode.Description, node.Description)
	}
}

func TestGetTree(t *testing.T) {
	db := newTestDB(t)
	moduleID, _ := db.SaveModule("TEST-MIB", "")

	nodes := []*Node{
		{OID: ".1", Name: "iso"},
		{OID: ".1.3", Name: "org", ParentOID: ".1"},
		{OID: ".1.3.6", Name: "dod", ParentOID: ".1.3"},
	}

	if err := db.SaveNodes(nodes, moduleID); err != nil {
		t.Fatalf("SaveNodes() error = %v", err)
	}

	tree, err := db.GetTree()
	if err != nil {
		t.Fatalf("GetTree() error = %v", err)
	}

	if len(tree) != 1 {
		t.Fatalf("GetTree() returned %d root nodes, want 1", len(tree))
	}

	if tree[0].Name != "iso" {
		t.Errorf("Root node name = %s, want iso", tree[0].Name)
	}

	if len(tree[0].Children) != 1 {
		t.Fatalf("iso should have 1 child, got %d", len(tree[0].Children))
	}

	if tree[0].Children[0].Name != "org" {
		t.Errorf("Child node name = %s, want org", tree[0].Children[0].Name)
	}
}

func TestGetNodeVariantsAndAncestors(t *testing.T) {
	db := newTestDB(t)
	moduleID, _ := db.SaveModule("TEST-MIB", "")

	nodes := []*Node{
		{OID: "1.3", Name: "org"},
		{OID: "1.3.6", Name: "dod", ParentOID: "1.3"},
		{OID: "1.3.6.1", Name: "internet", ParentOID: "1.3.6"},
		{OID: "1.3.6.1.2", Name: "mgmt", ParentOID: "1.3.6.1"},
		{OID: "1.3.6.1.2.1", Name: "mib-2", ParentOID: "1.3.6.1.2"},
		{OID: "1.3.6.1.2.1.1", Name: "system", ParentOID: "1.3.6.1.2.1"},
		{OID: "1.3.6.1.2.1.1.4", Name: "sysContact", ParentOID: "1.3.6.1.2.1.1"},
	}

	if err := db.SaveNodes(nodes, moduleID); err != nil {
		t.Fatalf("SaveNodes() error = %v", err)
	}

	node, err := db.GetNode(".1.3.6.1.2.1.1.4.0")
	if err != nil {
		t.Fatalf("GetNode() with variant failed: %v", err)
	}
	if node.Name != "sysContact" {
		t.Errorf("GetNode() Name = %s, want sysContact", node.Name)
	}

	ancestors, err := db.GetNodeAncestors("1.3.6.1.2.1.1.4.0")
	if err != nil {
		t.Fatalf("GetNodeAncestors() error = %v", err)
	}

	if len(ancestors) < 3 {
		t.Fatalf("expected at least 3 ancestors, got %d", len(ancestors))
	}
	if ancestors[0].Name != "sysContact" {
		t.Errorf("first ancestor = %s, want sysContact", ancestors[0].Name)
	}
	if ancestors[1].Name != "system" {
		t.Errorf("second ancestor = %s, want system", ancestors[1].Name)
	}
	if ancestors[2].Name != "mib-2" {
		t.Errorf("third ancestor = %s, want mib-2", ancestors[2].Name)
	}
}

func TestListModules(t *testing.T) {
	db := newTestDB(t)
	db.SaveModule("B-MIB", "")
	db.SaveModule("A-MIB", "")
	db.SaveModule("C-MIB", "")

	modules, err := db.ListModules()
	if err != nil {
		t.Fatalf("ListModules() error = %v", err)
	}

	if len(modules) != 3 {
		t.Fatalf("ListModules() returned %d modules, want 3", len(modules))
	}

	names := []string{modules[0].Name, modules[1].Name, modules[2].Name}
	want := []string{"A-MIB", "B-MIB", "C-MIB"}
	if !reflect.DeepEqual(names, want) {
		t.Errorf("ListModules() names = %v, want %v", names, want)
	}

	for _, summary := range modules {
		if summary.NodeCount != 0 || summary.ScalarCount != 0 || summary.TableCount != 0 || summary.TypeCount != 0 {
			t.Errorf("expected zeroed stats for module %s, got %+v", summary.Name, summary)
		}
	}
}

func TestDeleteModule(t *testing.T) {
	db := newTestDB(t)
	moduleID, _ := db.SaveModule("TEST-MIB", "")
	db.SaveNode(&Node{OID: ".1", Name: "iso"}, moduleID)

	if err := db.DeleteModule("TEST-MIB"); err != nil {
		t.Fatalf("DeleteModule() error = %v", err)
	}

	_, err := db.GetModuleID("TEST-MIB")
	if err == nil {
		t.Error("GetModuleID() should have failed after deletion, but it didn't")
	}

	// Check if nodes are also deleted (due to CASCADE)
	_, err = db.GetNode(".1")
	if err == nil {
		t.Error("GetNode() should have failed after module deletion, but it didn't")
	}
}

func TestGetStats(t *testing.T) {
	db := newTestDB(t)
	mod1, _ := db.SaveModule("MIB-1", "")
	mod2, _ := db.SaveModule("MIB-2", "")

	nodes := []*Node{
		{OID: ".1", Name: "iso", Type: "node"},
		{OID: ".1.3", Name: "org", Type: "node"},
		{OID: ".1.3.6", Name: "dod", Type: "scalar"},
		{OID: ".1.3.6.1", Name: "internet", Type: "table"},
	}

	db.SaveNodes(nodes[:2], mod1)
	db.SaveNodes(nodes[2:], mod2)

	stats, err := db.GetStats()
	if err != nil {
		t.Fatalf("GetStats() error = %v", err)
	}

	want := map[string]int{
		"modules":     2,
		"total_nodes": 4,
		"node":        2,
		"scalar":      1,
		"table":       1,
	}

	if !reflect.DeepEqual(stats, want) {
		t.Errorf("GetStats() = %v, want %v", stats, want)
	}
}

func TestModuleSummaryAndTree(t *testing.T) {
	db := newTestDB(t)

	modID, _ := db.SaveModule("TEST-MIB", "")

	nodes := []*Node{
		{OID: "1.3.6", Name: "dod", Type: "node", Module: "TEST-MIB"},
		{OID: "1.3.6.1", Name: "internet", ParentOID: "1.3.6", Type: "scalar", Module: "TEST-MIB"},
	}

	if err := db.SaveNodes(nodes, modID); err != nil {
		t.Fatalf("SaveNodes() error = %v", err)
	}

	stats := ModuleStats{
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
	if err := db.UpdateModuleMetadata("TEST-MIB", 5, missing); err != nil {
		t.Fatalf("UpdateModuleMetadata() error = %v", err)
	}

	summary, err := db.GetModuleSummary("TEST-MIB")
	if err != nil {
		t.Fatalf("GetModuleSummary() error = %v", err)
	}

	if summary.NodeCount != 2 || summary.ScalarCount != 1 || summary.TypeCount != 3 {
		t.Errorf("unexpected summary stats: %+v", summary)
	}
	if summary.SkippedNodes != 5 {
		t.Errorf("summary.SkippedNodes = %d, want 5", summary.SkippedNodes)
	}
	if !reflect.DeepEqual(summary.MissingImports, missing) {
		t.Errorf("summary.MissingImports = %v, want %v", summary.MissingImports, missing)
	}

	tree, err := db.GetModuleTree("TEST-MIB")
	if err != nil {
		t.Fatalf("GetModuleTree() error = %v", err)
	}
	if len(tree) != 1 {
		t.Fatalf("expected 1 root node in module tree, got %d", len(tree))
	}
	if tree[0].Name != "dod" {
		t.Errorf("root node name = %s, want dod", tree[0].Name)
	}
	if len(tree[0].Children) != 1 || tree[0].Children[0].Name != "internet" {
		t.Errorf("unexpected children structure: %+v", tree[0].Children)
	}
	if tree[0].Module != "TEST-MIB" || tree[0].Children[0].Module != "TEST-MIB" {
		t.Error("module filtering failed, found nodes from other modules")
	}
}
