package mcpserver

import (
	"context"
	"encoding/json"
	"path/filepath"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/oshiruko19/mantis-mem/internal/store"
)

// connectInProcess wires an MCP client to a fresh mantis-mem server over the
// SDK's in-memory transport and returns the connected client session.
func connectInProcess(t *testing.T) (*mcp.ClientSession, context.Context) {
	t.Helper()
	ctx := context.Background()

	st, err := store.Open(filepath.Join(t.TempDir(), "mcp.db"))
	if err != nil {
		t.Fatalf("store.Open: %v", err)
	}
	t.Cleanup(func() { st.Close() })

	server := New(st, "test")
	serverT, clientT := mcp.NewInMemoryTransports()
	ss, err := server.Connect(ctx, serverT, nil)
	if err != nil {
		t.Fatalf("server.Connect: %v", err)
	}
	t.Cleanup(func() { _ = ss.Close() })

	client := mcp.NewClient(&mcp.Implementation{Name: "test-client", Version: "0"}, nil)
	cs, err := client.Connect(ctx, clientT, nil)
	if err != nil {
		t.Fatalf("client.Connect: %v", err)
	}
	t.Cleanup(func() { _ = cs.Close() })
	return cs, ctx
}

func callTool(t *testing.T, cs *mcp.ClientSession, ctx context.Context, name string, args map[string]any, out any) *mcp.CallToolResult {
	t.Helper()
	res, err := cs.CallTool(ctx, &mcp.CallToolParams{Name: name, Arguments: args})
	if err != nil {
		t.Fatalf("CallTool %s: %v", name, err)
	}
	if res.IsError {
		t.Fatalf("CallTool %s returned tool error: %s", name, contentText(res))
	}
	if out != nil {
		b, err := json.Marshal(res.StructuredContent)
		if err != nil {
			t.Fatalf("marshal structured content: %v", err)
		}
		if err := json.Unmarshal(b, out); err != nil {
			t.Fatalf("decode %s output: %v (raw=%s)", name, err, b)
		}
	}
	return res
}

func contentText(res *mcp.CallToolResult) string {
	for _, c := range res.Content {
		if tc, ok := c.(*mcp.TextContent); ok {
			return tc.Text
		}
	}
	return ""
}

func TestListToolsExposesAllMemTools(t *testing.T) {
	cs, ctx := connectInProcess(t)
	res, err := cs.ListTools(ctx, nil)
	if err != nil {
		t.Fatalf("ListTools: %v", err)
	}
	got := map[string]bool{}
	for _, tool := range res.Tools {
		got[tool.Name] = true
	}
	want := []string{
		"mem_current_project", "mem_context", "mem_search", "mem_timeline",
		"mem_get_observation", "mem_save", "mem_append", "mem_session_summary", "mem_session_history", "mem_suggest_topic_key",
	}
	for _, name := range want {
		if !got[name] {
			t.Errorf("tool %q not registered", name)
		}
	}
	if len(res.Tools) != len(want) {
		t.Errorf("expected %d tools, got %d", len(want), len(res.Tools))
	}
}

func TestSaveSearchGetRoundTripOverMCP(t *testing.T) {
	cs, ctx := connectInProcess(t)

	var saved struct {
		ID       int64  `json:"id"`
		TopicKey string `json:"topic_key"`
		Updated  bool   `json:"updated"`
	}
	callTool(t, cs, ctx, "mem_save", map[string]any{
		"kind":      "bug",
		"title":     "Retry-safe upload",
		"body":      "Reuse the request id as the idempotency key.",
		"topic_key": "bug/upload-dupes",
		"files":     []string{"internal/upload/handler.go"},
	}, &saved)
	if saved.ID == 0 {
		t.Fatalf("expected a non-zero observation id")
	}

	var search struct {
		Results []store.SearchResult `json:"results"`
	}
	callTool(t, cs, ctx, "mem_search", map[string]any{"query": "idempotency"}, &search)
	if len(search.Results) == 0 || search.Results[0].ID != saved.ID {
		t.Fatalf("search did not return the saved observation: %+v", search.Results)
	}

	var got struct {
		Observation *store.Observation `json:"observation"`
		Found       bool               `json:"found"`
	}
	callTool(t, cs, ctx, "mem_get_observation", map[string]any{"id": saved.ID}, &got)
	if !got.Found || got.Observation == nil {
		t.Fatalf("expected to find observation %d", saved.ID)
	}
	if got.Observation.Body != "Reuse the request id as the idempotency key." {
		t.Fatalf("unexpected body: %q", got.Observation.Body)
	}
}

func TestSaveWithCommitSHAOverMCP(t *testing.T) {
	cs, ctx := connectInProcess(t)
	var saved struct {
		ID        int64  `json:"id"`
		CommitSHA string `json:"commit_sha"`
	}
	callTool(t, cs, ctx, "mem_save", map[string]any{
		"kind":       "decision",
		"title":      "Use sqlite WAL",
		"body":       "WAL mode for concurrent access.",
		"commit_sha": "abc1234567890def",
	}, &saved)
	if saved.CommitSHA != "abc1234567890def" {
		t.Fatalf("expected commit sha returned on save, got %q", saved.CommitSHA)
	}

	var got struct {
		Observation   *store.Observation `json:"observation"`
		Found         bool               `json:"found"`
		CurrentCommit string             `json:"current_commit"`
	}
	callTool(t, cs, ctx, "mem_get_observation", map[string]any{"id": saved.ID}, &got)
	if !got.Found || got.Observation == nil {
		t.Fatalf("expected to find observation %d", saved.ID)
	}
	if got.Observation.CommitSHA != "abc1234567890def" {
		t.Fatalf("expected commit sha %q, got %q", "abc1234567890def", got.Observation.CommitSHA)
	}
}

func TestSaveDuplicateNudgeOverMCP(t *testing.T) {
	cs, ctx := connectInProcess(t)
	var first saveOutput
	callTool(t, cs, ctx, "mem_save", map[string]any{
		"kind":      "pattern",
		"title":     "OAuth2 authentication flow and token refresh",
		"body":      "Use refresh tokens before access tokens expire.",
		"topic_key": "architecture/oauth2-flow",
	}, &first)
	if first.ID == 0 {
		t.Fatalf("first save failed")
	}

	var second saveOutput
	callTool(t, cs, ctx, "mem_save", map[string]any{
		"kind":  "pattern",
		"title": "OAuth2 authentication token handling",
		"body":  "Store tokens securely.",
	}, &second)
	if second.ID == 0 {
		t.Fatalf("second save failed")
	}
	if len(second.Similar) == 0 {
		t.Fatalf("expected near-duplicate detection to return similar observations")
	}
	if second.Similar[0].TopicKey != "architecture/oauth2-flow" {
		t.Fatalf("expected top similar topic_key %q, got %q", "architecture/oauth2-flow", second.Similar[0].TopicKey)
	}
	if second.Nudge == "" {
		t.Fatalf("expected a nudge message nudging toward topic_key reuse")
	}
}

func TestSaveRejectsInvalidKind(t *testing.T) {
	cs, ctx := connectInProcess(t)
	res, err := cs.CallTool(ctx, &mcp.CallToolParams{
		Name:      "mem_save",
		Arguments: map[string]any{"kind": "nonsense", "title": "x", "body": "y"},
	})
	if err != nil {
		t.Fatalf("CallTool transport error: %v", err)
	}
	if !res.IsError {
		t.Fatalf("expected an invalid-kind tool error, got success")
	}
}

func TestCurrentProjectReportsSource(t *testing.T) {
	cs, ctx := connectInProcess(t)
	var out struct {
		Name   string `json:"name"`
		Source string `json:"source"`
		DBPath string `json:"db_path"`
	}
	callTool(t, cs, ctx, "mem_current_project", map[string]any{}, &out)
	if out.Name == "" || out.Source == "" {
		t.Fatalf("expected project name and source, got %+v", out)
	}
	if out.DBPath == "" {
		t.Fatalf("expected a db path in the response")
	}
}

func TestSessionSummaryAndHistoryOverMCP(t *testing.T) {
	cs, ctx := connectInProcess(t)
	var out struct {
		ID        int64 `json:"id"`
		ProjectID int64 `json:"project_id"`
	}
	callTool(t, cs, ctx, "mem_session_summary", map[string]any{
		"session_id": "sess-1",
		"goal":       "first goal",
	}, &out)
	callTool(t, cs, ctx, "mem_session_summary", map[string]any{
		"session_id": "sess-2",
		"goal":       "second goal",
	}, &out)

	var hist struct {
		Summaries []store.SessionSummary `json:"summaries"`
	}
	callTool(t, cs, ctx, "mem_session_history", map[string]any{"limit": 5}, &hist)
	if len(hist.Summaries) != 2 {
		t.Fatalf("expected 2 session summaries, got %d", len(hist.Summaries))
	}
	if hist.Summaries[0].SessionID != "sess-2" || hist.Summaries[1].SessionID != "sess-1" {
		t.Fatalf("unexpected summary order: %+v", hist.Summaries)
	}
}
