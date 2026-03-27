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
)

type Options struct {
	Path         string
	Output       OutputMode
	Sort         SortMode
	NonRecursive bool
	NoGitIgnore  bool
	Quiet        bool
	NoColor      bool
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
		jsonOutput  bool
		plainOutput bool
		sortValue   string
	)

	fs.BoolVar(&opts.ShowHelp, "help", false, "")
	fs.BoolVar(&opts.ShowHelp, "h", false, "")
	fs.BoolVar(&opts.ShowVersion, "version", false, "")
	fs.BoolVar(&jsonOutput, "json", false, "")
	fs.BoolVar(&plainOutput, "plain", false, "")
	fs.StringVar(&sortValue, "sort", string(SortTokensDesc), "")
	fs.BoolVar(&opts.NonRecursive, "non-recursive", false, "")
	fs.BoolVar(&opts.NoGitIgnore, "no-gitignore", false, "")
	fs.BoolVar(&opts.Quiet, "quiet", false, "")
	fs.BoolVar(&opts.Quiet, "q", false, "")
	fs.BoolVar(&opts.NoColor, "no-color", false, "")

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

	if jsonOutput && plainOutput {
		return Options{}, usageError("--json and --plain cannot be used together")
	}

	if jsonOutput {
		opts.Output = OutputJSON
	}
	if plainOutput {
		opts.Output = OutputPlain
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

	return nil
}

func Usage() string {
	return strings.TrimSpace(`
tu shows token usage for files so humans and agents can identify context-heavy files quickly.

Usage:
  tu [path] [--json | --plain] [--sort <mode>] [--non-recursive] [--no-gitignore]
  tu --help
  tu --version

Arguments:
  path                 File or directory target. Defaults to "."

Options:
  -h, --help           Show usage and exit
      --version        Print version and exit
      --json           Emit deterministic JSON to stdout
      --plain          Emit stable line-based text to stdout
      --sort           One of: tokens-desc, tokens-asc, path-asc, path-desc
      --non-recursive  Do not descend into child directories
      --no-gitignore   Include files ignored by .gitignore
  -q, --quiet          Suppress non-essential stderr messages
      --no-color       Disable color in human output
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
		case arg == "--sort":
			flagArgs = append(flagArgs, arg)
			if i+1 < len(args) {
				i++
				flagArgs = append(flagArgs, args[i])
			}
		case strings.HasPrefix(arg, "--sort="):
			flagArgs = append(flagArgs, arg)
		case strings.HasPrefix(arg, "-") && arg != "-":
			flagArgs = append(flagArgs, arg)
		default:
			positionalArgs = append(positionalArgs, arg)
		}
	}

	return flagArgs, positionalArgs
}
