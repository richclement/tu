package testfixture

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRepoRoot(t *testing.T) {
	t.Parallel()

	repoRoot, err := RepoRoot()
	if err != nil {
		t.Fatalf("RepoRoot returned error: %v", err)
	}

	if _, err := os.Stat(filepath.Join(repoRoot, "go.mod")); err != nil {
		t.Fatalf("expected go.mod at repo root: %v", err)
	}
}

func TestMaterializeCanonicalRepo(t *testing.T) {
	t.Parallel()

	parent := t.TempDir()
	repoRoot, err := MaterializeCanonicalRepo(parent)
	if err != nil {
		t.Fatalf("MaterializeCanonicalRepo returned error: %v", err)
	}

	for _, relativePath := range []string{
		".git",
		".gitignore",
		"README.md",
		"debug.tmp",
		filepath.Join("ignored", "secret.txt"),
		filepath.Join("nested", ".gitignore"),
		filepath.Join("nested", "local.log"),
	} {
		if _, err := os.Stat(filepath.Join(repoRoot, relativePath)); err != nil {
			t.Fatalf("expected fixture path %q to exist: %v", relativePath, err)
		}
	}
}
