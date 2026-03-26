package format

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/richclement/tu/internal/report"
)

func TestJSONGolden(t *testing.T) {
	t.Parallel()

	got, err := JSON(sampleReport())
	if err != nil {
		t.Fatalf("JSON returned error: %v", err)
	}

	want := readGoldenFile(t, "report.json.golden")
	if string(got) != string(want) {
		t.Fatalf("json golden mismatch\nwant:\n%s\ngot:\n%s", want, got)
	}
}

func TestPlainGolden(t *testing.T) {
	t.Parallel()

	got := Plain(sampleReport())
	want := readGoldenFile(t, "report.plain.golden")
	if string(got) != string(want) {
		t.Fatalf("plain golden mismatch\nwant:\n%s\ngot:\n%s", want, got)
	}
}

func TestPlainEmptyResults(t *testing.T) {
	t.Parallel()

	got := Plain(report.ScanReport{})
	if len(got) != 0 {
		t.Fatalf("expected empty output for empty results, got %q", got)
	}
}

func TestJSONEmptyResultsUsesArray(t *testing.T) {
	t.Parallel()

	got, err := JSON(report.ScanReport{SchemaVersion: report.SchemaVersionV1})
	if err != nil {
		t.Fatalf("JSON returned error: %v", err)
	}

	if !bytes.Contains(got, []byte(`"results": []`)) {
		t.Fatalf("expected empty results array, got %s", got)
	}

	var decoded map[string]any
	if err := json.Unmarshal(got, &decoded); err != nil {
		t.Fatalf("json unmarshal: %v", err)
	}

	results, ok := decoded["results"].([]any)
	if !ok {
		t.Fatalf("expected results to decode as array, got %T", decoded["results"])
	}
	if len(results) != 0 {
		t.Fatalf("expected empty results array, got %v", results)
	}
}

func TestPlainEscapesUnsafePathCharacters(t *testing.T) {
	t.Parallel()

	tokens := int64(7)
	method := report.MethodHeuristic
	got := Plain(report.ScanReport{
		Results: []report.Result{
			{
				Path:   "dir/with\tbreak\nname\\file.txt",
				Bytes:  99,
				Tokens: &tokens,
				Method: &method,
				Status: report.StatusCounted,
			},
		},
	})

	want := "7\t99\theuristic\tcounted\tdir/with\\tbreak\\nname\\\\file.txt\n"
	if string(got) != want {
		t.Fatalf("plain output mismatch\nwant:\n%s\ngot:\n%s", want, got)
	}
}

func readGoldenFile(t *testing.T, name string) []byte {
	t.Helper()

	path := filepath.Join("testdata", name)
	contents, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read golden file %s: %v", name, err)
	}

	return contents
}

func sampleReport() report.ScanReport {
	countedTokens := int64(321)
	countedMethod := report.MethodExact
	countedProvider := "openai"
	skippedReason := "binary"

	return report.ScanReport{
		SchemaVersion:    report.SchemaVersionV1,
		Target:           ".",
		Root:             ".",
		Recursive:        true,
		RespectGitIgnore: true,
		Sort:             "tokens-desc",
		Summary: report.Summary{
			FilesSeen:    2,
			FilesCounted: 1,
			FilesSkipped: 1,
			TotalBytes:   5801,
			TotalTokens:  321,
		},
		Results: []report.Result{
			{
				Path:     "README.md",
				Bytes:    1234,
				Tokens:   &countedTokens,
				Method:   &countedMethod,
				Provider: &countedProvider,
				Status:   report.StatusCounted,
				Reason:   nil,
			},
			{
				Path:     "assets/logo.png",
				Bytes:    4567,
				Tokens:   nil,
				Method:   nil,
				Provider: nil,
				Status:   report.StatusSkipped,
				Reason:   &skippedReason,
			},
		},
	}
}
