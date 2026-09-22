//go:build e2e

// End-to-end tests: compile the real mantis-mem binary and drive `serve` as a
// subprocess over actual stdio, using the MCP SDK's CommandTransport. This
// exercises the true production path (process launch, stdio JSON-RPC, DB on disk),
// unlike server_test.go which connects in-process.
//
// Run with:  go test -tags e2e ./internal/mcpserver/...
package mcpserver_test

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// buildBinary compiles the module's main package into a temp file.
func buildBinary(t *testing.T) string {
	t.Helper()
	bin := filepath.Join(t.TempDir(), "mantis-mem-e2e")
	build := exec.Command("go", "build", "-o", bin, "github.com/oshiruko19/mantis-mem")
	build.Env = append(os.Environ(), "CGO_ENABLED=0")
	if out, err := build.CombinedOutput(); err != nil {
		t.Fatalf("go build failed: %v\n%s", err, out)
	}
	return bin
}

func callE2E(t *testing.T, ctx context.Context, cs *mcp.ClientSession, name string, args map[string]any, out any) {
	t.Helper()
	res, err := cs.CallTool(ctx, &mcp.CallToolParams{Name: name, Arguments: args})
	if err != nil {
		t.Fatalf("CallTool %s: %v", name, err)
	}
	if res.IsError {
		t.Fatalf("CallTool %s returned an error result", name)
	}
	if out != nil {
		b, err := json.Marshal(res.StructuredContent)
		if err != nil {
			t.Fatalf("marshal %s output: %v", name, err)
		}
		if err := json.Unmarshal(b, out); err != nil {
			t.Fatalf("decode %s output: %v (raw=%s)", name, err, b)
		}
	}
}

func TestE2E_ServeOverStdio(t *testing.T) {
	bin := buildBinary(t)
	db := filepath.Join(t.TempDir(), "e2e.db")

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	cmd := exec.Command(bin, "serve")
	cmd.Env = append(os.Environ(), "MANTIS_DB="+db, "MANTIS_PROJECT=e2e")
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	t.Cleanup(func() {
		if t.Failed() && stderr.Len() > 0 {
			t.Logf("subprocess stderr:\n%s", stderr.String())
		}
	})

	client := mcp.NewClient(&mcp.Implementation{Name: "e2e-client", Version: "0"}, nil)
	cs, err := client.Connect(ctx, &mcp.CommandTransport{Command: cmd}, nil)
	if err != nil {
		t.Fatalf("connect to serve subprocess: %v", err)
	}
	t.Cleanup(func() { _ = cs.Close() })

	// tools/list — the real binary must advertise all 10 mem_* tools.
	lt, err := cs.ListTools(ctx, nil)
	if err != nil {
		t.Fatalf("ListTools: %v", err)
	}
	if len(lt.Tools) != 10 {
		t.Fatalf("expected 10 tools over stdio, got %d", len(lt.Tools))
	}

	// mem_save → mem_search → confirm the round-trip through a real process + disk DB.
	var saved struct {
		ID int64 `json:"id"`
	}
	callE2E(t, ctx, cs, "mem_save", map[string]any{
		"kind":  "discovery",
		"title": "e2e over stdio",
		"body":  "drove the real binary via CommandTransport",
	}, &saved)
	if saved.ID == 0 {
		t.Fatal("mem_save returned a zero id")
	}

	var search struct {
		Results []struct {
			ID int64 `json:"id"`
		} `json:"results"`
	}
	callE2E(t, ctx, cs, "mem_search", map[string]any{"query": "stdio"}, &search)
	if len(search.Results) == 0 || search.Results[0].ID != saved.ID {
		t.Fatalf("search did not return the saved observation: %+v", search.Results)
	}

	// mem_current_project must report the env-resolved project and the on-disk DB path.
	var proj struct {
		Name   string `json:"name"`
		Source string `json:"source"`
		DBPath string `json:"db_path"`
	}
	callE2E(t, ctx, cs, "mem_current_project", map[string]any{}, &proj)
	if proj.Source != "env" || proj.Name != "e2e" {
		t.Fatalf("unexpected project resolution: %+v", proj)
	}
	if proj.DBPath != db {
		t.Fatalf("db path mismatch: got %q want %q", proj.DBPath, db)
	}

	// The DB file must actually exist on disk after the round-trip.
	if _, err := os.Stat(db); err != nil {
		t.Fatalf("expected DB file at %s: %v", db, err)
	}
}
