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
		Recursive:        !opts.NonRecursive,
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
	switch opts.Output {
	case OutputJSON:
		output, err := format.JSON(scanReport)
		if err != nil {
			return err
		}
		_, err = stdout.Write(output)
		return err
	case OutputPlain:
		_, err := stdout.Write(format.Plain(scanReport))
		return err
	default:
		_, err := stdout.Write(format.Human(scanReport))
		return err
	}
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
			countHeuristic(scanReport.Results),
		)
		return
	}

	if scanReport.Summary.FilesSkipped > 0 {
		fmt.Fprintf(stderr, "warning: %d files skipped during scan\n", scanReport.Summary.FilesSkipped)
	}
}

func countHeuristic(results []report.Result) int64 {
	var total int64
	for _, result := range results {
		if result.Method != nil && *result.Method == report.MethodHeuristic {
			total++
		}
	}

	return total
}
