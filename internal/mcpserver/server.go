// Package mcpserver exposes the mantis-mem tools over an MCP stdio server.
package mcpserver

import (
	"context"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/oshiruko19/mantis-mem/internal/store"
)

// New builds an MCP server with all mem_* tools registered against st.
func New(st *store.Store, version string) *mcp.Server {
	h := &Handler{Store: st, Version: version}
	s := mcp.NewServer(&mcp.Implementation{Name: "mantis-mem", Version: version}, nil)

	mcp.AddTool(s, &mcp.Tool{
		Name:        "mem_current_project",
		Description: "Confirm the resolved project (name, canonical path, how it was detected) and the memory DB path. Call this first to orient.",
	}, h.MemCurrentProject)

	mcp.AddTool(s, &mcp.Tool{
		Name:        "mem_context",
		Description: "Recover recent history for the current project: the latest session summary plus recent observation previews.",
	}, h.MemContext)

	mcp.AddTool(s, &mcp.Tool{
		Name:        "mem_search",
		Description: "Full-text search the current project's memory. Returns ranked previews (not full records); use mem_get_observation for the full text.",
	}, h.MemSearch)

	mcp.AddTool(s, &mcp.Tool{
		Name:        "mem_timeline",
		Description: "List observations in chronological order for the current project, optionally scoped to a session or a start time.",
	}, h.MemTimeline)

	mcp.AddTool(s, &mcp.Tool{
		Name:        "mem_get_observation",
		Description: "Fetch the full, complete record for a single observation by id.",
	}, h.MemGetObservation)

	mcp.AddTool(s, &mcp.Tool{
		Name:        "mem_save",
		Description: "Save durable project knowledge (decision, bug fix, discovery, config change, pattern, constraint, feature, note). Reuse a topic_key to update an evolving topic in place. Do not save raw tool output or routine chatter.",
	}, h.MemSave)

	mcp.AddTool(s, &mcp.Tool{
		Name:        "mem_session_summary",
		Description: "Save or update a session handoff: goal, instructions, discoveries, accomplished work, next steps, and relevant files.",
	}, h.MemSessionSummary)

	mcp.AddTool(s, &mcp.Tool{
		Name:        "mem_suggest_topic_key",
		Description: "Suggest a stable topic_key slug (namespace/kebab-title) for an evolving topic when the key is unclear.",
	}, h.MemSuggestTopicKey)

	return s
}

// Serve opens the store at dbPath and runs the MCP server over stdio until the
// client disconnects. All diagnostics go to stderr; stdout carries JSON-RPC only.
func Serve(ctx context.Context, dbPath, version string) error {
	st, err := store.Open(dbPath)
	if err != nil {
		return err
	}
	defer st.Close()
	return New(st, version).Run(ctx, &mcp.StdioTransport{})
}
