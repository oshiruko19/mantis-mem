// Package store owns the SQLite database: schema, connection, and all queries.
package store

import (
	"database/sql"
	_ "embed"
	"fmt"
	"os"
	"path/filepath"

	_ "modernc.org/sqlite" // registers the cgo-free "sqlite" database/sql driver
)

//go:embed schema.sql
var schemaSQL string

// Store is a handle to the mantis-mem SQLite database.
type Store struct {
	db   *sql.DB
	path string
}

// DefaultDBPath returns the DB location: $MANTIS_DB, else ~/.mantis/mantis_mem.db.
func DefaultDBPath() (string, error) {
	if v := os.Getenv("MANTIS_DB"); v != "" {
		return v, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolve home dir: %w", err)
	}
	return filepath.Join(home, ".mantis", "mantis_mem.db"), nil
}

// Open opens (creating if needed) the database at dbPath. If dbPath is empty,
// DefaultDBPath is used. The parent directory is created, connection pragmas are
// applied per-connection, and the schema is migrated.
func Open(dbPath string) (*Store, error) {
	if dbPath == "" {
		var err error
		if dbPath, err = DefaultDBPath(); err != nil {
			return nil, err
		}
	}
	if dir := filepath.Dir(dbPath); dir != "" {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return nil, fmt.Errorf("create db dir %s: %w", dir, err)
		}
	}

	// Per-connection pragmas travel in the DSN so every pooled connection gets
	// them. WAL lets the MCP server and a CLI invocation share the file.
	dsn := fmt.Sprintf(
		"file:%s?_pragma=busy_timeout(5000)&_pragma=journal_mode(WAL)&_pragma=foreign_keys(1)",
		dbPath,
	)
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}
	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("ping sqlite: %w", err)
	}

	s := &Store{db: db, path: dbPath}
	if err := s.migrate(); err != nil {
		db.Close()
		return nil, err
	}
	return s, nil
}

func (s *Store) migrate() error {
	if _, err := s.db.Exec(schemaSQL); err != nil {
		return fmt.Errorf("apply schema: %w", err)
	}
	return nil
}

// Path returns the on-disk path of the database file.
func (s *Store) Path() string { return s.path }

// Close closes the underlying database.
func (s *Store) Close() error { return s.db.Close() }
