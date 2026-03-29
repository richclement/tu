package cli

import (
	"fmt"
	"io"
	"os"

	"github.com/richclement/tu/internal/format"
	"github.com/richclement/tu/internal/report"
	"github.com/richclement/tu/internal/scan"
)

func Run(args []string, stdout io.Writer, stderr io.Writer, version string) int {
	cwd, err := os.Getwd()
	if err != nil {
		fmt.Fprintf(stderr, "error: determine working directory: %v\n", err)
		return 1
	}

	return runWithCWD(args, stdout, stderr, version, cwd)
}

func runWithCWD(args []string, stdout io.Writer, stderr io.Writer, version string, cwd string) int {
	opts, err := ParseOptions(args)
	if err != nil {
		if IsUsageError(err) {
			fmt.Fprintf(stderr, "error: %s\n", err)
			fmt.Fprintln(stderr, "Run `tu --help` for usage.")
			return 2
		}

		fmt.Fprintf(stderr, "error: %v\n", err)
		return 1
	}

	if opts.ShowHelp {
		fmt.Fprintln(stdout, Usage())
		return 0
	}

	if opts.ShowVersion {
		fmt.Fprintln(stdout, version)
		return 0
	}

	scanReport, err := scan.BuildReport(scan.Config{
		CWD:              cwd,
		Target:           opts.Path,
		MaxDepth:         opts.Depth,
		Summarize:        opts.Summarize,
		RespectGitIgnore: !opts.NoGitIgnore,
		Sort:             string(opts.Sort),
	})
	if err != nil {
		fmt.Fprintf(stderr, "error: %v\n", err)
		return 1
	}

	if err := writePrimaryOutput(stdout, scanReport, opts); err != nil {
		fmt.Fprintf(stderr, "error: %v\n", err)
		return 1
	}

	writeSummary(stderr, scanReport, opts)
	return 0
}

func writePrimaryOutput(stdout io.Writer, scanReport report.ScanReport, opts Options) error {
	output, err := renderPrimaryOutput(scanReport, opts)
	if err != nil {
		return err
	}

	writer, closer, err := openOutputWriter(stdout, opts)
	if err != nil {
		return err
	}
	if closer != nil {
		defer closer.Close()
	}

	_, err = writer.Write(output)
	return err
}

func renderPrimaryOutput(scanReport report.ScanReport, opts Options) ([]byte, error) {
	switch opts.Output {
	case OutputJSON:
		return format.JSON(scanReport)
	case OutputPlain:
		return format.Plain(scanReport), nil
	case OutputCSV:
		return format.CSV(scanReport)
	default:
		return format.Human(scanReport), nil
	}
}

func openOutputWriter(stdout io.Writer, opts Options) (io.Writer, io.Closer, error) {
	if opts.File == "" || opts.File == "-" {
		return stdout, nil, nil
	}

	file, err := os.Create(opts.File)
	if err != nil {
		return nil, nil, fmt.Errorf("open output file %q: %w", opts.File, err)
	}

	return file, file, nil
}

func writeSummary(stderr io.Writer, scanReport report.ScanReport, opts Options) {
	if opts.Quiet {
		return
	}

	if opts.Output == OutputHuman {
		fmt.Fprintf(
			stderr,
			"files counted: %d, files skipped: %d, heuristic results: %d\n",
			scanReport.Summary.FilesCounted,
			scanReport.Summary.FilesSkipped,
			scanReport.Summary.HeuristicResults,
		)
		return
	}

	if scanReport.Summary.FilesSkipped > 0 {
		fmt.Fprintf(stderr, "warning: %d files skipped during scan\n", scanReport.Summary.FilesSkipped)
	}
}
