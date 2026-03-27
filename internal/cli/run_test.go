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
	summary, ok := decoded["summary"].(map[string]any)
	if !ok {
		t.Fatalf("expected summary object, got %T", decoded["summary"])
	}
	if _, ok := summary["total_bytes"]; ok {
		t.Fatalf("expected summary.total_bytes to be absent, got %v", summary)
	}
	results, ok := decoded["results"].([]any)
	if !ok {
		t.Fatalf("expected results array, got %T", decoded["results"])
	}
	for index, raw := range results {
		result, ok := raw.(map[string]any)
		if !ok {
			t.Fatalf("expected result object at index %d, got %T", index, raw)
		}
		if _, ok := result["bytes"]; ok {
			t.Fatalf("expected result[%d] bytes to be absent, got %v", index, result)
		}
	}
	if stderr.Len() != 0 {
		t.Fatalf("expected empty stderr in quiet machine mode, got %q", stderr.String())
	}
}

func TestRunPlainOutput(t *testing.T) {
	t.Parallel()

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	exitCode := runWithCWD(
		[]string{"repo", "--plain", "--quiet"},
		&stdout,
		&stderr,
		"dev",
		scanFixtureParentDir(t),
	)
	if exitCode != 0 {
		t.Fatalf("expected exit code 0, got %d", exitCode)
	}
	if stderr.Len() != 0 {
		t.Fatalf("expected empty stderr in quiet plain mode, got %q", stderr.String())
	}

	lines := strings.Split(strings.TrimSpace(stdout.String()), "\n")
	if len(lines) == 0 {
		t.Fatal("expected plain output lines")
	}
	for index, line := range lines {
		fields := strings.Split(line, "\t")
		if len(fields) != 4 {
			t.Fatalf("expected 4 plain output fields on line %d, got %d in %q", index, len(fields), line)
		}
		if fields[2] == "" {
			t.Fatalf("expected status field on line %d, got %q", index, line)
		}
		if fields[3] == "" {
			t.Fatalf("expected path field on line %d, got %q", index, line)
		}
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
	if strings.Contains(stdout.String(), "BYTES") {
		t.Fatalf("expected human output to omit BYTES column, got %q", stdout.String())
	}
	if !strings.Contains(stdout.String(), "METHOD") {
		t.Fatalf("expected human output to include METHOD column, got %q", stdout.String())
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
