package cli

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"strings"
)

type OutputMode string

const (
	OutputHuman OutputMode = "human"
	OutputJSON  OutputMode = "json"
	OutputPlain OutputMode = "plain"
	OutputCSV   OutputMode = "csv"
)

type SortMode string

const (
	SortTokensDesc SortMode = "tokens-desc"
	SortTokensAsc  SortMode = "tokens-asc"
	SortPathAsc    SortMode = "path-asc"
	SortPathDesc   SortMode = "path-desc"
)

var (
	errInvalidUsage = errors.New("invalid usage")
	validSortModes  = map[SortMode]struct{}{
		SortTokensDesc: {},
		SortTokensAsc:  {},
		SortPathAsc:    {},
		SortPathDesc:   {},
	}
	validOutputModes = map[OutputMode]struct{}{
		OutputHuman: {},
		OutputJSON:  {},
		OutputPlain: {},
		OutputCSV:   {},
	}
)

type Options struct {
	Path         string
	Output       OutputMode
	File         string
	Sort         SortMode
	NonRecursive bool
	NoGitIgnore  bool
	Quiet        bool
	ShowHelp     bool
	ShowVersion  bool
}

func DefaultOptions() Options {
	return Options{
		Path:   ".",
		Output: OutputHuman,
		Sort:   SortTokensDesc,
	}
}

func ParseOptions(args []string) (Options, error) {
	opts := DefaultOptions()
	normalizedArgs, positionalArgs := normalizeArgs(args)

	fs := flag.NewFlagSet("tu", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	var (
		formatValue string
		sortValue   string
	)

	fs.BoolVar(&opts.ShowHelp, "help", false, "")
	fs.BoolVar(&opts.ShowHelp, "h", false, "")
	fs.BoolVar(&opts.ShowVersion, "version", false, "")
	fs.StringVar(&formatValue, "format", string(OutputHuman), "")
	fs.StringVar(&opts.File, "file", "", "")
	fs.StringVar(&sortValue, "sort", string(SortTokensDesc), "")
	fs.BoolVar(&opts.NonRecursive, "non-recursive", false, "")
	fs.BoolVar(&opts.NoGitIgnore, "no-gitignore", false, "")
	fs.BoolVar(&opts.Quiet, "quiet", false, "")
	fs.BoolVar(&opts.Quiet, "q", false, "")

	if err := fs.Parse(normalizedArgs); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			opts.ShowHelp = true
			return opts, nil
		}

		return Options{}, usageError(err.Error())
	}

	if opts.ShowHelp || opts.ShowVersion {
		return opts, nil
	}

	if len(fs.Args()) > 0 {
		return Options{}, usageError("unexpected positional arguments after flag parsing")
	}

	if len(positionalArgs) > 0 {
		opts.Path = positionalArgs[0]
		if len(positionalArgs) > 1 {
			return Options{}, usageError("expected at most one path argument")
		}
	}

	opts.Sort = SortMode(sortValue)
	opts.Output = OutputMode(formatValue)

	if err := opts.Validate(); err != nil {
		return Options{}, err
	}

	return opts, nil
}

func (o Options) Validate() error {
	if o.ShowHelp || o.ShowVersion {
		return nil
	}

	if _, ok := validSortModes[o.Sort]; !ok {
		return usageError(fmt.Sprintf("unsupported sort mode %q", o.Sort))
	}
	if _, ok := validOutputModes[o.Output]; !ok {
		return usageError(fmt.Sprintf("unsupported format %q", o.Output))
	}

	return nil
}

func Usage() string {
	return strings.TrimSpace(`
tu shows token usage for files so humans and agents can identify context-heavy files quickly.

Usage:
  tu [path] [--format <human|json|plain|csv>] [--file <path|-] [--sort <mode>] [--non-recursive] [--no-gitignore]
  tu --help
  tu --version

Arguments:
  path                 File or directory target. Defaults to "."

Options:
  -h, --help           Show usage and exit
      --version        Print version and exit
      --format         One of: human, json, plain, csv
      --file           Write primary output to a file path or "-" for stdout
      --sort           One of: tokens-desc, tokens-asc, path-asc, path-desc
      --non-recursive  Do not descend into child directories
      --no-gitignore   Include files ignored by .gitignore
  -q, --quiet          Suppress non-essential stderr messages
`)
}

func usageError(message string) error {
	return fmt.Errorf("%w: %s", errInvalidUsage, message)
}

func IsUsageError(err error) bool {
	return errors.Is(err, errInvalidUsage)
}

func normalizeArgs(args []string) ([]string, []string) {
	flagArgs := make([]string, 0, len(args))
	positionalArgs := make([]string, 0, 1)

	for i := 0; i < len(args); i++ {
		arg := args[i]

		switch {
		case arg == "--":
			positionalArgs = append(positionalArgs, args[i+1:]...)
			return flagArgs, positionalArgs
		case arg == "--sort" || arg == "--format" || arg == "--file":
			flagArgs = append(flagArgs, arg)
			if i+1 < len(args) {
				i++
				flagArgs = append(flagArgs, args[i])
			}
		case strings.HasPrefix(arg, "--sort="),
			strings.HasPrefix(arg, "--format="),
			strings.HasPrefix(arg, "--file="):
			flagArgs = append(flagArgs, arg)
		case strings.HasPrefix(arg, "-") && arg != "-":
			flagArgs = append(flagArgs, arg)
		default:
			positionalArgs = append(positionalArgs, arg)
		}
	}

	return flagArgs, positionalArgs
}
