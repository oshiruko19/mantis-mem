// Package project resolves the current memory scope from the working directory.
package project

import (
	"os"
	"path/filepath"
	"strings"
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
