package cli

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/richclement/tu/internal/testfixture"
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

	parent := t.TempDir()
	if _, err := testfixture.MaterializeCanonicalRepo(parent); err != nil {
		t.Fatalf("materialize fixture repo: %v", err)
	}
	return parent
}
