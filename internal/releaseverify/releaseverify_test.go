package releaseverify

import (
	"bytes"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/richclement/tu/internal/testfixture"
)

func TestVerifyBuiltBinary(t *testing.T) {
	repoRoot, err := testfixture.RepoRoot()
	if err != nil {
		t.Fatalf("resolve repo root: %v", err)
	}

	binaryPath := filepath.Join(t.TempDir(), "tu-test-binary")
	cmd := exec.Command("go", "build", "-ldflags", "-X main.version=v0.0.0-test", "-o", binaryPath, "./cmd/tu")
	cmd.Dir = repoRoot

	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		t.Fatalf("build test binary: %v: %s", err, stderr.String())
	}

	if err := Verify(binaryPath, "v0.0.0-test"); err != nil {
		t.Fatalf("verify built binary: %v", err)
	}
}

func TestCompareResultsMismatch(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		wantErr string
		left    commandResult
		right   commandResult
	}{
		{
			name:    "exit code mismatch",
			wantErr: "exit code mismatch",
			left:    commandResult{exitCode: 0, stdout: "ok", stderr: ""},
			right:   commandResult{exitCode: 1, stdout: "ok", stderr: ""},
		},
		{
			name:    "stdout mismatch",
			wantErr: "stdout mismatch",
			left:    commandResult{exitCode: 0, stdout: "expected", stderr: ""},
			right:   commandResult{exitCode: 0, stdout: "actual", stderr: ""},
		},
		{
			name:    "stderr mismatch",
			wantErr: "stderr mismatch",
			left:    commandResult{exitCode: 0, stdout: "ok", stderr: "expected"},
			right:   commandResult{exitCode: 0, stdout: "ok", stderr: "actual"},
		},
	}

	for _, current := range tests {
		t.Run(current.name, func(t *testing.T) {
			err := compareResults("test-case", current.left, current.right)
			if err == nil {
				t.Fatal("expected compareResults to report a mismatch")
			}
			if !strings.Contains(err.Error(), current.wantErr) {
				t.Fatalf("expected error to contain %q, got %q", current.wantErr, err.Error())
			}
		})
	}
}

func TestCommandArgsForBinaryUsesDistinctOutputFiles(t *testing.T) {
	t.Parallel()

	currentCase := commandCase{
		name:       "csv-file",
		args:       []string{"repo", "--format", "csv", "--file", filepath.Join("/tmp", "report.csv"), "--quiet"},
		outputFile: filepath.Join("/tmp", "report.csv"),
	}

	expectedArgs, expectedOutput := commandArgsForBinary(currentCase, "reference")
	actualArgs, actualOutput := commandArgsForBinary(currentCase, "release")

	if expectedOutput == actualOutput {
		t.Fatalf("expected distinct output files, got %q", expectedOutput)
	}
	if expectedArgs[4] != expectedOutput {
		t.Fatalf("expected reference args to use %q, got %q", expectedOutput, expectedArgs[4])
	}
	if actualArgs[4] != actualOutput {
		t.Fatalf("expected release args to use %q, got %q", actualOutput, actualArgs[4])
	}
}
