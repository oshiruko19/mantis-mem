package store

import (
	"path/filepath"
	"testing"
)

func newTestStore(t *testing.T) *Store {
	t.Helper()
	dbPath := filepath.Join(t.TempDir(), "test.db")
	st, err := Open(dbPath)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() { st.Close() })
	return st
}

func mustProject(t *testing.T, st *Store) *Project {
	t.Helper()
	p, err := st.UpsertProject("proj", "/tmp/proj", "cwd")
	if err != nil {
		t.Fatalf("UpsertProject: %v", err)
	}
	return p
}

func TestUpsertProjectIsIdempotent(t *testing.T) {
	st := newTestStore(t)
	p1 := mustProject(t, st)
	p2, err := st.UpsertProject("proj-renamed", "/tmp/proj", "git")
	if err != nil {
		t.Fatalf("UpsertProject: %v", err)
	}
	if p1.ID != p2.ID {
		t.Fatalf("expected same project id, got %d and %d", p1.ID, p2.ID)
	}
	if p2.Name != "proj-renamed" || p2.Source != "git" {
		t.Fatalf("expected fields updated in place, got %+v", p2)
	}
}

func TestSaveGetRoundTrip(t *testing.T) {
	st := newTestStore(t)
	p := mustProject(t, st)
	saved, updated, err := st.SaveObservation(&Observation{
		ProjectID: p.ID, Kind: "bug", Title: "leak", Body: "fixed the leak",
		Files: []string{"a.go", "b.go"}, Tags: "mem",
	})
	if err != nil {
		t.Fatalf("SaveObservation: %v", err)
	}
	if updated {
		t.Fatalf("first save should not be an update")
	}
	got, err := st.GetObservation(saved.ID)
	if err != nil || got == nil {
		t.Fatalf("GetObservation: %v (got=%v)", err, got)
	}
	if got.Title != "leak" || got.Body != "fixed the leak" {
		t.Fatalf("unexpected observation: %+v", got)
	}
	if len(got.Files) != 2 || got.Files[0] != "a.go" {
		t.Fatalf("files did not round-trip: %+v", got.Files)
	}
}

func TestGetMissingReturnsNil(t *testing.T) {
	st := newTestStore(t)
	got, err := st.GetObservation(999)
	if err != nil {
		t.Fatalf("GetObservation error: %v", err)
	}
	if got != nil {
		t.Fatalf("expected nil for missing id, got %+v", got)
	}
}

func TestTopicKeyUpdatesInPlace(t *testing.T) {
	st := newTestStore(t)
	p := mustProject(t, st)
	first, _, err := st.SaveObservation(&Observation{
		ProjectID: p.ID, TopicKey: "architecture/auth", Kind: "pattern",
		Title: "auth v1", Body: "cookie sessions",
	})
	if err != nil {
		t.Fatalf("first save: %v", err)
	}
	second, updated, err := st.SaveObservation(&Observation{
		ProjectID: p.ID, TopicKey: "architecture/auth", Kind: "pattern",
		Title: "auth v2", Body: "jwt sessions",
	})
	if err != nil {
		t.Fatalf("second save: %v", err)
	}
	if !updated {
		t.Fatalf("second save with same topic_key should report updated=true")
	}
	if first.ID != second.ID {
		t.Fatalf("topic upsert should reuse row: %d vs %d", first.ID, second.ID)
	}
	recent, err := st.RecentObservations(p.ID, 10)
	if err != nil {
		t.Fatalf("RecentObservations: %v", err)
	}
	if len(recent) != 1 {
		t.Fatalf("expected exactly 1 row for the topic, got %d", len(recent))
	}
	if recent[0].Title != "auth v2" || recent[0].Body != "jwt sessions" {
		t.Fatalf("row not updated in place: %+v", recent[0])
	}
}

func TestSearchRanksAndPreviews(t *testing.T) {
	st := newTestStore(t)
	p := mustProject(t, st)
	_, _, _ = st.SaveObservation(&Observation{ProjectID: p.ID, Kind: "bug", Title: "database retry logic", Body: "retry retry retry on lock"})
	_, _, _ = st.SaveObservation(&Observation{ProjectID: p.ID, Kind: "note", Title: "unrelated", Body: "nothing here"})

	res, err := st.SearchObservations(p.ID, "retry", 10)
	if err != nil {
		t.Fatalf("SearchObservations: %v", err)
	}
	if len(res) != 1 {
		t.Fatalf("expected 1 match for 'retry', got %d", len(res))
	}
	if res[0].Title != "database retry logic" {
		t.Fatalf("unexpected top hit: %+v", res[0])
	}
	if res[0].Snippet == "" {
		t.Fatalf("expected a snippet preview")
	}
}

func TestSearchScopedByProject(t *testing.T) {
	st := newTestStore(t)
	a, _ := st.UpsertProject("a", "/tmp/a", "cwd")
	b, _ := st.UpsertProject("b", "/tmp/b", "cwd")
	_, _, _ = st.SaveObservation(&Observation{ProjectID: a.ID, Kind: "bug", Title: "widget", Body: "sprocket"})

	res, err := st.SearchObservations(b.ID, "sprocket", 10)
	if err != nil {
		t.Fatalf("SearchObservations: %v", err)
	}
	if len(res) != 0 {
		t.Fatalf("search must be project-scoped; got %d rows for other project", len(res))
	}
}

func TestSearchMalformedQueryIsSafe(t *testing.T) {
	st := newTestStore(t)
	p := mustProject(t, st)
	_, _, _ = st.SaveObservation(&Observation{ProjectID: p.ID, Kind: "bug", Title: "quotes and parens", Body: "text"})
	// Raw FTS5 operators/punctuation must not cause a syntax error.
	if _, err := st.SearchObservations(p.ID, `"((( AND OR NEAR`, 10); err != nil {
		t.Fatalf("malformed query should be sanitized, got error: %v", err)
	}
}

func TestSessionSummaryUpsertAndLatest(t *testing.T) {
	st := newTestStore(t)
	p := mustProject(t, st)
	if _, err := st.UpsertSessionSummary(&SessionSummary{ProjectID: p.ID, SessionID: "s1", Goal: "old goal"}); err != nil {
		t.Fatalf("upsert 1: %v", err)
	}
	if _, err := st.UpsertSessionSummary(&SessionSummary{ProjectID: p.ID, SessionID: "s1", Goal: "new goal", NextSteps: "ship it"}); err != nil {
		t.Fatalf("upsert 2: %v", err)
	}
	latest, err := st.LatestSessionSummary(p.ID)
	if err != nil || latest == nil {
		t.Fatalf("LatestSessionSummary: %v (got=%v)", err, latest)
	}
	if latest.Goal != "new goal" || latest.NextSteps != "ship it" {
		t.Fatalf("summary not upserted in place: %+v", latest)
	}
}
