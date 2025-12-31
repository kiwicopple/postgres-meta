package generators

import (
	"github.com/supabase/postgres-meta/cli/pkg/pgmeta"
)

// Generator is the interface that all type generators must implement
type Generator interface {
	// Generate generates code from the given metadata
	Generate(meta *pgmeta.GeneratorMetadata, opts Options) (string, error)
	// Name returns the name of the generator
	Name() string
}

// Options contains options for code generation
type Options struct {
	// Schema filtering
	IncludedSchemas []string
	ExcludedSchemas []string

	// TypeScript options
	DetectOneToOneRelationships bool
	PostgrestVersion            string

	// Swift options
	AccessControl string // "internal", "public", "private", "package"
}

// Common type mappings and utilities

// TypeMapping represents a mapping from PostgreSQL type to target language type
type TypeMapping struct {
	PgType     string
	TargetType string
}

// ident converts a PostgreSQL identifier to a safe identifier in the target language
func ident(name string) string {
	return name
}

// Helper function to check if a column is optional for insert
func isOptionalForInsert(col pgmeta.PostgresColumn) bool {
	return col.IsNullable || col.IsIdentity || col.IsGenerated || col.DefaultValue != nil
}

// Helper to get columns for a specific table
func getColumnsForTable(columns []pgmeta.PostgresColumn, schema, table string) []pgmeta.PostgresColumn {
	var result []pgmeta.PostgresColumn
	for _, col := range columns {
		if col.Schema == schema && col.Table == table {
			result = append(result, col)
		}
	}
	return result
}

// Helper to get relationships for a specific table
func getRelationshipsForTable(relationships []pgmeta.PostgresRelationship, schema, table string) []pgmeta.PostgresRelationship {
	var result []pgmeta.PostgresRelationship
	for _, rel := range relationships {
		if rel.Schema == schema && rel.Relation == table {
			result = append(result, rel)
		}
	}
	return result
}

// Helper to find a type by name and schema
func findType(types []pgmeta.PostgresType, schema, name string) *pgmeta.PostgresType {
	for _, t := range types {
		if t.Schema == schema && t.Name == name {
			return &t
		}
	}
	return nil
}

// Helper to check if a type is an enum
func isEnum(types []pgmeta.PostgresType, schema, format string) bool {
	for _, t := range types {
		if t.Schema == schema && t.Name == format && len(t.Enums) > 0 {
			return true
		}
	}
	return false
}

// Helper to check if a type is a composite type
func isComposite(types []pgmeta.PostgresType, schema, format string) bool {
	for _, t := range types {
		if t.Schema == schema && t.Name == format && len(t.Attributes) > 0 {
			return true
		}
	}
	return false
}

// GetGenerator returns a generator by name
func GetGenerator(name string) Generator {
	switch name {
	case "typescript":
		return &TypeScriptGenerator{}
	case "python":
		return &PythonGenerator{}
	case "go":
		return &GoGenerator{}
	case "swift":
		return &SwiftGenerator{}
	default:
		return nil
	}
}

// GetAvailableGenerators returns a list of available generator names
func GetAvailableGenerators() []string {
	return []string{"typescript", "python", "go", "swift"}
}
