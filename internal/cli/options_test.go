package cli

import (
	"math"
	"slices"
	"strconv"
	"testing"

	"github.com/richclement/tu/internal/report"
)

func TestParseOptionsDefaults(t *testing.T) {
	t.Parallel()

	opts, err := ParseOptions(nil)
	if err != nil {
		t.Fatalf("ParseOptions returned error: %v", err)
	}

	if opts.Path != "." {
		t.Fatalf("expected default path '.', got %q", opts.Path)
	}
	if opts.Output != OutputHuman {
		t.Fatalf("expected human output, got %q", opts.Output)
	}
	if opts.Sort != SortTokensDesc {
		t.Fatalf("expected default sort %q, got %q", SortTokensDesc, opts.Sort)
	}
	if opts.SymlinkMode != report.SymlinkModePhysical {
		t.Fatalf("expected default symlink mode %q, got %q", report.SymlinkModePhysical, opts.SymlinkMode)
	}
}

func TestParseOptionsWithPathAndFlags(t *testing.T) {
	t.Parallel()

	opts, err := ParseOptions([]string{"docs", "--format", "plain", "--file", "report.txt", "--sort", "path-asc", "--depth", "1", "--no-gitignore", "-q"})
	if err != nil {
		t.Fatalf("ParseOptions returned error: %v", err)
	}

	if opts.Path != "docs" {
		t.Fatalf("expected path docs, got %q", opts.Path)
	}
	if opts.Output != OutputPlain {
		t.Fatalf("expected plain output, got %q", opts.Output)
	}
	if opts.File != "report.txt" {
		t.Fatalf("expected file report.txt, got %q", opts.File)
	}
	if opts.Sort != SortPathAsc {
		t.Fatalf("expected sort %q, got %q", SortPathAsc, opts.Sort)
	}
	if opts.Depth == nil || *opts.Depth != 1 {
		t.Fatalf("expected depth 1, got %+v", opts.Depth)
	}
	if !opts.NoGitIgnore {
		t.Fatal("expected no-gitignore to be true")
	}
	if !opts.Quiet {
		t.Fatal("expected quiet to be true")
	}
}

func TestParseOptionsWithPathAfterFlags(t *testing.T) {
	t.Parallel()

	opts, err := ParseOptions([]string{"--format=json", "--sort=path-desc", "-d", "2", "docs"})
	if err != nil {
		t.Fatalf("ParseOptions returned error: %v", err)
	}

	if opts.Path != "docs" {
		t.Fatalf("expected path docs, got %q", opts.Path)
	}
	if opts.Output != OutputJSON {
		t.Fatalf("expected json output, got %q", opts.Output)
	}
	if opts.Sort != SortPathDesc {
		t.Fatalf("expected sort %q, got %q", SortPathDesc, opts.Sort)
	}
	if opts.Depth == nil || *opts.Depth != 2 {
		t.Fatalf("expected depth 2, got %+v", opts.Depth)
	}
}

func TestParseOptionsSymlinkModeLastFlagWins(t *testing.T) {
	t.Parallel()

	opts, err := ParseOptions([]string{"-L", "docs", "-P", "-H"})
	if err != nil {
		t.Fatalf("ParseOptions returned error: %v", err)
	}

	if opts.SymlinkMode != report.SymlinkModeCommandLine {
		t.Fatalf("expected command-line symlink mode, got %q", opts.SymlinkMode)
	}
}

func TestParseOptionsSupportsCSVAndStdoutFileShortcut(t *testing.T) {
	t.Parallel()

	opts, err := ParseOptions([]string{"--format=csv", "--file=-"})
	if err != nil {
		t.Fatalf("ParseOptions returned error: %v", err)
	}

	if opts.Output != OutputCSV {
		t.Fatalf("expected csv output, got %q", opts.Output)
	}
	if opts.File != "-" {
		t.Fatalf("expected file '-', got %q", opts.File)
	}
}

func TestParseOptionsAllowsDashPrefixedPathAfterDoubleDash(t *testing.T) {
	t.Parallel()

	opts, err := ParseOptions([]string{"--", "--literal-path"})
	if err != nil {
		t.Fatalf("ParseOptions returned error: %v", err)
	}

	if opts.Path != "--literal-path" {
		t.Fatalf("expected path --literal-path, got %q", opts.Path)
	}
}

func TestParseOptionsHelpAndVersion(t *testing.T) {
	t.Parallel()

	opts, err := ParseOptions([]string{"--help"})
	if err != nil {
		t.Fatalf("ParseOptions returned error: %v", err)
	}
	if !opts.ShowHelp {
		t.Fatal("expected help flag to be set")
	}

	opts, err = ParseOptions([]string{"--version"})
	if err != nil {
		t.Fatalf("ParseOptions returned error: %v", err)
	}
	if !opts.ShowVersion {
		t.Fatal("expected version flag to be set")
	}
}

func TestParseOptionsSummarizeSetsDepthZero(t *testing.T) {
	t.Parallel()

	opts, err := ParseOptions([]string{"--summarize"})
	if err != nil {
		t.Fatalf("ParseOptions returned error: %v", err)
	}

	if !opts.Summarize {
		t.Fatal("expected summarize to be true")
	}
	if opts.Depth == nil || *opts.Depth != 0 {
		t.Fatalf("expected depth 0, got %+v", opts.Depth)
	}
}

func TestParseOptionsAllowsSummarizeWithDepthZero(t *testing.T) {
	t.Parallel()

	opts, err := ParseOptions([]string{"-s", "--depth=0"})
	if err != nil {
		t.Fatalf("ParseOptions returned error: %v", err)
	}

	if !opts.Summarize {
		t.Fatal("expected summarize to be true")
	}
	if opts.Depth == nil || *opts.Depth != 0 {
		t.Fatalf("expected depth 0, got %+v", opts.Depth)
	}
}

func TestParseOptionsParsesThreshold(t *testing.T) {
	t.Parallel()

	opts, err := ParseOptions([]string{"repo", "--threshold", "12"})
	if err != nil {
		t.Fatalf("ParseOptions returned error: %v", err)
	}

	if opts.Threshold == nil || *opts.Threshold != 12 {
		t.Fatalf("expected threshold 12, got %+v", opts.Threshold)
	}
}

func TestParseOptionsParsesNegativeThresholdShortForm(t *testing.T) {
	t.Parallel()

	opts, err := ParseOptions([]string{"repo", "-t", "-5"})
	if err != nil {
		t.Fatalf("ParseOptions returned error: %v", err)
	}

	if opts.Threshold == nil || *opts.Threshold != -5 {
		t.Fatalf("expected threshold -5, got %+v", opts.Threshold)
	}
}

func TestParseOptionsParsesThresholdEqualsForms(t *testing.T) {
	t.Parallel()

	longOpts, err := ParseOptions([]string{"--threshold=7"})
	if err != nil {
		t.Fatalf("ParseOptions returned error for long form: %v", err)
	}
	if longOpts.Threshold == nil || *longOpts.Threshold != 7 {
		t.Fatalf("expected threshold 7, got %+v", longOpts.Threshold)
	}

	shortOpts, err := ParseOptions([]string{"-t=-9"})
	if err != nil {
		t.Fatalf("ParseOptions returned error for short form: %v", err)
	}
	if shortOpts.Threshold == nil || *shortOpts.Threshold != -9 {
		t.Fatalf("expected threshold -9, got %+v", shortOpts.Threshold)
	}
}

func TestParseOptionsParsesMaxFileSize(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name string
		args []string
		want int64
	}{
		{name: "raw-bytes", args: []string{"--max-file-size", "1000000"}, want: 1000000},
		{name: "decimal-unit", args: []string{"--max-file-size", "1MB"}, want: 1000000},
		{name: "binary-unit", args: []string{"--max-file-size", "1MiB"}, want: 1 << 20},
		{name: "decimal-binary-unit", args: []string{"--max-file-size", "1.5MiB"}, want: 1572864},
		{name: "space-and-case-insensitive", args: []string{"--max-file-size", "1.5 mib"}, want: 1572864},
		{name: "rounds-down", args: []string{"--max-file-size", "1.1KiB"}, want: 1126},
		{name: "equals-form", args: []string{"--max-file-size=1.5MB"}, want: 1500000},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			opts, err := ParseOptions(tc.args)
			if err != nil {
				t.Fatalf("ParseOptions returned error: %v", err)
			}
			if opts.MaxFileSize == nil || *opts.MaxFileSize != tc.want {
				t.Fatalf("expected max file size %d, got %+v", tc.want, opts.MaxFileSize)
			}
		})
	}
}

func TestParseOptionsParsesExcludeLongAndShortForms(t *testing.T) {
	t.Parallel()

	longOpts, err := ParseOptions([]string{"repo", "--exclude", "node_modules"})
	if err != nil {
		t.Fatalf("ParseOptions returned error for long form: %v", err)
	}
	if !slices.Equal(longOpts.Exclude, []string{"node_modules"}) {
		t.Fatalf("expected exclude node_modules, got %+v", longOpts.Exclude)
	}

	shortOpts, err := ParseOptions([]string{"repo", "-I", "*.min.js"})
	if err != nil {
		t.Fatalf("ParseOptions returned error for short form: %v", err)
	}
	if !slices.Equal(shortOpts.Exclude, []string{"*.min.js"}) {
		t.Fatalf("expected exclude *.min.js, got %+v", shortOpts.Exclude)
	}
}

func TestParseOptionsParsesExcludeEqualsForms(t *testing.T) {
	t.Parallel()

	longOpts, err := ParseOptions([]string{"--exclude=node_modules"})
	if err != nil {
		t.Fatalf("ParseOptions returned error for long form: %v", err)
	}
	if !slices.Equal(longOpts.Exclude, []string{"node_modules"}) {
		t.Fatalf("expected exclude node_modules, got %+v", longOpts.Exclude)
	}

	shortOpts, err := ParseOptions([]string{"-I=*.snap"})
	if err != nil {
		t.Fatalf("ParseOptions returned error for short form: %v", err)
	}
	if !slices.Equal(shortOpts.Exclude, []string{"*.snap"}) {
		t.Fatalf("expected exclude *.snap, got %+v", shortOpts.Exclude)
	}
}

func TestParseOptionsParsesRepeatedExcludeInOrder(t *testing.T) {
	t.Parallel()

	opts, err := ParseOptions([]string{"repo", "-I", "node_modules", "--exclude", "*.snap", "--exclude=dist"})
	if err != nil {
		t.Fatalf("ParseOptions returned error: %v", err)
	}

	if !slices.Equal(opts.Exclude, []string{"node_modules", "*.snap", "dist"}) {
		t.Fatalf("expected ordered excludes, got %+v", opts.Exclude)
	}
}

func TestParseOptionsParsesThresholdAndExcludeTogether(t *testing.T) {
	t.Parallel()

	opts, err := ParseOptions([]string{"repo", "--threshold", "12", "-I", "node_modules"})
	if err != nil {
		t.Fatalf("ParseOptions returned error: %v", err)
	}

	if opts.Threshold == nil || *opts.Threshold != 12 {
		t.Fatalf("expected threshold 12, got %+v", opts.Threshold)
	}
	if !slices.Equal(opts.Exclude, []string{"node_modules"}) {
		t.Fatalf("expected exclude node_modules, got %+v", opts.Exclude)
	}
}

func TestParseOptionsRejectsLegacyJSONFlag(t *testing.T) {
	t.Parallel()

	if _, err := ParseOptions([]string{"--json"}); err == nil {
		t.Fatal("expected legacy flag error, got nil")
	}
}

func TestParseOptionsRejectsLegacyPlainFlag(t *testing.T) {
	t.Parallel()

	if _, err := ParseOptions([]string{"--plain"}); err == nil {
		t.Fatal("expected legacy flag error, got nil")
	}
}

func TestParseOptionsRejectsLegacyNoColorFlag(t *testing.T) {
	t.Parallel()

	if _, err := ParseOptions([]string{"--no-color"}); err == nil {
		t.Fatal("expected legacy flag error, got nil")
	}
}

func TestParseOptionsRejectsRemovedNonRecursiveFlag(t *testing.T) {
	t.Parallel()

	if _, err := ParseOptions([]string{"--non-recursive"}); err == nil {
		t.Fatal("expected removed flag error, got nil")
	}
}

func TestParseOptionsRejectsInvalidFormat(t *testing.T) {
	t.Parallel()

	if _, err := ParseOptions([]string{"--format", "bogus"}); err == nil {
		t.Fatal("expected invalid format error, got nil")
	}
}

func TestParseOptionsRejectsInvalidSort(t *testing.T) {
	t.Parallel()

	if _, err := ParseOptions([]string{"--sort", "bogus"}); err == nil {
		t.Fatal("expected invalid sort error, got nil")
	}
}

func TestParseOptionsRejectsNegativeDepth(t *testing.T) {
	t.Parallel()

	if _, err := ParseOptions([]string{"--depth", "-1"}); err == nil {
		t.Fatal("expected negative depth error, got nil")
	}
}

func TestParseOptionsRejectsSummarizeWithNonZeroDepth(t *testing.T) {
	t.Parallel()

	if _, err := ParseOptions([]string{"--summarize", "--depth", "1"}); err == nil {
		t.Fatal("expected summarize/depth conflict error, got nil")
	}
}

func TestParseOptionsRejectsMissingSortValue(t *testing.T) {
	t.Parallel()

	if _, err := ParseOptions([]string{"--sort"}); err == nil {
		t.Fatal("expected missing sort value error, got nil")
	}
}

func TestParseOptionsRejectsMissingFormatValue(t *testing.T) {
	t.Parallel()

	if _, err := ParseOptions([]string{"--format"}); err == nil {
		t.Fatal("expected missing format value error, got nil")
	}
}

func TestParseOptionsRejectsMissingFileValue(t *testing.T) {
	t.Parallel()

	if _, err := ParseOptions([]string{"--file"}); err == nil {
		t.Fatal("expected missing file value error, got nil")
	}
}

func TestParseOptionsRejectsMissingDepthValue(t *testing.T) {
	t.Parallel()

	if _, err := ParseOptions([]string{"--depth"}); err == nil {
		t.Fatal("expected missing depth value error, got nil")
	}
}

func TestParseOptionsRejectsMissingThresholdValue(t *testing.T) {
	t.Parallel()

	if _, err := ParseOptions([]string{"--threshold"}); err == nil {
		t.Fatal("expected missing threshold value error, got nil")
	}
}

func TestParseOptionsRejectsMissingExcludeValue(t *testing.T) {
	t.Parallel()

	if _, err := ParseOptions([]string{"--exclude"}); err == nil {
		t.Fatal("expected missing exclude value error, got nil")
	}
}

func TestParseOptionsRejectsEmptyExcludeValue(t *testing.T) {
	t.Parallel()

	if _, err := ParseOptions([]string{"--exclude="}); err == nil {
		t.Fatal("expected empty exclude value error, got nil")
	}
}

func TestParseOptionsRejectsMalformedExcludePattern(t *testing.T) {
	t.Parallel()

	for _, args := range [][]string{
		{"--exclude", "["},
		{"--exclude", `\`},
		{"--exclude", "foo["},
	} {
		if _, err := ParseOptions(args); err == nil {
			t.Fatalf("expected malformed exclude error for args %v, got nil", args)
		}
	}
}

func TestParseOptionsRejectsInvalidThresholdValue(t *testing.T) {
	t.Parallel()

	if _, err := ParseOptions([]string{"--threshold", "bogus"}); err == nil {
		t.Fatal("expected invalid threshold error, got nil")
	}
}

func TestParseOptionsRejectsInvalidMaxFileSizeValue(t *testing.T) {
	t.Parallel()

	for _, args := range [][]string{
		{"--max-file-size", "0"},
		{"--max-file-size", "-1"},
		{"--max-file-size", "1.5"},
		{"--max-file-size", "bogus"},
		{"--max-file-size", "1XB"},
		{"--max-file-size", "9223372036854775808"},
		{"--max-file-size", "9223372036854775807PiB"},
	} {
		if _, err := ParseOptions(args); err == nil {
			t.Fatalf("expected invalid max-file-size error for args %v, got nil", args)
		}
	}
}

func TestParseOptionsRejectsMinInt64Threshold(t *testing.T) {
	t.Parallel()

	if _, err := ParseOptions([]string{"--threshold", strconv.FormatInt(math.MinInt64, 10)}); err == nil {
		t.Fatal("expected min-int64 threshold error, got nil")
	}
}

func TestParseOptionsRejectsUnknownFlag(t *testing.T) {
	t.Parallel()

	if _, err := ParseOptions([]string{"--bogus"}); err == nil {
		t.Fatal("expected unknown flag error, got nil")
	}
}

func TestParseOptionsRejectsExtraArgs(t *testing.T) {
	t.Parallel()

	if _, err := ParseOptions([]string{"a", "b"}); err == nil {
		t.Fatal("expected extra arg error, got nil")
	}
}
