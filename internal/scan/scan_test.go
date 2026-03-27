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
		Recursive:        true,
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
		Recursive:        true,
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

func TestBuildReportNonRecursiveSkipsNestedDirectories(t *testing.T) {
	t.Parallel()

	report, err := BuildReport(Config{
		CWD:              fixtureParentDir(t),
		Target:           "repo",
		Recursive:        false,
		RespectGitIgnore: true,
		Sort:             "path-asc",
	})
	if err != nil {
		t.Fatalf("BuildReport returned error: %v", err)
	}

	paths := resultPaths(report.Results)
	if slices.Contains(paths, "nested/child.txt") || slices.Contains(paths, "nested/local.txt") {
		t.Fatalf("expected nested files to be skipped in non-recursive mode, got %v", paths)
	}
}

func TestBuildReportFileTarget(t *testing.T) {
	t.Parallel()

	report, err := BuildReport(Config{
		CWD:              fixtureParentDir(t),
		Target:           filepath.ToSlash(filepath.Join("repo", "nested", "child.txt")),
		Recursive:        true,
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
		Recursive:        true,
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

func TestBuildReportSkipsTooLargeFiles(t *testing.T) {
	t.Parallel()

	root := tempRepo(t)
	writeFile(t, filepath.Join(root, "large.txt"), slices.Repeat([]byte("x"), int(largeFileThresholdBytes)+1))

	report, err := BuildReport(Config{
		CWD:              filepath.Dir(root),
		Target:           filepath.Base(root),
		Recursive:        true,
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
		Recursive:        true,
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
		Recursive:        true,
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
			Recursive:        true,
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
	results, err := scanDirectory(rootAbs, rootAbs, true, nil, func() *count.Counter {
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
	if left.Path != right.Path || left.Status != right.Status {
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
