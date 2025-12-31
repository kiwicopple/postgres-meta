package generators

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/supabase/postgres-meta/pg-meta-go/pkg/pgmeta"
)

// TypeScriptGenerator generates TypeScript type definitions
type TypeScriptGenerator struct{}

// Name returns the generator name
func (g *TypeScriptGenerator) Name() string {
	return "typescript"
}

// Generate generates TypeScript type definitions from the metadata
func (g *TypeScriptGenerator) Generate(meta *pgmeta.GeneratorMetadata, opts Options) (string, error) {
	ctx := newTsContext(meta, opts)
	return ctx.generate()
}

// tsContext holds the context for TypeScript generation
type tsContext struct {
	meta                        *pgmeta.GeneratorMetadata
	opts                        Options
	columnsByTableID            map[int64][]pgmeta.PostgresColumn
	tableNamesByTableID         map[int64]string
	typesById                   map[int64]pgmeta.PostgresType
	relationTypeByIDs           map[int64]pgmeta.PostgresType
	introspectionBySchema       map[string]*schemaIntrospection
	detectOneToOneRelationships bool
}

type schemaIntrospection struct {
	tables         []tableWithRelationships
	views          []viewWithRelationships
	functions      []functionWithArgs
	enums          []pgmeta.PostgresType
	compositeTypes []pgmeta.PostgresType
}

type tableWithRelationships struct {
	table         pgmeta.PostgresTable
	relationships []pgmeta.PostgresRelationship
}

type viewWithRelationships struct {
	view          pgmeta.PostgresView
	isUpdatable   bool
	relationships []pgmeta.PostgresRelationship
}

type functionWithArgs struct {
	fn     pgmeta.PostgresFunction
	inArgs []pgmeta.FunctionArg
}

func newTsContext(meta *pgmeta.GeneratorMetadata, opts Options) *tsContext {
	ctx := &tsContext{
		meta:                        meta,
		opts:                        opts,
		columnsByTableID:            make(map[int64][]pgmeta.PostgresColumn),
		tableNamesByTableID:         make(map[int64]string),
		typesById:                   make(map[int64]pgmeta.PostgresType),
		relationTypeByIDs:           make(map[int64]pgmeta.PostgresType),
		introspectionBySchema:       make(map[string]*schemaIntrospection),
		detectOneToOneRelationships: opts.DetectOneToOneRelationships,
	}
	ctx.buildIndexes()
	return ctx
}

func (ctx *tsContext) buildIndexes() {
	// Sort schemas
	sort.Slice(ctx.meta.Schemas, func(i, j int) bool {
		return ctx.meta.Schemas[i].Name < ctx.meta.Schemas[j].Name
	})

	// Initialize introspection for each schema
	for _, s := range ctx.meta.Schemas {
		ctx.introspectionBySchema[s.Name] = &schemaIntrospection{
			tables:         []tableWithRelationships{},
			views:          []viewWithRelationships{},
			functions:      []functionWithArgs{},
			enums:          []pgmeta.PostgresType{},
			compositeTypes: []pgmeta.PostgresType{},
		}
	}

	// Build table-like structures
	tablesLike := make([]struct {
		id     int64
		name   string
		schema string
	}, 0)

	for _, t := range ctx.meta.Tables {
		tablesLike = append(tablesLike, struct {
			id     int64
			name   string
			schema string
		}{t.ID, t.Name, t.Schema})
	}
	for _, t := range ctx.meta.ForeignTables {
		tablesLike = append(tablesLike, struct {
			id     int64
			name   string
			schema string
		}{t.ID, t.Name, t.Schema})
	}
	for _, v := range ctx.meta.Views {
		tablesLike = append(tablesLike, struct {
			id     int64
			name   string
			schema string
		}{v.ID, v.Name, v.Schema})
	}
	for _, v := range ctx.meta.MaterializedViews {
		tablesLike = append(tablesLike, struct {
			id     int64
			name   string
			schema string
		}{v.ID, v.Name, v.Schema})
	}

	for _, tl := range tablesLike {
		ctx.columnsByTableID[tl.id] = []pgmeta.PostgresColumn{}
		ctx.tableNamesByTableID[tl.id] = tl.name
	}

	// Group columns by table ID
	for _, col := range ctx.meta.Columns {
		if _, ok := ctx.columnsByTableID[col.TableID]; ok {
			ctx.columnsByTableID[col.TableID] = append(ctx.columnsByTableID[col.TableID], col)
		}
	}

	// Sort columns by name
	for tableID := range ctx.columnsByTableID {
		cols := ctx.columnsByTableID[tableID]
		sort.Slice(cols, func(i, j int) bool {
			return cols[i].Name < cols[j].Name
		})
		ctx.columnsByTableID[tableID] = cols
	}

	// Build types indexes
	for _, t := range ctx.meta.Types {
		ctx.typesById[t.ID] = t
		if t.TypeRelationID != nil && *t.TypeRelationID != 0 {
			ctx.relationTypeByIDs[t.ID] = t
		}
		if intro, ok := ctx.introspectionBySchema[t.Schema]; ok {
			if len(t.Enums) > 0 {
				intro.enums = append(intro.enums, t)
			}
			if len(t.Attributes) > 0 {
				intro.compositeTypes = append(intro.compositeTypes, t)
			}
		}
	}

	// Build tables and views
	for _, table := range ctx.meta.Tables {
		if intro, ok := ctx.introspectionBySchema[table.Schema]; ok {
			intro.tables = append(intro.tables, tableWithRelationships{
				table:         table,
				relationships: ctx.getRelationships(table.Schema, table.Name),
			})
		}
	}

	for _, table := range ctx.meta.ForeignTables {
		if intro, ok := ctx.introspectionBySchema[table.Schema]; ok {
			intro.tables = append(intro.tables, tableWithRelationships{
				table: pgmeta.PostgresTable{
					ID:     table.ID,
					Schema: table.Schema,
					Name:   table.Name,
				},
				relationships: ctx.getRelationships(table.Schema, table.Name),
			})
		}
	}

	for _, view := range ctx.meta.Views {
		if intro, ok := ctx.introspectionBySchema[view.Schema]; ok {
			intro.views = append(intro.views, viewWithRelationships{
				view:          view,
				isUpdatable:   view.IsUpdatable,
				relationships: ctx.getRelationships(view.Schema, view.Name),
			})
		}
	}

	for _, view := range ctx.meta.MaterializedViews {
		if intro, ok := ctx.introspectionBySchema[view.Schema]; ok {
			intro.views = append(intro.views, viewWithRelationships{
				view: pgmeta.PostgresView{
					ID:          view.ID,
					Schema:      view.Schema,
					Name:        view.Name,
					IsUpdatable: false,
				},
				isUpdatable:   false,
				relationships: ctx.getRelationships(view.Schema, view.Name),
			})
		}
	}

	// Build functions
	validFnArgsMode := map[string]bool{"in": true, "inout": true, "variadic": true}
	validUnnamedArgTypes := map[int64]bool{114: true, 3802: true, 25: true} // json, jsonb, text

	for _, fn := range ctx.meta.Functions {
		if intro, ok := ctx.introspectionBySchema[fn.Schema]; ok {
			// Filter in args
			inArgs := []pgmeta.FunctionArg{}
			for _, arg := range fn.Args {
				if validFnArgsMode[arg.Mode] {
					inArgs = append(inArgs, arg)
				}
			}

			// Check if function should be included
			include := false
			if len(inArgs) == 0 {
				include = true
			} else if !hasUnnamedArgs(inArgs) {
				include = true
			} else if allUnnamedArgsHaveDefaults(inArgs, validUnnamedArgTypes) {
				include = true
			} else if len(inArgs) == 1 && inArgs[0].Name == "" {
				if validUnnamedArgTypes[inArgs[0].TypeID] {
					include = true
				} else if _, ok := ctx.relationTypeByIDs[inArgs[0].TypeID]; ok {
					include = true
				}
			}

			if include {
				intro.functions = append(intro.functions, functionWithArgs{
					fn:     fn,
					inArgs: inArgs,
				})
			}
		}
	}

	// Sort everything
	for _, intro := range ctx.introspectionBySchema {
		sort.Slice(intro.tables, func(i, j int) bool {
			return intro.tables[i].table.Name < intro.tables[j].table.Name
		})
		sort.Slice(intro.views, func(i, j int) bool {
			return intro.views[i].view.Name < intro.views[j].view.Name
		})
		sort.Slice(intro.functions, func(i, j int) bool {
			return intro.functions[i].fn.Name < intro.functions[j].fn.Name
		})
		sort.Slice(intro.enums, func(i, j int) bool {
			return intro.enums[i].Name < intro.enums[j].Name
		})
		sort.Slice(intro.compositeTypes, func(i, j int) bool {
			return intro.compositeTypes[i].Name < intro.compositeTypes[j].Name
		})
	}
}

func hasUnnamedArgs(args []pgmeta.FunctionArg) bool {
	for _, arg := range args {
		if arg.Name == "" {
			return true
		}
	}
	return false
}

func allUnnamedArgsHaveDefaults(args []pgmeta.FunctionArg, validTypes map[int64]bool) bool {
	for _, arg := range args {
		if arg.Name == "" {
			if !arg.HasDefault || !validTypes[arg.TypeID] {
				return false
			}
		}
	}
	return true
}

func (ctx *tsContext) getRelationships(schema, name string) []pgmeta.PostgresRelationship {
	var result []pgmeta.PostgresRelationship
	for _, rel := range ctx.meta.Relationships {
		if rel.Schema == schema && rel.ReferencedSchema == schema && rel.Relation == name {
			result = append(result, rel)
		}
	}
	return result
}

func (ctx *tsContext) generate() (string, error) {
	var sb strings.Builder

	sb.WriteString(`export type Json = string | number | boolean | null | { [key: string]: Json | undefined } | Json[]

export type Database = {
`)

	// Add internal supabase schema if postgrest version is specified
	if ctx.opts.PostgrestVersion != "" {
		sb.WriteString(fmt.Sprintf(`  __InternalSupabase: {
    PostgrestVersion: '%s'
  }
`, ctx.opts.PostgrestVersion))
	}

	for _, schema := range ctx.meta.Schemas {
		intro := ctx.introspectionBySchema[schema.Name]
		sb.WriteString(fmt.Sprintf("  %s: {\n", jsonString(schema.Name)))
		ctx.generateTables(&sb, schema, intro)
		ctx.generateViews(&sb, schema, intro)
		ctx.generateFunctions(&sb, schema, intro)
		ctx.generateEnums(&sb, intro)
		ctx.generateCompositeTypes(&sb, schema, intro)
		sb.WriteString("  }\n")
	}

	sb.WriteString("}\n\n")

	// Add helper types
	sb.WriteString(ctx.generateHelperTypes())

	// Add Constants
	sb.WriteString(ctx.generateConstants())

	return sb.String(), nil
}

func (ctx *tsContext) generateTables(sb *strings.Builder, schema pgmeta.PostgresSchema, intro *schemaIntrospection) {
	sb.WriteString("    Tables: {\n")
	if len(intro.tables) == 0 {
		sb.WriteString("      [_ in never]: never\n")
	} else {
		for _, tw := range intro.tables {
			sb.WriteString(fmt.Sprintf("      %s: {\n", jsonString(tw.table.Name)))
			ctx.generateTableRow(sb, schema, tw.table.ID, intro)
			ctx.generateTableInsert(sb, schema, tw.table.ID)
			ctx.generateTableUpdate(sb, schema, tw.table.ID)
			ctx.generateRelationships(sb, tw.relationships)
			sb.WriteString("      }\n")
		}
	}
	sb.WriteString("    }\n")
}

func (ctx *tsContext) generateTableRow(sb *strings.Builder, schema pgmeta.PostgresSchema, tableID int64, intro *schemaIntrospection) {
	sb.WriteString("        Row: {\n")
	cols := ctx.columnsByTableID[tableID]
	for _, col := range cols {
		tsType := ctx.pgTypeToTsType(schema, col.Format)
		sb.WriteString(fmt.Sprintf("          %s: %s\n", jsonString(col.Name), ctx.generateNullableUnion(tsType, col.IsNullable)))
	}
	sb.WriteString("        }\n")
}

func (ctx *tsContext) generateTableInsert(sb *strings.Builder, schema pgmeta.PostgresSchema, tableID int64) {
	sb.WriteString("        Insert: {\n")
	cols := ctx.columnsByTableID[tableID]
	for _, col := range cols {
		if col.IdentityGeneration != nil && *col.IdentityGeneration == "ALWAYS" {
			sb.WriteString(fmt.Sprintf("          %s?: never\n", jsonString(col.Name)))
		} else {
			tsType := ctx.pgTypeToTsType(schema, col.Format)
			isOptional := col.IsNullable || col.IsIdentity || col.DefaultValue != nil
			optMarker := ""
			if isOptional {
				optMarker = "?"
			}
			sb.WriteString(fmt.Sprintf("          %s%s: %s\n", jsonString(col.Name), optMarker, ctx.generateNullableUnion(tsType, col.IsNullable)))
		}
	}
	sb.WriteString("        }\n")
}

func (ctx *tsContext) generateTableUpdate(sb *strings.Builder, schema pgmeta.PostgresSchema, tableID int64) {
	sb.WriteString("        Update: {\n")
	cols := ctx.columnsByTableID[tableID]
	for _, col := range cols {
		if col.IdentityGeneration != nil && *col.IdentityGeneration == "ALWAYS" {
			sb.WriteString(fmt.Sprintf("          %s?: never\n", jsonString(col.Name)))
		} else {
			tsType := ctx.pgTypeToTsType(schema, col.Format)
			sb.WriteString(fmt.Sprintf("          %s?: %s\n", jsonString(col.Name), ctx.generateNullableUnion(tsType, col.IsNullable)))
		}
	}
	sb.WriteString("        }\n")
}

func (ctx *tsContext) generateRelationships(sb *strings.Builder, relationships []pgmeta.PostgresRelationship) {
	sb.WriteString("        Relationships: [\n")
	for _, rel := range relationships {
		sb.WriteString("          {\n")
		sb.WriteString(fmt.Sprintf("            foreignKeyName: %s\n", jsonString(rel.ConstraintName)))
		colsJSON, _ := json.Marshal(rel.Columns)
		sb.WriteString(fmt.Sprintf("            columns: %s\n", string(colsJSON)))
		if ctx.detectOneToOneRelationships {
			sb.WriteString(fmt.Sprintf("            isOneToOne: %t\n", rel.IsOneToOne))
		}
		sb.WriteString(fmt.Sprintf("            referencedRelation: %s\n", jsonString(rel.ReferencedRelation)))
		refColsJSON, _ := json.Marshal(rel.ReferencedColumns)
		sb.WriteString(fmt.Sprintf("            referencedColumns: %s\n", string(refColsJSON)))
		sb.WriteString("          }\n")
	}
	sb.WriteString("        ]\n")
}

func (ctx *tsContext) generateViews(sb *strings.Builder, schema pgmeta.PostgresSchema, intro *schemaIntrospection) {
	sb.WriteString("    Views: {\n")
	if len(intro.views) == 0 {
		sb.WriteString("      [_ in never]: never\n")
	} else {
		for _, vw := range intro.views {
			sb.WriteString(fmt.Sprintf("      %s: {\n", jsonString(vw.view.Name)))
			sb.WriteString("        Row: {\n")
			cols := ctx.columnsByTableID[vw.view.ID]
			for _, col := range cols {
				tsType := ctx.pgTypeToTsType(schema, col.Format)
				sb.WriteString(fmt.Sprintf("          %s: %s\n", jsonString(col.Name), ctx.generateNullableUnion(tsType, col.IsNullable)))
			}
			sb.WriteString("        }\n")

			if vw.isUpdatable {
				sb.WriteString("        Insert: {\n")
				for _, col := range cols {
					if !col.IsUpdatable {
						sb.WriteString(fmt.Sprintf("          %s?: never\n", jsonString(col.Name)))
					} else {
						tsType := ctx.pgTypeToTsType(schema, col.Format)
						sb.WriteString(fmt.Sprintf("          %s?: %s\n", jsonString(col.Name), ctx.generateNullableUnion(tsType, true)))
					}
				}
				sb.WriteString("        }\n")

				sb.WriteString("        Update: {\n")
				for _, col := range cols {
					if !col.IsUpdatable {
						sb.WriteString(fmt.Sprintf("          %s?: never\n", jsonString(col.Name)))
					} else {
						tsType := ctx.pgTypeToTsType(schema, col.Format)
						sb.WriteString(fmt.Sprintf("          %s?: %s\n", jsonString(col.Name), ctx.generateNullableUnion(tsType, true)))
					}
				}
				sb.WriteString("        }\n")
			}

			ctx.generateRelationships(sb, vw.relationships)
			sb.WriteString("      }\n")
		}
	}
	sb.WriteString("    }\n")
}

func (ctx *tsContext) generateFunctions(sb *strings.Builder, schema pgmeta.PostgresSchema, intro *schemaIntrospection) {
	sb.WriteString("    Functions: {\n")
	if len(intro.functions) == 0 {
		sb.WriteString("      [_ in never]: never\n")
	} else {
		// Group functions by name
		fnsByName := make(map[string][]functionWithArgs)
		for _, fwa := range intro.functions {
			fnsByName[fwa.fn.Name] = append(fnsByName[fwa.fn.Name], fwa)
		}

		fnNames := make([]string, 0, len(fnsByName))
		for name := range fnsByName {
			fnNames = append(fnNames, name)
		}
		sort.Strings(fnNames)

		for _, fnName := range fnNames {
			fns := fnsByName[fnName]
			sb.WriteString(fmt.Sprintf("      %s: ", jsonString(fnName)))
			if len(fns) == 1 {
				ctx.generateFunctionSignature(sb, schema, fns[0])
			} else {
				sb.WriteString("\n")
				for i, fwa := range fns {
					sb.WriteString("        | ")
					ctx.generateFunctionSignature(sb, schema, fwa)
					if i < len(fns)-1 {
						sb.WriteString("\n")
					}
				}
			}
			sb.WriteString("\n")
		}
	}
	sb.WriteString("    }\n")
}

func (ctx *tsContext) generateFunctionSignature(sb *strings.Builder, schema pgmeta.PostgresSchema, fwa functionWithArgs) {
	sb.WriteString("{\n")

	// Generate Args
	if len(fwa.inArgs) == 0 {
		sb.WriteString("        Args: Record<PropertyKey, never>\n")
	} else {
		sb.WriteString("        Args: {\n")
		for _, arg := range fwa.inArgs {
			argType := "unknown"
			if t, ok := ctx.typesById[arg.TypeID]; ok {
				argType = ctx.pgTypeToTsType(schema, t.Name)
			}
			optMarker := ""
			if arg.HasDefault {
				optMarker = "?"
			}
			sb.WriteString(fmt.Sprintf("          %s%s: %s\n", jsonString(arg.Name), optMarker, argType))
		}
		sb.WriteString("        }\n")
	}

	// Generate Returns
	returnType := ctx.getFunctionReturnType(schema, fwa.fn)
	if fwa.fn.IsSetReturningFunction {
		sb.WriteString(fmt.Sprintf("        Returns: %s[]\n", returnType))
	} else {
		sb.WriteString(fmt.Sprintf("        Returns: %s\n", returnType))
	}

	sb.WriteString("      }")
}

func (ctx *tsContext) getFunctionReturnType(schema pgmeta.PostgresSchema, fn pgmeta.PostgresFunction) string {
	// Check if returns table
	tableArgs := []pgmeta.FunctionArg{}
	for _, arg := range fn.Args {
		if arg.Mode == "table" {
			tableArgs = append(tableArgs, arg)
		}
	}

	if len(tableArgs) > 0 {
		var parts []string
		for _, arg := range tableArgs {
			argType := "unknown"
			if t, ok := ctx.typesById[arg.TypeID]; ok {
				argType = ctx.pgTypeToTsType(schema, t.Name)
			}
			parts = append(parts, fmt.Sprintf("%s: %s", jsonString(arg.Name), argType))
		}
		return fmt.Sprintf("{ %s }", strings.Join(parts, ", "))
	}

	// Check if returns a relation's row type
	if fn.ReturnTypeRelationID != nil {
		for _, tw := range ctx.introspectionBySchema[schema.Name].tables {
			if tw.table.ID == *fn.ReturnTypeRelationID {
				cols := ctx.columnsByTableID[tw.table.ID]
				var parts []string
				for _, col := range cols {
					tsType := ctx.pgTypeToTsType(schema, col.Format)
					parts = append(parts, fmt.Sprintf("%s: %s", jsonString(col.Name), ctx.generateNullableUnion(tsType, col.IsNullable)))
				}
				return fmt.Sprintf("{ %s }", strings.Join(parts, ", "))
			}
		}
		for _, vw := range ctx.introspectionBySchema[schema.Name].views {
			if vw.view.ID == *fn.ReturnTypeRelationID {
				cols := ctx.columnsByTableID[vw.view.ID]
				var parts []string
				for _, col := range cols {
					tsType := ctx.pgTypeToTsType(schema, col.Format)
					parts = append(parts, fmt.Sprintf("%s: %s", jsonString(col.Name), ctx.generateNullableUnion(tsType, col.IsNullable)))
				}
				return fmt.Sprintf("{ %s }", strings.Join(parts, ", "))
			}
		}
	}

	// Return base type
	if t, ok := ctx.typesById[fn.ReturnTypeID]; ok {
		return ctx.pgTypeToTsType(schema, t.Name)
	}

	return "unknown"
}

func (ctx *tsContext) generateEnums(sb *strings.Builder, intro *schemaIntrospection) {
	sb.WriteString("    Enums: {\n")
	if len(intro.enums) == 0 {
		sb.WriteString("      [_ in never]: never\n")
	} else {
		for _, enum := range intro.enums {
			variants := make([]string, len(enum.Enums))
			for i, v := range enum.Enums {
				variants[i] = jsonString(v)
			}
			sb.WriteString(fmt.Sprintf("      %s: %s\n", jsonString(enum.Name), strings.Join(variants, " | ")))
		}
	}
	sb.WriteString("    }\n")
}

func (ctx *tsContext) generateCompositeTypes(sb *strings.Builder, schema pgmeta.PostgresSchema, intro *schemaIntrospection) {
	sb.WriteString("    CompositeTypes: {\n")
	if len(intro.compositeTypes) == 0 {
		sb.WriteString("      [_ in never]: never\n")
	} else {
		for _, ct := range intro.compositeTypes {
			sb.WriteString(fmt.Sprintf("      %s: {\n", jsonString(ct.Name)))
			for _, attr := range ct.Attributes {
				attrType := "unknown"
				if t, ok := ctx.typesById[attr.TypeID]; ok {
					attrType = ctx.pgTypeToTsType(schema, t.Name)
				}
				sb.WriteString(fmt.Sprintf("        %s: %s\n", jsonString(attr.Name), ctx.generateNullableUnion(attrType, true)))
			}
			sb.WriteString("      }\n")
		}
	}
	sb.WriteString("    }\n")
}

func (ctx *tsContext) generateNullableUnion(tsType string, isNullable bool) string {
	if tsType == "unknown" || tsType == "any" || !isNullable {
		return tsType
	}
	return fmt.Sprintf("%s | null", tsType)
}

func (ctx *tsContext) pgTypeToTsType(schema pgmeta.PostgresSchema, pgType string) string {
	switch pgType {
	case "bool":
		return "boolean"
	case "int2", "int4", "int8", "float4", "float8", "numeric":
		return "number"
	case "bytea", "bpchar", "varchar", "date", "text", "citext", "time", "timetz", "timestamp", "timestamptz", "uuid", "vector":
		return "string"
	case "json", "jsonb":
		return "Json"
	case "void":
		return "undefined"
	case "record":
		return "Record<string, unknown>"
	}

	// Handle array types
	if strings.HasPrefix(pgType, "_") {
		innerType := ctx.pgTypeToTsType(schema, pgType[1:])
		return fmt.Sprintf("(%s)[]", innerType)
	}

	// Check for enum type
	for _, t := range ctx.meta.Types {
		if t.Name == pgType && len(t.Enums) > 0 {
			// Prefer the type from the same schema
			if t.Schema == schema.Name {
				return fmt.Sprintf("Database[%s]['Enums'][%s]", jsonString(t.Schema), jsonString(t.Name))
			}
		}
	}

	// Check for enum in other schemas
	for _, t := range ctx.meta.Types {
		if t.Name == pgType && len(t.Enums) > 0 {
			for _, s := range ctx.meta.Schemas {
				if s.Name == t.Schema {
					return fmt.Sprintf("Database[%s]['Enums'][%s]", jsonString(t.Schema), jsonString(t.Name))
				}
			}
			// Return inline enum if schema is not in the list
			variants := make([]string, len(t.Enums))
			for i, v := range t.Enums {
				variants[i] = jsonString(v)
			}
			return strings.Join(variants, " | ")
		}
	}

	// Check for composite type
	for _, t := range ctx.meta.Types {
		if t.Name == pgType && len(t.Attributes) > 0 {
			if t.Schema == schema.Name {
				return fmt.Sprintf("Database[%s]['CompositeTypes'][%s]", jsonString(t.Schema), jsonString(t.Name))
			}
		}
	}

	// Check for table row type
	for _, t := range ctx.meta.Tables {
		if t.Name == pgType {
			for _, s := range ctx.meta.Schemas {
				if s.Name == t.Schema {
					return fmt.Sprintf("Database[%s]['Tables'][%s]['Row']", jsonString(t.Schema), jsonString(t.Name))
				}
			}
		}
	}

	// Check for view row type
	for _, v := range ctx.meta.Views {
		if v.Name == pgType {
			for _, s := range ctx.meta.Schemas {
				if s.Name == v.Schema {
					return fmt.Sprintf("Database[%s]['Views'][%s]['Row']", jsonString(v.Schema), jsonString(v.Name))
				}
			}
		}
	}

	return "unknown"
}

func (ctx *tsContext) generateHelperTypes() string {
	return `type PublicSchema = Database[Extract<keyof Database, "public">]

export type Tables<
  PublicTableNameOrOptions extends
    | keyof (PublicSchema["Tables"] & PublicSchema["Views"])
    | { schema: keyof Database },
  TableName extends PublicTableNameOrOptions extends { schema: keyof Database }
    ? keyof (Database[PublicTableNameOrOptions["schema"]]["Tables"] &
        Database[PublicTableNameOrOptions["schema"]]["Views"])
    : never = never,
> = PublicTableNameOrOptions extends { schema: keyof Database }
  ? (Database[PublicTableNameOrOptions["schema"]]["Tables"] &
      Database[PublicTableNameOrOptions["schema"]]["Views"])[TableName] extends {
      Row: infer R
    }
    ? R
    : never
  : PublicTableNameOrOptions extends keyof (PublicSchema["Tables"] &
        PublicSchema["Views"])
    ? (PublicSchema["Tables"] &
        PublicSchema["Views"])[PublicTableNameOrOptions] extends {
        Row: infer R
      }
      ? R
      : never
    : never

export type TablesInsert<
  PublicTableNameOrOptions extends
    | keyof PublicSchema["Tables"]
    | { schema: keyof Database },
  TableName extends PublicTableNameOrOptions extends { schema: keyof Database }
    ? keyof Database[PublicTableNameOrOptions["schema"]]["Tables"]
    : never = never,
> = PublicTableNameOrOptions extends { schema: keyof Database }
  ? Database[PublicTableNameOrOptions["schema"]]["Tables"][TableName] extends {
      Insert: infer I
    }
    ? I
    : never
  : PublicTableNameOrOptions extends keyof PublicSchema["Tables"]
    ? PublicSchema["Tables"][PublicTableNameOrOptions] extends {
        Insert: infer I
      }
      ? I
      : never
    : never

export type TablesUpdate<
  PublicTableNameOrOptions extends
    | keyof PublicSchema["Tables"]
    | { schema: keyof Database },
  TableName extends PublicTableNameOrOptions extends { schema: keyof Database }
    ? keyof Database[PublicTableNameOrOptions["schema"]]["Tables"]
    : never = never,
> = PublicTableNameOrOptions extends { schema: keyof Database }
  ? Database[PublicTableNameOrOptions["schema"]]["Tables"][TableName] extends {
      Update: infer U
    }
    ? U
    : never
  : PublicTableNameOrOptions extends keyof PublicSchema["Tables"]
    ? PublicSchema["Tables"][PublicTableNameOrOptions] extends {
        Update: infer U
      }
      ? U
      : never
    : never

export type Enums<
  PublicEnumNameOrOptions extends
    | keyof PublicSchema["Enums"]
    | { schema: keyof Database },
  EnumName extends PublicEnumNameOrOptions extends { schema: keyof Database }
    ? keyof Database[PublicEnumNameOrOptions["schema"]]["Enums"]
    : never = never,
> = PublicEnumNameOrOptions extends { schema: keyof Database }
  ? Database[PublicEnumNameOrOptions["schema"]]["Enums"][EnumName]
  : PublicEnumNameOrOptions extends keyof PublicSchema["Enums"]
    ? PublicSchema["Enums"][PublicEnumNameOrOptions]
    : never

export type CompositeTypes<
  PublicCompositeTypeNameOrOptions extends
    | keyof PublicSchema["CompositeTypes"]
    | { schema: keyof Database },
  CompositeTypeName extends PublicCompositeTypeNameOrOptions extends {
    schema: keyof Database
  }
    ? keyof Database[PublicCompositeTypeNameOrOptions["schema"]]["CompositeTypes"]
    : never = never,
> = PublicCompositeTypeNameOrOptions extends { schema: keyof Database }
  ? Database[PublicCompositeTypeNameOrOptions["schema"]]["CompositeTypes"][CompositeTypeName]
  : PublicCompositeTypeNameOrOptions extends keyof PublicSchema["CompositeTypes"]
    ? PublicSchema["CompositeTypes"][PublicCompositeTypeNameOrOptions]
    : never

`
}

func (ctx *tsContext) generateConstants() string {
	var sb strings.Builder
	sb.WriteString("export const Constants = {\n")
	for _, schema := range ctx.meta.Schemas {
		intro := ctx.introspectionBySchema[schema.Name]
		sb.WriteString(fmt.Sprintf("  %s: {\n", jsonString(schema.Name)))
		sb.WriteString("    Enums: {\n")
		for _, enum := range intro.enums {
			variants := make([]string, len(enum.Enums))
			for i, v := range enum.Enums {
				variants[i] = jsonString(v)
			}
			sb.WriteString(fmt.Sprintf("      %s: [%s],\n", jsonString(enum.Name), strings.Join(variants, ", ")))
		}
		sb.WriteString("    },\n")
		sb.WriteString("  },\n")
	}
	sb.WriteString("} as const\n")
	return sb.String()
}

func jsonString(s string) string {
	b, _ := json.Marshal(s)
	return string(b)
}
