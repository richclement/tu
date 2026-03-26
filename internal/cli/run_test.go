package cli

import (
	"bytes"
	"encoding/json"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunHelp(t *testing.T) {
	t.Parallel()

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	exitCode := Run([]string{"--help"}, &stdout, &stderr, "dev")
	if exitCode != 0 {
		t.Fatalf("expected exit code 0, got %d", exitCode)
	}
	if !strings.Contains(stdout.String(), "Usage:") {
		t.Fatalf("expected usage output, got %q", stdout.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("expected empty stderr, got %q", stderr.String())
	}
}

func TestRunVersion(t *testing.T) {
	t.Parallel()

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	exitCode := Run([]string{"--version"}, &stdout, &stderr, "1.2.3")
	if exitCode != 0 {
		t.Fatalf("expected exit code 0, got %d", exitCode)
	}
	if strings.TrimSpace(stdout.String()) != "1.2.3" {
		t.Fatalf("expected version output, got %q", stdout.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("expected empty stderr, got %q", stderr.String())
	}
}

func TestRunInvalidUsage(t *testing.T) {
	t.Parallel()

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	exitCode := Run([]string{"--json", "--plain"}, &stdout, &stderr, "dev")
	if exitCode != 2 {
		t.Fatalf("expected exit code 2, got %d", exitCode)
	}
	if stdout.Len() != 0 {
		t.Fatalf("expected empty stdout, got %q", stdout.String())
	}
	if !strings.Contains(stderr.String(), "Run `tu --help` for usage.") {
		t.Fatalf("expected help hint in stderr, got %q", stderr.String())
	}
}

func TestRunJSONOutput(t *testing.T) {
	t.Parallel()

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	exitCode := runWithCWD(
		[]string{"repo", "--json", "--quiet"},
		&stdout,
		&stderr,
		"dev",
		scanFixtureParentDir(t),
	)
	if exitCode != 0 {
		t.Fatalf("expected exit code 0, got %d", exitCode)
	}

	var decoded map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &decoded); err != nil {
		t.Fatalf("unmarshal json output: %v", err)
	}

	if decoded["schema_version"] != "v1" {
		t.Fatalf("expected schema_version v1, got %v", decoded["schema_version"])
	}
	if stderr.Len() != 0 {
		t.Fatalf("expected empty stderr in quiet machine mode, got %q", stderr.String())
	}
}

func TestRunHumanOutput(t *testing.T) {
	t.Parallel()

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	exitCode := runWithCWD([]string{"repo"}, &stdout, &stderr, "dev", scanFixtureParentDir(t))
	if exitCode != 0 {
		t.Fatalf("expected exit code 0, got %d", exitCode)
	}
	if !strings.Contains(stdout.String(), "TOKENS") {
		t.Fatalf("expected human table header, got %q", stdout.String())
	}
	if !strings.Contains(stderr.String(), "files counted:") {
		t.Fatalf("expected human summary on stderr, got %q", stderr.String())
	}
}

func TestRunRuntimeFailure(t *testing.T) {
	t.Parallel()

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	exitCode := runWithCWD([]string{"missing.txt"}, &stdout, &stderr, "dev", scanFixtureParentDir(t))
	if exitCode != 1 {
		t.Fatalf("expected exit code 1, got %d", exitCode)
	}
	if stdout.Len() != 0 {
		t.Fatalf("expected empty stdout, got %q", stdout.String())
	}
	if !strings.Contains(stderr.String(), "stat target") {
		t.Fatalf("expected stat target error, got %q", stderr.String())
	}
}

func scanFixtureParentDir(t *testing.T) string {
	t.Helper()

	sourceRoot, err := filepath.Abs(filepath.Join("..", "scan", "testdata", "repo"))
	if err != nil {
		t.Fatalf("abs scan fixture dir: %v", err)
	}

	parent := t.TempDir()
	destRoot := filepath.Join(parent, "repo")
	copyFixtureTree(t, sourceRoot, destRoot)
	if err := os.MkdirAll(filepath.Join(destRoot, ".git"), 0o755); err != nil {
		t.Fatalf("mkdir .git: %v", err)
	}

	writeFixtureFile(t, filepath.Join(destRoot, "debug.tmp"), []byte("This file should be ignored when gitignore rules are enabled.\n"))
	writeFixtureFile(t, filepath.Join(destRoot, "ignored", "secret.txt"), []byte("This ignored file should only appear when --no-gitignore is used.\n"))
	writeFixtureFile(t, filepath.Join(destRoot, "nested", "local.log"), []byte("This file is ignored by the nested .gitignore rule.\n"))

	return parent
}

func copyFixtureTree(t *testing.T, sourceRoot string, destRoot string) {
	t.Helper()

	err := filepath.WalkDir(sourceRoot, func(currentPath string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}

		relPath, err := filepath.Rel(sourceRoot, currentPath)
		if err != nil {
			return err
		}

		destPath := filepath.Join(destRoot, relPath)
		if entry.IsDir() {
			return os.MkdirAll(destPath, 0o755)
		}

		contents, err := os.ReadFile(currentPath)
		if err != nil {
			return err
		}

		return os.WriteFile(destPath, contents, 0o600)
	})
	if err != nil {
		t.Fatalf("copy fixture tree: %v", err)
	}
}

func writeFixtureFile(t *testing.T, path string, contents []byte) {
	t.Helper()

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", path, err)
	}
	if err := os.WriteFile(path, contents, 0o600); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}
