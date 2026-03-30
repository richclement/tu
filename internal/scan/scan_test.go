package scan

import (
	"os"
	"path/filepath"
	"slices"
	"sync/atomic"
	"testing"

	"github.com/richclement/tu/internal/count"
	reportpkg "github.com/richclement/tu/internal/report"
	"github.com/richclement/tu/internal/testfixture"
)

func TestBuildReportDirectoryRespectsGitIgnore(t *testing.T) {
	t.Parallel()

	report, err := BuildReport(Config{
		CWD:              fixtureParentDir(t),
		Target:           "repo",
		RespectGitIgnore: true,
		Sort:             "path-asc",
	})
	if err != nil {
		t.Fatalf("BuildReport returned error: %v", err)
	}

	if report.Target != "repo" {
		t.Fatalf("expected target repo, got %q", report.Target)
	}
	if report.Root != "repo" {
		t.Fatalf("expected root repo, got %q", report.Root)
	}
	if !report.Recursive {
		t.Fatal("expected recursive report")
	}
	if !report.RespectGitIgnore {
		t.Fatal("expected gitignore to be respected")
	}

	paths := resultPaths(report.Results)
	if slices.Contains(paths, "ignored/secret.txt") {
		t.Fatalf("expected ignored file to be excluded, got %v", paths)
	}
	if slices.Contains(paths, "debug.tmp") {
		t.Fatalf("expected tmp file to be excluded, got %v", paths)
	}
	if slices.Contains(paths, "nested/local.log") {
		t.Fatalf("expected nested ignored file to be excluded, got %v", paths)
	}
	if !slices.Contains(paths, "README.md") || !slices.Contains(paths, "nested/child.txt") {
		t.Fatalf("expected counted files in report, got %v", paths)
	}
}

func TestBuildReportNoGitIgnoreIncludesIgnoredFiles(t *testing.T) {
	t.Parallel()

	report, err := BuildReport(Config{
		CWD:              fixtureParentDir(t),
		Target:           "repo",
		RespectGitIgnore: false,
		Sort:             "path-asc",
	})
	if err != nil {
		t.Fatalf("BuildReport returned error: %v", err)
	}

	paths := resultPaths(report.Results)
	for _, expected := range []string{"ignored/secret.txt", "debug.tmp", "nested/local.log"} {
		if !slices.Contains(paths, expected) {
			t.Fatalf("expected %q in results, got %v", expected, paths)
		}
	}
}

func TestBuildReportDepthOneSkipsNestedDirectories(t *testing.T) {
	t.Parallel()

	report, err := BuildReport(Config{
		CWD:              fixtureParentDir(t),
		Target:           "repo",
		MaxDepth:         intPtr(1),
		RespectGitIgnore: true,
		Sort:             "path-asc",
	})
	if err != nil {
		t.Fatalf("BuildReport returned error: %v", err)
	}

	paths := resultPaths(report.Results)
	if slices.Contains(paths, "nested/child.txt") || slices.Contains(paths, "nested/local.txt") {
		t.Fatalf("expected nested files to be skipped at depth 1, got %v", paths)
	}
	if report.Recursive {
		t.Fatal("expected depth-1 report to be non-recursive")
	}
}

func TestBuildReportDepthTwoIncludesNestedFiles(t *testing.T) {
	t.Parallel()

	report, err := BuildReport(Config{
		CWD:              fixtureParentDir(t),
		Target:           "repo",
		MaxDepth:         intPtr(2),
		RespectGitIgnore: true,
		Sort:             "path-asc",
	})
	if err != nil {
		t.Fatalf("BuildReport returned error: %v", err)
	}

	paths := resultPaths(report.Results)
	if !slices.Contains(paths, "nested/child.txt") || !slices.Contains(paths, "nested/local.txt") {
		t.Fatalf("expected nested files at depth 2, got %v", paths)
	}
	if !report.Recursive {
		t.Fatal("expected depth-2 report to be recursive")
	}
}

func TestBuildReportSummarizeReturnsSingleSummaryRow(t *testing.T) {
	t.Parallel()

	report, err := BuildReport(Config{
		CWD:              fixtureParentDir(t),
		Target:           "repo",
		MaxDepth:         intPtr(0),
		Summarize:        true,
		RespectGitIgnore: true,
		Sort:             "path-asc",
	})
	if err != nil {
		t.Fatalf("BuildReport returned error: %v", err)
	}

	if len(report.Results) != 1 {
		t.Fatalf("expected one summary row, got %d", len(report.Results))
	}
	result := report.Results[0]
	if result.Kind != reportpkg.ResultKindSummary {
		t.Fatalf("expected summary kind, got %+v", result)
	}
	if result.Path != "repo" {
		t.Fatalf("expected summary path repo, got %q", result.Path)
	}
	if result.Status != reportpkg.StatusCounted {
		t.Fatalf("expected counted summary result, got %+v", result)
	}
	if result.Method != nil || result.Provider != nil || result.Reason != nil {
		t.Fatalf("expected summary row to omit method/provider/reason, got %+v", result)
	}
	if result.Tokens == nil || *result.Tokens != report.Summary.TotalTokens {
		t.Fatalf("expected summary tokens to match report summary, got %+v", result)
	}
	if !report.Recursive {
		t.Fatal("expected summarize report to stay recursive")
	}
}

func TestBuildReportFileTarget(t *testing.T) {
	t.Parallel()

	report, err := BuildReport(Config{
		CWD:              fixtureParentDir(t),
		Target:           filepath.ToSlash(filepath.Join("repo", "nested", "child.txt")),
		MaxDepth:         intPtr(0),
		Summarize:        true,
		RespectGitIgnore: true,
		Sort:             "tokens-desc",
	})
	if err != nil {
		t.Fatalf("BuildReport returned error: %v", err)
	}

	if report.Root != "repo/nested" {
		t.Fatalf("expected file target root repo/nested, got %q", report.Root)
	}
	if len(report.Results) != 1 {
		t.Fatalf("expected one file result, got %d", len(report.Results))
	}
	if report.Results[0].Path != "child.txt" {
		t.Fatalf("expected child.txt path, got %q", report.Results[0].Path)
	}
	if report.Results[0].Kind != reportpkg.ResultKindFile {
		t.Fatalf("expected file result kind, got %+v", report.Results[0])
	}
	if report.Results[0].Status != reportpkg.StatusCounted {
		t.Fatalf("expected counted result, got %q", report.Results[0].Status)
	}
	if report.Results[0].Method == nil || *report.Results[0].Method != reportpkg.MethodExact {
		t.Fatalf("expected exact method for file target, got %+v", report.Results[0].Method)
	}
	if report.Results[0].Provider == nil || *report.Results[0].Provider != "openai" {
		t.Fatalf("expected openai provider for file target, got %+v", report.Results[0].Provider)
	}
}

func TestBuildReportClassifiesSkippedFiles(t *testing.T) {
	t.Parallel()

	root := tempRepo(t)
	writeFile(t, filepath.Join(root, "README.md"), []byte("plain text for heuristic counting\n"))
	writeFile(t, filepath.Join(root, "binary.dat"), []byte{0x00, 0x01, 0x02, 0x03})
	writeFile(t, filepath.Join(root, "invalid.txt"), []byte{0xff, 0xfe, 'a', 'b'})

	report, err := BuildReport(Config{
		CWD:              filepath.Dir(root),
		Target:           filepath.Base(root),
		RespectGitIgnore: true,
		Sort:             "path-asc",
	})
	if err != nil {
		t.Fatalf("BuildReport returned error: %v", err)
	}

	byPath := resultsByPath(report.Results)
	if byPath["binary.dat"].Reason == nil || *byPath["binary.dat"].Reason != "binary" {
		t.Fatalf("expected binary.dat to be skipped as binary, got %+v", byPath["binary.dat"])
	}
	if byPath["invalid.txt"].Reason == nil || *byPath["invalid.txt"].Reason != "decode-failed" {
		t.Fatalf("expected invalid.txt to be skipped as decode-failed, got %+v", byPath["invalid.txt"])
	}
	if byPath["README.md"].Status != reportpkg.StatusCounted {
		t.Fatalf("expected README.md to be counted, got %+v", byPath["README.md"])
	}
	if byPath["README.md"].Method == nil || *byPath["README.md"].Method != reportpkg.MethodExact {
		t.Fatalf("expected README.md to use exact counting, got %+v", byPath["README.md"])
	}
	if report.Summary.FilesSkipped != 2 {
		t.Fatalf("expected 2 skipped files, got %d", report.Summary.FilesSkipped)
	}
}

func TestFilterResultsByThresholdPositiveKeepsOnlyHigherTokenCounts(t *testing.T) {
	t.Parallel()

	lowTokens := int64(3)
	highTokens := int64(12)
	filtered := filterResultsByThreshold([]reportpkg.Result{
		{Path: "low.txt", Tokens: &lowTokens},
		{Path: "high.txt", Tokens: &highTokens},
		{Path: "skipped.bin", Tokens: nil},
	}, 5)

	if paths := resultPaths(filtered); !slices.Equal(paths, []string{"high.txt"}) {
		t.Fatalf("expected only high.txt to remain, got %v", paths)
	}
}

func TestFilterResultsByThresholdNegativeKeepsOnlyLowerTokenCounts(t *testing.T) {
	t.Parallel()

	lowTokens := int64(3)
	highTokens := int64(12)
	filtered := filterResultsByThreshold([]reportpkg.Result{
		{Path: "low.txt", Tokens: &lowTokens},
		{Path: "high.txt", Tokens: &highTokens},
	}, -10)

	if paths := resultPaths(filtered); !slices.Equal(paths, []string{"low.txt"}) {
		t.Fatalf("expected only low.txt to remain, got %v", paths)
	}
}

func TestBuildReportThresholdKeepsSummaryTotals(t *testing.T) {
	t.Parallel()

	parent := fixtureParentDir(t)
	unfiltered, err := BuildReport(Config{
		CWD:              parent,
		Target:           "repo",
		RespectGitIgnore: true,
		Sort:             "tokens-desc",
	})
	if err != nil {
		t.Fatalf("BuildReport returned error: %v", err)
	}

	filtered, err := BuildReport(Config{
		CWD:              parent,
		Target:           "repo",
		Threshold:        int64Ptr(unfiltered.Summary.TotalTokens),
		RespectGitIgnore: true,
		Sort:             "tokens-desc",
	})
	if err != nil {
		t.Fatalf("BuildReport returned error: %v", err)
	}

	if len(filtered.Results) != 0 {
		t.Fatalf("expected no filtered results, got %+v", filtered.Results)
	}
	if filtered.Summary != unfiltered.Summary {
		t.Fatalf("expected summary to remain unfiltered, got %+v", filtered.Summary)
	}
	if filtered.Threshold == nil || *filtered.Threshold != unfiltered.Summary.TotalTokens {
		t.Fatalf("expected threshold metadata %d, got %+v", unfiltered.Summary.TotalTokens, filtered.Threshold)
	}
}

func TestBuildReportThresholdOmitsSkippedRows(t *testing.T) {
	t.Parallel()

	root := tempRepo(t)
	writeFile(t, filepath.Join(root, "README.md"), []byte("text for counting\n"))
	writeFile(t, filepath.Join(root, "binary.dat"), []byte{0x00, 0x01, 0x02, 0x03})

	report, err := BuildReport(Config{
		CWD:              filepath.Dir(root),
		Target:           filepath.Base(root),
		Threshold:        int64Ptr(0),
		RespectGitIgnore: true,
		Sort:             "path-asc",
	})
	if err != nil {
		t.Fatalf("BuildReport returned error: %v", err)
	}

	if paths := resultPaths(report.Results); !slices.Equal(paths, []string{"README.md"}) {
		t.Fatalf("expected only counted rows to remain, got %v", paths)
	}
	if report.Summary.FilesSkipped != 1 {
		t.Fatalf("expected skipped summary to remain intact, got %+v", report.Summary)
	}
}

func TestBuildReportSummaryThresholdAppliesToAggregateRow(t *testing.T) {
	t.Parallel()

	parent := fixtureParentDir(t)
	baseline, err := BuildReport(Config{
		CWD:              parent,
		Target:           "repo",
		MaxDepth:         intPtr(0),
		Summarize:        true,
		RespectGitIgnore: true,
		Sort:             "tokens-desc",
	})
	if err != nil {
		t.Fatalf("BuildReport returned error: %v", err)
	}

	if len(baseline.Results) != 1 || baseline.Results[0].Tokens == nil {
		t.Fatalf("expected one summary row, got %+v", baseline.Results)
	}

	filtered, err := BuildReport(Config{
		CWD:              parent,
		Target:           "repo",
		MaxDepth:         intPtr(0),
		Threshold:        int64Ptr(*baseline.Results[0].Tokens),
		Summarize:        true,
		RespectGitIgnore: true,
		Sort:             "tokens-desc",
	})
	if err != nil {
		t.Fatalf("BuildReport returned error: %v", err)
	}

	if len(filtered.Results) != 0 {
		t.Fatalf("expected summary row to be filtered out, got %+v", filtered.Results)
	}
	if filtered.Summary.TotalTokens != baseline.Summary.TotalTokens {
		t.Fatalf("expected summary total tokens to remain unchanged, got %+v", filtered.Summary)
	}
}

func TestBuildReportThresholdFiltersFileTarget(t *testing.T) {
	t.Parallel()

	parent := fixtureParentDir(t)
	target := filepath.ToSlash(filepath.Join("repo", "nested", "child.txt"))
	baseline, err := BuildReport(Config{
		CWD:              parent,
		Target:           target,
		RespectGitIgnore: true,
		Sort:             "tokens-desc",
	})
	if err != nil {
		t.Fatalf("BuildReport returned error: %v", err)
	}

	if len(baseline.Results) != 1 || baseline.Results[0].Tokens == nil {
		t.Fatalf("expected one file result, got %+v", baseline.Results)
	}

	filtered, err := BuildReport(Config{
		CWD:              parent,
		Target:           target,
		Threshold:        int64Ptr(*baseline.Results[0].Tokens),
		RespectGitIgnore: true,
		Sort:             "tokens-desc",
	})
	if err != nil {
		t.Fatalf("BuildReport returned error: %v", err)
	}

	if len(filtered.Results) != 0 {
		t.Fatalf("expected file result to be filtered out, got %+v", filtered.Results)
	}
	if filtered.Summary.TotalTokens != baseline.Summary.TotalTokens {
		t.Fatalf("expected summary total tokens to remain unchanged, got %+v", filtered.Summary)
	}
}

func TestSummarizeCountsHeuristicResults(t *testing.T) {
	t.Parallel()

	exactTokens := int64(10)
	heuristicTokensOne := int64(7)
	heuristicTokensTwo := int64(5)
	exactMethod := reportpkg.MethodExact
	heuristicMethod := reportpkg.MethodHeuristic
	skippedReason := "binary"

	summary := summarize([]reportpkg.Result{
		{
			Kind:   reportpkg.ResultKindFile,
			Path:   "exact.txt",
			Tokens: &exactTokens,
			Method: &exactMethod,
			Status: reportpkg.StatusCounted,
		},
		{
			Kind:   reportpkg.ResultKindFile,
			Path:   "heuristic-one.txt",
			Tokens: &heuristicTokensOne,
			Method: &heuristicMethod,
			Status: reportpkg.StatusCounted,
		},
		{
			Kind:   reportpkg.ResultKindFile,
			Path:   "heuristic-two.txt",
			Tokens: &heuristicTokensTwo,
			Method: &heuristicMethod,
			Status: reportpkg.StatusCounted,
		},
		{
			Kind:   reportpkg.ResultKindFile,
			Path:   "skipped.bin",
			Status: reportpkg.StatusSkipped,
			Reason: &skippedReason,
		},
	})

	if summary.FilesSeen != 4 {
		t.Fatalf("expected 4 files seen, got %d", summary.FilesSeen)
	}
	if summary.FilesCounted != 3 {
		t.Fatalf("expected 3 files counted, got %d", summary.FilesCounted)
	}
	if summary.FilesSkipped != 1 {
		t.Fatalf("expected 1 file skipped, got %d", summary.FilesSkipped)
	}
	if summary.HeuristicResults != 2 {
		t.Fatalf("expected 2 heuristic results, got %d", summary.HeuristicResults)
	}
	if summary.TotalTokens != exactTokens+heuristicTokensOne+heuristicTokensTwo {
		t.Fatalf("expected total tokens %d, got %d", exactTokens+heuristicTokensOne+heuristicTokensTwo, summary.TotalTokens)
	}
}

func TestBuildReportSkipsTooLargeFiles(t *testing.T) {
	t.Parallel()

	root := tempRepo(t)
	writeFile(t, filepath.Join(root, "large.txt"), slices.Repeat([]byte("x"), int(largeFileThresholdBytes)+1))

	report, err := BuildReport(Config{
		CWD:              filepath.Dir(root),
		Target:           filepath.Base(root),
		RespectGitIgnore: true,
		Sort:             "path-asc",
	})
	if err != nil {
		t.Fatalf("BuildReport returned error: %v", err)
	}

	result := resultsByPath(report.Results)["large.txt"]
	if result.Reason == nil || *result.Reason != "too-large" {
		t.Fatalf("expected too-large skip, got %+v", result)
	}
	if result.Method != nil || result.Provider != nil || result.Tokens != nil {
		t.Fatalf("expected too-large skip to have no count metadata, got %+v", result)
	}
}

func TestBuildReportClassifiesPermissionDeniedFiles(t *testing.T) {
	t.Parallel()

	root := tempRepo(t)
	protectedPath := filepath.Join(root, "protected.txt")
	writeFile(t, protectedPath, []byte("secret\n"))
	if err := os.Chmod(protectedPath, 0); err != nil {
		t.Fatalf("chmod protected file: %v", err)
	}
	defer func() {
		_ = os.Chmod(protectedPath, 0o600)
	}()

	report, err := BuildReport(Config{
		CWD:              filepath.Dir(root),
		Target:           filepath.Base(root),
		RespectGitIgnore: true,
		Sort:             "path-asc",
	})
	if err != nil {
		t.Fatalf("BuildReport returned error: %v", err)
	}

	result := resultsByPath(report.Results)["protected.txt"]
	if result.Reason == nil || *result.Reason != "permission-denied" {
		t.Fatalf("expected permission-denied skip, got %+v", result)
	}
}

func TestBuildReportSortPathDesc(t *testing.T) {
	t.Parallel()

	report, err := BuildReport(Config{
		CWD:              fixtureParentDir(t),
		Target:           "repo",
		RespectGitIgnore: true,
		Sort:             "path-desc",
	})
	if err != nil {
		t.Fatalf("BuildReport returned error: %v", err)
	}

	paths := resultPaths(report.Results)
	if !slices.IsSortedFunc(paths, func(left string, right string) int {
		switch {
		case left > right:
			return -1
		case left < right:
			return 1
		default:
			return 0
		}
	}) {
		t.Fatalf("expected path-desc order, got %v", paths)
	}
}

func TestBuildReportDeterministicUnderConcurrency(t *testing.T) {
	t.Parallel()

	parent := fixtureParentDir(t)
	var baseline []reportpkg.Result

	for i := 0; i < 5; i++ {
		report, err := BuildReport(Config{
			CWD:              parent,
			Target:           "repo",
			RespectGitIgnore: true,
			Sort:             "tokens-desc",
		})
		if err != nil {
			t.Fatalf("BuildReport returned error on run %d: %v", i, err)
		}

		if i == 0 {
			baseline = report.Results
			continue
		}

		if !slices.EqualFunc(baseline, report.Results, sameResult) {
			t.Fatalf("expected deterministic results\nbaseline: %+v\ncurrent: %+v", baseline, report.Results)
		}
	}
}

func TestBuildReportDeterministicUnderConcurrencyWithThreshold(t *testing.T) {
	t.Parallel()

	parent := fixtureParentDir(t)
	var baseline []reportpkg.Result

	for i := 0; i < 5; i++ {
		report, err := BuildReport(Config{
			CWD:              parent,
			Target:           "repo",
			Threshold:        int64Ptr(0),
			RespectGitIgnore: true,
			Sort:             "tokens-desc",
		})
		if err != nil {
			t.Fatalf("BuildReport returned error on run %d: %v", i, err)
		}

		if i == 0 {
			baseline = report.Results
			continue
		}

		if !slices.EqualFunc(baseline, report.Results, sameResult) {
			t.Fatalf("expected deterministic thresholded results\nbaseline: %+v\ncurrent: %+v", baseline, report.Results)
		}
	}
}

func TestScanSingleFileCreatesOneCounter(t *testing.T) {
	t.Parallel()

	parent := fixtureParentDir(t)
	rootAbs := filepath.Join(parent, "repo")
	fileAbs := filepath.Join(rootAbs, "README.md")

	var factoryCalls atomic.Int32
	result := scanSingleFile(fileAbs, rootAbs, func() *count.Counter {
		factoryCalls.Add(1)
		return count.NewCounter()
	})

	if factoryCalls.Load() != 1 {
		t.Fatalf("expected one counter creation for single-file scan, got %d", factoryCalls.Load())
	}
	if result.Status != reportpkg.StatusCounted {
		t.Fatalf("expected counted result, got %+v", result)
	}
}

func TestScanDirectoryCreatesOneCounterPerWorker(t *testing.T) {
	t.Parallel()

	parent := fixtureParentDir(t)
	rootAbs := filepath.Join(parent, "repo")

	var factoryCalls atomic.Int32
	results, err := scanDirectory(rootAbs, rootAbs, nil, nil, func() *count.Counter {
		factoryCalls.Add(1)
		return count.NewCounter()
	})
	if err != nil {
		t.Fatalf("scanDirectory returned error: %v", err)
	}

	if factoryCalls.Load() != int32(defaultWorkerCount()) {
		t.Fatalf("expected %d counter creations for worker pool, got %d", defaultWorkerCount(), factoryCalls.Load())
	}
	if len(results) == 0 {
		t.Fatal("expected scanDirectory to return results")
	}
}

func TestNewIgnoreMatcherLoadsNestedGitIgnoresLazily(t *testing.T) {
	t.Parallel()

	root := tempRepo(t)
	nestedDir := filepath.Join(root, "nested")
	if err := os.MkdirAll(nestedDir, 0o755); err != nil {
		t.Fatalf("mkdir nested dir: %v", err)
	}

	writeFile(t, filepath.Join(root, ".gitignore"), []byte("debug.tmp\n"))
	writeFile(t, filepath.Join(nestedDir, ".gitignore"), []byte("local.log\n"))

	matcher, err := newIgnoreMatcher(root, true)
	if err != nil {
		t.Fatalf("newIgnoreMatcher returned error: %v", err)
	}

	rootIgnorePath := filepath.Join(root, ".gitignore")
	nestedIgnorePath := filepath.Join(nestedDir, ".gitignore")
	if _, ok := matcher.loaded[rootIgnorePath]; !ok {
		t.Fatalf("expected root .gitignore to be loaded eagerly")
	}
	if _, ok := matcher.loaded[nestedIgnorePath]; ok {
		t.Fatalf("expected nested .gitignore to load lazily")
	}

	nestedFilePath := filepath.Join(nestedDir, "local.log")
	if matcher.shouldIgnore(nestedFilePath, false) {
		t.Fatalf("expected nested rule to be inactive before directory entry")
	}

	if err := matcher.prepareForDir(nestedDir); err != nil {
		t.Fatalf("prepareForDir returned error: %v", err)
	}

	if _, ok := matcher.loaded[nestedIgnorePath]; !ok {
		t.Fatalf("expected nested .gitignore to be loaded after prepareForDir")
	}
	if !matcher.shouldIgnore(nestedFilePath, false) {
		t.Fatalf("expected nested rule to apply after prepareForDir")
	}
}

func fixtureParentDir(t *testing.T) string {
	t.Helper()

	parent := t.TempDir()
	if _, err := testfixture.MaterializeCanonicalRepo(parent); err != nil {
		t.Fatalf("materialize fixture repo: %v", err)
	}
	return parent
}

func tempRepo(t *testing.T) string {
	t.Helper()

	root := filepath.Join(t.TempDir(), "repo")
	if err := os.MkdirAll(filepath.Join(root, ".git"), 0o755); err != nil {
		t.Fatalf("mkdir .git: %v", err)
	}

	return root
}

func writeFile(t *testing.T, path string, contents []byte) {
	t.Helper()

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", path, err)
	}
	if err := os.WriteFile(path, contents, 0o600); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

func resultPaths(results []reportpkg.Result) []string {
	paths := make([]string, 0, len(results))
	for _, result := range results {
		paths = append(paths, result.Path)
	}

	return paths
}

func resultsByPath(results []reportpkg.Result) map[string]reportpkg.Result {
	byPath := make(map[string]reportpkg.Result, len(results))
	for _, result := range results {
		byPath[result.Path] = result
	}

	return byPath
}

func sameResult(left reportpkg.Result, right reportpkg.Result) bool {
	if left.Kind != right.Kind || left.Path != right.Path || left.Status != right.Status {
		return false
	}
	if !sameNullableInt64(left.Tokens, right.Tokens) {
		return false
	}
	if !sameNullableMethod(left.Method, right.Method) {
		return false
	}
	if !sameNullableString(left.Provider, right.Provider) {
		return false
	}

	return sameNullableString(left.Reason, right.Reason)
}

func sameNullableInt64(left *int64, right *int64) bool {
	if left == nil || right == nil {
		return left == nil && right == nil
	}

	return *left == *right
}

func sameNullableMethod(left *reportpkg.Method, right *reportpkg.Method) bool {
	if left == nil || right == nil {
		return left == nil && right == nil
	}

	return *left == *right
}

func sameNullableString(left *string, right *string) bool {
	if left == nil || right == nil {
		return left == nil && right == nil
	}

	return *left == *right
}

func intPtr(value int) *int {
	return &value
}

func int64Ptr(value int64) *int64 {
	return &value
}
