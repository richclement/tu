package releaseverify

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/richclement/tu/internal/testfixture"
)

type commandResult struct {
	exitCode          int
	stdout            string
	stderr            string
	outputFileContent string
}

type commandCase struct {
	name       string
	args       []string
	cwd        string
	outputFile string
}

func Verify(binaryPath string, version string) error {
	repoRoot, err := testfixture.RepoRoot()
	if err != nil {
		return fmt.Errorf("resolve repo root: %w", err)
	}

	binaryAbs, err := filepath.Abs(binaryPath)
	if err != nil {
		return fmt.Errorf("resolve binary path: %w", err)
	}

	tempDir, err := os.MkdirTemp("", "tu-release-verify-*")
	if err != nil {
		return fmt.Errorf("create temp dir: %w", err)
	}
	defer os.RemoveAll(tempDir)

	referenceBinary := filepath.Join(tempDir, "tu-reference"+filepath.Ext(binaryAbs))
	if err := buildReferenceBinary(repoRoot, referenceBinary, version); err != nil {
		return err
	}

	fixtureParent := filepath.Join(tempDir, "fixture-parent")
	if err := os.MkdirAll(fixtureParent, 0o755); err != nil {
		return fmt.Errorf("create fixture parent: %w", err)
	}

	fixtureRoot, err := testfixture.MaterializeCanonicalRepo(fixtureParent)
	if err != nil {
		return fmt.Errorf("materialize fixture repo: %w", err)
	}
	fixtureCWD := filepath.Dir(fixtureRoot)

	cases := []commandCase{
		{name: "version", args: []string{"--version"}, cwd: repoRoot},
		{name: "human-directory", args: []string{"repo"}, cwd: fixtureCWD},
		{name: "json-directory", args: []string{"repo", "--format", "json", "--quiet"}, cwd: fixtureCWD},
		{name: "plain-directory", args: []string{"repo", "--format", "plain", "--quiet"}, cwd: fixtureCWD},
		{name: "json-file", args: []string{filepath.ToSlash(filepath.Join("repo", "nested", "child.txt")), "--format", "json", "--quiet"}, cwd: fixtureCWD},
		{name: "csv-file", args: []string{"repo", "--format", "csv", "--file", filepath.Join(tempDir, "report.csv"), "--quiet"}, cwd: fixtureCWD, outputFile: filepath.Join(tempDir, "report.csv")},
		{name: "invalid-format", args: []string{"--format", "bogus"}, cwd: fixtureCWD},
		{name: "legacy-flag", args: []string{"--json"}, cwd: fixtureCWD},
		{name: "runtime-failure", args: []string{"missing.txt"}, cwd: fixtureCWD},
	}

	for _, currentCase := range cases {
		expected, err := runCommand(referenceBinary, currentCase.args, currentCase.cwd, currentCase.outputFile)
		if err != nil {
			return fmt.Errorf("run reference binary for %s: %w", currentCase.name, err)
		}

		actual, err := runCommand(binaryAbs, currentCase.args, currentCase.cwd, currentCase.outputFile)
		if err != nil {
			return fmt.Errorf("run release binary for %s: %w", currentCase.name, err)
		}

		if err := compareResults(currentCase.name, expected, actual); err != nil {
			return err
		}
	}

	return nil
}

func buildReferenceBinary(repoRoot string, outputPath string, version string) error {
	args := []string{"build", "-o", outputPath}
	if version != "" {
		args = append(args, "-ldflags", "-X main.version="+version)
	}
	args = append(args, "./cmd/tu")

	cmd := exec.Command("go", args...)
	cmd.Dir = repoRoot

	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("build reference binary: %w: %s", err, stderr.String())
	}

	return nil
}

func runCommand(binary string, args []string, cwd string, outputFile string) (commandResult, error) {
	cmd := exec.Command(binary, args...)
	cmd.Dir = cwd

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	result := commandResult{
		exitCode: 0,
		stdout:   stdout.String(),
		stderr:   stderr.String(),
	}
	if outputFile != "" && outputFile != "-" {
		contents, readErr := os.ReadFile(outputFile)
		if readErr != nil {
			return commandResult{}, fmt.Errorf("read output file %q: %w", outputFile, readErr)
		}
		result.outputFileContent = string(contents)
	}
	if err == nil {
		return result, nil
	}

	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		result.exitCode = exitErr.ExitCode()
		return result, nil
	}

	return commandResult{}, err
}

func compareResults(name string, expected commandResult, actual commandResult) error {
	if expected.exitCode != actual.exitCode {
		return fmt.Errorf("%s exit code mismatch: expected %d, got %d", name, expected.exitCode, actual.exitCode)
	}
	if expected.stdout != actual.stdout {
		return fmt.Errorf("%s stdout mismatch\nexpected:\n%s\nactual:\n%s", name, expected.stdout, actual.stdout)
	}
	if expected.stderr != actual.stderr {
		return fmt.Errorf("%s stderr mismatch\nexpected:\n%s\nactual:\n%s", name, expected.stderr, actual.stderr)
	}
	if expected.outputFileContent != actual.outputFileContent {
		return fmt.Errorf("%s output file mismatch\nexpected:\n%s\nactual:\n%s", name, expected.outputFileContent, actual.outputFileContent)
	}

	return nil
}
