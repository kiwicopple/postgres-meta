package main

import (
	"context"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/supabase/postgres-meta/cli/pkg/generators"
	"github.com/supabase/postgres-meta/cli/pkg/pgmeta"
)

var (
	// Database connection flags
	dbHost     string
	dbPort     int
	dbName     string
	dbUser     string
	dbPassword string
	dbURL      string
	sslMode    string

	// Generator flags
	includedSchemas []string
	excludedSchemas []string

	// TypeScript flags
	detectOneToOneRelationships bool
	postgrestVersion            string

	// Swift flags
	accessControl string
)

func main() {
	rootCmd := &cobra.Command{
		Use:   "pg-meta",
		Short: "PostgreSQL metadata and code generation tool",
		Long:  `pg-meta is a CLI tool for introspecting PostgreSQL databases and generating type definitions for various languages.`,
	}

	// Add global database connection flags
	rootCmd.PersistentFlags().StringVar(&dbHost, "host", "localhost", "Database host")
	rootCmd.PersistentFlags().IntVar(&dbPort, "port", 5432, "Database port")
	rootCmd.PersistentFlags().StringVar(&dbName, "db", "", "Database name")
	rootCmd.PersistentFlags().StringVar(&dbUser, "user", "", "Database user")
	rootCmd.PersistentFlags().StringVar(&dbPassword, "password", "", "Database password")
	rootCmd.PersistentFlags().StringVar(&dbURL, "db-url", "", "Database connection URL (overrides individual flags)")
	rootCmd.PersistentFlags().StringVar(&sslMode, "ssl-mode", "prefer", "SSL mode (disable, prefer, require)")

	// Generate command
	generateCmd := &cobra.Command{
		Use:   "generate",
		Short: "Generate code from database schema",
		Long:  `Generate type definitions, models, or code from your PostgreSQL database schema.`,
	}

	// Types subcommand
	typesCmd := &cobra.Command{
		Use:   "types [language]",
		Short: "Generate type definitions for a language",
		Long: `Generate type definitions for your database schema in the specified language.

Supported languages:
  - typescript  Generate TypeScript types for Supabase clients
  - python      Generate Python Pydantic models
  - go          Generate Go struct definitions
  - swift       Generate Swift struct definitions`,
		Args: cobra.ExactArgs(1),
		ValidArgs: []string{"typescript", "python", "go", "swift"},
		RunE: runGenerateTypes,
	}

	// Add generator flags
	typesCmd.Flags().StringSliceVar(&includedSchemas, "included-schemas", nil, "Schemas to include (comma-separated)")
	typesCmd.Flags().StringSliceVar(&excludedSchemas, "excluded-schemas", nil, "Schemas to exclude (comma-separated)")
	typesCmd.Flags().BoolVar(&detectOneToOneRelationships, "detect-one-to-one", false, "Detect one-to-one relationships (TypeScript only)")
	typesCmd.Flags().StringVar(&postgrestVersion, "postgrest-version", "", "PostgREST version for TypeScript types")
	typesCmd.Flags().StringVar(&accessControl, "access-control", "internal", "Access control modifier (Swift only: internal, public, private, package)")

	generateCmd.AddCommand(typesCmd)

	// Scaffold command (placeholder for future)
	scaffoldCmd := &cobra.Command{
		Use:   "scaffold [language]",
		Short: "Generate scaffold code for a framework",
		Long: `Generate scaffold code for your database schema targeting a specific framework.

Coming soon:
  - typescript --drizzle-orm
  - python --django
  - elixir --phoenix`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return fmt.Errorf("scaffold command is not yet implemented")
		},
	}

	generateCmd.AddCommand(scaffoldCmd)
	rootCmd.AddCommand(generateCmd)

	// Version command
	versionCmd := &cobra.Command{
		Use:   "version",
		Short: "Print the version",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println("pg-meta version 0.1.0")
		},
	}
	rootCmd.AddCommand(versionCmd)

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func runGenerateTypes(cmd *cobra.Command, args []string) error {
	language := args[0]

	// Validate language
	gen := generators.GetGenerator(language)
	if gen == nil {
		return fmt.Errorf("unsupported language: %s. Supported languages: %v", language, generators.GetAvailableGenerators())
	}

	// Create database client
	ctx := context.Background()
	client, err := createClient(ctx)
	if err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}
	defer client.Close()

	// Get generator metadata
	opts := pgmeta.GeneratorOptions{
		IncludedSchemas: includedSchemas,
		ExcludedSchemas: excludedSchemas,
	}

	meta, err := client.GetGeneratorMetadata(opts)
	if err != nil {
		return fmt.Errorf("failed to get metadata: %w", err)
	}

	// Generate code
	genOpts := generators.Options{
		IncludedSchemas:             includedSchemas,
		ExcludedSchemas:             excludedSchemas,
		DetectOneToOneRelationships: detectOneToOneRelationships,
		PostgrestVersion:            postgrestVersion,
		AccessControl:               accessControl,
	}

	output, err := gen.Generate(meta, genOpts)
	if err != nil {
		return fmt.Errorf("failed to generate types: %w", err)
	}

	fmt.Print(output)
	return nil
}

func createClient(ctx context.Context) (*pgmeta.Client, error) {
	if dbURL != "" {
		return pgmeta.NewClientFromConnectionString(ctx, dbURL)
	}

	// Check for required fields
	if dbName == "" {
		// Try to get from environment
		dbURL = os.Getenv("DATABASE_URL")
		if dbURL != "" {
			return pgmeta.NewClientFromConnectionString(ctx, dbURL)
		}
		return nil, fmt.Errorf("database name is required (use --db or --db-url or set DATABASE_URL)")
	}

	if dbUser == "" {
		dbUser = os.Getenv("PGUSER")
		if dbUser == "" {
			return nil, fmt.Errorf("database user is required (use --user or set PGUSER)")
		}
	}

	if dbPassword == "" {
		dbPassword = os.Getenv("PGPASSWORD")
	}

	if dbHost == "localhost" {
		if envHost := os.Getenv("PGHOST"); envHost != "" {
			dbHost = envHost
		}
	}

	if dbPort == 5432 {
		if envPort := os.Getenv("PGPORT"); envPort != "" {
			fmt.Sscanf(envPort, "%d", &dbPort)
		}
	}

	return pgmeta.NewClient(ctx, pgmeta.Config{
		Host:     dbHost,
		Port:     dbPort,
		Database: dbName,
		User:     dbUser,
		Password: dbPassword,
		SSLMode:  sslMode,
	})
}
