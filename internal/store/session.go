package store

import (
	"database/sql"
	"errors"
	"fmt"
)

// UpsertSessionSummary stores the handoff for a session, replacing any prior
// summary for the same (project_id, session_id). The stored row is returned.
func (s *Store) UpsertSessionSummary(ss *SessionSummary) (*SessionSummary, error) {
	files := marshalFiles(ss.Files)
	const q = `
INSERT INTO session_summaries(project_id, session_id, goal, instructions, discoveries, accomplished, next_steps, files)
VALUES(?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(project_id, session_id) DO UPDATE SET
    goal = excluded.goal,
    instructions = excluded.instructions,
    discoveries = excluded.discoveries,
    accomplished = excluded.accomplished,
    next_steps = excluded.next_steps,
    files = excluded.files,
    updated_at = strftime('%Y-%m-%dT%H:%M:%fZ','now')
RETURNING id, project_id, session_id, goal, instructions, discoveries, accomplished, next_steps, files, created_at, updated_at;`
	row := s.db.QueryRow(q, ss.ProjectID, ss.SessionID, ss.Goal, ss.Instructions,
		ss.Discoveries, ss.Accomplished, ss.NextSteps, files)
	return scanSessionSummary(row)
}

// LatestSessionSummary returns the most recently updated summary for a project, or nil.
func (s *Store) LatestSessionSummary(projectID int64) (*SessionSummary, error) {
	const q = `
SELECT id, project_id, session_id, goal, instructions, discoveries, accomplished, next_steps, files, created_at, updated_at
FROM session_summaries WHERE project_id = ?
ORDER BY updated_at DESC, id DESC
LIMIT 1;`
	ss, err := scanSessionSummary(s.db.QueryRow(q, projectID))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	return ss, err
}

// SessionSummaryHistory returns past session summaries for a project in reverse
// chronological order (most recent first).
func (s *Store) SessionSummaryHistory(projectID int64, limit int) ([]SessionSummary, error) {
	if limit <= 0 {
		limit = 10
	}
	const q = `
SELECT id, project_id, session_id, goal, instructions, discoveries, accomplished, next_steps, files, created_at, updated_at
FROM session_summaries WHERE project_id = ?
ORDER BY updated_at DESC, id DESC
LIMIT ?;`
	rows, err := s.db.Query(q, projectID, limit)
	if err != nil {
		return nil, fmt.Errorf("session summary history: %w", err)
	}
	defer rows.Close()

	out := []SessionSummary{}
	for rows.Next() {
		ss, err := scanSessionSummary(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *ss)
	}
	return out, rows.Err()
}

func scanSessionSummary(sc scanner) (*SessionSummary, error) {
	var ss SessionSummary
	var files string
	if err := sc.Scan(
		&ss.ID, &ss.ProjectID, &ss.SessionID, &ss.Goal, &ss.Instructions,
		&ss.Discoveries, &ss.Accomplished, &ss.NextSteps, &files, &ss.CreatedAt, &ss.UpdatedAt,
	); err != nil {
		return nil, err
	}
	ss.Files = unmarshalFiles(files)
	return &ss, nil
}
