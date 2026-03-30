package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/richclement/tu/internal/report"
	"github.com/richclement/tu/internal/scan"
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

	exitCode := Run([]string{"--format", "bogus"}, &stdout, &stderr, "dev")
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

func TestRunLegacyFlagUsageError(t *testing.T) {
	t.Parallel()

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	exitCode := Run([]string{"--json"}, &stdout, &stderr, "dev")
	if exitCode != 2 {
		t.Fatalf("expected exit code 2, got %d", exitCode)
	}
	if !strings.Contains(stderr.String(), "Run `tu --help` for usage.") {
		t.Fatalf("expected help hint in stderr, got %q", stderr.String())
	}
}

func TestRunMalformedExcludeUsageError(t *testing.T) {
	t.Parallel()

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	exitCode := Run([]string{"--exclude", "["}, &stdout, &stderr, "dev")
	if exitCode != 2 {
		t.Fatalf("expected exit code 2, got %d", exitCode)
	}
	if stdout.Len() != 0 {
		t.Fatalf("expected empty stdout, got %q", stdout.String())
	}
	if !strings.Contains(stderr.String(), "malformed") {
		t.Fatalf("expected malformed pattern error in stderr, got %q", stderr.String())
	}
}

func TestRunJSONOutput(t *testing.T) {
	t.Parallel()

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	exitCode := runWithCWD(
		[]string{"repo", "--format", "json", "--quiet"},
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
	if _, ok := decoded["threshold"]; ok {
		t.Fatalf("expected threshold to be omitted when unset, got %v", decoded["threshold"])
	}
	if _, ok := decoded["exclude"]; ok {
		t.Fatalf("expected exclude to be omitted when unset, got %v", decoded["exclude"])
	}
	summary, ok := decoded["summary"].(map[string]any)
	if !ok {
		t.Fatalf("expected summary object, got %T", decoded["summary"])
	}
	if _, ok := summary["total_bytes"]; ok {
		t.Fatalf("expected summary.total_bytes to be absent, got %v", summary)
	}
	if _, ok := summary["heuristic_results"]; ok {
		t.Fatalf("expected summary.heuristic_results to be absent, got %v", summary)
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
		if result["kind"] == nil {
			t.Fatalf("expected result[%d] kind to be present, got %v", index, result)
		}
		if _, ok := result["bytes"]; ok {
			t.Fatalf("expected result[%d] bytes to be absent, got %v", index, result)
		}
	}
	if stderr.Len() != 0 {
		t.Fatalf("expected empty stderr in quiet machine mode, got %q", stderr.String())
	}
}

func TestRunJSONOutputIncludesThreshold(t *testing.T) {
	t.Parallel()

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	exitCode := runWithCWD(
		[]string{"repo", "--format", "json", "--threshold", "10", "--quiet"},
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

	if decoded["threshold"] != float64(10) {
		t.Fatalf("expected threshold 10 in json output, got %v", decoded["threshold"])
	}
}

func TestRunJSONOutputIncludesExclude(t *testing.T) {
	t.Parallel()

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	exitCode := runWithCWD(
		[]string{"repo", "--format", "json", "--exclude", "README.md", "--exclude", "*.tmp", "--quiet"},
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

	rawExclude, ok := decoded["exclude"].([]any)
	if !ok {
		t.Fatalf("expected exclude array, got %T", decoded["exclude"])
	}
	got := make([]string, 0, len(rawExclude))
	for _, value := range rawExclude {
		got = append(got, value.(string))
	}
	if !slices.Equal(got, []string{"README.md", "*.tmp"}) {
		t.Fatalf("expected exclude metadata to preserve order, got %v", got)
	}
}

func TestRunPlainOutput(t *testing.T) {
	t.Parallel()

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	exitCode := runWithCWD(
		[]string{"repo", "--format", "plain", "--quiet"},
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

func TestRunCSVOutput(t *testing.T) {
	t.Parallel()

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	exitCode := runWithCWD(
		[]string{"repo", "--format", "csv", "--quiet"},
		&stdout,
		&stderr,
		"dev",
		scanFixtureParentDir(t),
	)
	if exitCode != 0 {
		t.Fatalf("expected exit code 0, got %d", exitCode)
	}
	if stderr.Len() != 0 {
		t.Fatalf("expected empty stderr in quiet csv mode, got %q", stderr.String())
	}

	lines := strings.Split(strings.TrimSpace(stdout.String()), "\n")
	if len(lines) < 2 {
		t.Fatalf("expected csv header plus rows, got %q", stdout.String())
	}
	if lines[0] != "kind,path,tokens,method,provider,status,reason" {
		t.Fatalf("unexpected csv header %q", lines[0])
	}
}

func TestRunSummarizeHumanOutput(t *testing.T) {
	t.Parallel()

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	exitCode := runWithCWD([]string{"repo", "--summarize"}, &stdout, &stderr, "dev", scanFixtureParentDir(t))
	if exitCode != 0 {
		t.Fatalf("expected exit code 0, got %d", exitCode)
	}

	lines := strings.Split(strings.TrimSpace(stdout.String()), "\n")
	if len(lines) != 2 {
		t.Fatalf("expected human header plus one summary row, got %q", stdout.String())
	}
	if !strings.Contains(lines[1], "repo") {
		t.Fatalf("expected summary row path in human output, got %q", lines[1])
	}
	if !strings.Contains(stderr.String(), "files counted:") {
		t.Fatalf("expected stderr summary, got %q", stderr.String())
	}
}

func TestWriteSummaryUsesPrecomputedHeuristicCount(t *testing.T) {
	t.Parallel()

	var stderr bytes.Buffer
	totalTokens := int64(12)

	writeSummary(&stderr, report.ScanReport{
		Summary: report.Summary{
			FilesCounted:     4,
			FilesSkipped:     1,
			HeuristicResults: 3,
		},
		Results: []report.Result{
			{
				Kind:   report.ResultKindSummary,
				Path:   "repo",
				Tokens: &totalTokens,
				Status: report.StatusCounted,
			},
		},
	}, Options{
		Output: OutputHuman,
		Quiet:  false,
	})

	if !strings.Contains(stderr.String(), "heuristic results: 3") {
		t.Fatalf("expected precomputed heuristic count in stderr, got %q", stderr.String())
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

func TestRunHumanOutputShowsNoMatchesForThreshold(t *testing.T) {
	t.Parallel()

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	parent := scanFixtureParentDir(t)

	unfiltered, err := scan.BuildReport(scan.Config{
		CWD:              parent,
		Target:           "repo",
		RespectGitIgnore: true,
		Sort:             "tokens-desc",
	})
	if err != nil {
		t.Fatalf("BuildReport returned error: %v", err)
	}

	exitCode := runWithCWD(
		[]string{"repo", "--threshold", strconv.FormatInt(unfiltered.Summary.TotalTokens, 10)},
		&stdout,
		&stderr,
		"dev",
		parent,
	)
	if exitCode != 0 {
		t.Fatalf("expected exit code 0, got %d", exitCode)
	}
	if stdout.String() != "No entries matched threshold.\n" {
		t.Fatalf("expected no-match threshold message, got %q", stdout.String())
	}
	expectedSummary := "files counted: " + strconv.FormatInt(unfiltered.Summary.FilesCounted, 10)
	if !strings.Contains(stderr.String(), expectedSummary) {
		t.Fatalf("expected stderr summary to use full scan totals, got %q", stderr.String())
	}
}

func TestRunHumanOutputExcludesMatchingEntriesAndUpdatesSummary(t *testing.T) {
	t.Parallel()

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	parent := scanFixtureParentDir(t)

	baseline, err := scan.BuildReport(scan.Config{
		CWD:              parent,
		Target:           "repo",
		Exclude:          []string{"README.md"},
		RespectGitIgnore: true,
		Sort:             "tokens-desc",
	})
	if err != nil {
		t.Fatalf("BuildReport returned error: %v", err)
	}

	exitCode := runWithCWD(
		[]string{"repo", "--exclude", "README.md"},
		&stdout,
		&stderr,
		"dev",
		parent,
	)
	if exitCode != 0 {
		t.Fatalf("expected exit code 0, got %d", exitCode)
	}
	if strings.Contains(stdout.String(), "README.md") {
		t.Fatalf("expected excluded entry to be absent from human output, got %q", stdout.String())
	}
	expectedSummary := "files counted: " + strconv.FormatInt(baseline.Summary.FilesCounted, 10) +
		", files skipped: " + strconv.FormatInt(baseline.Summary.FilesSkipped, 10) +
		", heuristic results: " + strconv.FormatInt(baseline.Summary.HeuristicResults, 10)
	if !strings.Contains(stderr.String(), expectedSummary) {
		t.Fatalf("expected summary to reflect only included files, got %q", stderr.String())
	}
}

func TestRunWritesJSONToFile(t *testing.T) {
	t.Parallel()

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	outputPath := filepath.Join(t.TempDir(), "report.json")

	exitCode := runWithCWD(
		[]string{"repo", "--format", "json", "--file", outputPath, "--quiet"},
		&stdout,
		&stderr,
		"dev",
		scanFixtureParentDir(t),
	)
	if exitCode != 0 {
		t.Fatalf("expected exit code 0, got %d", exitCode)
	}
	if stdout.Len() != 0 {
		t.Fatalf("expected empty stdout, got %q", stdout.String())
	}

	contents, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("read output file: %v", err)
	}

	var decoded map[string]any
	if err := json.Unmarshal(contents, &decoded); err != nil {
		t.Fatalf("unmarshal json output file: %v", err)
	}
}

func TestRunWritesHumanToFile(t *testing.T) {
	t.Parallel()

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	outputPath := filepath.Join(t.TempDir(), "report.txt")

	exitCode := runWithCWD(
		[]string{"repo", "--format", "human", "--file", outputPath},
		&stdout,
		&stderr,
		"dev",
		scanFixtureParentDir(t),
	)
	if exitCode != 0 {
		t.Fatalf("expected exit code 0, got %d", exitCode)
	}
	if stdout.Len() != 0 {
		t.Fatalf("expected empty stdout, got %q", stdout.String())
	}

	contents, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("read output file: %v", err)
	}
	if !strings.Contains(string(contents), "TOKENS") {
		t.Fatalf("expected human output in file, got %q", string(contents))
	}
	if !strings.Contains(stderr.String(), "files counted:") {
		t.Fatalf("expected human summary on stderr, got %q", stderr.String())
	}
}

func TestRunWritesCSVToFile(t *testing.T) {
	t.Parallel()

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	outputPath := filepath.Join(t.TempDir(), "report.csv")

	exitCode := runWithCWD(
		[]string{"repo", "--format", "csv", "--file", outputPath, "--quiet"},
		&stdout,
		&stderr,
		"dev",
		scanFixtureParentDir(t),
	)
	if exitCode != 0 {
		t.Fatalf("expected exit code 0, got %d", exitCode)
	}
	if stdout.Len() != 0 {
		t.Fatalf("expected empty stdout, got %q", stdout.String())
	}

	contents, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("read output file: %v", err)
	}
	if !strings.HasPrefix(string(contents), "kind,path,tokens,method,provider,status,reason\n") {
		t.Fatalf("expected csv header in file, got %q", string(contents))
	}
}

func TestRunCSVThresholdNoMatchesStillWritesHeader(t *testing.T) {
	t.Parallel()

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	parent := scanFixtureParentDir(t)

	unfiltered, err := scan.BuildReport(scan.Config{
		CWD:              parent,
		Target:           "repo",
		RespectGitIgnore: true,
		Sort:             "tokens-desc",
	})
	if err != nil {
		t.Fatalf("BuildReport returned error: %v", err)
	}

	exitCode := runWithCWD(
		[]string{"repo", "--format", "csv", "--threshold", strconv.FormatInt(unfiltered.Summary.TotalTokens, 10), "--quiet"},
		&stdout,
		&stderr,
		"dev",
		parent,
	)
	if exitCode != 0 {
		t.Fatalf("expected exit code 0, got %d", exitCode)
	}
	if stdout.String() != "kind,path,tokens,method,provider,status,reason\n" {
		t.Fatalf("expected header-only csv output, got %q", stdout.String())
	}
}

func TestRunCSVOutputExcludesMatchingEntries(t *testing.T) {
	t.Parallel()

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	exitCode := runWithCWD(
		[]string{"repo", "--format", "csv", "--exclude", "README.md", "--quiet"},
		&stdout,
		&stderr,
		"dev",
		scanFixtureParentDir(t),
	)
	if exitCode != 0 {
		t.Fatalf("expected exit code 0, got %d", exitCode)
	}
	if strings.Contains(stdout.String(), "README.md") {
		t.Fatalf("expected excluded entry to be absent from csv output, got %q", stdout.String())
	}
}

func TestRunThresholdAppliesAfterExclude(t *testing.T) {
	t.Parallel()

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	parent := scanFixtureParentDir(t)

	baseline, err := scan.BuildReport(scan.Config{
		CWD:              parent,
		Target:           "repo",
		Exclude:          []string{"README.md"},
		RespectGitIgnore: true,
		Sort:             "tokens-desc",
	})
	if err != nil {
		t.Fatalf("BuildReport returned error: %v", err)
	}

	exitCode := runWithCWD(
		[]string{"repo", "--exclude", "README.md", "--threshold", strconv.FormatInt(baseline.Summary.TotalTokens, 10)},
		&stdout,
		&stderr,
		"dev",
		parent,
	)
	if exitCode != 0 {
		t.Fatalf("expected exit code 0, got %d", exitCode)
	}
	if stdout.String() != "No entries matched threshold.\n" {
		t.Fatalf("expected no-match threshold message, got %q", stdout.String())
	}
	expectedSummary := "files counted: " + strconv.FormatInt(baseline.Summary.FilesCounted, 10) +
		", files skipped: " + strconv.FormatInt(baseline.Summary.FilesSkipped, 10) +
		", heuristic results: " + strconv.FormatInt(baseline.Summary.HeuristicResults, 10)
	if !strings.Contains(stderr.String(), expectedSummary) {
		t.Fatalf("expected summary to reflect post-exclude totals, got %q", stderr.String())
	}
}

func TestRunExcludeTargetWithThresholdShowsNoFilesFound(t *testing.T) {
	t.Parallel()

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	exitCode := runWithCWD(
		[]string{"repo", "--exclude", "repo", "--threshold", "0"},
		&stdout,
		&stderr,
		"dev",
		scanFixtureParentDir(t),
	)
	if exitCode != 0 {
		t.Fatalf("expected exit code 0, got %d", exitCode)
	}
	if stdout.String() != "No files found.\n" {
		t.Fatalf("expected no-files message for excluded target, got %q", stdout.String())
	}
}

func TestRunFileDashWritesToStdout(t *testing.T) {
	t.Parallel()

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	exitCode := runWithCWD(
		[]string{"repo", "--format", "json", "--file", "-", "--quiet"},
		&stdout,
		&stderr,
		"dev",
		scanFixtureParentDir(t),
	)
	if exitCode != 0 {
		t.Fatalf("expected exit code 0, got %d", exitCode)
	}
	if stdout.Len() == 0 {
		t.Fatal("expected stdout output when --file=- is used")
	}
}

func TestRunOutputFilePathFailure(t *testing.T) {
	t.Parallel()

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	outputPath := filepath.Join(t.TempDir(), "missing", "report.json")

	exitCode := runWithCWD(
		[]string{"repo", "--format", "json", "--file", outputPath},
		&stdout,
		&stderr,
		"dev",
		scanFixtureParentDir(t),
	)
	if exitCode != 1 {
		t.Fatalf("expected exit code 1, got %d", exitCode)
	}
	if !strings.Contains(stderr.String(), "open output file") {
		t.Fatalf("expected output file error, got %q", stderr.String())
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
