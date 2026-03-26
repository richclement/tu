package scan

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"unicode/utf8"

	"github.com/richclement/tu/internal/report"
)

const heuristicProvider = "heuristic"

type Config struct {
	CWD              string
	Target           string
	Recursive        bool
	RespectGitIgnore bool
	Sort             string
}

func BuildReport(cfg Config) (report.ScanReport, error) {
	targetArg := cfg.Target
	if targetArg == "" {
		targetArg = "."
	}

	targetAbs := targetArg
	if !filepath.IsAbs(targetAbs) {
		targetAbs = filepath.Join(cfg.CWD, targetAbs)
	}
	targetAbs = filepath.Clean(targetAbs)

	info, err := os.Stat(targetAbs)
	if err != nil {
		return report.ScanReport{}, fmt.Errorf("stat target %q: %w", targetArg, err)
	}

	rootAbs := targetAbs
	rootDisplay := normalizeRoot(targetArg, info.IsDir())
	recursive := cfg.Recursive && info.IsDir()
	respectGitIgnore := cfg.RespectGitIgnore && info.IsDir()
	if !info.IsDir() {
		rootAbs = filepath.Dir(targetAbs)
		recursive = false
	}

	scanReport := report.ScanReport{
		SchemaVersion:    report.SchemaVersionV1,
		Target:           normalizeTarget(targetArg),
		Root:             rootDisplay,
		Recursive:        recursive,
		RespectGitIgnore: respectGitIgnore,
		Sort:             cfg.Sort,
		Results:          []report.Result{},
	}

	matcher, err := newIgnoreMatcher(rootAbs, info.IsDir(), respectGitIgnore)
	if err != nil {
		return report.ScanReport{}, err
	}

	if info.IsDir() {
		scanReport.Results, err = scanDirectory(targetAbs, rootAbs, recursive, matcher)
		if err != nil {
			return report.ScanReport{}, err
		}
	} else {
		result := scanFile(targetAbs, rootAbs)
		scanReport.Results = append(scanReport.Results, result)
	}

	sortResults(scanReport.Results, cfg.Sort)
	scanReport.Summary = summarize(scanReport.Results)

	return scanReport, nil
}

func scanDirectory(targetAbs string, rootAbs string, recursive bool, matcher *ignoreMatcher) ([]report.Result, error) {
	results := make([]report.Result, 0)

	err := filepath.WalkDir(targetAbs, func(currentPath string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			if currentPath == targetAbs {
				return walkErr
			}

			results = append(results, skippedFromError(relativePath(rootAbs, currentPath), 0, walkErr))
			if entry != nil && entry.IsDir() {
				return filepath.SkipDir
			}

			return nil
		}

		if currentPath == targetAbs {
			return nil
		}

		if entry.IsDir() {
			if entry.Name() == ".git" {
				return filepath.SkipDir
			}

			if matcher != nil && matcher.shouldIgnore(currentPath, true) {
				return filepath.SkipDir
			}

			if !recursive {
				return filepath.SkipDir
			}

			return nil
		}

		if matcher != nil && matcher.shouldIgnore(currentPath, false) {
			return nil
		}

		results = append(results, scanFile(currentPath, rootAbs))
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("walk target %q: %w", normalizeTarget(targetAbs), err)
	}

	return results, nil
}

func scanFile(absPath string, rootAbs string) report.Result {
	displayPath := relativePath(rootAbs, absPath)

	info, err := os.Stat(absPath)
	if err != nil {
		return skippedFromError(displayPath, 0, err)
	}

	if !info.Mode().IsRegular() {
		return skippedResult(displayPath, info.Size(), "unreadable")
	}

	contents, err := os.ReadFile(absPath)
	if err != nil {
		return skippedFromError(displayPath, info.Size(), err)
	}

	if reason, ok := classifyContents(contents); ok {
		return skippedResult(displayPath, info.Size(), reason)
	}

	tokens := estimateTokens(contents)
	method := report.MethodHeuristic
	provider := heuristicProvider

	return report.Result{
		Path:     displayPath,
		Bytes:    info.Size(),
		Tokens:   &tokens,
		Method:   &method,
		Provider: &provider,
		Status:   report.StatusCounted,
	}
}

func summarize(results []report.Result) report.Summary {
	summary := report.Summary{}

	for _, result := range results {
		summary.FilesSeen++
		summary.TotalBytes += result.Bytes

		switch result.Status {
		case report.StatusCounted:
			summary.FilesCounted++
			if result.Tokens != nil {
				summary.TotalTokens += *result.Tokens
			}
		case report.StatusSkipped:
			summary.FilesSkipped++
		}
	}

	return summary
}

func sortResults(results []report.Result, mode string) {
	sort.SliceStable(results, func(i int, j int) bool {
		left := results[i]
		right := results[j]

		switch mode {
		case "path-asc":
			return left.Path < right.Path
		case "path-desc":
			return left.Path > right.Path
		case "tokens-asc":
			return compareTokens(left, right, true)
		case "tokens-desc":
			fallthrough
		default:
			return compareTokens(left, right, false)
		}
	})
}

func compareTokens(left report.Result, right report.Result, ascending bool) bool {
	leftHasTokens := left.Tokens != nil
	rightHasTokens := right.Tokens != nil
	if leftHasTokens != rightHasTokens {
		return leftHasTokens
	}

	if leftHasTokens && rightHasTokens && *left.Tokens != *right.Tokens {
		if ascending {
			return *left.Tokens < *right.Tokens
		}

		return *left.Tokens > *right.Tokens
	}

	return left.Path < right.Path
}

func estimateTokens(contents []byte) int64 {
	if len(contents) == 0 {
		return 0
	}

	runeCount := utf8.RuneCount(contents)
	tokens := int64(runeCount / 4)
	if runeCount%4 != 0 {
		tokens++
	}
	if tokens == 0 {
		return 1
	}

	return tokens
}

func classifyContents(contents []byte) (string, bool) {
	if looksBinary(contents) {
		return "binary", true
	}

	if !utf8.Valid(contents) {
		return "decode-failed", true
	}

	return "", false
}

func looksBinary(contents []byte) bool {
	if len(contents) == 0 {
		return false
	}

	sample := contents
	if len(sample) > 8*1024 {
		sample = sample[:8*1024]
	}

	if bytes.IndexByte(sample, 0) >= 0 {
		return true
	}

	controlBytes := 0
	for _, current := range sample {
		if current < 0x09 || (current > 0x0D && current < 0x20) {
			controlBytes++
		}
	}

	return controlBytes*100/len(sample) >= 10
}

func skippedFromError(displayPath string, size int64, err error) report.Result {
	reason := "unreadable"
	if errors.Is(err, fs.ErrPermission) || os.IsPermission(err) {
		reason = "permission-denied"
	}

	return skippedResult(displayPath, size, reason)
}

func skippedResult(displayPath string, size int64, reason string) report.Result {
	return report.Result{
		Path:   displayPath,
		Bytes:  size,
		Status: report.StatusSkipped,
		Reason: &reason,
	}
}

func normalizeTarget(target string) string {
	if target == "" {
		return "."
	}

	return filepath.ToSlash(filepath.Clean(target))
}

func normalizeRoot(target string, isDir bool) string {
	normalizedTarget := normalizeTarget(target)
	if isDir {
		if filepath.IsAbs(target) {
			return filepath.ToSlash(filepath.Base(normalizedTarget))
		}

		return normalizedTarget
	}

	root := filepath.Dir(normalizedTarget)
	if root == "" || root == "." {
		return "."
	}

	if filepath.IsAbs(target) {
		return filepath.ToSlash(filepath.Base(root))
	}

	return filepath.ToSlash(root)
}

func relativePath(rootAbs string, currentPath string) string {
	relPath, err := filepath.Rel(rootAbs, currentPath)
	if err != nil {
		return filepath.ToSlash(filepath.Base(currentPath))
	}

	return filepath.ToSlash(relPath)
}

type ignoreMatcher struct {
	repoRoot string
	files    []ignoreFile
	loaded   map[string]struct{}
}

type ignoreFile struct {
	baseRel string
	depth   int
	rules   []ignoreRule
}

type ignoreRule struct {
	pattern  string
	negated  bool
	dirOnly  bool
	hasSlash bool
}

func newIgnoreMatcher(scanRootAbs string, isDir bool, enabled bool) (*ignoreMatcher, error) {
	if !enabled {
		return nil, nil
	}

	repoRoot, ok := findRepoRoot(scanRootAbs)
	if !ok {
		return nil, nil
	}

	matcher := &ignoreMatcher{
		repoRoot: repoRoot,
		loaded:   map[string]struct{}{},
	}

	for _, currentDir := range ancestorDirs(repoRoot, scanRootAbs) {
		if err := matcher.loadFile(filepath.Join(currentDir, ".gitignore")); err != nil {
			return nil, err
		}
	}

	if !isDir {
		return matcher, nil
	}

	err := filepath.WalkDir(scanRootAbs, func(currentPath string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			if entry != nil && entry.IsDir() {
				return filepath.SkipDir
			}

			return nil
		}

		if entry.IsDir() && entry.Name() == ".git" {
			return filepath.SkipDir
		}

		if !entry.IsDir() && entry.Name() == ".gitignore" {
			return matcher.loadFile(currentPath)
		}

		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("load .gitignore rules: %w", err)
	}

	sort.SliceStable(matcher.files, func(i int, j int) bool {
		if matcher.files[i].depth != matcher.files[j].depth {
			return matcher.files[i].depth < matcher.files[j].depth
		}

		return matcher.files[i].baseRel < matcher.files[j].baseRel
	})

	return matcher, nil
}

func (matcher *ignoreMatcher) loadFile(ignorePath string) error {
	if _, ok := matcher.loaded[ignorePath]; ok {
		return nil
	}

	matcher.loaded[ignorePath] = struct{}{}

	contents, err := os.ReadFile(ignorePath)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil
		}

		return fmt.Errorf("read %s: %w", ignorePath, err)
	}

	baseAbs := filepath.Dir(ignorePath)
	baseRel, err := filepath.Rel(matcher.repoRoot, baseAbs)
	if err != nil {
		return fmt.Errorf("rel base for %s: %w", ignorePath, err)
	}

	baseRel = filepath.ToSlash(baseRel)
	if baseRel == "." {
		baseRel = ""
	}

	rules := make([]ignoreRule, 0)
	scanner := bufio.NewScanner(strings.NewReader(string(contents)))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		rule := ignoreRule{}
		if strings.HasPrefix(line, "!") {
			rule.negated = true
			line = strings.TrimSpace(strings.TrimPrefix(line, "!"))
			if line == "" {
				continue
			}
		}

		if strings.HasSuffix(line, "/") {
			rule.dirOnly = true
			line = strings.TrimSuffix(line, "/")
		}

		rule.pattern = filepath.ToSlash(strings.TrimPrefix(line, "/"))
		rule.hasSlash = strings.Contains(rule.pattern, "/")
		if rule.pattern == "" {
			continue
		}

		rules = append(rules, rule)
	}
	if err := scanner.Err(); err != nil {
		return fmt.Errorf("scan %s: %w", ignorePath, err)
	}

	matcher.files = append(matcher.files, ignoreFile{
		baseRel: baseRel,
		depth:   strings.Count(baseRel, "/"),
		rules:   rules,
	})

	return nil
}

func (matcher *ignoreMatcher) shouldIgnore(absPath string, isDir bool) bool {
	relPath, err := filepath.Rel(matcher.repoRoot, absPath)
	if err != nil {
		return false
	}

	relPath = filepath.ToSlash(relPath)
	if relPath == "." {
		return false
	}

	ignored := false
	for _, ignoreFile := range matcher.files {
		relativeToBase, ok := ignoreFile.relativeToBase(relPath)
		if !ok {
			continue
		}

		for _, rule := range ignoreFile.rules {
			if !rule.matches(relativeToBase, isDir) {
				continue
			}

			ignored = !rule.negated
		}
	}

	return ignored
}

func (ignoreFile ignoreFile) relativeToBase(relPath string) (string, bool) {
	if ignoreFile.baseRel == "" {
		return relPath, true
	}

	prefix := ignoreFile.baseRel + "/"
	if !strings.HasPrefix(relPath, prefix) {
		return "", false
	}

	return strings.TrimPrefix(relPath, prefix), true
}

func (rule ignoreRule) matches(relPath string, isDir bool) bool {
	if relPath == "" {
		return false
	}

	if rule.dirOnly && !isDir {
		return false
	}

	candidate := relPath
	if !rule.hasSlash {
		candidate = path.Base(relPath)
	}

	matched, err := path.Match(rule.pattern, candidate)
	if err != nil {
		return rule.pattern == candidate
	}

	return matched
}

func findRepoRoot(startAbs string) (string, bool) {
	current := startAbs
	for {
		if _, err := os.Stat(filepath.Join(current, ".git")); err == nil {
			return current, true
		}

		parent := filepath.Dir(current)
		if parent == current {
			return "", false
		}

		current = parent
	}
}

func ancestorDirs(repoRoot string, scanRootAbs string) []string {
	current := scanRootAbs
	ancestors := []string{current}
	for current != repoRoot {
		current = filepath.Dir(current)
		ancestors = append(ancestors, current)
	}

	for left, right := 0, len(ancestors)-1; left < right; left, right = left+1, right-1 {
		ancestors[left], ancestors[right] = ancestors[right], ancestors[left]
	}

	return ancestors
}
