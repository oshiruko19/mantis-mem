package project

import (
	"os"
	"path/filepath"
	"testing"
)

func eval(t *testing.T, p string) string {
	t.Helper()
	r, err := filepath.EvalSymlinks(p)
	if err != nil {
		t.Fatalf("EvalSymlinks(%s): %v", p, err)
	}
	return r
}

func TestResolveEnvOverride(t *testing.T) {
	t.Setenv("MANTIS_PROJECT", "myproj")
	got, err := Resolve()
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if got.Name != "myproj" || got.Source != "env" || got.Path != "env:myproj" {
		t.Fatalf("env override wrong: %+v", got)
	}
}

func TestResolveGitRoot(t *testing.T) {
	t.Setenv("MANTIS_PROJECT", "") // ensure the env override is inactive
	repo := t.TempDir()
	if err := os.Mkdir(filepath.Join(repo, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	sub := filepath.Join(repo, "a", "b")
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	t.Chdir(sub)

	got, err := Resolve()
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if got.Source != "git" {
		t.Fatalf("expected source git, got %q", got.Source)
	}
	if want := eval(t, repo); got.Path != want {
		t.Fatalf("git root path: got %q want %q", got.Path, want)
	}
	if got.Name != filepath.Base(repo) {
		t.Fatalf("git project name: got %q want %q", got.Name, filepath.Base(repo))
	}
}

func TestResolveCwdWhenNoGit(t *testing.T) {
	t.Setenv("MANTIS_PROJECT", "")
	dir := t.TempDir()
	t.Chdir(dir)

	got, err := Resolve()
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if got.Source != "cwd" {
		t.Fatalf("expected source cwd, got %q (path %q)", got.Source, got.Path)
	}
	if want := eval(t, dir); got.Path != want {
		t.Fatalf("cwd path: got %q want %q", got.Path, want)
	}
}

func TestEnvOverridesGit(t *testing.T) {
	// Even inside a git repo, the env override wins.
	repo := t.TempDir()
	if err := os.Mkdir(filepath.Join(repo, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Chdir(repo)
	t.Setenv("MANTIS_PROJECT", "override")

	got, err := Resolve()
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if got.Source != "env" || got.Name != "override" {
		t.Fatalf("env should take precedence over git: %+v", got)
	}
}

func TestFindGitRootDir(t *testing.T) {
	repo := t.TempDir()
	if err := os.Mkdir(filepath.Join(repo, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	sub := filepath.Join(repo, "x", "y")
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	if got := findGitRoot(repo); got != repo {
		t.Fatalf("findGitRoot at root: got %q want %q", got, repo)
	}
	if got := findGitRoot(sub); got != repo {
		t.Fatalf("findGitRoot from subdir: got %q want %q", got, repo)
	}
}

func TestFindGitRootFile(t *testing.T) {
	// A .git file (worktree/submodule style) must also be detected.
	repo := t.TempDir()
	if err := os.WriteFile(filepath.Join(repo, ".git"), []byte("gitdir: ../real"), 0o644); err != nil {
		t.Fatal(err)
	}
	if got := findGitRoot(repo); got != repo {
		t.Fatalf("findGitRoot should detect a .git file: got %q", got)
	}
}

func TestFindGitRootNone(t *testing.T) {
	dir := t.TempDir()
	if got := findGitRoot(dir); got != "" {
		t.Fatalf("expected no git root, got %q", got)
	}
}

func TestCanonicalIsAbsolute(t *testing.T) {
	if got := canonical("."); !filepath.IsAbs(got) {
		t.Fatalf("canonical(.) should be absolute, got %q", got)
	}
}
