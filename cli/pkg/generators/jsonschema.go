package generators

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/supabase/postgres-meta/cli/pkg/pgmeta"
)

// JSONSchemaGenerator generates JSON Schema definitions
type JSONSchemaGenerator struct{}

// Name returns the generator name
func (g *JSONSchemaGenerator) Name() string {
	return "jsonschema"
}

// Generate generates JSON Schema definitions from the metadata
func (g *JSONSchemaGenerator) Generate(meta *pgmeta.GeneratorMetadata, opts Options) (string, error) {
	ctx := newJSONSchemaContext(meta)
	return ctx.generate()
}

// jsonSchemaContext holds the context for JSON Schema generation
type jsonSchemaContext struct {
	meta        *pgmeta.GeneratorMetadata
	types       map[string]pgmeta.PostgresType
	userEnums   map[string][]string
	columns     map[int64][]pgmeta.PostgresColumn
	schemas     map[string]pgmeta.PostgresSchema
	schemaNames map[string]bool
}

func newJSONSchemaContext(meta *pgmeta.GeneratorMetadata) *jsonSchemaContext {
	ctx := &jsonSchemaContext{
		meta:        meta,
		types:       make(map[string]pgmeta.PostgresType),
		userEnums:   make(map[string][]string),
		columns:     make(map[int64][]pgmeta.PostgresColumn),
		schemas:     make(map[string]pgmeta.PostgresSchema),
		schemaNames: make(map[string]bool),
	}

	// Build indexes
	for _, schema := range meta.Schemas {
		ctx.schemas[schema.Name] = schema
		ctx.schemaNames[schema.Name] = true
	}

	for _, t := range meta.Types {
		ctx.types[t.Name] = t
		if len(t.Enums) > 0 {
			ctx.userEnums[t.Schema+"."+t.Name] = t.Enums
		}
	}

	// Sort and group columns by table_id
	sortedColumns := make([]pgmeta.PostgresColumn, len(meta.Columns))
	copy(sortedColumns, meta.Columns)
	sort.Slice(sortedColumns, func(i, j int) bool {
		return sortedColumns[i].OrdinalPosition < sortedColumns[j].OrdinalPosition
	})

	for _, col := range sortedColumns {
		ctx.columns[col.TableID] = append(ctx.columns[col.TableID], col)
	}

	return ctx
}

// JSONSchema represents a JSON Schema document
type JSONSchema struct {
	Schema      string                 `json:"$schema"`
	ID          string                 `json:"$id,omitempty"`
	Title       string                 `json:"title,omitempty"`
	Description string                 `json:"description,omitempty"`
	Type        string                 `json:"type,omitempty"`
	Definitions map[string]*JSONSchema `json:"$defs,omitempty"`
	Properties  map[string]*JSONSchema `json:"properties,omitempty"`
	Required    []string               `json:"required,omitempty"`
	Enum        []string               `json:"enum,omitempty"`
	Items       *JSONSchema            `json:"items,omitempty"`
	Format      string                 `json:"format,omitempty"`
	Ref         string                 `json:"$ref,omitempty"`
	OneOf       []*JSONSchema          `json:"oneOf,omitempty"`
	AnyOf       []*JSONSchema          `json:"anyOf,omitempty"`
}

func (ctx *jsonSchemaContext) generate() (string, error) {
	root := &JSONSchema{
		Schema:      "https://json-schema.org/draft/2020-12/schema",
		Title:       "Database Schema",
		Description: "JSON Schema definitions for database tables",
		Type:        "object",
		Definitions: make(map[string]*JSONSchema),
		Properties:  make(map[string]*JSONSchema),
	}

	// Generate enum definitions
	for _, t := range ctx.meta.Types {
		if len(t.Enums) > 0 && ctx.schemaNames[t.Schema] {
			defName := toSchemaName(t.Schema, t.Name)
			root.Definitions[defName] = &JSONSchema{
				Title:       t.Name,
				Description: fmt.Sprintf("Enum type: %s.%s", t.Schema, t.Name),
				Type:        "string",
				Enum:        t.Enums,
			}
		}
	}

	// Generate table schemas
	for _, table := range ctx.meta.Tables {
		if !ctx.schemaNames[table.Schema] {
			continue
		}

		tableName := toSchemaName(table.Schema, table.Name)

		// Row schema (for reading)
		rowSchema := ctx.generateTableSchema(table.ID, table.Schema, table.Name, "row")
		root.Definitions[tableName+"Row"] = rowSchema

		// Insert schema (for creating)
		insertSchema := ctx.generateTableSchema(table.ID, table.Schema, table.Name, "insert")
		root.Definitions[tableName+"Insert"] = insertSchema

		// Update schema (for updating)
		updateSchema := ctx.generateTableSchema(table.ID, table.Schema, table.Name, "update")
		root.Definitions[tableName+"Update"] = updateSchema

		// Add reference to properties
		root.Properties[tableName] = &JSONSchema{
			Type: "object",
			Properties: map[string]*JSONSchema{
				"Row":    {Ref: "#/$defs/" + tableName + "Row"},
				"Insert": {Ref: "#/$defs/" + tableName + "Insert"},
				"Update": {Ref: "#/$defs/" + tableName + "Update"},
			},
		}
	}

	// Generate view schemas
	for _, view := range ctx.meta.Views {
		if !ctx.schemaNames[view.Schema] {
			continue
		}
		viewName := toSchemaName(view.Schema, view.Name)
		viewSchema := ctx.generateViewSchema(view.ID, view.Schema, view.Name)
		root.Definitions[viewName] = viewSchema
		root.Properties[viewName] = &JSONSchema{
			Ref: "#/$defs/" + viewName,
		}
	}

	// Generate materialized view schemas
	for _, matview := range ctx.meta.MaterializedViews {
		if !ctx.schemaNames[matview.Schema] {
			continue
		}
		matviewName := toSchemaName(matview.Schema, matview.Name)
		matviewSchema := ctx.generateMatViewSchema(matview.ID, matview.Schema, matview.Name)
		root.Definitions[matviewName] = matviewSchema
		root.Properties[matviewName] = &JSONSchema{
			Ref: "#/$defs/" + matviewName,
		}
	}

	// Marshal to JSON with indentation
	output, err := json.MarshalIndent(root, "", "  ")
	if err != nil {
		return "", err
	}

	return string(output) + "\n", nil
}

func (ctx *jsonSchemaContext) generateTableSchema(tableID int64, schema, name, mode string) *JSONSchema {
	cols := ctx.columns[tableID]

	tableSchema := &JSONSchema{
		Title:      fmt.Sprintf("%s.%s (%s)", schema, name, mode),
		Type:       "object",
		Properties: make(map[string]*JSONSchema),
		Required:   []string{},
	}

	for _, col := range cols {
		propSchema := ctx.columnToSchema(col)

		// Handle nullability
		if col.IsNullable {
			propSchema = &JSONSchema{
				AnyOf: []*JSONSchema{
					propSchema,
					{Type: "null"},
				},
			}
		}

		tableSchema.Properties[col.Name] = propSchema

		// Determine required fields based on mode
		switch mode {
		case "row":
			// All non-nullable fields are required in row
			if !col.IsNullable {
				tableSchema.Required = append(tableSchema.Required, col.Name)
			}
		case "insert":
			// Required for insert if not nullable, not identity, not generated, no default
			if !col.IsNullable && !col.IsIdentity && !col.IsGenerated && col.DefaultValue == nil {
				tableSchema.Required = append(tableSchema.Required, col.Name)
			}
		case "update":
			// Nothing required for update (all fields optional)
		}
	}

	if len(tableSchema.Required) == 0 {
		tableSchema.Required = nil
	}

	return tableSchema
}

func (ctx *jsonSchemaContext) generateViewSchema(viewID int64, schema, name string) *JSONSchema {
	cols := ctx.columns[viewID]

	viewSchema := &JSONSchema{
		Title:       fmt.Sprintf("%s.%s (view)", schema, name),
		Description: "View (read-only)",
		Type:        "object",
		Properties:  make(map[string]*JSONSchema),
		Required:    []string{},
	}

	for _, col := range cols {
		propSchema := ctx.columnToSchema(col)

		if col.IsNullable {
			propSchema = &JSONSchema{
				AnyOf: []*JSONSchema{
					propSchema,
					{Type: "null"},
				},
			}
		}

		viewSchema.Properties[col.Name] = propSchema

		if !col.IsNullable {
			viewSchema.Required = append(viewSchema.Required, col.Name)
		}
	}

	if len(viewSchema.Required) == 0 {
		viewSchema.Required = nil
	}

	return viewSchema
}

func (ctx *jsonSchemaContext) generateMatViewSchema(matviewID int64, schema, name string) *JSONSchema {
	cols := ctx.columns[matviewID]

	matviewSchema := &JSONSchema{
		Title:       fmt.Sprintf("%s.%s (materialized view)", schema, name),
		Description: "Materialized view (read-only)",
		Type:        "object",
		Properties:  make(map[string]*JSONSchema),
		Required:    []string{},
	}

	for _, col := range cols {
		propSchema := ctx.columnToSchema(col)

		if col.IsNullable {
			propSchema = &JSONSchema{
				AnyOf: []*JSONSchema{
					propSchema,
					{Type: "null"},
				},
			}
		}

		matviewSchema.Properties[col.Name] = propSchema

		if !col.IsNullable {
			matviewSchema.Required = append(matviewSchema.Required, col.Name)
		}
	}

	if len(matviewSchema.Required) == 0 {
		matviewSchema.Required = nil
	}

	return matviewSchema
}

func (ctx *jsonSchemaContext) columnToSchema(col pgmeta.PostgresColumn) *JSONSchema {
	format := col.Format

	// Check if it's an array type
	if strings.HasPrefix(format, "_") {
		innerType := format[1:]
		return &JSONSchema{
			Type:  "array",
			Items: ctx.pgTypeToSchema(innerType, col.Schema),
		}
	}

	return ctx.pgTypeToSchema(format, col.Schema)
}

func (ctx *jsonSchemaContext) pgTypeToSchema(pgType, schema string) *JSONSchema {
	// Check if it's an enum
	if enums, ok := ctx.userEnums[schema+"."+pgType]; ok {
		return &JSONSchema{
			Type: "string",
			Enum: enums,
		}
	}

	// Map PostgreSQL types to JSON Schema types
	switch pgType {
	// Boolean
	case "bool":
		return &JSONSchema{Type: "boolean"}

	// Integer types
	case "int2", "int4", "int8":
		return &JSONSchema{Type: "integer"}

	// Floating point
	case "float4", "float8", "numeric":
		return &JSONSchema{Type: "number"}

	// String types
	case "varchar", "bpchar", "text", "citext", "name":
		return &JSONSchema{Type: "string"}

	// Date/Time types
	case "date":
		return &JSONSchema{Type: "string", Format: "date"}
	case "time", "timetz":
		return &JSONSchema{Type: "string", Format: "time"}
	case "timestamp", "timestamptz":
		return &JSONSchema{Type: "string", Format: "date-time"}

	// UUID
	case "uuid":
		return &JSONSchema{Type: "string", Format: "uuid"}

	// JSON types
	case "json", "jsonb":
		return &JSONSchema{} // Any type

	// Binary
	case "bytea":
		return &JSONSchema{Type: "string", Format: "byte"}

	// Network types
	case "inet", "cidr":
		return &JSONSchema{Type: "string", Format: "ipv4"}
	case "macaddr", "macaddr8":
		return &JSONSchema{Type: "string"}

	// Geometric types
	case "point", "line", "lseg", "box", "path", "polygon", "circle":
		return &JSONSchema{Type: "string"}

	// Range types
	case "int4range", "int8range", "numrange", "tsrange", "tstzrange", "daterange":
		return &JSONSchema{Type: "string"}

	// Other types
	case "money":
		return &JSONSchema{Type: "string"}
	case "interval":
		return &JSONSchema{Type: "string", Format: "duration"}
	case "xml":
		return &JSONSchema{Type: "string"}
	case "tsvector", "tsquery":
		return &JSONSchema{Type: "string"}

	default:
		// For unknown types, use string
		return &JSONSchema{Type: "string"}
	}
}

func toSchemaName(schema, name string) string {
	if schema == "public" {
		return toPascalCase(name)
	}
	return toPascalCase(schema) + toPascalCase(name)
}

func toPascalCase(s string) string {
	var result strings.Builder
	capitalizeNext := true

	for _, r := range s {
		if r == '_' || r == '-' || r == ' ' {
			capitalizeNext = true
			continue
		}
		if capitalizeNext {
			result.WriteRune(toUpper(r))
			capitalizeNext = false
		} else {
			result.WriteRune(r)
		}
	}
	return result.String()
}

func toUpper(r rune) rune {
	if r >= 'a' && r <= 'z' {
		return r - 32
	}
	return r
}
