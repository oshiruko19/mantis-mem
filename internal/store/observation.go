package store

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"unicode"
)

// SaveObservation inserts a new observation. When TopicKey is non-empty and a row
// already exists for (project_id, topic_key), that row is updated in place instead.
// The stored row and whether it was an update are returned.
func (s *Store) SaveObservation(o *Observation) (saved *Observation, updated bool, err error) {
	files := marshalFiles(o.Files)

	if strings.TrimSpace(o.TopicKey) == "" {
		const q = `
INSERT INTO observations(project_id, session_id, commit_sha, topic_key, kind, title, body, files, tags)
VALUES(?, ?, ?, NULL, ?, ?, ?, ?, ?)
RETURNING id, project_id, session_id, commit_sha, COALESCE(topic_key,''), kind, title, body, files, tags, created_at, updated_at;`
		row := s.db.QueryRow(q, o.ProjectID, o.SessionID, o.CommitSHA, o.Kind, o.Title, o.Body, files, o.Tags)
		saved, err = scanObservation(row)
		return saved, false, err
	}

	// Existence check + upsert in one transaction so "updated" is reported
	// deterministically (a timestamp comparison races within the same millisecond).
	tx, err := s.db.Begin()
	if err != nil {
		return nil, false, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck // no-op after Commit

	var existingID int64
	err = tx.QueryRow(
		`SELECT id FROM observations WHERE project_id = ? AND topic_key = ?;`,
		o.ProjectID, o.TopicKey,
	).Scan(&existingID)
	exists := err == nil
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, false, fmt.Errorf("check topic: %w", err)
	}

	// Upsert against the partial unique index (topic_key IS NOT NULL).
	const q = `
INSERT INTO observations(project_id, session_id, commit_sha, topic_key, kind, title, body, files, tags)
VALUES(?, ?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(project_id, topic_key) WHERE topic_key IS NOT NULL DO UPDATE SET
    session_id = excluded.session_id,
    commit_sha = CASE WHEN excluded.commit_sha != '' THEN excluded.commit_sha ELSE observations.commit_sha END,
    kind = excluded.kind,
    title = excluded.title,
    body = excluded.body,
    files = excluded.files,
    tags = excluded.tags,
    updated_at = strftime('%Y-%m-%dT%H:%M:%fZ','now')
RETURNING id, project_id, session_id, commit_sha, COALESCE(topic_key,''), kind, title, body, files, tags, created_at, updated_at;`
	row := tx.QueryRow(q, o.ProjectID, o.SessionID, o.CommitSHA, o.TopicKey, o.Kind, o.Title, o.Body, files, o.Tags)
	saved, err = scanObservation(row)
	if err != nil {
		return nil, false, err
	}
	if err = tx.Commit(); err != nil {
		return nil, false, fmt.Errorf("commit: %w", err)
	}
	return saved, exists, nil
}

// GetObservation returns the full observation by id, or nil if not found.
func (s *Store) GetObservation(id int64) (*Observation, error) {
	const q = `
SELECT id, project_id, session_id, commit_sha, COALESCE(topic_key,''), kind, title, body, files, tags, created_at, updated_at
FROM observations WHERE id = ?;`
	o, err := scanObservation(s.db.QueryRow(q, id))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	return o, err
}

// SearchObservations runs a full-text search scoped to a project, returning
// bm25-ranked previews (best first). The query is sanitized into a safe FTS5 expression.
func (s *Store) SearchObservations(projectID int64, query string, limit int) ([]SearchResult, error) {
	match := ftsQuery(query)
	if match == "" {
		return []SearchResult{}, nil
	}
	if limit <= 0 {
		limit = 20
	}
	const q = `
SELECT o.id, o.title,
       snippet(observations_fts, 1, '[', ']', '…', 12) AS snip,
       o.kind, COALESCE(o.topic_key,''), bm25(observations_fts) AS score, o.created_at
FROM observations_fts
JOIN observations o ON o.id = observations_fts.rowid
WHERE observations_fts MATCH ? AND o.project_id = ?
ORDER BY score
LIMIT ?;`
	rows, err := s.db.Query(q, match, projectID, limit)
	if err != nil {
		return nil, fmt.Errorf("search: %w", err)
	}
	defer rows.Close()

	out := []SearchResult{}
	for rows.Next() {
		var r SearchResult
		if err := rows.Scan(&r.ID, &r.Title, &r.Snippet, &r.Kind, &r.TopicKey, &r.Score, &r.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan search row: %w", err)
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// FindSimilar finds existing observations in the same project that have high
// text similarity to the provided title, excluding excludeID (if non-zero).
// Useful for detecting duplicate or near-duplicate observations and nudging
// toward topic_key reuse.
func (s *Store) FindSimilar(projectID int64, title string, excludeID int64, limit int) ([]SearchResult, error) {
	if limit <= 0 {
		limit = 3
	}
	words := extractSignificantWords(title)
	if len(words) == 0 {
		return []SearchResult{}, nil
	}
	quoted := make([]string, 0, len(words))
	for _, w := range words {
		quoted = append(quoted, `"`+w+`"`)
	}
	match := strings.Join(quoted, " OR ")

	const q = `
SELECT o.id, o.title,
       snippet(observations_fts, 1, '[', ']', '…', 12) AS snip,
       o.kind, COALESCE(o.topic_key,''), bm25(observations_fts, 5.0, 1.0, 2.0) AS score, o.created_at
FROM observations_fts
JOIN observations o ON o.id = observations_fts.rowid
WHERE observations_fts MATCH ? AND o.project_id = ? AND o.id != ?
ORDER BY score
LIMIT ?;`
	rows, err := s.db.Query(q, match, projectID, excludeID, limit)
	if err != nil {
		return nil, fmt.Errorf("find similar: %w", err)
	}
	defer rows.Close()

	out := []SearchResult{}
	for rows.Next() {
		var r SearchResult
		if err := rows.Scan(&r.ID, &r.Title, &r.Snippet, &r.Kind, &r.TopicKey, &r.Score, &r.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan similar row: %w", err)
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

func extractSignificantWords(raw string) []string {
	fields := strings.FieldsFunc(raw, func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsNumber(r)
	})
	var words []string
	for _, f := range fields {
		if len(f) >= 3 {
			words = append(words, f)
		}
	}
	if len(words) == 0 {
		for _, f := range fields {
			if len(f) >= 2 {
				words = append(words, f)
			}
		}
	}
	return words
}

// RecentObservations returns the most recently created observations for a project.
func (s *Store) RecentObservations(projectID int64, limit int) ([]Observation, error) {
	if limit <= 0 {
		limit = 10
	}
	const q = `
SELECT id, project_id, session_id, commit_sha, COALESCE(topic_key,''), kind, title, body, files, tags, created_at, updated_at
FROM observations WHERE project_id = ?
ORDER BY created_at DESC, id DESC
LIMIT ?;`
	return s.queryObservations(q, projectID, limit)
}

// Timeline returns observations in chronological order for a project, optionally
// filtered to a session and/or a lower time bound (RFC3339 string).
func (s *Store) Timeline(projectID int64, sessionID, since string, limit int) ([]Observation, error) {
	if limit <= 0 {
		limit = 50
	}
	var (
		q    strings.Builder
		args []any
	)
	q.WriteString(`
SELECT id, project_id, session_id, commit_sha, COALESCE(topic_key,''), kind, title, body, files, tags, created_at, updated_at
FROM observations WHERE project_id = ?`)
	args = append(args, projectID)
	if strings.TrimSpace(sessionID) != "" {
		q.WriteString(" AND session_id = ?")
		args = append(args, sessionID)
	}
	if strings.TrimSpace(since) != "" {
		q.WriteString(" AND created_at >= ?")
		args = append(args, since)
	}
	q.WriteString(" ORDER BY created_at ASC, id ASC LIMIT ?;")
	args = append(args, limit)
	return s.queryObservations(q.String(), args...)
}

func (s *Store) queryObservations(q string, args ...any) ([]Observation, error) {
	rows, err := s.db.Query(q, args...)
	if err != nil {
		return nil, fmt.Errorf("query observations: %w", err)
	}
	defer rows.Close()
	out := []Observation{}
	for rows.Next() {
		o, err := scanObservation(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *o)
	}
	return out, rows.Err()
}

// scanner is satisfied by both *sql.Row and *sql.Rows.
type scanner interface {
	Scan(dest ...any) error
}

func scanObservation(sc scanner) (*Observation, error) {
	var o Observation
	var files string
	if err := sc.Scan(
		&o.ID, &o.ProjectID, &o.SessionID, &o.CommitSHA, &o.TopicKey, &o.Kind,
		&o.Title, &o.Body, &files, &o.Tags, &o.CreatedAt, &o.UpdatedAt,
	); err != nil {
		return nil, err
	}
	o.Files = unmarshalFiles(files)
	return &o, nil
}

func marshalFiles(files []string) string {
	if len(files) == 0 {
		return "[]"
	}
	b, err := json.Marshal(files)
	if err != nil {
		return "[]"
	}
	return string(b)
}

func unmarshalFiles(s string) []string {
	if s == "" {
		return []string{}
	}
	var out []string
	if err := json.Unmarshal([]byte(s), &out); err != nil {
		return []string{}
	}
	return out
}

// ftsQuery turns arbitrary user text into a safe FTS5 MATCH expression: each token
// becomes a quoted phrase joined by implicit AND. This avoids MATCH syntax errors
// from stray operators and is injection-safe.
func ftsQuery(raw string) string {
	fields := strings.FieldsFunc(raw, func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsNumber(r)
	})
	quoted := make([]string, 0, len(fields))
	for _, f := range fields {
		quoted = append(quoted, `"`+f+`"`)
	}
	return strings.Join(quoted, " ")
}
