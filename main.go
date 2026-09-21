// Command mantis-mem is a single-binary curated project memory store: SQLite +
// FTS5 behind both an MCP stdio server (for agents) and a human CLI.
package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"strings"

	"github.com/oshiruko19/mantis-mem/internal/cli"
	"github.com/oshiruko19/mantis-mem/internal/mcpserver"
)

// version is overridden at build time via -ldflags "-X main.version=...".
var version = "dev"

func main() {
	args := os.Args[1:]
	// Allow a leading global --db that applies to any subcommand (incl. serve).
	// It is exported as MANTIS_DB, which store.Open uses as the default.
	args = stripGlobalDB(args)

	if len(args) > 0 && args[0] == "serve" {
		if err := runServe(args[1:]); err != nil {
			fmt.Fprintf(os.Stderr, "mantis-mem: %v\n", err)
			os.Exit(1)
		}
		return
	}

	if err := cli.Run(args, version, os.Stdout); err != nil {
		fmt.Fprintf(os.Stderr, "mantis-mem: %v\n", err)
		os.Exit(1)
	}
}

// stripGlobalDB consumes a leading "--db PATH" / "-db PATH" / "--db=PATH" option
// and exports it as MANTIS_DB, returning the remaining args. Anything else is
// left untouched for the subcommand's own flag parsing.
func stripGlobalDB(args []string) []string {
	if len(args) == 0 {
		return args
	}
	switch {
	case args[0] == "--db" || args[0] == "-db":
		if len(args) >= 2 {
			_ = os.Setenv("MANTIS_DB", args[1])
			return args[2:]
		}
		return args[1:]
	case strings.HasPrefix(args[0], "--db="):
		_ = os.Setenv("MANTIS_DB", strings.TrimPrefix(args[0], "--db="))
		return args[1:]
	case strings.HasPrefix(args[0], "-db="):
		_ = os.Setenv("MANTIS_DB", strings.TrimPrefix(args[0], "-db="))
		return args[1:]
	}
	return args
}

// runServe launches the MCP stdio server. It accepts an optional --db path.
func runServe(args []string) error {
	dbPath := os.Getenv("MANTIS_DB")
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--db", "-db":
			if i+1 >= len(args) {
				return fmt.Errorf("--db requires a value")
			}
			dbPath = args[i+1]
			i++
		default:
			return fmt.Errorf("unknown serve flag %q", args[i])
		}
	}

	// Terminate cleanly on interrupt; the SDK's Run returns when ctx is cancelled.
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	return mcpserver.Serve(ctx, dbPath, version)
}
