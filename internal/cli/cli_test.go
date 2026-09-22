package cli

import (
	"bytes"
	"path/filepath"
	"strings"
	"testing"

	"github.com/oshiruko19/mantis-mem/internal/store"
)

// newEnv points the CLI at a fresh temp DB and a deterministic (env-resolved)
// project, so tests are hermetic and observation ids start at 1.
func newEnv(t *testing.T) {
	t.Helper()
	t.Setenv("MANTIS_DB", filepath.Join(t.TempDir(), "cli.db"))
	t.Setenv("MANTIS_PROJECT", "clitest")
}

func run(t *testing.T, args ...string) (string, error) {
	t.Helper()
	var buf bytes.Buffer
	err := Run(args, "9.9.9", &buf)
	return buf.String(), err
}

func TestVersion(t *testing.T) {
	out, err := run(t, "version")
	if err != nil {
		t.Fatalf("version: %v", err)
	}
	if !strings.Contains(out, "mantis-mem 9.9.9") {
		t.Fatalf("version output: %q", out)
	}
}

func TestNoCommand(t *testing.T) {
	if _, err := run(t); err == nil {
		t.Fatal("expected an error when no command is given")
	}
}

func TestUnknownCommand(t *testing.T) {
	if _, err := run(t, "frobnicate"); err == nil {
		t.Fatal("expected an error for an unknown command")
	}
}

func TestSaveThenSearch(t *testing.T) {
	newEnv(t)
	out, err := run(t, "save", "--kind", "bug", "--title", "Login crash", "--body", "fixed the crash", "--commit", "deadbeef12345678")
	if err != nil {
		t.Fatalf("save: %v", err)
	}
	if !strings.Contains(out, "saved observation #1") || !strings.Contains(out, "commit deadbeef") {
		t.Fatalf("save output: %q", out)
	}

	out, err = run(t, "search", "Login")
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	if !strings.Contains(out, "Login crash") {
		t.Fatalf("search did not find the saved observation: %q", out)
	}
}

func TestAppendToObservation(t *testing.T) {
	newEnv(t)
	if _, err := run(t, "save", "--kind", "feature", "--title", "Streaming notes", "--body", "What: initial plan"); err != nil {
		t.Fatalf("save: %v", err)
	}
	out, err := run(t, "append", "--id", "1", "--note", "Progress: added the append path", "--session", "s1")
	if err != nil {
		t.Fatalf("append: %v", err)
	}
	if !strings.Contains(out, "appended to observation #1") {
		t.Fatalf("append output: %q", out)
	}
	out, err = run(t, "get", "1")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if !strings.Contains(out, "What: initial plan") || !strings.Contains(out, "Progress: added the append path") {
		t.Fatalf("append should preserve prior body and add the note: %q", out)
	}
}

func TestAppendMissingObservation(t *testing.T) {
	newEnv(t)
	if _, err := run(t, "append", "--id", "42", "--note", "x"); err == nil {
		t.Fatal("expected an error appending to a missing observation")
	}
}

func TestAppendRequiresIDAndNote(t *testing.T) {
	newEnv(t)
	if _, err := run(t, "append", "--note", "x"); err == nil {
		t.Fatal("expected an error when --id is missing")
	}
	if _, err := run(t, "append", "--id", "1", "--note", ""); err == nil {
		t.Fatal("expected an error when --note is empty")
	}
}

func TestSaveDuplicateNudge(t *testing.T) {
	newEnv(t)
	_, err := run(t, "save", "--kind", "pattern", "--title", "WebSocket reconnection protocol", "--body", "exponential backoff", "--topic", "net/ws-reconnect")
	if err != nil {
		t.Fatalf("save 1: %v", err)
	}

	out, err := run(t, "save", "--kind", "pattern", "--title", "WebSocket reconnection handling", "--body", "retry with jitter")
	if err != nil {
		t.Fatalf("save 2: %v", err)
	}
	if !strings.Contains(out, "similar observation(s) found") {
		t.Fatalf("expected near-duplicate notice in output: %q", out)
	}
	if !strings.Contains(out, "net/ws-reconnect") {
		t.Fatalf("expected suggestion to reuse topic net/ws-reconnect: %q", out)
	}
}

func TestSearchNoMatches(t *testing.T) {
	newEnv(t)
	out, err := run(t, "search", "nothingheresorry")
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	if !strings.Contains(out, "no matches") {
		t.Fatalf("expected 'no matches', got %q", out)
	}
}

func TestSaveInvalidKind(t *testing.T) {
	newEnv(t)
	_, err := run(t, "save", "--kind", "nonsense", "--title", "x", "--body", "y")
	if err == nil || !strings.Contains(err.Error(), "invalid") {
		t.Fatalf("expected invalid-kind error, got %v", err)
	}
}

func TestSaveRequiresTitleAndBody(t *testing.T) {
	newEnv(t)
	if _, err := run(t, "save", "--kind", "bug", "--title", "", "--body", ""); err == nil {
		t.Fatal("expected error when title and body are empty")
	}
}

func TestGet(t *testing.T) {
	newEnv(t)
	if _, err := run(t, "save", "--kind", "note", "--title", "Convention", "--body", "use tabs not spaces"); err != nil {
		t.Fatalf("save: %v", err)
	}
	out, err := run(t, "get", "1")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if !strings.Contains(out, "Convention") || !strings.Contains(out, "use tabs not spaces") {
		t.Fatalf("get output missing content: %q", out)
	}
}

func TestGetNotFound(t *testing.T) {
	newEnv(t)
	_, err := run(t, "get", "999")
	if err == nil || !strings.Contains(err.Error(), "not found") {
		t.Fatalf("expected not-found error, got %v", err)
	}
}

func TestGetInvalidID(t *testing.T) {
	newEnv(t)
	if _, err := run(t, "get", "abc"); err == nil {
		t.Fatal("expected error for non-numeric id")
	}
}

func TestProject(t *testing.T) {
	newEnv(t)
	out, err := run(t, "project")
	if err != nil {
		t.Fatalf("project: %v", err)
	}
	if !strings.Contains(out, "clitest") || !strings.Contains(out, "env") {
		t.Fatalf("project output: %q", out)
	}
}

func TestContext(t *testing.T) {
	newEnv(t)
	if _, err := run(t, "save", "--kind", "decision", "--title", "Chose WAL", "--body", "concurrent access"); err != nil {
		t.Fatalf("save: %v", err)
	}
	out, err := run(t, "context")
	if err != nil {
		t.Fatalf("context: %v", err)
	}
	if !strings.Contains(out, "clitest") || !strings.Contains(out, "Chose WAL") {
		t.Fatalf("context output: %q", out)
	}
}

func TestTimeline(t *testing.T) {
	newEnv(t)
	if _, err := run(t, "save", "--kind", "feature", "--title", "Bulk import", "--body", "csv endpoint"); err != nil {
		t.Fatalf("save: %v", err)
	}
	out, err := run(t, "timeline")
	if err != nil {
		t.Fatalf("timeline: %v", err)
	}
	if !strings.Contains(out, "Bulk import") {
		t.Fatalf("timeline output: %q", out)
	}
}

func TestSuggestTopic(t *testing.T) {
	out, err := run(t, "suggest-topic", "--kind", "pattern", "--title", "Auth Model")
	if err != nil {
		t.Fatalf("suggest-topic: %v", err)
	}
	if strings.TrimSpace(out) != "architecture/auth-model" {
		t.Fatalf("suggest-topic output: %q", out)
	}
}

func TestSuggestTopicRequiresTitle(t *testing.T) {
	if _, err := run(t, "suggest-topic", "--kind", "bug"); err == nil {
		t.Fatal("expected error when --title is missing")
	}
}

func TestSessions(t *testing.T) {
	newEnv(t)
	out, err := run(t, "sessions")
	if err != nil {
		t.Fatalf("sessions: %v", err)
	}
	if !strings.Contains(out, "no session summaries found") {
		t.Fatalf("expected no summaries, got: %q", out)
	}

	st, err := openStore("")
	if err != nil {
		t.Fatalf("openStore: %v", err)
	}
	defer st.Close()
	p, err := resolveProject(st)
	if err != nil {
		t.Fatalf("resolveProject: %v", err)
	}
	if _, err := st.UpsertSessionSummary(&store.SessionSummary{
		ProjectID: p.ID, SessionID: "sess-abc", Goal: "refactor memory",
	}); err != nil {
		t.Fatalf("UpsertSessionSummary: %v", err)
	}

	out, err = run(t, "sessions")
	if err != nil {
		t.Fatalf("sessions after insert: %v", err)
	}
	if !strings.Contains(out, "sess-abc") || !strings.Contains(out, "refactor memory") {
		t.Fatalf("sessions output missing summary: %q", out)
	}
}

func TestInit(t *testing.T) {
	newEnv(t)
	out, err := run(t, "init")
	if err != nil {
		t.Fatalf("init: %v", err)
	}
	if !strings.Contains(out, "initialized") {
		t.Fatalf("init output: %q", out)
	}
}

func TestSaveTopicUpdateInPlace(t *testing.T) {
	newEnv(t)
	if _, err := run(t, "save", "--kind", "pattern", "--title", "Auth v1", "--body", "cookies", "--topic", "architecture/auth"); err != nil {
		t.Fatalf("save v1: %v", err)
	}
	out, err := run(t, "save", "--kind", "pattern", "--title", "Auth v2", "--body", "jwt", "--topic", "architecture/auth")
	if err != nil {
		t.Fatalf("save v2: %v", err)
	}
	if !strings.Contains(out, "updated observation #1") {
		t.Fatalf("expected in-place update of #1, got %q", out)
	}
}
