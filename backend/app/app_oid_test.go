package app

import (
	"testing"

	"mib-to-the-future/backend/mib"
	"mib-to-the-future/backend/snmp"
)

func setupTestAppWithNodes(t *testing.T, nodes ...*mib.Node) *App {
	t.Helper()

	dataDir := t.TempDir()
	db, err := mib.NewDatabase(dataDir)
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

	for _, node := range nodes {
		if node == nil {
			continue
		}
		if err := db.SaveNode(node, moduleID); err != nil {
			t.Fatalf("SaveNode(%s) error = %v", node.OID, err)
		}
	}

	app := NewApp()
	app.mibDB = db
	return app
}

func TestEnrichResultAddsResolvedNameForScalarOID(t *testing.T) {
	app := setupTestAppWithNodes(
		t,
		&mib.Node{OID: "1.3.6.1.2.1.1", Name: "system", Type: "node"},
		&mib.Node{OID: "1.3.6.1.2.1.1.5", Name: "sysName", Type: "scalar", ParentOID: "1.3.6.1.2.1.1"},
	)

	result := &snmp.Result{
		OID:   "1.3.6.1.2.1.1.5.0",
		Value: "laboratorio",
		Type:  "OctetString",
	}

	app.enrichResult(result)

	if result.ResolvedName != "sysName" {
		t.Fatalf("ResolvedName = %q, want %q", result.ResolvedName, "sysName")
	}
}

func TestEnrichResultAddsResolvedNameForColumnOID(t *testing.T) {
	app := setupTestAppWithNodes(
		t,
		&mib.Node{OID: "1.3.6.1.2.1.2", Name: "interfaces", Type: "node"},
		&mib.Node{OID: "1.3.6.1.2.1.2.2", Name: "ifTable", Type: "table", ParentOID: "1.3.6.1.2.1.2"},
		&mib.Node{OID: "1.3.6.1.2.1.2.2.1", Name: "ifEntry", Type: "row", ParentOID: "1.3.6.1.2.1.2.2"},
		&mib.Node{OID: "1.3.6.1.2.1.2.2.1.2", Name: "ifDescr", Type: "column", ParentOID: "1.3.6.1.2.1.2.2.1"},
	)

	result := &snmp.Result{
		OID:   "1.3.6.1.2.1.2.2.1.2.10",
		Value: "ethernet0/10",
		Type:  "OctetString",
	}

	app.enrichResult(result)

	if result.ResolvedName != "ifDescr[10]" {
		t.Fatalf("ResolvedName = %q, want %q", result.ResolvedName, "ifDescr[10]")
	}
}

func TestEnrichResultAddsResolvedNameWithMultipleIndexSegments(t *testing.T) {
	app := setupTestAppWithNodes(
		t,
		&mib.Node{OID: "1.3.6.1.4.1.9999", Name: "exampleRoot", Type: "node"},
		&mib.Node{OID: "1.3.6.1.4.1.9999.1", Name: "metrics", Type: "node", ParentOID: "1.3.6.1.4.1.9999"},
		&mib.Node{OID: "1.3.6.1.4.1.9999.1.2", Name: "metricsTable", Type: "table", ParentOID: "1.3.6.1.4.1.9999.1"},
		&mib.Node{OID: "1.3.6.1.4.1.9999.1.2.1", Name: "metricsEntry", Type: "row", ParentOID: "1.3.6.1.4.1.9999.1.2"},
		&mib.Node{OID: "1.3.6.1.4.1.9999.1.2.1.3", Name: "metricValue", Type: "column", ParentOID: "1.3.6.1.4.1.9999.1.2.1"},
	)

	result := &snmp.Result{
		OID:   "1.3.6.1.4.1.9999.1.2.1.3.10.42",
		Value: "128",
		Type:  "Integer",
	}

	app.enrichResult(result)

	if result.ResolvedName != "metricValue[10.42]" {
		t.Fatalf("ResolvedName = %q, want %q", result.ResolvedName, "metricValue[10.42]")
	}
}
