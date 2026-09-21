-- mantis-mem schema. Applied idempotently on every Open().

CREATE TABLE IF NOT EXISTS projects (
    id             INTEGER PRIMARY KEY AUTOINCREMENT,
    name           TEXT NOT NULL,
    path           TEXT NOT NULL UNIQUE,               -- canonical identity; "env:<name>" for env override
    source         TEXT NOT NULL,                      -- git | cwd | env
    created_at     TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now')),
    last_active_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now'))
);

CREATE TABLE IF NOT EXISTS observations (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    project_id  INTEGER NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    session_id  TEXT NOT NULL DEFAULT '',
    topic_key   TEXT,                                  -- NULL for one-off notes; stable slug for evolving topics
    kind        TEXT NOT NULL,                         -- decision|bug|discovery|config|pattern|constraint|feature|note
    title       TEXT NOT NULL,
    body        TEXT NOT NULL,                         -- structured What/Why/Where/Learned text
    files       TEXT NOT NULL DEFAULT '[]',            -- JSON array of file paths
    tags        TEXT NOT NULL DEFAULT '',              -- space-separated tags
    created_at  TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now')),
    updated_at  TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now'))
);

CREATE INDEX IF NOT EXISTS idx_obs_project_created ON observations(project_id, created_at);

-- One row per (project, topic_key): re-saving an evolving topic updates in place.
CREATE UNIQUE INDEX IF NOT EXISTS idx_obs_topic ON observations(project_id, topic_key) WHERE topic_key IS NOT NULL;

CREATE TABLE IF NOT EXISTS session_summaries (
    id           INTEGER PRIMARY KEY AUTOINCREMENT,
    project_id   INTEGER NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    session_id   TEXT NOT NULL,
    goal         TEXT NOT NULL DEFAULT '',
    instructions TEXT NOT NULL DEFAULT '',
    discoveries  TEXT NOT NULL DEFAULT '',
    accomplished TEXT NOT NULL DEFAULT '',
    next_steps   TEXT NOT NULL DEFAULT '',
    files        TEXT NOT NULL DEFAULT '[]',
    created_at   TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now')),
    updated_at   TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now'))
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_session_summary ON session_summaries(project_id, session_id);

-- Full-text index over observations (external-content table kept in sync by triggers).
CREATE VIRTUAL TABLE IF NOT EXISTS observations_fts USING fts5(
    title, body, tags,
    content='observations',
    content_rowid='id',
    tokenize='porter unicode61'
);

CREATE TRIGGER IF NOT EXISTS observations_ai AFTER INSERT ON observations BEGIN
    INSERT INTO observations_fts(rowid, title, body, tags)
    VALUES (new.id, new.title, new.body, new.tags);
END;

CREATE TRIGGER IF NOT EXISTS observations_ad AFTER DELETE ON observations BEGIN
    INSERT INTO observations_fts(observations_fts, rowid, title, body, tags)
    VALUES ('delete', old.id, old.title, old.body, old.tags);
END;

CREATE TRIGGER IF NOT EXISTS observations_au AFTER UPDATE ON observations BEGIN
    INSERT INTO observations_fts(observations_fts, rowid, title, body, tags)
    VALUES ('delete', old.id, old.title, old.body, old.tags);
    INSERT INTO observations_fts(rowid, title, body, tags)
    VALUES (new.id, new.title, new.body, new.tags);
END;
