package cli

import "testing"

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
