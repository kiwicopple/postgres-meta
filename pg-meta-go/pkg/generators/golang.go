package generators

import (
	"fmt"
	"sort"
	"strings"

	"github.com/supabase/postgres-meta/pg-meta-go/pkg/pgmeta"
)

// GoGenerator generates Go struct definitions
type GoGenerator struct{}

// Name returns the generator name
func (g *GoGenerator) Name() string {
	return "go"
}

// Generate generates Go struct definitions from the metadata
func (g *GoGenerator) Generate(meta *pgmeta.GeneratorMetadata, opts Options) (string, error) {
	ctx := newGoContext(meta)
	return ctx.generate()
}

// goContext holds the context for Go generation
type goContext struct {
	meta    *pgmeta.GeneratorMetadata
	columns map[int64][]pgmeta.PostgresColumn
}

func newGoContext(meta *pgmeta.GeneratorMetadata) *goContext {
	ctx := &goContext{
		meta:    meta,
		columns: make(map[int64][]pgmeta.PostgresColumn),
	}

	// Sort and group columns by table_id
	sortedColumns := make([]pgmeta.PostgresColumn, len(meta.Columns))
	copy(sortedColumns, meta.Columns)
	sort.Slice(sortedColumns, func(i, j int) bool {
		return sortedColumns[i].Name < sortedColumns[j].Name
	})

	for _, col := range sortedColumns {
		ctx.columns[col.TableID] = append(ctx.columns[col.TableID], col)
	}

	return ctx
}

func (ctx *goContext) generate() (string, error) {
	var sb strings.Builder

	sb.WriteString("package database\n\n")

	// Generate table structs
	for _, table := range ctx.meta.Tables {
		if !ctx.schemaExists(table.Schema) {
			continue
		}
		schema := ctx.findSchema(table.Schema)
		cols := ctx.columns[table.ID]

		sb.WriteString(ctx.generateTableStruct(schema, table.Name, cols, "Select"))
		sb.WriteString("\n\n")
		sb.WriteString(ctx.generateTableStruct(schema, table.Name, cols, "Insert"))
		sb.WriteString("\n\n")
		sb.WriteString(ctx.generateTableStruct(schema, table.Name, cols, "Update"))
		sb.WriteString("\n\n")
	}

	// Generate view structs
	for _, view := range ctx.meta.Views {
		if !ctx.schemaExists(view.Schema) {
			continue
		}
		schema := ctx.findSchema(view.Schema)
		cols := ctx.columns[view.ID]

		sb.WriteString(ctx.generateTableStruct(schema, view.Name, cols, "Select"))
		sb.WriteString("\n\n")
	}

	// Generate materialized view structs
	for _, matview := range ctx.meta.MaterializedViews {
		if !ctx.schemaExists(matview.Schema) {
			continue
		}
		schema := ctx.findSchema(matview.Schema)
		cols := ctx.columns[matview.ID]

		sb.WriteString(ctx.generateTableStruct(schema, matview.Name, cols, "Select"))
		sb.WriteString("\n\n")
	}

	// Generate composite type structs
	for _, t := range ctx.meta.Types {
		if len(t.Attributes) > 0 && ctx.schemaExists(t.Schema) {
			schema := ctx.findSchema(t.Schema)
			sb.WriteString(ctx.generateCompositeTypeStruct(schema, t))
			sb.WriteString("\n\n")
		}
	}

	return strings.TrimRight(sb.String(), "\n") + "\n", nil
}

func (ctx *goContext) schemaExists(schemaName string) bool {
	for _, s := range ctx.meta.Schemas {
		if s.Name == schemaName {
			return true
		}
	}
	return false
}

func (ctx *goContext) findSchema(name string) pgmeta.PostgresSchema {
	for _, s := range ctx.meta.Schemas {
		if s.Name == name {
			return s
		}
	}
	return pgmeta.PostgresSchema{}
}

type goColumnEntry struct {
	formattedName string
	goType        string
	jsonName      string
}

func (ctx *goContext) generateTableStruct(schema pgmeta.PostgresSchema, tableName string, columns []pgmeta.PostgresColumn, operation string) string {
	structName := formatForGoTypeName(schema.Name) + formatForGoTypeName(tableName) + operation

	entries := make([]goColumnEntry, len(columns))
	maxNameLen := 0
	maxTypeLen := 0

	for i, col := range columns {
		var nullable bool
		switch operation {
		case "Insert":
			nullable = col.IsNullable || col.IsIdentity || col.IsGenerated || col.DefaultValue != nil
		case "Update":
			nullable = true
		default:
			nullable = col.IsNullable
		}

		formattedName := formatForGoTypeName(col.Name)
		goType := ctx.pgTypeToGoType(col.Format, nullable)

		entries[i] = goColumnEntry{
			formattedName: formattedName,
			goType:        goType,
			jsonName:      col.Name,
		}

		if len(formattedName) > maxNameLen {
			maxNameLen = len(formattedName)
		}
		if len(goType) > maxTypeLen {
			maxTypeLen = len(goType)
		}
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("type %s struct {\n", structName))
	for _, entry := range entries {
		sb.WriteString(fmt.Sprintf("  %-*s %-*s `json:\"%s\"`\n",
			maxNameLen, entry.formattedName,
			maxTypeLen, entry.goType,
			entry.jsonName))
	}
	sb.WriteString("}")

	return sb.String()
}

func (ctx *goContext) generateCompositeTypeStruct(schema pgmeta.PostgresSchema, t pgmeta.PostgresType) string {
	structName := formatForGoTypeName(schema.Name) + formatForGoTypeName(t.Name)

	entries := make([]goColumnEntry, len(t.Attributes))
	maxNameLen := 0
	maxTypeLen := 0

	for i, attr := range t.Attributes {
		// Find the type by ID
		var typeName string
		for _, tt := range ctx.meta.Types {
			if tt.ID == attr.TypeID {
				typeName = tt.Name
				break
			}
		}
		if typeName == "" {
			typeName = "interface{}"
		}

		formattedName := formatForGoTypeName(attr.Name)
		goType := ctx.pgTypeToGoType(typeName, false)

		entries[i] = goColumnEntry{
			formattedName: formattedName,
			goType:        goType,
			jsonName:      attr.Name,
		}

		if len(formattedName) > maxNameLen {
			maxNameLen = len(formattedName)
		}
		if len(goType) > maxTypeLen {
			maxTypeLen = len(goType)
		}
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("type %s struct {\n", structName))
	for _, entry := range entries {
		sb.WriteString(fmt.Sprintf("  %-*s %-*s `json:\"%s\"`\n",
			maxNameLen, entry.formattedName,
			maxTypeLen, entry.goType,
			entry.jsonName))
	}
	sb.WriteString("}")

	return sb.String()
}

func (ctx *goContext) pgTypeToGoType(pgType string, nullable bool) string {
	var goType string

	// Check base type map
	if t, ok := goTypeMap[pgType]; ok {
		goType = t
	}

	// Check for enum type
	if goType == "" {
		for _, t := range ctx.meta.Types {
			if t.Name == pgType && len(t.Enums) > 0 {
				goType = "string"
				break
			}
		}
	}

	// Handle nullable types
	if goType != "" {
		if nullable {
			if nt, ok := goNullableTypeMap[goType]; ok {
				return nt
			}
		}
		return goType
	}

	// Check for composite type
	for _, t := range ctx.meta.Types {
		if t.Name == pgType && len(t.Attributes) > 0 {
			return "map[string]interface{}"
		}
	}

	// Handle array types
	if strings.HasPrefix(pgType, "_") {
		innerType := ctx.pgTypeToGoType(pgType[1:], false)
		return "[]" + innerType
	}

	// Fallback
	return "interface{}"
}

func formatForGoTypeName(name string) string {
	parts := nonAlphanumericRegex.Split(name, -1)
	var result strings.Builder
	for _, part := range parts {
		if len(part) > 0 {
			result.WriteString(strings.ToUpper(part[:1]))
			if len(part) > 1 {
				result.WriteString(part[1:])
			}
		}
	}
	return result.String()
}

var goTypeMap = map[string]string{
	// Bool
	"bool": "bool",

	// Numbers
	"int2":    "int16",
	"int4":    "int32",
	"int8":    "int64",
	"float4":  "float32",
	"float8":  "float64",
	"numeric": "float64",

	// Strings
	"bytea":       "[]byte",
	"bpchar":      "string",
	"varchar":     "string",
	"date":        "string",
	"text":        "string",
	"citext":      "string",
	"time":        "string",
	"timetz":      "string",
	"timestamp":   "string",
	"timestamptz": "string",
	"uuid":        "string",
	"vector":      "string",

	// JSON
	"json":  "interface{}",
	"jsonb": "interface{}",

	// Range
	"int4range":      "string",
	"int4multirange": "string",
	"int8range":      "string",
	"int8multirange": "string",
	"numrange":       "string",
	"nummultirange":  "string",
	"tsrange":        "string",
	"tsmultirange":   "string",
	"tstzrange":      "string",
	"tstzmultirange": "string",
	"daterange":      "string",
	"datemultirange": "string",

	// Misc
	"void":   "interface{}",
	"record": "map[string]interface{}",
}

var goNullableTypeMap = map[string]string{
	"string":                  "*string",
	"bool":                    "*bool",
	"int16":                   "*int16",
	"int32":                   "*int32",
	"int64":                   "*int64",
	"float32":                 "*float32",
	"float64":                 "*float64",
	"[]byte":                  "[]byte",
	"interface{}":             "interface{}",
	"map[string]interface{}":  "map[string]interface{}",
}
