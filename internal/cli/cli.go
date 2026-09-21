// Package cli implements the human-facing subcommands that mirror the MCP tools.
package cli

import (
	"flag"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/oshiruko19/mantis-mem/internal/mcpserver"
	"github.com/oshiruko19/mantis-mem/internal/project"
	"github.com/oshiruko19/mantis-mem/internal/store"
)

// Run dispatches a CLI subcommand. args is os.Args[1:]; args[0] is the subcommand.
func Run(args []string, version string, out io.Writer) error {
	if len(args) == 0 {
		usage(out)
		return fmt.Errorf("no command given")
	}
	cmd, rest := args[0], args[1:]
	switch cmd {
	case "search":
		return cmdSearch(rest, out)
	case "save":
		return cmdSave(rest, out)
	case "context":
		return cmdContext(rest, out)
	case "timeline":
		return cmdTimeline(rest, out)
	case "get":
		return cmdGet(rest, out)
	case "project":
		return cmdProject(rest, out)
	case "suggest-topic":
		return cmdSuggestTopic(rest, out)
	case "init":
		return cmdInit(rest, out)
	case "version", "--version", "-v":
		fmt.Fprintf(out, "mantis-mem %s\n", version)
		return nil
	case "help", "--help", "-h":
		usage(out)
		return nil
	default:
		usage(out)
		return fmt.Errorf("unknown command %q", cmd)
	}
}

func usage(out io.Writer) {
	fmt.Fprint(out, `mantis-mem — curated project memory (SQLite + FTS5)

Usage:
  mantis-mem serve                 run the MCP stdio server (for agents)
  mantis-mem init                  create/open the database
  mantis-mem project               show the resolved current project
  mantis-mem save   --kind K --title T --body B [--topic K] [--tags "a b"] [--files "p1,p2"]
  mantis-mem search QUERY          full-text search the current project
  mantis-mem context [--limit N]   recent history for the current project
  mantis-mem timeline [--session S] [--since RFC3339] [--limit N]
  mantis-mem get ID                show a full observation
  mantis-mem suggest-topic --kind K --title T
  mantis-mem version

Global:
  --db PATH    database file (default $MANTIS_DB or ~/.mantis/mantis_mem.db)
`)
}

// openStore opens the DB honoring a --db flag registered on the FlagSet.
func openStore(dbPath string) (*store.Store, error) {
	return store.Open(dbPath)
}

// resolveProject resolves + records the current project against an open store.
func resolveProject(st *store.Store) (*store.Project, error) {
	r, err := project.Resolve()
	if err != nil {
		return nil, err
	}
	return st.UpsertProject(r.Name, r.Path, r.Source)
}

func cmdInit(args []string, out io.Writer) error {
	fs := flag.NewFlagSet("init", flag.ContinueOnError)
	db := fs.String("db", "", "database file path")
	if err := fs.Parse(args); err != nil {
		return err
	}
	st, err := openStore(*db)
	if err != nil {
		return err
	}
	defer st.Close()
	fmt.Fprintf(out, "initialized %s\n", st.Path())
	return nil
}

func cmdProject(args []string, out io.Writer) error {
	fs := flag.NewFlagSet("project", flag.ContinueOnError)
	db := fs.String("db", "", "database file path")
	if err := fs.Parse(args); err != nil {
		return err
	}
	st, err := openStore(*db)
	if err != nil {
		return err
	}
	defer st.Close()
	p, err := resolveProject(st)
	if err != nil {
		return err
	}
	fmt.Fprintf(out, "project: %s\nsource:  %s\npath:    %s\ndb:      %s\n", p.Name, p.Source, p.Path, st.Path())
	return nil
}

func cmdSave(args []string, out io.Writer) error {
	fs := flag.NewFlagSet("save", flag.ContinueOnError)
	db := fs.String("db", "", "database file path")
	kind := fs.String("kind", "", "decision|bug|discovery|config|pattern|constraint|feature|note")
	title := fs.String("title", "", "concise title")
	body := fs.String("body", "", "structured note (What/Why/Where/Learned)")
	topic := fs.String("topic", "", "stable topic_key to update in place")
	tags := fs.String("tags", "", "space-separated tags")
	files := fs.String("files", "", "comma-separated file paths")
	session := fs.String("session", "", "session id")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if !store.ValidKind(*kind) {
		return fmt.Errorf("invalid --kind %q; must be one of %s", *kind, strings.Join(store.Kinds, ", "))
	}
	if strings.TrimSpace(*title) == "" || strings.TrimSpace(*body) == "" {
		return fmt.Errorf("--title and --body are required")
	}
	st, err := openStore(*db)
	if err != nil {
		return err
	}
	defer st.Close()
	p, err := resolveProject(st)
	if err != nil {
		return err
	}
	saved, updated, err := st.SaveObservation(&store.Observation{
		ProjectID: p.ID,
		SessionID: *session,
		TopicKey:  strings.TrimSpace(*topic),
		Kind:      *kind,
		Title:     *title,
		Body:      *body,
		Files:     splitList(*files),
		Tags:      *tags,
	})
	if err != nil {
		return err
	}
	verb := "saved"
	if updated {
		verb = "updated"
	}
	fmt.Fprintf(out, "%s observation #%d", verb, saved.ID)
	if saved.TopicKey != "" {
		fmt.Fprintf(out, " [%s]", saved.TopicKey)
	}
	fmt.Fprintln(out)
	return nil
}

func cmdSearch(args []string, out io.Writer) error {
	fs := flag.NewFlagSet("search", flag.ContinueOnError)
	db := fs.String("db", "", "database file path")
	limit := fs.Int("limit", 20, "maximum results")
	if err := fs.Parse(args); err != nil {
		return err
	}
	query := strings.TrimSpace(strings.Join(fs.Args(), " "))
	if query == "" {
		return fmt.Errorf("usage: mantis-mem search QUERY")
	}
	st, err := openStore(*db)
	if err != nil {
		return err
	}
	defer st.Close()
	p, err := resolveProject(st)
	if err != nil {
		return err
	}
	results, err := st.SearchObservations(p.ID, query, *limit)
	if err != nil {
		return err
	}
	if len(results) == 0 {
		fmt.Fprintln(out, "no matches")
		return nil
	}
	for _, r := range results {
		topic := ""
		if r.TopicKey != "" {
			topic = " [" + r.TopicKey + "]"
		}
		fmt.Fprintf(out, "#%d  %-10s %s%s\n    %s\n", r.ID, r.Kind, r.Title, topic, r.Snippet)
	}
	return nil
}

func cmdContext(args []string, out io.Writer) error {
	fs := flag.NewFlagSet("context", flag.ContinueOnError)
	db := fs.String("db", "", "database file path")
	limit := fs.Int("limit", 10, "recent observations to show")
	if err := fs.Parse(args); err != nil {
		return err
	}
	st, err := openStore(*db)
	if err != nil {
		return err
	}
	defer st.Close()
	p, err := resolveProject(st)
	if err != nil {
		return err
	}
	fmt.Fprintf(out, "project: %s (%s)\n", p.Name, p.Source)
	summary, err := st.LatestSessionSummary(p.ID)
	if err != nil {
		return err
	}
	if summary != nil {
		fmt.Fprintf(out, "\nlatest session summary [%s]:\n", summary.SessionID)
		printField(out, "  goal", summary.Goal)
		printField(out, "  next", summary.NextSteps)
	}
	recent, err := st.RecentObservations(p.ID, *limit)
	if err != nil {
		return err
	}
	fmt.Fprintf(out, "\nrecent observations (%d):\n", len(recent))
	for _, o := range recent {
		fmt.Fprintf(out, "  #%d  %-10s %s\n", o.ID, o.Kind, o.Title)
	}
	return nil
}

func cmdTimeline(args []string, out io.Writer) error {
	fs := flag.NewFlagSet("timeline", flag.ContinueOnError)
	db := fs.String("db", "", "database file path")
	session := fs.String("session", "", "restrict to a session id")
	since := fs.String("since", "", "lower time bound (RFC3339)")
	limit := fs.Int("limit", 50, "maximum observations")
	if err := fs.Parse(args); err != nil {
		return err
	}
	st, err := openStore(*db)
	if err != nil {
		return err
	}
	defer st.Close()
	p, err := resolveProject(st)
	if err != nil {
		return err
	}
	obs, err := st.Timeline(p.ID, *session, *since, *limit)
	if err != nil {
		return err
	}
	for _, o := range obs {
		fmt.Fprintf(out, "%s  #%d  %-10s %s\n", o.CreatedAt, o.ID, o.Kind, o.Title)
	}
	return nil
}

func cmdGet(args []string, out io.Writer) error {
	fs := flag.NewFlagSet("get", flag.ContinueOnError)
	db := fs.String("db", "", "database file path")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() != 1 {
		return fmt.Errorf("usage: mantis-mem get ID")
	}
	id, err := strconv.ParseInt(fs.Arg(0), 10, 64)
	if err != nil {
		return fmt.Errorf("invalid id %q", fs.Arg(0))
	}
	st, err := openStore(*db)
	if err != nil {
		return err
	}
	defer st.Close()
	o, err := st.GetObservation(id)
	if err != nil {
		return err
	}
	if o == nil {
		return fmt.Errorf("observation #%d not found", id)
	}
	fmt.Fprintf(out, "#%d  %s\n", o.ID, o.Title)
	fmt.Fprintf(out, "kind: %s", o.Kind)
	if o.TopicKey != "" {
		fmt.Fprintf(out, "   topic: %s", o.TopicKey)
	}
	fmt.Fprintf(out, "\ncreated: %s   updated: %s\n", o.CreatedAt, o.UpdatedAt)
	if len(o.Files) > 0 {
		fmt.Fprintf(out, "files: %s\n", strings.Join(o.Files, ", "))
	}
	if o.Tags != "" {
		fmt.Fprintf(out, "tags: %s\n", o.Tags)
	}
	fmt.Fprintf(out, "\n%s\n", o.Body)
	return nil
}

func cmdSuggestTopic(args []string, out io.Writer) error {
	fs := flag.NewFlagSet("suggest-topic", flag.ContinueOnError)
	kind := fs.String("kind", "", "observation kind")
	title := fs.String("title", "", "title to slugify")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if strings.TrimSpace(*title) == "" {
		return fmt.Errorf("--title is required")
	}
	fmt.Fprintln(out, mcpserver.SuggestTopicKey(*kind, *title))
	return nil
}

func printField(out io.Writer, label, val string) {
	if strings.TrimSpace(val) != "" {
		fmt.Fprintf(out, "%s: %s\n", label, val)
	}
}

func splitList(s string) []string {
	if strings.TrimSpace(s) == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if t := strings.TrimSpace(p); t != "" {
			out = append(out, t)
		}
	}
	return out
}
