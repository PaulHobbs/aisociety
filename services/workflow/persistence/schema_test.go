package persistence

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/jackc/pgx/v4"
)

func TestSchemaTablesExist(t *testing.T) {
	connStr := os.Getenv("TEST_DATABASE_URL")
	if connStr == "" {
		t.Fatal("TEST_DATABASE_URL not set")
	}

	conn, err := pgx.Connect(context.Background(), connStr)
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}
	defer conn.Close(context.Background())

	// List all tables for debugging
	rows, err := conn.Query(context.Background(), `
		SELECT table_name
		FROM information_schema.tables
		WHERE table_schema = 'public'
	`)
	if err != nil {
		t.Fatalf("Failed to list tables: %v", err)
	}
	defer rows.Close()

	var existingTables []string
	for rows.Next() {
		var table string
		if err := rows.Scan(&table); err != nil {
			t.Fatalf("Failed to scan table name: %v", err)
		}
		existingTables = append(existingTables, table)
	}
	fmt.Printf("Existing tables: %v\n", existingTables)

	expectedTables := []string{"workflows", "nodes", "node_edges"}

	for _, table := range expectedTables {
		found := false
		for _, existing := range existingTables {
			if existing == table {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Table %s does not exist", table)
		}
	}
}
