package sqlite

import (
	"database/sql"

	_ "github.com/mattn/go-sqlite3"
)

type Database interface {
}

type database struct {
	conn *sql.DB
}

func OpenDatabase(path string) (Database, error) {
	conn, err := sql.Open("sqlite3", path)
	if err != nil {
		return nil, err
	}
	return &database{conn: conn}, nil
}
