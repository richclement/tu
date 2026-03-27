package releaseverify

import (
	"bytes"
	"os/exec"
	"path/filepath"
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
