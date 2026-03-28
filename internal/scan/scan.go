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
	"runtime"
	"sort"
	"strings"
	"sync"
	"unicode/utf8"

	"github.com/richclement/tu/internal/count"
	"github.com/richclement/tu/internal/report"
)

const (
	largeFileThresholdBytes int64 = 1 << 20
	maxWorkerCount                = 4
)

type Config struct {
	CWD              string
	Target           string
	Recursive        bool
	RespectGitIgnore bool
	Sort             string
}

type counterFactory func() *count.Counter

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

	matcher, err := newIgnoreMatcher(rootAbs, respectGitIgnore)
	if err != nil {
		return report.ScanReport{}, err
	}

	if info.IsDir() {
		scanReport.Results, err = scanDirectory(targetAbs, rootAbs, recursive, matcher, count.NewCounter)
		if err != nil {
			return report.ScanReport{}, err
		}
	} else {
		result := scanSingleFile(targetAbs, rootAbs, count.NewCounter)
		scanReport.Results = append(scanReport.Results, result)
	}

	sortResults(scanReport.Results, cfg.Sort)
	scanReport.Summary = summarize(scanReport.Results)

	return scanReport, nil
}

func scanDirectory(targetAbs string, rootAbs string, recursive bool, matcher *ignoreMatcher, newCounter counterFactory) ([]report.Result, error) {
	results := make([]report.Result, 0)
	resultCh := make(chan report.Result, defaultWorkerCount()*2)
	tasks := make(chan string, defaultWorkerCount()*2)

	var resultsMu sync.Mutex
	var collectorWG sync.WaitGroup
	collectorWG.Add(1)
	go func() {
		defer collectorWG.Done()
		for result := range resultCh {
			resultsMu.Lock()
			results = append(results, result)
			resultsMu.Unlock()
		}
	}()

	var workerWG sync.WaitGroup
	for i := 0; i < defaultWorkerCount(); i++ {
		workerWG.Add(1)
		go func() {
			defer workerWG.Done()

			workerCounter := newCounter()

			for taskPath := range tasks {
				resultCh <- scanFile(taskPath, rootAbs, workerCounter)
			}
		}()
	}

	err := filepath.WalkDir(targetAbs, func(currentPath string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			if currentPath == targetAbs {
				return walkErr
			}

			resultCh <- skippedFromError(relativePath(rootAbs, currentPath), walkErr)
			if entry != nil && entry.IsDir() {
				return filepath.SkipDir
			}

			return nil
		}

		if entry.IsDir() && matcher != nil {
			if err := matcher.prepareForDir(currentPath); err != nil {
				return err
			}
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

		tasks <- currentPath
		return nil
	})
	close(tasks)
	workerWG.Wait()
	close(resultCh)
	collectorWG.Wait()

	if err != nil {
		return nil, fmt.Errorf("walk target %q: %w", normalizeTarget(targetAbs), err)
	}

	return results, nil
}

func scanSingleFile(absPath string, rootAbs string, newCounter counterFactory) report.Result {
	return scanFile(absPath, rootAbs, newCounter())
}

func defaultWorkerCount() int {
	workers := runtime.GOMAXPROCS(0)
	if workers < 1 {
		return 1
	}
	if workers > maxWorkerCount {
		return maxWorkerCount
	}

	return workers
}

func scanFile(absPath string, rootAbs string, counter *count.Counter) report.Result {
	displayPath := relativePath(rootAbs, absPath)

	info, err := os.Stat(absPath)
	if err != nil {
		return skippedFromError(displayPath, err)
	}

	if !info.Mode().IsRegular() {
		return skippedResult(displayPath, "unreadable")
	}

	if info.Size() > largeFileThresholdBytes {
		return skippedResult(displayPath, "too-large")
	}

	contents, err := os.ReadFile(absPath)
	if err != nil {
		return skippedFromError(displayPath, err)
	}

	if reason, ok := classifyContents(contents); ok {
		return skippedResult(displayPath, reason)
	}

	counted := counter.CountText(string(contents))
	tokens := counted.Tokens
	method := counted.Method
	provider := counted.Provider

	return report.Result{
		Path:     displayPath,
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

func skippedFromError(displayPath string, err error) report.Result {
	reason := "unreadable"
	if errors.Is(err, fs.ErrPermission) || os.IsPermission(err) {
		reason = "permission-denied"
	}

	return skippedResult(displayPath, reason)
}

func skippedResult(displayPath string, reason string) report.Result {
	return report.Result{
		Path:   displayPath,
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

func newIgnoreMatcher(scanRootAbs string, enabled bool) (*ignoreMatcher, error) {
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

	sort.SliceStable(matcher.files, func(i int, j int) bool {
		if matcher.files[i].depth != matcher.files[j].depth {
			return matcher.files[i].depth < matcher.files[j].depth
		}

		return matcher.files[i].baseRel < matcher.files[j].baseRel
	})

	return matcher, nil
}

func (matcher *ignoreMatcher) prepareForDir(dirPath string) error {
	return matcher.loadFile(filepath.Join(dirPath, ".gitignore"))
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
