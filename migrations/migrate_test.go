package migrations

import (
	"strings"
	"testing"
)

func TestEmbeddedInitialMigration(t *testing.T) {
	body, err := files.ReadFile("001_create_movies.sql")
	if err != nil {
		t.Fatalf("initial migration must be embedded in the binary: %v", err)
	}
	if !strings.Contains(string(body), "CREATE TABLE movies") {
		t.Fatal("initial migration must create the movies table")
	}
}
