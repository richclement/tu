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
	if bytes.Contains(got, []byte(`"bytes"`)) {
		t.Fatalf("expected json output to omit per-result bytes, got %s", got)
	}
	if bytes.Contains(got, []byte(`"total_bytes"`)) {
		t.Fatalf("expected json output to omit total_bytes, got %s", got)
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

func TestCSVGolden(t *testing.T) {
	t.Parallel()

	got, err := CSV(sampleReport())
	if err != nil {
		t.Fatalf("CSV returned error: %v", err)
	}

	want := readGoldenFile(t, "report.csv.golden")
	if string(got) != string(want) {
		t.Fatalf("csv golden mismatch\nwant:\n%s\ngot:\n%s", want, got)
	}
}

func TestPlainEmptyResults(t *testing.T) {
	t.Parallel()

	got := Plain(report.ScanReport{})
	if len(got) != 0 {
		t.Fatalf("expected empty output for empty results, got %q", got)
	}
}

func TestCSVEmptyResultsIncludesHeader(t *testing.T) {
	t.Parallel()

	got, err := CSV(report.ScanReport{})
	if err != nil {
		t.Fatalf("CSV returned error: %v", err)
	}

	want := "kind,path,tokens,method,provider,status,reason\n"
	if string(got) != want {
		t.Fatalf("expected header-only csv output, got %q", got)
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
	if !bytes.Contains(got, []byte(`"symlink_mode": "physical"`)) {
		t.Fatalf("expected default symlink_mode in json output, got %s", got)
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

func TestJSONIncludesThresholdWhenSet(t *testing.T) {
	t.Parallel()

	threshold := int64(9)
	got, err := JSON(report.ScanReport{
		SchemaVersion: report.SchemaVersionV1,
		Threshold:     &threshold,
	})
	if err != nil {
		t.Fatalf("JSON returned error: %v", err)
	}

	if !bytes.Contains(got, []byte(`"threshold": 9`)) {
		t.Fatalf("expected threshold in json output, got %s", got)
	}
}

func TestJSONIncludesMaxFileSizeWhenSet(t *testing.T) {
	t.Parallel()

	maxFileSize := int64(1572864)
	got, err := JSON(report.ScanReport{
		SchemaVersion:    report.SchemaVersionV1,
		MaxFileSizeBytes: &maxFileSize,
	})
	if err != nil {
		t.Fatalf("JSON returned error: %v", err)
	}

	if !bytes.Contains(got, []byte(`"max_file_size_bytes": 1572864`)) {
		t.Fatalf("expected max_file_size_bytes in json output, got %s", got)
	}
}

func TestHumanEmptyResultsUsesThresholdMessageWhenSet(t *testing.T) {
	t.Parallel()

	got := Human(report.ScanReport{ThresholdEmptied: true})
	if string(got) != "No entries matched threshold.\n" {
		t.Fatalf("expected threshold-aware human message, got %q", got)
	}
}

func TestHumanEmptyResultsUsesNoFilesMessageWhenThresholdDidNotEmptyResults(t *testing.T) {
	t.Parallel()

	threshold := int64(4)
	got := Human(report.ScanReport{Threshold: &threshold})
	if string(got) != "No files found.\n" {
		t.Fatalf("expected no-files message when threshold did not empty results, got %q", got)
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
				Tokens: &tokens,
				Method: &method,
				Status: report.StatusCounted,
			},
		},
	})

	want := "7\theuristic\tcounted\tdir/with\\tbreak\\nname\\\\file.txt\n"
	if string(got) != want {
		t.Fatalf("plain output mismatch\nwant:\n%s\ngot:\n%s", want, got)
	}
}

func TestCSVEscapesUnsafeCharacters(t *testing.T) {
	t.Parallel()

	tokens := int64(7)
	method := report.MethodHeuristic
	provider := "local,quoted"
	reason := "line one\nline two"
	got, err := CSV(report.ScanReport{
		Results: []report.Result{
			{
				Kind:     report.ResultKindFile,
				Path:     "dir/with,break\"name.txt",
				Tokens:   &tokens,
				Method:   &method,
				Provider: &provider,
				Status:   report.StatusSkipped,
				Reason:   &reason,
			},
		},
	})
	if err != nil {
		t.Fatalf("CSV returned error: %v", err)
	}

	want := "kind,path,tokens,method,provider,status,reason\nfile,\"dir/with,break\"\"name.txt\",7,heuristic,\"local,quoted\",skipped,\"line one\nline two\"\n"
	if string(got) != want {
		t.Fatalf("csv output mismatch\nwant:\n%s\ngot:\n%s", want, got)
	}
}

func TestCSVFormatsNilFieldsAsEmpty(t *testing.T) {
	t.Parallel()

	got, err := CSV(report.ScanReport{
		Results: []report.Result{
			{
				Kind:   report.ResultKindFile,
				Path:   "assets/logo.png",
				Status: report.StatusSkipped,
			},
		},
	})
	if err != nil {
		t.Fatalf("CSV returned error: %v", err)
	}

	want := "kind,path,tokens,method,provider,status,reason\nfile,assets/logo.png,,,,skipped,\n"
	if string(got) != want {
		t.Fatalf("expected empty csv fields for nil values, got %q", got)
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
		SymlinkMode:      report.SymlinkModePhysical,
		Sort:             "tokens-desc",
		Summary: report.Summary{
			FilesSeen:    2,
			FilesCounted: 1,
			FilesSkipped: 1,
			TotalTokens:  321,
		},
		Results: []report.Result{
			{
				Kind:     report.ResultKindFile,
				Path:     "README.md",
				Tokens:   &countedTokens,
				Method:   &countedMethod,
				Provider: &countedProvider,
				Status:   report.StatusCounted,
				Reason:   nil,
			},
			{
				Kind:     report.ResultKindFile,
				Path:     "assets/logo.png",
				Tokens:   nil,
				Method:   nil,
				Provider: nil,
				Status:   report.StatusSkipped,
				Reason:   &skippedReason,
			},
		},
	}
}
