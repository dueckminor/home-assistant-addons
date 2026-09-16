package sqlite

import (
	"testing"
)

func TestOpenWALDatabaseAllowsWriteDuringRead(t *testing.T) {
	db, err := OpenWALDatabase(t.TempDir() + "/metrics.db")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	if _, err := db.Exec(`CREATE TABLE metrics (value INTEGER)`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO metrics VALUES (1)`); err != nil {
		t.Fatal(err)
	}

	rows, err := db.Query(`SELECT value FROM metrics`)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	if !rows.Next() {
		t.Fatal("expected a row")
	}

	if _, err := db.Exec(`INSERT INTO metrics VALUES (2)`); err != nil {
		t.Fatalf("write while read cursor is open: %v", err)
	}
}
