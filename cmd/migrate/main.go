package main

import (
	"database/sql"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	_ "github.com/lib/pq"

	"weather-platform/internal/config"
)

func main() {
	direction := flag.String("direction", "up", "Migration direction: up or down")
	flag.Parse()

	// Load configuration
	cfg, err := config.LoadConfig()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load configuration: %v\n", err)
		os.Exit(1)
	}

	// Connect to database
	dsn := fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		cfg.Database.Host,
		cfg.Database.Port,
		cfg.Database.User,
		cfg.Database.Password,
		cfg.Database.Database,
		cfg.Database.SSLMode,
	)

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to connect to database: %v\n", err)
		os.Exit(1)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to ping database: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Connected to database successfully")

	// Read migration file
	var migrationFile string
	if *direction == "up" {
		migrationFile = "migrations/001_create_schema.up.sql"
	} else {
		migrationFile = "migrations/001_create_schema.down.sql"
	}

	migrationPath := filepath.Join(".", migrationFile)
	content, err := os.ReadFile(migrationPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to read migration file: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Running migration: %s\n", migrationFile)

	// Execute migration
	_, err = db.Exec(string(content))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to execute migration: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Migration completed successfully")
}
