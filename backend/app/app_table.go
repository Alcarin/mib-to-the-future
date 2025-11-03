package app

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
	"unicode"

	"mib-to-the-future/backend/mib"
	"mib-to-the-future/backend/snmp"
)

// TableColumn descrive una colonna di una tabella SNMP con i metadati derivati dal MIB.
type TableColumn struct {
	Key         string `json:"key"`
	Label       string `json:"label"`
	OID         string `json:"oid"`
	Type        string `json:"type"`
	Syntax      string `json:"syntax,omitempty"`
	Access      string `json:"access,omitempty"`
	Description string `json:"description,omitempty"`
}

// TableRow rappresenta un record della tabella dove ogni chiave corrisponde a una colonna.
type TableRow map[string]string

// TableDataResponse incapsula metadati e righe della tabella SNMP richiesta dal frontend.
type TableDataResponse struct {
	TableOID string        `json:"tableOid"`
	EntryOID string        `json:"entryOid"`
	Columns  []TableColumn `json:"columns"`
	Rows     []TableRow    `json:"rows"`
}

// FetchTableData esegue un WALK sull'entry della tabella per restituire righe e colonne formattate per il frontend.
// Parametri:
//   - config: configurazione SNMP da utilizzare per la connessione.
//   - tableOID: l'OID del nodo tabella (o di un suo discendente) da interrogare.
//
// Ritorna i metadati della tabella e le righe ottenute dal dispositivo SNMP.
func (a *App) FetchTableData(config snmp.Config, tableOID string) (*TableDataResponse, error) {
	if a.mibDB == nil {
		return nil, a.mibNotInitializedErr()
	}

	normalized := normalizeOIDKey(tableOID)
	if normalized == "" {
		return nil, fmt.Errorf("table OID is required")
	}

	node, err := a.mibDB.GetNode(normalized)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve table %s: %w", normalized, err)
	}

	tableNode, rowNode, columns, err := a.resolveTableSchema(node)
	if err != nil {
		return nil, err
	}

	results, err := a.SNMPWalk(config, rowNode.OID)
	if err != nil {
		return nil, err
	}

	response := &TableDataResponse{
		TableOID: tableNode.OID,
		EntryOID: rowNode.OID,
		Columns:  make([]TableColumn, len(columns)),
	}

	for i, column := range columns {
		label := makeColumnLabel(column.Name)
		if label == "" {
			label = column.Name
		}

		response.Columns[i] = TableColumn{
			Key:         column.Name,
			Label:       label,
			OID:         column.OID,
			Type:        inferColumnValueType(column.Syntax),
			Syntax:      column.Syntax,
			Access:      column.Access,
			Description: column.Description,
		}
	}

	response.Rows = buildTableRows(results, columns)
	return response, nil
}

// resolveTableSchema risolve lo schema di una tabella SNMP partendo da un nodo table, row o column.
func (a *App) resolveTableSchema(node *mib.Node) (*mib.Node, *mib.Node, []*mib.Node, error) {
	if node == nil {
		return nil, nil, nil, fmt.Errorf("table node is nil")
	}

	switch node.Type {
	case "table":
		rowNode, columns, err := a.resolveTableRowAndColumns(node)
		if err != nil {
			return nil, nil, nil, err
		}
		return node, rowNode, columns, nil
	case "row":
		columns, err := a.resolveRowColumns(node)
		if err != nil {
			return nil, nil, nil, err
		}

		parentOID := normalizeOIDKey(node.ParentOID)
		if parentOID == "" {
			return nil, nil, nil, fmt.Errorf("row %s è privo di tabella padre", node.Name)
		}

		tableNode, err := a.mibDB.GetNode(parentOID)
		if err != nil {
			return nil, nil, nil, fmt.Errorf("failed to resolve table for row %s: %w", node.Name, err)
		}
		return tableNode, node, columns, nil
	case "column":
		parentRowOID := normalizeOIDKey(node.ParentOID)
		if parentRowOID == "" {
			return nil, nil, nil, fmt.Errorf("column %s è privo di nodo row padre", node.Name)
		}

		rowNode, err := a.mibDB.GetNode(parentRowOID)
		if err != nil {
			return nil, nil, nil, fmt.Errorf("failed to resolve row for column %s: %w", node.Name, err)
		}

		columns, err := a.resolveRowColumns(rowNode)
		if err != nil {
			return nil, nil, nil, err
		}

		tableOID := normalizeOIDKey(rowNode.ParentOID)
		if tableOID == "" {
			return nil, nil, nil, fmt.Errorf("row %s è privo di tabella padre", rowNode.Name)
		}

		tableNode, err := a.mibDB.GetNode(tableOID)
		if err != nil {
			return nil, nil, nil, fmt.Errorf("failed to resolve table for column %s: %w", node.Name, err)
		}
		return tableNode, rowNode, columns, nil
	default:
		return nil, nil, nil, fmt.Errorf("node %s (%s) non rappresenta una tabella", node.Name, node.Type)
	}
}

// resolveTableRowAndColumns trova il nodo row e le colonne di una tabella.
func (a *App) resolveTableRowAndColumns(tableNode *mib.Node) (*mib.Node, []*mib.Node, error) {
	children, err := a.mibDB.GetChildren(tableNode.OID)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to load childrens for table %s: %w", tableNode.Name, err)
	}

	var rowNode *mib.Node
	for _, child := range children {
		if child.Type == "row" {
			rowNode = child
			break
		}
	}

	if rowNode == nil {
		return nil, nil, fmt.Errorf("table %s (%s) non definisce un nodo row", tableNode.Name, tableNode.OID)
	}

	columns, err := a.resolveRowColumns(rowNode)
	if err != nil {
		return nil, nil, err
	}

	return rowNode, columns, nil
}

// resolveRowColumns recupera tutte le colonne di un nodo row.
func (a *App) resolveRowColumns(rowNode *mib.Node) ([]*mib.Node, error) {
	children, err := a.mibDB.GetChildren(rowNode.OID)
	if err != nil {
		return nil, fmt.Errorf("failed to load columns for row %s: %w", rowNode.Name, err)
	}

	var columns []*mib.Node
	for _, child := range children {
		if child.Type == "column" {
			columns = append(columns, child)
		}
	}

	if len(columns) == 0 {
		return nil, fmt.Errorf("row %s (%s) non contiene colonne accessibili", rowNode.Name, rowNode.OID)
	}

	sort.Slice(columns, func(i, j int) bool {
		return mib.CompareOIDs(columns[i].OID, columns[j].OID) < 0
	})

	return columns, nil
}

// buildTableRows costruisce le righe della tabella dai risultati SNMP.
func buildTableRows(results []snmp.Result, columns []*mib.Node) []TableRow {
	if len(results) == 0 || len(columns) == 0 {
		return []TableRow{}
	}

	type columnInfo struct {
		name string
		oid  string
	}

	infos := make([]columnInfo, 0, len(columns))
	for _, column := range columns {
		baseOID := normalizeOIDKey(column.OID)
		if baseOID == "" {
			continue
		}
		infos = append(infos, columnInfo{
			name: column.Name,
			oid:  baseOID,
		})
	}

	rows := make(map[string]TableRow)
	order := make([]string, 0)

	for _, result := range results {
		normalizedOID := normalizeOIDKey(result.OID)
		if normalizedOID == "" {
			continue
		}

		for _, info := range infos {
			if normalizedOID != info.oid && !strings.HasPrefix(normalizedOID, info.oid+".") {
				continue
			}

			suffix := strings.TrimPrefix(normalizedOID, info.oid)
			suffix = strings.TrimPrefix(suffix, ".")
			if suffix == "" {
				suffix = "0"
			}

			row, ok := rows[suffix]
			if !ok {
				row = make(TableRow)
				row["__instance"] = suffix
				rows[suffix] = row
				order = append(order, suffix)
			}

			display := result.DisplayValue
			if display == "" {
				display = result.Value
			}
			rawValue := result.RawValue
			if rawValue == "" {
				rawValue = result.Value
			}

			row[info.name] = display
			row[fmt.Sprintf("%s__raw", info.name)] = rawValue
			break
		}
	}

	if len(rows) == 0 {
		return []TableRow{}
	}

	sortInstanceKeys(order)

	formatted := make([]TableRow, 0, len(order))
	for _, key := range order {
		formatted = append(formatted, rows[key])
	}

	return formatted
}

// makeColumnLabel genera un'etichetta leggibile dal nome di una colonna MIB.
func makeColumnLabel(name string) string {
	cleaned := strings.TrimSpace(name)
	if cleaned == "" {
		return ""
	}

	cleaned = strings.ReplaceAll(cleaned, "_", " ")
	cleaned = strings.ReplaceAll(cleaned, "-", " ")

	runes := []rune(cleaned)
	var builder strings.Builder
	builder.Grow(len(runes) + 4)

	prevWasSpace := false
	prevWasLowerOrDigit := false

	for i, r := range runes {
		if unicode.IsSpace(r) {
			if !prevWasSpace && builder.Len() > 0 {
				builder.WriteRune(' ')
				prevWasSpace = true
			}
			prevWasLowerOrDigit = false
			continue
		}

		if i == 0 {
			builder.WriteRune(unicode.ToUpper(r))
		} else if unicode.IsUpper(r) && !prevWasSpace && (prevWasLowerOrDigit || (i+1 < len(runes) && unicode.IsLower(runes[i+1]))) {
			builder.WriteRune(' ')
			builder.WriteRune(r)
		} else {
			builder.WriteRune(r)
		}

		prevWasSpace = false
		prevWasLowerOrDigit = unicode.IsLower(r) || unicode.IsDigit(r)
	}

	return builder.String()
}

// inferColumnValueType deduce il tipo di dato di una colonna dalla sua sintassi.
func inferColumnValueType(syntax string) string {
	if syntax == "" {
		return "string"
	}

	lowered := strings.ToLower(syntax)
	hints := []string{
		"int",
		"counter",
		"gauge",
		"unsigned",
		"timeticks",
		"time ticks",
		"float",
		"double",
		"bits",
		"enum",
		"numeric",
	}

	for _, hint := range hints {
		if strings.Contains(lowered, hint) {
			return "number"
		}
	}

	return "string"
}

// compareIndexPaths confronta due indici di tabella numericamente.
func compareIndexPaths(a, b string) int {
	if a == b {
		return 0
	}

	partsA := strings.Split(a, ".")
	partsB := strings.Split(b, ".")
	limit := len(partsA)
	if len(partsB) < limit {
		limit = len(partsB)
	}

	for i := 0; i < limit; i++ {
		segmentA := partsA[i]
		segmentB := partsB[i]

		intA, errA := strconv.Atoi(segmentA)
		intB, errB := strconv.Atoi(segmentB)

		if errA == nil && errB == nil {
			if intA < intB {
				return -1
			}
			if intA > intB {
				return 1
			}
			continue
		}

		if segmentA < segmentB {
			return -1
		}
		if segmentA > segmentB {
			return 1
		}
	}

	if len(partsA) < len(partsB) {
		return -1
	}
	if len(partsA) > len(partsB) {
		return 1
	}
	return 0
}

// sortInstanceKeys ordina le chiavi di istanza in ordine naturale.
func sortInstanceKeys(keys []string) {
	sort.Slice(keys, func(i, j int) bool {
		return compareIndexPaths(keys[i], keys[j]) < 0
	})
}
