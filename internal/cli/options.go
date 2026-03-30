package cli

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"math"
	"path"
	"strconv"
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
	Path        string
	Output      OutputMode
	File        string
	Sort        SortMode
	Depth       *int
	Threshold   *int64
	Exclude     []string
	Summarize   bool
	NoGitIgnore bool
	Quiet       bool
	ShowHelp    bool
	ShowVersion bool
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
		formatValue    string
		sortValue      string
		depthValue     intFlag
		thresholdValue int64Flag
		excludeValue   stringSliceFlag
	)

	fs.BoolVar(&opts.ShowHelp, "help", false, "")
	fs.BoolVar(&opts.ShowHelp, "h", false, "")
	fs.BoolVar(&opts.ShowVersion, "version", false, "")
	fs.StringVar(&formatValue, "format", string(OutputHuman), "")
	fs.StringVar(&opts.File, "file", "", "")
	fs.StringVar(&sortValue, "sort", string(SortTokensDesc), "")
	fs.Var(&depthValue, "depth", "")
	fs.Var(&depthValue, "d", "")
	fs.Var(&thresholdValue, "threshold", "")
	fs.Var(&thresholdValue, "t", "")
	fs.Var(&excludeValue, "exclude", "")
	fs.Var(&excludeValue, "I", "")
	fs.BoolVar(&opts.Summarize, "summarize", false, "")
	fs.BoolVar(&opts.Summarize, "s", false, "")
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
	if depthValue.set {
		depth := depthValue.value
		opts.Depth = &depth
	}
	if thresholdValue.set {
		threshold := thresholdValue.value
		opts.Threshold = &threshold
	}
	if excludeValue.set {
		opts.Exclude = append(opts.Exclude, excludeValue.values...)
	}
	if opts.Summarize && opts.Depth == nil {
		depth := 0
		opts.Depth = &depth
	}

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
	if o.Depth != nil && *o.Depth < 0 {
		return usageError(fmt.Sprintf("depth must be >= 0, got %d", *o.Depth))
	}
	if o.Threshold != nil && *o.Threshold == math.MinInt64 {
		return usageError("threshold must be greater than -9223372036854775808")
	}
	for _, pattern := range o.Exclude {
		if pattern == "" {
			return usageError("exclude must not be empty")
		}
		if _, err := path.Match(pattern, ""); err != nil {
			return usageError(fmt.Sprintf("exclude pattern %q is malformed", pattern))
		}
	}
	if o.Summarize && o.Depth != nil && *o.Depth > 0 {
		return usageError("--summarize requires --depth 0 when --depth is also provided")
	}

	return nil
}

func Usage() string {
	return strings.TrimSpace(`
tu shows token usage for files so humans and agents can identify context-heavy files quickly.

Usage:
  tu [path] [--format <human|json|plain|csv>] [--file <path|-] [--sort <mode>] [--depth <n>] [--threshold <tokens>] [--exclude <glob>]... [--summarize] [--no-gitignore]
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
  -d, --depth          Limit file results by depth; use 0 for summary-only, 1 for top-level files
  -t, --threshold      Filter displayed rows by token count; negative values keep rows below abs(threshold)
  -I, --exclude        Ignore files and directories whose basename matches the glob; repeatable
  -s, --summarize      Alias for --depth 0
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
		case arg == "--sort" || arg == "--format" || arg == "--file" || arg == "--depth" || arg == "-d" || arg == "--threshold" || arg == "-t" || arg == "--exclude" || arg == "-I":
			flagArgs = append(flagArgs, arg)
			if i+1 < len(args) {
				i++
				flagArgs = append(flagArgs, args[i])
			}
		case strings.HasPrefix(arg, "--sort="),
			strings.HasPrefix(arg, "--format="),
			strings.HasPrefix(arg, "--file="),
			strings.HasPrefix(arg, "--depth="),
			strings.HasPrefix(arg, "-d="),
			strings.HasPrefix(arg, "--threshold="),
			strings.HasPrefix(arg, "-t="),
			strings.HasPrefix(arg, "--exclude="),
			strings.HasPrefix(arg, "-I="):
			flagArgs = append(flagArgs, arg)
		case strings.HasPrefix(arg, "-") && arg != "-":
			flagArgs = append(flagArgs, arg)
		default:
			positionalArgs = append(positionalArgs, arg)
		}
	}

	return flagArgs, positionalArgs
}

type intFlag struct {
	set   bool
	value int
}

func (f *intFlag) String() string {
	if !f.set {
		return ""
	}

	return strconv.Itoa(f.value)
}

func (f *intFlag) Set(value string) error {
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return err
	}

	f.set = true
	f.value = parsed
	return nil
}

type stringSliceFlag struct {
	set    bool
	values []string
}

func (f *stringSliceFlag) String() string {
	return strings.Join(f.values, ",")
}

func (f *stringSliceFlag) Set(value string) error {
	f.set = true
	f.values = append(f.values, value)
	return nil
}

type int64Flag struct {
	set   bool
	value int64
}

func (f *int64Flag) String() string {
	if !f.set {
		return ""
	}

	return strconv.FormatInt(f.value, 10)
}

func (f *int64Flag) Set(value string) error {
	parsed, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		return err
	}

	f.set = true
	f.value = parsed
	return nil
}
