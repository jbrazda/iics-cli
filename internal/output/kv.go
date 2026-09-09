package output

// KVRow is a single property/value pair for vertical (attribute listing) tables.
type KVRow struct {
	Property string `json:"property"`
	Value    string `json:"value"`
}

// KVCols is the standard two-column layout for a vertical property/value table.
var KVCols = []Column{
	{Header: "PROPERTY", Field: "property", Width: 24},
	{Header: "VALUE", Field: "value"},
}

// KV builds a KVRow.
func KV(property, value string) KVRow {
	return KVRow{Property: property, Value: value}
}
