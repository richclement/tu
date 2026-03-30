package cli

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"math"
	"math/big"
	"path"
	"regexp"
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
	rawByteSizePattern = regexp.MustCompile(`^\d+$`)
	humanSizePattern   = regexp.MustCompile(`^(\d+(?:\.\d+)?)\s*([A-Za-z]+)$`)
	sizeUnits          = map[string]int64{
		"b":   1,
		"kb":  1000,
		"mb":  1000 * 1000,
		"gb":  1000 * 1000 * 1000,
		"tb":  1000 * 1000 * 1000 * 1000,
		"pb":  1000 * 1000 * 1000 * 1000 * 1000,
		"kib": 1 << 10,
		"mib": 1 << 20,
		"gib": 1 << 30,
		"tib": 1 << 40,
		"pib": 1 << 50,
	}
)

type Options struct {
	Path        string
	Output      OutputMode
	File        string
	Sort        SortMode
	Depth       *int
	Threshold   *int64
	MaxFileSize *int64
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
		maxFileSize    byteSizeFlag
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
	fs.Var(&maxFileSize, "max-file-size", "")
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
	if maxFileSize.set {
		size := maxFileSize.value
		opts.MaxFileSize = &size
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
	if o.MaxFileSize != nil && *o.MaxFileSize <= 0 {
		return usageError("max-file-size must be greater than 0")
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
  tu [path] [--format <human|json|plain|csv>] [--file <path|-] [--sort <mode>] [--depth <n>] [--threshold <tokens>] [--max-file-size <bytes|size>] [--exclude <glob>]... [--summarize] [--no-gitignore]
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
      --max-file-size  Skip files larger than this limit before reading; accepts bytes or sizes like 1MiB, 1.5MB, 512KiB
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
		case arg == "--sort" || arg == "--format" || arg == "--file" || arg == "--depth" || arg == "-d" || arg == "--threshold" || arg == "-t" || arg == "--max-file-size" || arg == "--exclude" || arg == "-I":
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
			strings.HasPrefix(arg, "--max-file-size="),
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

type byteSizeFlag struct {
	set   bool
	value int64
}

func (f *byteSizeFlag) String() string {
	if !f.set {
		return ""
	}

	return strconv.FormatInt(f.value, 10)
}

func (f *byteSizeFlag) Set(value string) error {
	parsed, err := parseByteSize(value)
	if err != nil {
		return err
	}

	f.set = true
	f.value = parsed
	return nil
}

func parseByteSize(value string) (int64, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return 0, fmt.Errorf(`invalid max-file-size %q: expected integer bytes or a size like 1.5MiB, 1MB, 512KiB`, value)
	}
	if strings.HasPrefix(trimmed, "-") {
		return 0, errors.New("max-file-size must be greater than 0")
	}
	if rawByteSizePattern.MatchString(trimmed) {
		parsed, err := strconv.ParseInt(trimmed, 10, 64)
		if err != nil {
			return 0, fmt.Errorf(`invalid max-file-size %q: value overflows int64 bytes`, value)
		}
		if parsed <= 0 {
			return 0, errors.New("max-file-size must be greater than 0")
		}
		return parsed, nil
	}

	matches := humanSizePattern.FindStringSubmatch(trimmed)
	if matches == nil {
		return 0, fmt.Errorf(`invalid max-file-size %q: expected integer bytes or a size like 1.5MiB, 1MB, 512KiB`, value)
	}

	unit := strings.ToLower(matches[2])
	multiplier, ok := sizeUnits[unit]
	if !ok {
		return 0, fmt.Errorf(`invalid max-file-size %q: unknown size unit`, value)
	}

	number := new(big.Rat)
	if _, ok := number.SetString(matches[1]); !ok {
		return 0, fmt.Errorf(`invalid max-file-size %q: expected integer bytes or a size like 1.5MiB, 1MB, 512KiB`, value)
	}

	product := new(big.Rat).Mul(number, new(big.Rat).SetInt64(multiplier))
	floor := new(big.Int).Quo(product.Num(), product.Denom())
	if floor.Sign() <= 0 {
		return 0, errors.New("max-file-size must be greater than 0")
	}
	if !floor.IsInt64() {
		return 0, fmt.Errorf(`invalid max-file-size %q: value overflows int64 bytes`, value)
	}

	return floor.Int64(), nil
}
