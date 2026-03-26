package cli

import (
	"fmt"
	"io"
)

func Run(args []string, stdout io.Writer, stderr io.Writer, version string) int {
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

	fmt.Fprintln(stderr, "scan execution is not implemented yet; slice 1 only covers CLI parsing, help/version, and validation")
	return 1
}
