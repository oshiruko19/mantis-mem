package store

import (
	"database/sql"
	"errors"
	"fmt"
)

// UpsertProject inserts the project keyed by canonical path, or updates its name,
// source, and last-active timestamp if it already exists. The stored row is returned.
func (s *Store) UpsertProject(name, path, source string) (*Project, error) {
	const q = `
INSERT INTO projects(name, path, source) VALUES(?, ?, ?)
ON CONFLICT(path) DO UPDATE SET
    name = excluded.name,
    source = excluded.source,
    last_active_at = strftime('%Y-%m-%dT%H:%M:%fZ','now')
RETURNING id, name, path, source, created_at, last_active_at;`
	var p Project
	err := s.db.QueryRow(q, name, path, source).Scan(
		&p.ID, &p.Name, &p.Path, &p.Source, &p.CreatedAt, &p.LastActiveAt,
	)
	if err != nil {
		return nil, fmt.Errorf("upsert project: %w", err)
	}
	return &p, nil
}

// GetProjectByPath returns the project with the given canonical path, or nil if none.
func (s *Store) GetProjectByPath(path string) (*Project, error) {
	const q = `SELECT id, name, path, source, created_at, last_active_at FROM projects WHERE path = ?;`
	var p Project
	err := s.db.QueryRow(q, path).Scan(
		&p.ID, &p.Name, &p.Path, &p.Source, &p.CreatedAt, &p.LastActiveAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get project: %w", err)
	}
	return &p, nil
}
