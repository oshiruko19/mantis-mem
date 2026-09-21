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
		"mem_get_observation", "mem_save", "mem_session_summary", "mem_suggest_topic_key",
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
