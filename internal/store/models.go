package store

// Kinds are the allowed observation categories.
var Kinds = []string{"decision", "bug", "discovery", "config", "pattern", "constraint", "feature", "note"}

// ValidKind reports whether k is a recognized observation kind.
func ValidKind(k string) bool {
	for _, v := range Kinds {
		if v == k {
			return true
		}
	}
	return false
}

// Project is a resolved memory scope (a codebase / working context).
type Project struct {
	ID           int64  `json:"id"`
	Name         string `json:"name"`
	Path         string `json:"path"`
	Source       string `json:"source"`
	CreatedAt    string `json:"created_at"`
	LastActiveAt string `json:"last_active_at"`
}

// Observation is a single curated memory record.
type Observation struct {
	ID        int64    `json:"id"`
	ProjectID int64    `json:"project_id"`
	SessionID string   `json:"session_id,omitempty"`
	CommitSHA string   `json:"commit_sha,omitempty"`
	TopicKey  string   `json:"topic_key,omitempty"`
	Kind      string   `json:"kind"`
	Title     string   `json:"title"`
	Body      string   `json:"body"`
	Files     []string `json:"files"`
	Tags      string   `json:"tags,omitempty"`
	CreatedAt string   `json:"created_at"`
	UpdatedAt string   `json:"updated_at"`
}

// SearchResult is a lightweight preview returned by full-text search.
type SearchResult struct {
	ID        int64   `json:"id"`
	Title     string  `json:"title"`
	Snippet   string  `json:"snippet"`
	Kind      string  `json:"kind"`
	TopicKey  string  `json:"topic_key,omitempty"`
	Score     float64 `json:"score"`
	CreatedAt string  `json:"created_at"`
}

// SessionSummary is the handoff record for a working session.
type SessionSummary struct {
	ID           int64    `json:"id"`
	ProjectID    int64    `json:"project_id"`
	SessionID    string   `json:"session_id"`
	Goal         string   `json:"goal,omitempty"`
	Instructions string   `json:"instructions,omitempty"`
	Discoveries  string   `json:"discoveries,omitempty"`
	Accomplished string   `json:"accomplished,omitempty"`
	NextSteps    string   `json:"next_steps,omitempty"`
	Files        []string `json:"files"`
	CreatedAt    string   `json:"created_at"`
	UpdatedAt    string   `json:"updated_at"`
}
