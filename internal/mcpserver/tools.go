package mcpserver

import (
	"context"
	"fmt"
	"strings"
	"unicode"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/oshiruko19/mantis-mem/internal/project"
	"github.com/oshiruko19/mantis-mem/internal/store"
)

// Handler holds the dependencies shared by all mem_* tool handlers.
type Handler struct {
	Store   *store.Store
	Version string
}

// currentProject resolves the working directory to a project and records activity.
func (h *Handler) currentProject() (*store.Project, *project.Resolved, error) {
	r, err := project.Resolve()
	if err != nil {
		return nil, nil, fmt.Errorf("resolve project: %w", err)
	}
	p, err := h.Store.UpsertProject(r.Name, r.Path, r.Source)
	if err != nil {
		return nil, nil, err
	}
	return p, r, nil
}

// --- shared DTOs ---------------------------------------------------------------

type observationPreview struct {
	ID        int64  `json:"id"`
	Kind      string `json:"kind"`
	Title     string `json:"title"`
	TopicKey  string `json:"topic_key,omitempty"`
	SessionID string `json:"session_id,omitempty"`
	CreatedAt string `json:"created_at"`
}

func toPreview(o store.Observation) observationPreview {
	return observationPreview{
		ID: o.ID, Kind: o.Kind, Title: o.Title,
		TopicKey: o.TopicKey, SessionID: o.SessionID, CreatedAt: o.CreatedAt,
	}
}

// --- mem_current_project -------------------------------------------------------

type emptyInput struct{}

type currentProjectOutput struct {
	Name      string `json:"name"`
	Path      string `json:"path"`
	Source    string `json:"source"`
	ProjectID int64  `json:"project_id"`
	DBPath    string `json:"db_path"`
}

func (h *Handler) MemCurrentProject(_ context.Context, _ *mcp.CallToolRequest, _ emptyInput) (*mcp.CallToolResult, currentProjectOutput, error) {
	p, _, err := h.currentProject()
	if err != nil {
		return nil, currentProjectOutput{}, err
	}
	return nil, currentProjectOutput{
		Name: p.Name, Path: p.Path, Source: p.Source, ProjectID: p.ID, DBPath: h.Store.Path(),
	}, nil
}

// --- mem_context ---------------------------------------------------------------

type contextInput struct {
	Limit int `json:"limit,omitempty" jsonschema:"how many recent observations to include (default 10)"`
}

type contextOutput struct {
	Project       currentProjectOutput  `json:"project"`
	LatestSummary *store.SessionSummary `json:"latest_summary,omitempty"`
	RecentObs     []observationPreview  `json:"recent_observations"`
}

func (h *Handler) MemContext(_ context.Context, _ *mcp.CallToolRequest, in contextInput) (*mcp.CallToolResult, contextOutput, error) {
	p, _, err := h.currentProject()
	if err != nil {
		return nil, contextOutput{}, err
	}
	summary, err := h.Store.LatestSessionSummary(p.ID)
	if err != nil {
		return nil, contextOutput{}, err
	}
	recent, err := h.Store.RecentObservations(p.ID, in.Limit)
	if err != nil {
		return nil, contextOutput{}, err
	}
	previews := make([]observationPreview, 0, len(recent))
	for _, o := range recent {
		previews = append(previews, toPreview(o))
	}
	return nil, contextOutput{
		Project:       currentProjectOutput{Name: p.Name, Path: p.Path, Source: p.Source, ProjectID: p.ID, DBPath: h.Store.Path()},
		LatestSummary: summary,
		RecentObs:     previews,
	}, nil
}

// --- mem_search ----------------------------------------------------------------

type searchInput struct {
	Query string `json:"query" jsonschema:"full-text search terms scoped to the current project"`
	Limit int    `json:"limit,omitempty" jsonschema:"maximum results to return (default 20)"`
}

type searchOutput struct {
	Results []store.SearchResult `json:"results"`
}

func (h *Handler) MemSearch(_ context.Context, _ *mcp.CallToolRequest, in searchInput) (*mcp.CallToolResult, searchOutput, error) {
	if strings.TrimSpace(in.Query) == "" {
		return nil, searchOutput{}, fmt.Errorf("query is required")
	}
	p, _, err := h.currentProject()
	if err != nil {
		return nil, searchOutput{}, err
	}
	results, err := h.Store.SearchObservations(p.ID, in.Query, in.Limit)
	if err != nil {
		return nil, searchOutput{}, err
	}
	return nil, searchOutput{Results: results}, nil
}

// --- mem_timeline --------------------------------------------------------------

type timelineInput struct {
	SessionID string `json:"session_id,omitempty" jsonschema:"restrict to a single session"`
	Since     string `json:"since,omitempty" jsonschema:"lower time bound, RFC3339 (e.g. 2026-09-01T00:00:00Z)"`
	Limit     int    `json:"limit,omitempty" jsonschema:"maximum observations to return (default 50)"`
}

type timelineOutput struct {
	Observations []observationPreview `json:"observations"`
}

func (h *Handler) MemTimeline(_ context.Context, _ *mcp.CallToolRequest, in timelineInput) (*mcp.CallToolResult, timelineOutput, error) {
	p, _, err := h.currentProject()
	if err != nil {
		return nil, timelineOutput{}, err
	}
	obs, err := h.Store.Timeline(p.ID, in.SessionID, in.Since, in.Limit)
	if err != nil {
		return nil, timelineOutput{}, err
	}
	previews := make([]observationPreview, 0, len(obs))
	for _, o := range obs {
		previews = append(previews, toPreview(o))
	}
	return nil, timelineOutput{Observations: previews}, nil
}

// --- mem_get_observation -------------------------------------------------------

type getInput struct {
	ID int64 `json:"id" jsonschema:"the observation id to fetch"`
}

type getOutput struct {
	Observation *store.Observation `json:"observation"`
	Found       bool               `json:"found"`
}

func (h *Handler) MemGetObservation(_ context.Context, _ *mcp.CallToolRequest, in getInput) (*mcp.CallToolResult, getOutput, error) {
	o, err := h.Store.GetObservation(in.ID)
	if err != nil {
		return nil, getOutput{}, err
	}
	return nil, getOutput{Observation: o, Found: o != nil}, nil
}

// --- mem_save ------------------------------------------------------------------

type saveInput struct {
	Kind      string   `json:"kind" jsonschema:"one of: decision, bug, discovery, config, pattern, constraint, feature, note"`
	Title     string   `json:"title" jsonschema:"a concise, searchable title"`
	Body      string   `json:"body" jsonschema:"structured note; prefer What/Why/Where/Learned lines"`
	TopicKey  string   `json:"topic_key,omitempty" jsonschema:"stable slug (e.g. architecture/auth-model) to update an evolving topic in place"`
	Files     []string `json:"files,omitempty" jsonschema:"relevant file paths"`
	Tags      string   `json:"tags,omitempty" jsonschema:"space-separated tags for retrieval"`
	SessionID string   `json:"session_id,omitempty" jsonschema:"the current session identifier"`
}

type saveOutput struct {
	ID       int64  `json:"id"`
	TopicKey string `json:"topic_key,omitempty"`
	Updated  bool   `json:"updated"` // true when an existing topic was updated in place
}

func (h *Handler) MemSave(_ context.Context, _ *mcp.CallToolRequest, in saveInput) (*mcp.CallToolResult, saveOutput, error) {
	if strings.TrimSpace(in.Title) == "" || strings.TrimSpace(in.Body) == "" {
		return nil, saveOutput{}, fmt.Errorf("title and body are required")
	}
	if !store.ValidKind(in.Kind) {
		return nil, saveOutput{}, fmt.Errorf("invalid kind %q; must be one of %s", in.Kind, strings.Join(store.Kinds, ", "))
	}
	p, _, err := h.currentProject()
	if err != nil {
		return nil, saveOutput{}, err
	}
	saved, updated, err := h.Store.SaveObservation(&store.Observation{
		ProjectID: p.ID,
		SessionID: in.SessionID,
		TopicKey:  strings.TrimSpace(in.TopicKey),
		Kind:      in.Kind,
		Title:     in.Title,
		Body:      in.Body,
		Files:     in.Files,
		Tags:      in.Tags,
	})
	if err != nil {
		return nil, saveOutput{}, err
	}
	return nil, saveOutput{ID: saved.ID, TopicKey: saved.TopicKey, Updated: updated}, nil
}

// --- mem_session_summary -------------------------------------------------------

type sessionSummaryInput struct {
	SessionID    string   `json:"session_id" jsonschema:"the current session identifier"`
	Goal         string   `json:"goal,omitempty" jsonschema:"what this session set out to do"`
	Instructions string   `json:"instructions,omitempty" jsonschema:"durable user instructions/constraints"`
	Discoveries  string   `json:"discoveries,omitempty" jsonschema:"key things learned"`
	Accomplished string   `json:"accomplished,omitempty" jsonschema:"work completed"`
	NextSteps    string   `json:"next_steps,omitempty" jsonschema:"what to do next"`
	Files        []string `json:"files,omitempty" jsonschema:"relevant file paths"`
}

type sessionSummaryOutput struct {
	ID        int64 `json:"id"`
	ProjectID int64 `json:"project_id"`
}

func (h *Handler) MemSessionSummary(_ context.Context, _ *mcp.CallToolRequest, in sessionSummaryInput) (*mcp.CallToolResult, sessionSummaryOutput, error) {
	if strings.TrimSpace(in.SessionID) == "" {
		return nil, sessionSummaryOutput{}, fmt.Errorf("session_id is required")
	}
	p, _, err := h.currentProject()
	if err != nil {
		return nil, sessionSummaryOutput{}, err
	}
	ss, err := h.Store.UpsertSessionSummary(&store.SessionSummary{
		ProjectID:    p.ID,
		SessionID:    in.SessionID,
		Goal:         in.Goal,
		Instructions: in.Instructions,
		Discoveries:  in.Discoveries,
		Accomplished: in.Accomplished,
		NextSteps:    in.NextSteps,
		Files:        in.Files,
	})
	if err != nil {
		return nil, sessionSummaryOutput{}, err
	}
	return nil, sessionSummaryOutput{ID: ss.ID, ProjectID: ss.ProjectID}, nil
}

// --- mem_suggest_topic_key -----------------------------------------------------

type suggestInput struct {
	Kind  string `json:"kind,omitempty" jsonschema:"observation kind, used as the namespace"`
	Title string `json:"title" jsonschema:"the title to derive a slug from"`
}

type suggestOutput struct {
	TopicKey string `json:"topic_key"`
}

func (h *Handler) MemSuggestTopicKey(_ context.Context, _ *mcp.CallToolRequest, in suggestInput) (*mcp.CallToolResult, suggestOutput, error) {
	return nil, suggestOutput{TopicKey: SuggestTopicKey(in.Kind, in.Title)}, nil
}

// SuggestTopicKey builds a stable "<namespace>/<kebab-title>" slug. The namespace
// maps kinds to durable topic areas; the title is slugified.
func SuggestTopicKey(kind, title string) string {
	ns := map[string]string{
		"decision":   "decision",
		"bug":        "bug",
		"discovery":  "discovery",
		"config":     "config",
		"pattern":    "architecture",
		"constraint": "constraint",
		"feature":    "feature",
		"note":       "note",
	}[kind]
	if ns == "" {
		ns = "topic"
	}
	return ns + "/" + slugify(title)
}

func slugify(s string) string {
	var b strings.Builder
	lastDash := true // avoid leading dash
	for _, r := range strings.ToLower(s) {
		switch {
		case unicode.IsLetter(r) || unicode.IsNumber(r):
			b.WriteRune(r)
			lastDash = false
		case !lastDash:
			b.WriteRune('-')
			lastDash = true
		}
	}
	out := strings.Trim(b.String(), "-")
	if out == "" {
		return "untitled"
	}
	// keep slugs short and readable
	const maxLen = 48
	if len(out) > maxLen {
		out = strings.Trim(out[:maxLen], "-")
	}
	return out
}
