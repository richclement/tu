package scan

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io/fs"
	"math"
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
	maxWorkerCount = 4
)

type Config struct {
	CWD              string
	Target           string
	SymlinkMode      report.SymlinkMode
	MaxDepth         *int
	Threshold        *int64
	MaxFileSizeBytes *int64
	Exclude          []string
	Summarize        bool
	RespectGitIgnore bool
	Sort             string
}

type counterFactory func() *count.Counter

type scanTask struct {
	physicalAbs string
	displayPath string
}

type resolvedTarget struct {
	physicalAbs  string
	isDir        bool
	rootIsDir    bool
	rootLeafPath string
	skipReason   string
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

	symlinkMode := normalizeSymlinkMode(cfg.SymlinkMode)
	resolved, err := resolveTarget(targetAbs, targetArg, symlinkMode)
	if err != nil {
		return report.ScanReport{}, fmt.Errorf("stat target %q: %w", targetArg, err)
	}

	maxDepth := normalizeMaxDepth(cfg.MaxDepth, cfg.Summarize)
	recursive := shouldReportRecursive(resolved.isDir, maxDepth)
	respectGitIgnore := cfg.RespectGitIgnore && resolved.isDir

	scanReport := report.ScanReport{
		SchemaVersion:    report.SchemaVersionV1,
		Target:           normalizeTarget(targetArg),
		Root:             normalizeRoot(targetArg, resolved.rootIsDir),
		Recursive:        recursive,
		RespectGitIgnore: respectGitIgnore,
		SymlinkMode:      symlinkMode,
		Sort:             cfg.Sort,
		Threshold:        cfg.Threshold,
		MaxFileSizeBytes: cfg.MaxFileSizeBytes,
		Exclude:          copyStringSlice(cfg.Exclude),
		Results:          []report.Result{},
	}

	excluder := newExcludeMatcher(cfg.Exclude)
	if excluder != nil && excluder.shouldExclude(normalizeTarget(targetArg)) {
		return scanReport, nil
	}

	matcher, err := newIgnoreMatcher(resolved.physicalAbs, respectGitIgnore)
	if err != nil {
		return report.ScanReport{}, err
	}

	switch {
	case resolved.skipReason != "":
		scanReport.Results = append(scanReport.Results, skippedResult(resolved.rootLeafPath, resolved.skipReason))
	case resolved.isDir:
		switch symlinkMode {
		case report.SymlinkModeLogical:
			scanReport.Results, err = scanLogicalDirectory(
				targetArg,
				resolved.physicalAbs,
				maxDepth,
				matcher,
				excluder,
				cfg.MaxFileSizeBytes,
				count.NewCounter,
			)
		default:
			scanReport.Results, err = scanPhysicalDirectory(
				targetArg,
				resolved.physicalAbs,
				maxDepth,
				matcher,
				excluder,
				cfg.MaxFileSizeBytes,
				count.NewCounter,
			)
		}
		if err != nil {
			return report.ScanReport{}, err
		}
	default:
		scanReport.Results = append(scanReport.Results, scanSingleFile(
			scanTask{physicalAbs: resolved.physicalAbs, displayPath: resolved.rootLeafPath},
			cfg.MaxFileSizeBytes,
			count.NewCounter,
		))
	}

	scanReport.Summary = summarize(scanReport.Results)
	if isSummaryOnly(maxDepth) && resolved.isDir {
		scanReport.Results = []report.Result{summaryResult(scanReport.Root, scanReport.Summary.TotalTokens)}
	}
	if cfg.Threshold != nil {
		resultCountBeforeThreshold := len(scanReport.Results)
		scanReport.Results = filterResultsByThreshold(scanReport.Results, *cfg.Threshold)
		scanReport.ThresholdEmptied = resultCountBeforeThreshold > 0 && len(scanReport.Results) == 0
	}
	sortResults(scanReport.Results, cfg.Sort)

	return scanReport, nil
}

func normalizeSymlinkMode(mode report.SymlinkMode) report.SymlinkMode {
	switch mode {
	case report.SymlinkModeCommandLine, report.SymlinkModeLogical:
		return mode
	default:
		return report.SymlinkModePhysical
	}
}

func resolveTarget(targetAbs string, targetArg string, mode report.SymlinkMode) (resolvedTarget, error) {
	info, err := os.Lstat(targetAbs)
	if err != nil {
		return resolvedTarget{}, err
	}

	target := resolvedTarget{
		physicalAbs:  targetAbs,
		isDir:        info.IsDir(),
		rootIsDir:    info.IsDir(),
		rootLeafPath: targetLeafPath(targetArg),
	}

	if info.Mode()&fs.ModeSymlink == 0 {
		if mode == report.SymlinkModeLogical {
			resolvedAbs, err := filepath.EvalSymlinks(targetAbs)
			if err != nil {
				target.skipReason = classifyFollowError(err)
				target.isDir = false
				return target, nil
			}
			target.physicalAbs = resolvedAbs
		}
		return target, nil
	}

	if mode == report.SymlinkModePhysical {
		target.isDir = false
		if resolvedInfo, err := os.Stat(targetAbs); err == nil {
			target.rootIsDir = resolvedInfo.IsDir()
		} else {
			target.rootIsDir = true
		}
		target.skipReason = "symlink"
		return target, nil
	}

	resolvedAbs, err := filepath.EvalSymlinks(targetAbs)
	if err != nil {
		target.isDir = false
		target.rootIsDir = true
		target.skipReason = classifyFollowError(err)
		return target, nil
	}

	resolvedInfo, err := os.Stat(resolvedAbs)
	if err != nil {
		target.physicalAbs = resolvedAbs
		target.isDir = false
		target.rootIsDir = true
		target.skipReason = classifyFollowError(err)
		return target, nil
	}

	target.physicalAbs = resolvedAbs
	target.isDir = resolvedInfo.IsDir()
	target.rootIsDir = resolvedInfo.IsDir()
	return target, nil
}

func targetLeafPath(target string) string {
	return path.Base(normalizeTarget(target))
}

func filterResultsByThreshold(results []report.Result, threshold int64) []report.Result {
	filtered := make([]report.Result, 0, len(results))
	for _, result := range results {
		if result.Tokens == nil {
			continue
		}
		if matchesThreshold(*result.Tokens, threshold) {
			filtered = append(filtered, result)
		}
	}

	return filtered
}

func matchesThreshold(tokens int64, threshold int64) bool {
	if threshold >= 0 {
		return tokens > threshold
	}
	if threshold == math.MinInt64 {
		return false
	}

	return uint64(tokens) < absThresholdMagnitude(threshold)
}

func absThresholdMagnitude(threshold int64) uint64 {
	return uint64(-(threshold + 1)) + 1
}

func scanPhysicalDirectory(
	walkTarget string,
	targetAbs string,
	maxDepth *int,
	matcher *ignoreMatcher,
	excluder *excludeMatcher,
	maxFileSizeBytes *int64,
	newCounter counterFactory,
) ([]report.Result, error) {
	results, err := runScanWithWorkers(maxFileSizeBytes, newCounter, func(tasks chan<- scanTask, resultCh chan<- report.Result) error {
		return filepath.WalkDir(targetAbs, func(currentPath string, entry fs.DirEntry, walkErr error) error {
			displayPath := relativePath(targetAbs, currentPath)

			if walkErr != nil {
				if currentPath == targetAbs {
					return walkErr
				}

				resultCh <- skippedFromError(displayPath, walkErr)
				if entry != nil && entry.IsDir() {
					return filepath.SkipDir
				}

				return nil
			}

			if currentPath == targetAbs {
				return nil
			}

			if excluder != nil && excluder.shouldExclude(displayPath) {
				if entry.IsDir() {
					return filepath.SkipDir
				}

				return nil
			}

			if entry.Type()&fs.ModeSymlink != 0 {
				resultCh <- skippedResult(displayPath, "symlink")
				return nil
			}

			if entry.IsDir() {
				if entry.Name() == ".git" {
					return filepath.SkipDir
				}

				if matcher != nil {
					if err := matcher.prepareForDir(currentPath); err != nil {
						return err
					}
					if matcher.shouldIgnore(currentPath, true) {
						return filepath.SkipDir
					}
				}

				if shouldSkipDirAtDepth(displayPath, maxDepth) {
					return filepath.SkipDir
				}

				return nil
			}

			if matcher != nil && matcher.shouldIgnore(currentPath, false) {
				return nil
			}
			if shouldSkipFileAtDepth(displayPath, maxDepth) {
				return nil
			}

			tasks <- scanTask{physicalAbs: currentPath, displayPath: displayPath}
			return nil
		})
	})
	if err != nil {
		return nil, fmt.Errorf("walk target %q: %w", normalizeTarget(walkTarget), err)
	}

	return results, nil
}

func scanLogicalDirectory(
	walkTarget string,
	targetAbs string,
	maxDepth *int,
	matcher *ignoreMatcher,
	excluder *excludeMatcher,
	maxFileSizeBytes *int64,
	newCounter counterFactory,
) ([]report.Result, error) {
	rootCanonical, err := filepath.EvalSymlinks(targetAbs)
	if err != nil {
		return nil, fmt.Errorf("walk target %q: %w", normalizeTarget(walkTarget), err)
	}

	results, err := runScanWithWorkers(maxFileSizeBytes, newCounter, func(tasks chan<- scanTask, resultCh chan<- report.Result) error {
		return walkLogicalDirectory(rootCanonical, "", []string{rootCanonical}, maxDepth, matcher, excluder, tasks, resultCh)
	})
	if err != nil {
		return nil, fmt.Errorf("walk target %q: %w", normalizeTarget(walkTarget), err)
	}

	return results, nil
}

func runScanWithWorkers(
	maxFileSizeBytes *int64,
	newCounter counterFactory,
	producer func(chan<- scanTask, chan<- report.Result) error,
) ([]report.Result, error) {
	results := make([]report.Result, 0)
	resultCh := make(chan report.Result, defaultWorkerCount()*2)
	tasks := make(chan scanTask, defaultWorkerCount()*2)

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
			for task := range tasks {
				resultCh <- scanSingleFile(task, maxFileSizeBytes, func() *count.Counter {
					return workerCounter
				})
			}
		}()
	}

	err := producer(tasks, resultCh)
	close(tasks)
	workerWG.Wait()
	close(resultCh)
	collectorWG.Wait()

	return results, err
}

func walkLogicalDirectory(
	currentDirAbs string,
	currentDisplay string,
	branchDirs []string,
	maxDepth *int,
	matcher *ignoreMatcher,
	excluder *excludeMatcher,
	tasks chan<- scanTask,
	resultCh chan<- report.Result,
) error {
	entries, err := os.ReadDir(currentDirAbs)
	if err != nil {
		if currentDisplay == "" {
			return err
		}

		resultCh <- skippedFromError(currentDisplay, err)
		return nil
	}

	for _, entry := range entries {
		displayPath := joinDisplayPath(currentDisplay, entry.Name())
		if excluder != nil && excluder.shouldExclude(displayPath) {
			continue
		}

		physicalPath := filepath.Join(currentDirAbs, entry.Name())
		isSymlink := entry.Type()&fs.ModeSymlink != 0
		if entry.Name() == ".git" && (entry.IsDir() || isSymlink) {
			continue
		}

		if isSymlink {
			resolvedPath, err := filepath.EvalSymlinks(physicalPath)
			if err != nil {
				if matcher != nil && matcher.shouldIgnoreRelative(displayPath, false) {
					continue
				}
				resultCh <- skippedResult(displayPath, "broken-symlink")
				continue
			}

			resolvedInfo, err := os.Stat(resolvedPath)
			if err != nil {
				if matcher != nil && matcher.shouldIgnoreRelative(displayPath, false) {
					continue
				}
				resultCh <- skippedResult(displayPath, classifyFollowError(err))
				continue
			}
			if matcher != nil && matcher.shouldIgnoreRelative(displayPath, resolvedInfo.IsDir()) {
				continue
			}

			if resolvedInfo.IsDir() {
				if shouldSkipDirAtDepth(displayPath, maxDepth) {
					continue
				}
				if containsPath(branchDirs, resolvedPath) {
					resultCh <- skippedResult(displayPath, "symlink-cycle")
					continue
				}
				if matcher != nil {
					if err := matcher.prepareForDir(resolvedPath); err != nil {
						return err
					}
					if matcher.shouldIgnore(resolvedPath, true) {
						continue
					}
				}

				if err := walkLogicalDirectory(resolvedPath, displayPath, append(branchDirs, resolvedPath), maxDepth, matcher, excluder, tasks, resultCh); err != nil {
					return err
				}
				continue
			}

			if matcher != nil && matcher.shouldIgnore(resolvedPath, false) {
				continue
			}
			if shouldSkipFileAtDepth(displayPath, maxDepth) {
				continue
			}

			tasks <- scanTask{physicalAbs: resolvedPath, displayPath: displayPath}
			continue
		}

		if entry.IsDir() {
			if shouldSkipDirAtDepth(displayPath, maxDepth) {
				continue
			}
			if matcher != nil {
				if err := matcher.prepareForDir(physicalPath); err != nil {
					return err
				}
				if matcher.shouldIgnore(physicalPath, true) {
					continue
				}
			}

			if err := walkLogicalDirectory(physicalPath, displayPath, append(branchDirs, physicalPath), maxDepth, matcher, excluder, tasks, resultCh); err != nil {
				return err
			}
			continue
		}

		if matcher != nil && matcher.shouldIgnore(physicalPath, false) {
			continue
		}
		if shouldSkipFileAtDepth(displayPath, maxDepth) {
			continue
		}

		tasks <- scanTask{physicalAbs: physicalPath, displayPath: displayPath}
	}

	return nil
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

func scanSingleFile(task scanTask, maxFileSizeBytes *int64, newCounter counterFactory) report.Result {
	return scanTaskFile(task, maxFileSizeBytes, newCounter())
}

func scanTaskFile(task scanTask, maxFileSizeBytes *int64, counter *count.Counter) report.Result {
	info, err := os.Lstat(task.physicalAbs)
	if err != nil {
		return skippedFromError(task.displayPath, err)
	}

	if info.Mode()&fs.ModeSymlink != 0 {
		return skippedResult(task.displayPath, "symlink")
	}
	if !info.Mode().IsRegular() {
		return skippedResult(task.displayPath, "unreadable")
	}

	if maxFileSizeBytes != nil && info.Size() > *maxFileSizeBytes {
		return skippedResult(task.displayPath, "too-large")
	}

	contents, err := os.ReadFile(task.physicalAbs)
	if err != nil {
		return skippedFromError(task.displayPath, err)
	}

	if reason, ok := classifyContents(contents); ok {
		return skippedResult(task.displayPath, reason)
	}

	counted := counter.CountText(string(contents))
	tokens := counted.Tokens
	method := counted.Method
	provider := counted.Provider

	return report.Result{
		Kind:     report.ResultKindFile,
		Path:     task.displayPath,
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
			if result.Method != nil && *result.Method == report.MethodHeuristic {
				summary.HeuristicResults++
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

func classifyFollowError(err error) string {
	if errors.Is(err, fs.ErrPermission) || os.IsPermission(err) {
		return "permission-denied"
	}
	if isSymlinkCycleError(err) {
		return "symlink-cycle"
	}

	return "broken-symlink"
}

func isSymlinkCycleError(err error) bool {
	return strings.Contains(strings.ToLower(err.Error()), "too many links")
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
		Kind:   report.ResultKindFile,
		Path:   displayPath,
		Status: report.StatusSkipped,
		Reason: &reason,
	}
}

func normalizeMaxDepth(maxDepth *int, summarize bool) *int {
	if summarize && maxDepth == nil {
		depth := 0
		return &depth
	}

	return maxDepth
}

func shouldReportRecursive(isDir bool, maxDepth *int) bool {
	if !isDir {
		return false
	}
	if maxDepth == nil {
		return true
	}
	if *maxDepth == 0 {
		return true
	}

	return *maxDepth > 1
}

func isSummaryOnly(maxDepth *int) bool {
	return maxDepth != nil && *maxDepth == 0
}

func shouldSkipDirAtDepth(displayPath string, maxDepth *int) bool {
	if maxDepth == nil || *maxDepth == 0 {
		return false
	}

	return relativeDepth(displayPath) >= *maxDepth
}

func shouldSkipFileAtDepth(displayPath string, maxDepth *int) bool {
	if maxDepth == nil || *maxDepth == 0 {
		return false
	}

	return relativeDepth(displayPath) > *maxDepth
}

func relativeDepth(relPath string) int {
	if relPath == "" || relPath == "." {
		return 0
	}

	return strings.Count(relPath, "/") + 1
}

func summaryResult(path string, totalTokens int64) report.Result {
	return report.Result{
		Kind:   report.ResultKindSummary,
		Path:   path,
		Tokens: &totalTokens,
		Status: report.StatusCounted,
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

func joinDisplayPath(parent string, name string) string {
	if parent == "" {
		return name
	}

	return parent + "/" + name
}

func containsPath(paths []string, target string) bool {
	for _, current := range paths {
		if current == target {
			return true
		}
	}

	return false
}

func copyStringSlice(values []string) []string {
	if len(values) == 0 {
		return nil
	}

	return append([]string(nil), values...)
}

type excludeMatcher struct {
	patterns []string
}

func newExcludeMatcher(patterns []string) *excludeMatcher {
	if len(patterns) == 0 {
		return nil
	}

	return &excludeMatcher{patterns: copyStringSlice(patterns)}
}

func (matcher *excludeMatcher) shouldExclude(pathLike string) bool {
	name := path.Base(filepath.ToSlash(pathLike))
	if name == "." || name == "" {
		return false
	}

	for _, pattern := range matcher.patterns {
		matched, _ := path.Match(pattern, name)
		if matched {
			return true
		}
	}

	return false
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

	if err := matcher.prepareForDir(scanRootAbs); err != nil {
		return nil, err
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
	if matcher == nil || !matcher.containsPath(dirPath) {
		return nil
	}

	for _, currentDir := range ancestorDirs(matcher.repoRoot, filepath.Clean(dirPath)) {
		if err := matcher.loadFile(filepath.Join(currentDir, ".gitignore")); err != nil {
			return err
		}
	}

	sort.SliceStable(matcher.files, func(i int, j int) bool {
		if matcher.files[i].depth != matcher.files[j].depth {
			return matcher.files[i].depth < matcher.files[j].depth
		}

		return matcher.files[i].baseRel < matcher.files[j].baseRel
	})

	return nil
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
	if matcher == nil || !matcher.containsPath(absPath) {
		return false
	}

	relPath, err := filepath.Rel(matcher.repoRoot, absPath)
	if err != nil {
		return false
	}

	return matcher.shouldIgnoreRelative(relPath, isDir)
}

func (matcher *ignoreMatcher) shouldIgnoreRelative(relPath string, isDir bool) bool {
	if matcher == nil {
		return false
	}

	relPath = filepath.ToSlash(filepath.Clean(relPath))
	if relPath == "." || relPath == "" {
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

func (matcher *ignoreMatcher) containsPath(absPath string) bool {
	relPath, err := filepath.Rel(matcher.repoRoot, absPath)
	if err != nil {
		return false
	}

	return relPath == "." || (relPath != ".." && !strings.HasPrefix(relPath, ".."+string(filepath.Separator)))
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
	current := filepath.Clean(startAbs)
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
	current := filepath.Clean(scanRootAbs)
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
