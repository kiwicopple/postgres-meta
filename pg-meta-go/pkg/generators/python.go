package generators

import (
	"fmt"
	"regexp"
	"sort"
	"strings"

	"github.com/supabase/postgres-meta/pg-meta-go/pkg/pgmeta"
)

// PythonGenerator generates Python Pydantic type definitions
type PythonGenerator struct{}

// Name returns the generator name
func (g *PythonGenerator) Name() string {
	return "python"
}

// Generate generates Python type definitions from the metadata
func (g *PythonGenerator) Generate(meta *pgmeta.GeneratorMetadata, opts Options) (string, error) {
	ctx := newPyContext(meta)
	return ctx.generate()
}

// pyContext holds the context for Python generation
type pyContext struct {
	meta         *pgmeta.GeneratorMetadata
	types        map[string]pgmeta.PostgresType
	userEnums    map[string]*pyEnum
	columns      map[int64][]pgmeta.PostgresColumn
	schemas      map[string]pgmeta.PostgresSchema
	schemaNames  map[string]bool
}

func newPyContext(meta *pgmeta.GeneratorMetadata) *pyContext {
	ctx := &pyContext{
		meta:        meta,
		types:       make(map[string]pgmeta.PostgresType),
		userEnums:   make(map[string]*pyEnum),
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
			ctx.userEnums[t.Name] = &pyEnum{
				name:     formatForPyClassName(t.Schema) + formatForPyClassName(t.Name),
				variants: t.Enums,
			}
		}
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

func (ctx *pyContext) generate() (string, error) {
	var sb strings.Builder

	sb.WriteString(`from __future__ import annotations

import datetime
import uuid
from typing import (
    Annotated,
    Any,
    List,
    Literal,
    NotRequired,
    Optional,
    TypeAlias,
    TypedDict,
)

from pydantic import BaseModel, Field, Json

`)

	// Generate enums
	for _, t := range ctx.meta.Types {
		if len(t.Enums) > 0 && ctx.schemaNames[t.Schema] {
			enum := ctx.userEnums[t.Name]
			sb.WriteString(enum.serialize())
			sb.WriteString("\n\n")
		}
	}

	// Generate table classes
	for _, table := range ctx.meta.Tables {
		if !ctx.schemaNames[table.Schema] {
			continue
		}
		selectModel := ctx.tableToSelectModel(table)
		insertDict := ctx.tableToInsertDict(table)
		updateDict := ctx.tableToUpdateDict(table)
		sb.WriteString(selectModel.serialize())
		sb.WriteString("\n\n")
		sb.WriteString(insertDict.serialize())
		sb.WriteString("\n\n")
		sb.WriteString(updateDict.serialize())
		sb.WriteString("\n\n")
	}

	// Generate view classes
	for _, view := range ctx.meta.Views {
		if !ctx.schemaNames[view.Schema] {
			continue
		}
		model := ctx.viewToModel(view)
		sb.WriteString(model.serialize())
		sb.WriteString("\n\n")
	}

	// Generate materialized view classes
	for _, matview := range ctx.meta.MaterializedViews {
		if !ctx.schemaNames[matview.Schema] {
			continue
		}
		model := ctx.matViewToModel(matview)
		sb.WriteString(model.serialize())
		sb.WriteString("\n\n")
	}

	// Generate composite type classes
	for _, t := range ctx.meta.Types {
		if len(t.Attributes) > 0 && ctx.schemaNames[t.Schema] {
			model := ctx.typeToModel(t)
			sb.WriteString(model.serialize())
			sb.WriteString("\n\n")
		}
	}

	return strings.TrimRight(sb.String(), "\n") + "\n", nil
}

func (ctx *pyContext) resolveTypeName(name string) string {
	if enum, ok := ctx.userEnums[name]; ok {
		return enum.name
	}
	if pyType, ok := pyTypeMap[name]; ok {
		return pyType
	}
	if t, ok := ctx.types[name]; ok {
		return formatForPyClassName(t.Schema) + formatForPyClassName(t.Name)
	}
	return "Any"
}

func (ctx *pyContext) parsePgType(pgType string) pyType {
	if strings.HasPrefix(pgType, "_") {
		inner := ctx.parsePgType(pgType[1:])
		return &pyListType{inner: inner}
	}
	typeName := ctx.resolveTypeName(pgType)
	return &pySimpleType{name: typeName}
}

func (ctx *pyContext) columnsToClassAttrs(tableID int64) []*pyBaseModelAttr {
	cols := ctx.columns[tableID]
	attrs := make([]*pyBaseModelAttr, len(cols))
	for i, col := range cols {
		attrs[i] = &pyBaseModelAttr{
			name:     formatForPyAttributeName(col.Name),
			pgName:   col.Name,
			pyType:   ctx.parsePgType(col.Format),
			nullable: col.IsNullable,
		}
	}
	return attrs
}

func (ctx *pyContext) columnsToDictAttrs(tableID int64, allNotRequired bool) []*pyTypedDictAttr {
	cols := ctx.columns[tableID]
	attrs := make([]*pyTypedDictAttr, len(cols))
	for i, col := range cols {
		attrs[i] = &pyTypedDictAttr{
			name:        formatForPyAttributeName(col.Name),
			pgName:      col.Name,
			pyType:      ctx.parsePgType(col.Format),
			nullable:    col.IsNullable,
			notRequired: allNotRequired || col.IsNullable || col.IsIdentity || col.DefaultValue != nil,
		}
	}
	return attrs
}

func (ctx *pyContext) tableToSelectModel(table pgmeta.PostgresTable) *pyBaseModel {
	schema := ctx.schemas[table.Schema]
	return &pyBaseModel{
		name:       formatForPyClassName(schema.Name) + formatForPyClassName(table.Name),
		tableName:  table.Name,
		schema:     schema,
		attributes: ctx.columnsToClassAttrs(table.ID),
	}
}

func (ctx *pyContext) tableToInsertDict(table pgmeta.PostgresTable) *pyTypedDict {
	schema := ctx.schemas[table.Schema]
	return &pyTypedDict{
		name:       formatForPyClassName(schema.Name) + formatForPyClassName(table.Name),
		tableName:  table.Name,
		operation:  "Insert",
		schema:     schema,
		attributes: ctx.columnsToDictAttrs(table.ID, false),
	}
}

func (ctx *pyContext) tableToUpdateDict(table pgmeta.PostgresTable) *pyTypedDict {
	schema := ctx.schemas[table.Schema]
	return &pyTypedDict{
		name:       formatForPyClassName(schema.Name) + formatForPyClassName(table.Name),
		tableName:  table.Name,
		operation:  "Update",
		schema:     schema,
		attributes: ctx.columnsToDictAttrs(table.ID, true),
	}
}

func (ctx *pyContext) viewToModel(view pgmeta.PostgresView) *pyBaseModel {
	schema := ctx.schemas[view.Schema]
	return &pyBaseModel{
		name:       formatForPyClassName(schema.Name) + formatForPyClassName(view.Name),
		tableName:  view.Name,
		schema:     schema,
		attributes: ctx.columnsToClassAttrs(view.ID),
	}
}

func (ctx *pyContext) matViewToModel(matview pgmeta.PostgresMaterializedView) *pyBaseModel {
	schema := ctx.schemas[matview.Schema]
	return &pyBaseModel{
		name:       formatForPyClassName(schema.Name) + formatForPyClassName(matview.Name),
		tableName:  matview.Name,
		schema:     schema,
		attributes: ctx.columnsToClassAttrs(matview.ID),
	}
}

func (ctx *pyContext) typeToModel(t pgmeta.PostgresType) *pyBaseModel {
	schema := ctx.schemas[t.Schema]
	attrs := make([]*pyBaseModelAttr, len(t.Attributes))
	for i, attr := range t.Attributes {
		attrType := ctx.types[attr.Name]
		typeName := "Any"
		for _, tt := range ctx.meta.Types {
			if tt.ID == attr.TypeID {
				typeName = tt.Name
				break
			}
		}
		attrs[i] = &pyBaseModelAttr{
			name:     formatForPyAttributeName(attr.Name),
			pgName:   attr.Name,
			pyType:   ctx.parsePgType(typeName),
			nullable: false,
		}
		_ = attrType // unused but kept for potential future use
	}
	return &pyBaseModel{
		name:       formatForPyClassName(schema.Name) + formatForPyClassName(t.Name),
		tableName:  t.Name,
		schema:     schema,
		attributes: attrs,
	}
}

// Python type interfaces
type pyType interface {
	serialize() string
}

type pySimpleType struct {
	name string
}

func (t *pySimpleType) serialize() string {
	return t.name
}

type pyListType struct {
	inner pyType
}

func (t *pyListType) serialize() string {
	return fmt.Sprintf("List[%s]", t.inner.serialize())
}

// pyEnum represents a Python enum type alias
type pyEnum struct {
	name     string
	variants []string
}

func (e *pyEnum) serialize() string {
	variants := make([]string, len(e.variants))
	for i, v := range e.variants {
		variants[i] = fmt.Sprintf(`"%s"`, v)
	}
	return fmt.Sprintf("%s: TypeAlias = Literal[%s]", e.name, strings.Join(variants, ", "))
}

// pyBaseModelAttr represents a Pydantic BaseModel attribute
type pyBaseModelAttr struct {
	name     string
	pgName   string
	pyType   pyType
	nullable bool
}

func (a *pyBaseModelAttr) serialize() string {
	typeStr := a.pyType.serialize()
	if a.nullable {
		typeStr = fmt.Sprintf("Optional[%s]", typeStr)
	}
	return fmt.Sprintf("    %s: %s = Field(alias=\"%s\")", a.name, typeStr, a.pgName)
}

// pyBaseModel represents a Pydantic BaseModel class
type pyBaseModel struct {
	name       string
	tableName  string
	schema     pgmeta.PostgresSchema
	attributes []*pyBaseModelAttr
}

func (m *pyBaseModel) serialize() string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("class %s(BaseModel):\n", m.name))
	if len(m.attributes) == 0 {
		sb.WriteString("    pass")
	} else {
		for i, attr := range m.attributes {
			sb.WriteString(attr.serialize())
			if i < len(m.attributes)-1 {
				sb.WriteString("\n")
			}
		}
	}
	return sb.String()
}

// pyTypedDictAttr represents a TypedDict attribute
type pyTypedDictAttr struct {
	name        string
	pgName      string
	pyType      pyType
	nullable    bool
	notRequired bool
}

func (a *pyTypedDictAttr) serialize() string {
	annotation := fmt.Sprintf("Annotated[%s, Field(alias=\"%s\")]", a.pyType.serialize(), a.pgName)
	rhs := annotation
	if a.notRequired {
		rhs = fmt.Sprintf("NotRequired[%s]", annotation)
	}
	return fmt.Sprintf("    %s: %s", a.name, rhs)
}

// pyTypedDict represents a TypedDict class
type pyTypedDict struct {
	name       string
	tableName  string
	operation  string
	schema     pgmeta.PostgresSchema
	attributes []*pyTypedDictAttr
}

func (d *pyTypedDict) serialize() string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("class %s%s(TypedDict):\n", d.name, d.operation))
	if len(d.attributes) == 0 {
		sb.WriteString("    pass")
	} else {
		for i, attr := range d.attributes {
			sb.WriteString(attr.serialize())
			if i < len(d.attributes)-1 {
				sb.WriteString("\n")
			}
		}
	}
	return sb.String()
}

// Helper functions

var nonAlphanumericRegex = regexp.MustCompile(`[^a-zA-Z0-9]+`)

func formatForPyClassName(name string) string {
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

func formatForPyAttributeName(name string) string {
	parts := nonAlphanumericRegex.Split(name, -1)
	for i := range parts {
		parts[i] = strings.ToLower(parts[i])
	}
	return strings.Join(parts, "_")
}

var pyTypeMap = map[string]string{
	// Bool
	"bool": "bool",

	// Numbers
	"int2":    "int",
	"int4":    "int",
	"int8":    "int",
	"float4":  "float",
	"float8":  "float",
	"numeric": "float",

	// Strings
	"bytea":       "bytes",
	"bpchar":      "str",
	"varchar":     "str",
	"string":      "str",
	"date":        "datetime.date",
	"text":        "str",
	"citext":      "str",
	"time":        "datetime.time",
	"timetz":      "datetime.time",
	"timestamp":   "datetime.datetime",
	"timestamptz": "datetime.datetime",
	"uuid":        "uuid.UUID",
	"vector":      "list[Any]",

	// JSON
	"json":  "Json[Any]",
	"jsonb": "Json[Any]",

	// Range types
	"int4range":      "str",
	"int4multirange": "str",
	"int8range":      "str",
	"int8multirange": "str",
	"numrange":       "str",
	"nummultirange":  "str",
	"tsrange":        "str",
	"tsmultirange":   "str",
	"tstzrange":      "str",
	"tstzmultirange": "str",
	"daterange":      "str",
	"datemultirange": "str",

	// Miscellaneous
	"void":   "None",
	"record": "dict[str, Any]",
}
