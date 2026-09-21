// Package project resolves the current memory scope from the working directory.
package project

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// Resolved describes the current project and which rule identified it.
type Resolved struct {
	Name   string // human-facing project name
	Path   string // canonical identity key ("env:<name>" for the env override)
	Source string // git | cwd | env
}

// Resolve determines the current project, in precedence order:
//  1. $MANTIS_PROJECT  -> source "env"  (directory-independent identity)
//  2. nearest ancestor containing .git -> source "git"
//  3. current working directory         -> source "cwd"
func Resolve() (*Resolved, error) {
	wd, err := os.Getwd()
	if err != nil {
		return nil, err
	}
	abs := canonical(wd)

	if v := strings.TrimSpace(os.Getenv("MANTIS_PROJECT")); v != "" {
		return &Resolved{Name: v, Path: "env:" + v, Source: "env"}, nil
	}
	if root := findGitRoot(abs); root != "" {
		return &Resolved{Name: filepath.Base(root), Path: root, Source: "git"}, nil
	}
	return &Resolved{Name: filepath.Base(abs), Path: abs, Source: "cwd"}, nil
}

// canonical returns an absolute, symlink-resolved path, falling back gracefully.
func canonical(p string) string {
	if abs, err := filepath.Abs(p); err == nil {
		p = abs
	}
	if resolved, err := filepath.EvalSymlinks(p); err == nil {
		return resolved
	}
	return p
}

// findGitRoot walks up from dir looking for a .git entry (directory or file, to
// support worktrees and submodules). Returns "" if none is found.
func findGitRoot(dir string) string {
	for {
		if _, err := os.Stat(filepath.Join(dir, ".git")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return ""
		}
		dir = parent
	}
}

// CurrentCommit discovers the current Git commit SHA for dir (or its ancestors).
// It runs "git rev-parse HEAD" in dir. If git is unavailable, dir is not in a git
// repo, or there are no commits yet, it returns "" without failing.
func CurrentCommit(dir string) string {
	if strings.HasPrefix(dir, "env:") || dir == "" {
		if wd, err := os.Getwd(); err == nil {
			dir = wd
		}
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "git", "rev-parse", "HEAD")
	cmd.Dir = dir
	out, err := cmd.Output()
	if err != nil {
		return ""
	}
	sha := strings.TrimSpace(string(out))
	if len(sha) >= 7 && len(sha) <= 64 && isHex(sha) {
		return sha
	}
	return ""
}

func isHex(s string) bool {
	for _, r := range s {
		if (r < '0' || r > '9') && (r < 'a' || r > 'f') && (r < 'A' || r > 'F') {
			return false
		}
	}
	return true
}
