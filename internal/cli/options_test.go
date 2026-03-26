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

	opts, err := ParseOptions([]string{"docs", "--plain", "--sort", "path-asc", "--non-recursive", "--no-gitignore", "-q", "--no-color"})
	if err != nil {
		t.Fatalf("ParseOptions returned error: %v", err)
	}

	if opts.Path != "docs" {
		t.Fatalf("expected path docs, got %q", opts.Path)
	}
	if opts.Output != OutputPlain {
		t.Fatalf("expected plain output, got %q", opts.Output)
	}
	if opts.Sort != SortPathAsc {
		t.Fatalf("expected sort %q, got %q", SortPathAsc, opts.Sort)
	}
	if !opts.NonRecursive {
		t.Fatal("expected non-recursive to be true")
	}
	if !opts.NoGitIgnore {
		t.Fatal("expected no-gitignore to be true")
	}
	if !opts.Quiet {
		t.Fatal("expected quiet to be true")
	}
	if !opts.NoColor {
		t.Fatal("expected no-color to be true")
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

func TestParseOptionsRejectsConflictingOutputModes(t *testing.T) {
	t.Parallel()

	if _, err := ParseOptions([]string{"--json", "--plain"}); err == nil {
		t.Fatal("expected conflict error, got nil")
	}
}

func TestParseOptionsRejectsInvalidSort(t *testing.T) {
	t.Parallel()

	if _, err := ParseOptions([]string{"--sort", "bogus"}); err == nil {
		t.Fatal("expected invalid sort error, got nil")
	}
}

func TestParseOptionsRejectsExtraArgs(t *testing.T) {
	t.Parallel()

	if _, err := ParseOptions([]string{"a", "b"}); err == nil {
		t.Fatal("expected extra arg error, got nil")
	}
}
