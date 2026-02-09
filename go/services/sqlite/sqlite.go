package sqlite

import (
	"context"
	"database/sql"

	_ "modernc.org/sqlite"
)

type Database interface {
	QueryRow(query string, args ...any) *sql.Row
	Query(query string, args ...any) (*sql.Rows, error)
	Exec(query string, args ...any) (sql.Result, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	Close() error
}

type database struct {
	*sql.DB
}

func OpenDatabase(path string) (Database, error) {
	conn, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}
	return &database{DB: conn}, nil
}
