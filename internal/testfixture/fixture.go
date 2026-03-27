package testfixture

import (
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
)

func RepoRoot() (string, error) {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		return "", os.ErrNotExist
	}

	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", "..")), nil
}

func CanonicalRepoSourceRoot() (string, error) {
	repoRoot, err := RepoRoot()
	if err != nil {
		return "", err
	}

	return filepath.Join(repoRoot, "internal", "scan", "testdata", "repo"), nil
}

func MaterializeCanonicalRepo(parentDir string) (string, error) {
	sourceRoot, err := CanonicalRepoSourceRoot()
	if err != nil {
		return "", err
	}

	destRoot := filepath.Join(parentDir, "repo")
	if err := copyTree(sourceRoot, destRoot); err != nil {
		return "", err
	}
	if err := os.MkdirAll(filepath.Join(destRoot, ".git"), 0o755); err != nil {
		return "", err
	}

	extras := map[string][]byte{
		filepath.Join(destRoot, "debug.tmp"):             []byte("This file should be ignored when gitignore rules are enabled.\n"),
		filepath.Join(destRoot, "ignored", "secret.txt"): []byte("This ignored file should only appear when --no-gitignore is used.\n"),
		filepath.Join(destRoot, "nested", "local.log"):   []byte("This file is ignored by the nested .gitignore rule.\n"),
	}
	for path, contents := range extras {
		if err := writeFile(path, contents); err != nil {
			return "", err
		}
	}

	return destRoot, nil
}

func copyTree(sourceRoot string, destRoot string) error {
	return filepath.WalkDir(sourceRoot, func(currentPath string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}

		relPath, err := filepath.Rel(sourceRoot, currentPath)
		if err != nil {
			return err
		}

		destPath := filepath.Join(destRoot, relPath)
		if entry.IsDir() {
			return os.MkdirAll(destPath, 0o755)
		}

		contents, err := os.ReadFile(currentPath)
		if err != nil {
			return err
		}

		return os.WriteFile(destPath, contents, 0o600)
	})
}

func writeFile(path string, contents []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}

	return os.WriteFile(path, contents, 0o600)
}
