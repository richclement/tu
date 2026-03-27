package scan

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/richclement/tu/internal/testfixture"
)

func BenchmarkBuildReportFixtureRepo(b *testing.B) {
	parent := fixtureParentDirForBenchmark(b)

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := BuildReport(Config{
			CWD:              parent,
			Target:           "repo",
			Recursive:        true,
			RespectGitIgnore: true,
			Sort:             "tokens-desc",
		}); err != nil {
			b.Fatalf("BuildReport returned error: %v", err)
		}
	}
}

func BenchmarkBuildReportSyntheticRepo(b *testing.B) {
	parent := syntheticBenchmarkParentDir(b)

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := BuildReport(Config{
			CWD:              parent,
			Target:           "repo",
			Recursive:        true,
			RespectGitIgnore: true,
			Sort:             "tokens-desc",
		}); err != nil {
			b.Fatalf("BuildReport returned error: %v", err)
		}
	}
}

func fixtureParentDirForBenchmark(b *testing.B) string {
	b.Helper()

	parent := b.TempDir()
	if _, err := testfixture.MaterializeCanonicalRepo(parent); err != nil {
		b.Fatalf("materialize fixture repo: %v", err)
	}

	return parent
}

func syntheticBenchmarkParentDir(b *testing.B) string {
	b.Helper()

	parent := b.TempDir()
	destRoot := filepath.Join(parent, "repo")
	mustMakeRepoFiles(b, destRoot)

	for dirIndex := 0; dirIndex < 8; dirIndex++ {
		for fileIndex := 0; fileIndex < 40; fileIndex++ {
			path := filepath.Join(destRoot, "docs", "set", "section", benchmarkName(dirIndex, fileIndex)+".md")
			writeFixtureFileForBenchmark(b, path, []byte("This benchmark file exists to exercise concurrent counting in Milestone 5.\n"))
		}
	}

	return parent
}

func benchmarkName(dirIndex int, fileIndex int) string {
	return fmt.Sprintf("group-%02d-file-%02d", dirIndex, fileIndex)
}

func mustMakeRepoFiles(b *testing.B, destRoot string) {
	b.Helper()

	if err := os.MkdirAll(filepath.Join(destRoot, ".git"), 0o755); err != nil {
		b.Fatalf("mkdir .git: %v", err)
	}

	writeFixtureFileForBenchmark(b, filepath.Join(destRoot, "debug.tmp"), []byte("This file should be ignored when gitignore rules are enabled.\n"))
	writeFixtureFileForBenchmark(b, filepath.Join(destRoot, "ignored", "secret.txt"), []byte("This ignored file should only appear when --no-gitignore is used.\n"))
	writeFixtureFileForBenchmark(b, filepath.Join(destRoot, "nested", "local.log"), []byte("This file is ignored by the nested .gitignore rule.\n"))
}

func writeFixtureFileForBenchmark(b *testing.B, path string, contents []byte) {
	b.Helper()

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		b.Fatalf("mkdir %s: %v", path, err)
	}
	if err := os.WriteFile(path, contents, 0o600); err != nil {
		b.Fatalf("write %s: %v", path, err)
	}
}
