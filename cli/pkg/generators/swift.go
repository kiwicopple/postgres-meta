package generators

import (
	"fmt"
	"sort"
	"strings"

	"github.com/supabase/postgres-meta/cli/pkg/pgmeta"
)

// SwiftGenerator generates Swift struct definitions
type SwiftGenerator struct{}

// Name returns the generator name
func (g *SwiftGenerator) Name() string {
	return "swift"
}

// Generate generates Swift struct definitions from the metadata
func (g *SwiftGenerator) Generate(meta *pgmeta.GeneratorMetadata, opts Options) (string, error) {
	ctx := newSwiftContext(meta, opts)
	return ctx.generate()
}

// swiftContext holds the context for Swift generation
type swiftContext struct {
	meta          *pgmeta.GeneratorMetadata
	columns       map[int64][]pgmeta.PostgresColumn
	accessControl string
}

func newSwiftContext(meta *pgmeta.GeneratorMetadata, opts Options) *swiftContext {
	ctx := &swiftContext{
		meta:          meta,
		columns:       make(map[int64][]pgmeta.PostgresColumn),
		accessControl: opts.AccessControl,
	}

	if ctx.accessControl == "" {
		ctx.accessControl = "internal"
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

func (ctx *swiftContext) generate() (string, error) {
	var lines []string

	lines = append(lines, "import Foundation")
	lines = append(lines, "import Supabase")
	lines = append(lines, "")

	// Sort schemas
	schemas := make([]pgmeta.PostgresSchema, len(ctx.meta.Schemas))
	copy(schemas, ctx.meta.Schemas)
	sort.Slice(schemas, func(i, j int) bool {
		return schemas[i].Name < schemas[j].Name
	})

	for _, schema := range schemas {
		schemaLines := ctx.generateSchemaEnum(schema)
		lines = append(lines, schemaLines...)
	}

	return strings.Join(lines, "\n"), nil
}

func (ctx *swiftContext) generateSchemaEnum(schema pgmeta.PostgresSchema) []string {
	var lines []string

	schemaName := formatForSwiftSchemaName(schema.Name)
	lines = append(lines, fmt.Sprintf("%s enum %s {", ctx.accessControl, schemaName))

	// Get tables for this schema
	var schemaTables []pgmeta.PostgresTable
	for _, t := range ctx.meta.Tables {
		if t.Schema == schema.Name {
			schemaTables = append(schemaTables, t)
		}
	}
	sort.Slice(schemaTables, func(i, j int) bool {
		return schemaTables[i].Name < schemaTables[j].Name
	})

	// Get foreign tables for this schema
	var schemaForeignTables []pgmeta.PostgresForeignTable
	for _, t := range ctx.meta.ForeignTables {
		if t.Schema == schema.Name {
			schemaForeignTables = append(schemaForeignTables, t)
		}
	}
	sort.Slice(schemaForeignTables, func(i, j int) bool {
		return schemaForeignTables[i].Name < schemaForeignTables[j].Name
	})

	// Get views for this schema
	var schemaViews []pgmeta.PostgresView
	for _, v := range ctx.meta.Views {
		if v.Schema == schema.Name {
			schemaViews = append(schemaViews, v)
		}
	}
	sort.Slice(schemaViews, func(i, j int) bool {
		return schemaViews[i].Name < schemaViews[j].Name
	})

	// Get materialized views
	var schemaMatViews []pgmeta.PostgresMaterializedView
	for _, v := range ctx.meta.MaterializedViews {
		if v.Schema == schema.Name {
			schemaMatViews = append(schemaMatViews, v)
		}
	}
	sort.Slice(schemaMatViews, func(i, j int) bool {
		return schemaMatViews[i].Name < schemaMatViews[j].Name
	})

	// Get enums for this schema
	var schemaEnums []pgmeta.PostgresType
	for _, t := range ctx.meta.Types {
		if t.Schema == schema.Name && len(t.Enums) > 0 {
			schemaEnums = append(schemaEnums, t)
		}
	}
	sort.Slice(schemaEnums, func(i, j int) bool {
		return schemaEnums[i].Name < schemaEnums[j].Name
	})

	// Get composite types for this schema
	var schemaCompositeTypes []pgmeta.PostgresType
	for _, t := range ctx.meta.Types {
		if t.Schema == schema.Name && len(t.Attributes) > 0 {
			schemaCompositeTypes = append(schemaCompositeTypes, t)
		}
	}
	sort.Slice(schemaCompositeTypes, func(i, j int) bool {
		return schemaCompositeTypes[i].Name < schemaCompositeTypes[j].Name
	})

	// Generate enums
	for _, enum := range schemaEnums {
		enumLines := ctx.generateEnum(enum, 1)
		lines = append(lines, enumLines...)
	}

	// Generate table structs
	for _, table := range schemaTables {
		cols := ctx.columns[table.ID]
		for _, op := range []string{"Select", "Insert", "Update"} {
			structLines := ctx.generateStruct(table.Name, cols, op, 1)
			lines = append(lines, structLines...)
		}
	}

	// Generate foreign table structs
	for _, table := range schemaForeignTables {
		cols := ctx.columns[table.ID]
		for _, op := range []string{"Select", "Insert", "Update"} {
			structLines := ctx.generateStruct(table.Name, cols, op, 1)
			lines = append(lines, structLines...)
		}
	}

	// Generate view structs (Select only)
	for _, view := range schemaViews {
		cols := ctx.columns[view.ID]
		structLines := ctx.generateStruct(view.Name, cols, "Select", 1)
		lines = append(lines, structLines...)
	}

	// Generate materialized view structs (Select only)
	for _, matview := range schemaMatViews {
		cols := ctx.columns[matview.ID]
		structLines := ctx.generateStruct(matview.Name, cols, "Select", 1)
		lines = append(lines, structLines...)
	}

	// Generate composite type structs
	for _, ct := range schemaCompositeTypes {
		structLines := ctx.generateCompositeTypeStruct(ct, 1)
		lines = append(lines, structLines...)
	}

	lines = append(lines, "}")

	return lines
}

func (ctx *swiftContext) generateEnum(enum pgmeta.PostgresType, level int) []string {
	indent := strings.Repeat("  ", level)
	var lines []string

	enumName := formatForSwiftTypeName(enum.Name)
	lines = append(lines, fmt.Sprintf("%s%s enum %s: String, Codable, Hashable, Sendable {",
		indent, ctx.accessControl, enumName))

	for _, variant := range enum.Enums {
		caseName := formatForSwiftPropertyName(variant)
		lines = append(lines, fmt.Sprintf("%s  case %s = \"%s\"", indent, caseName, variant))
	}

	lines = append(lines, fmt.Sprintf("%s}", indent))

	return lines
}

type swiftAttribute struct {
	formattedName string
	formattedType string
	rawName       string
	isIdentity    bool
}

func (ctx *swiftContext) generateStruct(tableName string, columns []pgmeta.PostgresColumn, operation string, level int) []string {
	indent := strings.Repeat("  ", level)
	var lines []string

	structName := formatForSwiftTypeName(tableName) + operation

	// Build attributes
	var attrs []swiftAttribute
	for _, col := range columns {
		var nullable bool
		switch operation {
		case "Insert":
			nullable = col.IsNullable || col.IsIdentity || col.IsGenerated || col.DefaultValue != nil
		case "Update":
			nullable = true
		default:
			nullable = col.IsNullable
		}

		attrs = append(attrs, swiftAttribute{
			formattedName: formatForSwiftPropertyName(col.Name),
			formattedType: ctx.pgTypeToSwiftType(col.Format, nullable),
			rawName:       col.Name,
			isIdentity:    col.IsIdentity,
		})
	}

	// Determine protocol conformances
	protocols := []string{"Codable", "Hashable", "Sendable"}
	var identityAttr *swiftAttribute
	for i := range attrs {
		if attrs[i].isIdentity {
			identityAttr = &attrs[i]
			break
		}
	}
	if identityAttr != nil {
		protocols = append(protocols, "Identifiable")
	}

	lines = append(lines, fmt.Sprintf("%s%s struct %s: %s {",
		indent, ctx.accessControl, structName, strings.Join(protocols, ", ")))

	// Add id computed property if identity column is not named "id"
	if identityAttr != nil && identityAttr.formattedName != "id" {
		lines = append(lines, fmt.Sprintf("%s  %s var id: %s { %s }",
			indent, ctx.accessControl, identityAttr.formattedType, identityAttr.formattedName))
	}

	// Add properties
	for _, attr := range attrs {
		lines = append(lines, fmt.Sprintf("%s  %s let %s: %s",
			indent, ctx.accessControl, attr.formattedName, attr.formattedType))
	}

	// Add CodingKeys enum if there are attributes
	if len(attrs) > 0 {
		lines = append(lines, fmt.Sprintf("%s  %s enum CodingKeys: String, CodingKey {",
			indent, ctx.accessControl))
		for _, attr := range attrs {
			lines = append(lines, fmt.Sprintf("%s    case %s = \"%s\"",
				indent, attr.formattedName, attr.rawName))
		}
		lines = append(lines, fmt.Sprintf("%s  }", indent))
	}

	lines = append(lines, fmt.Sprintf("%s}", indent))

	return lines
}

func (ctx *swiftContext) generateCompositeTypeStruct(t pgmeta.PostgresType, level int) []string {
	indent := strings.Repeat("  ", level)
	var lines []string

	structName := formatForSwiftTypeName(t.Name)

	// Build attributes
	var attrs []swiftAttribute
	for _, attr := range t.Attributes {
		// Find type by ID
		var typeName string
		for _, tt := range ctx.meta.Types {
			if tt.ID == attr.TypeID {
				typeName = tt.Name
				break
			}
		}
		if typeName == "" {
			typeName = "AnyJSON"
		}

		attrs = append(attrs, swiftAttribute{
			formattedName: formatForSwiftTypeName(attr.Name),
			formattedType: ctx.pgTypeToSwiftType(typeName, false),
			rawName:       attr.Name,
			isIdentity:    false,
		})
	}

	protocols := []string{"Codable", "Hashable", "Sendable"}

	lines = append(lines, fmt.Sprintf("%s%s struct %s: %s {",
		indent, ctx.accessControl, structName, strings.Join(protocols, ", ")))

	// Add properties
	for _, attr := range attrs {
		lines = append(lines, fmt.Sprintf("%s  %s let %s: %s",
			indent, ctx.accessControl, attr.formattedName, attr.formattedType))
	}

	// Add CodingKeys enum if there are attributes
	if len(attrs) > 0 {
		lines = append(lines, fmt.Sprintf("%s  %s enum CodingKeys: String, CodingKey {",
			indent, ctx.accessControl))
		for _, attr := range attrs {
			lines = append(lines, fmt.Sprintf("%s    case %s = \"%s\"",
				indent, attr.formattedName, attr.rawName))
		}
		lines = append(lines, fmt.Sprintf("%s  }", indent))
	}

	lines = append(lines, fmt.Sprintf("%s}", indent))

	return lines
}

func (ctx *swiftContext) pgTypeToSwiftType(pgType string, nullable bool) string {
	var swiftType string

	switch pgType {
	case "bool":
		swiftType = "Bool"
	case "int2":
		swiftType = "Int16"
	case "int4":
		swiftType = "Int32"
	case "int8":
		swiftType = "Int64"
	case "float4":
		swiftType = "Float"
	case "float8":
		swiftType = "Double"
	case "numeric", "decimal":
		swiftType = "Decimal"
	case "uuid":
		swiftType = "UUID"
	case "bytea", "bpchar", "varchar", "date", "text", "citext", "time", "timetz", "timestamp", "timestamptz", "vector":
		swiftType = "String"
	case "json", "jsonb":
		swiftType = "AnyJSON"
	case "void":
		swiftType = "Void"
	case "record":
		swiftType = "JSONObject"
	default:
		// Handle array types
		if strings.HasPrefix(pgType, "_") {
			innerType := ctx.pgTypeToSwiftType(pgType[1:], false)
			swiftType = fmt.Sprintf("[%s]", innerType)
		} else {
			// Check for enum type
			for _, t := range ctx.meta.Types {
				if t.Name == pgType && len(t.Enums) > 0 {
					swiftType = formatForSwiftTypeName(t.Name)
					break
				}
			}

			// Check for composite type or table
			if swiftType == "" {
				for _, t := range ctx.meta.Types {
					if t.Name == pgType {
						swiftType = formatForSwiftTypeName(t.Name) + "Select"
						break
					}
				}
			}
			if swiftType == "" {
				for _, t := range ctx.meta.Tables {
					if t.Name == pgType {
						swiftType = formatForSwiftTypeName(t.Name) + "Select"
						break
					}
				}
			}
			if swiftType == "" {
				for _, v := range ctx.meta.Views {
					if v.Name == pgType {
						swiftType = formatForSwiftTypeName(v.Name) + "Select"
						break
					}
				}
			}

			if swiftType == "" {
				swiftType = "AnyJSON"
			}
		}
	}

	if nullable {
		return swiftType + "?"
	}
	return swiftType
}

func formatForSwiftSchemaName(name string) string {
	return formatForSwiftTypeName(name) + "Schema"
}

func formatForSwiftTypeName(name string) string {
	// Preserve initial underscore if it exists
	prefix := ""
	if strings.HasPrefix(name, "_") {
		prefix = "_"
		name = name[1:]
	}

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
	return prefix + result.String()
}

var swiftKeywords = map[string]bool{
	"in":      true,
	"default": true,
	"case":    true,
}

func formatForSwiftPropertyName(name string) string {
	parts := nonAlphanumericRegex.Split(name, -1)
	var result strings.Builder
	for i, part := range parts {
		if len(part) > 0 {
			lower := strings.ToLower(part)
			if i != 0 {
				result.WriteString(strings.ToUpper(lower[:1]))
				if len(lower) > 1 {
					result.WriteString(lower[1:])
				}
			} else {
				result.WriteString(lower)
			}
		}
	}

	propName := result.String()
	if swiftKeywords[propName] {
		return "`" + propName + "`"
	}
	return propName
}
