// Package projectfs provides project-root-contained filesystem operations.
package projectfs

import (
	"errors"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"
)

// Root is the resolved physical boundary for all project filesystem access.
type Root struct {
	path string
}

// Open resolves a project root, including symlinks in its existing ancestors.
func Open(rootPath string) (Root, error) {
	if strings.TrimSpace(rootPath) == "" {
		return Root{}, errors.New("project root is required")
	}
	absolute, err := filepath.Abs(rootPath)
	if err != nil {
		return Root{}, fmt.Errorf("resolve absolute project root: %w", err)
	}
	resolved, err := resolveWithMissingTail(filepath.Clean(absolute))
	if err != nil {
		return Root{}, fmt.Errorf("resolve project root: %w", err)
	}
	return Root{path: resolved}, nil
}

// Path returns the physical absolute root path.
func (root Root) Path() string {
	return root.path
}

// Resolve validates a portable relative path and returns its physical target.
func (root Root) Resolve(relativePath string) (string, error) {
	if strings.TrimSpace(relativePath) == "" {
		return "", errors.New("relative path is required")
	}
	if strings.ContainsRune(relativePath, '\x00') {
		return "", errors.New("relative path contains a null byte")
	}
	if strings.Contains(relativePath, "\\") {
		return "", errors.New("relative path must use forward slashes")
	}
	cleaned := path.Clean(relativePath)
	if cleaned == "." || path.IsAbs(cleaned) || strings.HasPrefix(cleaned, "../") {
		return "", fmt.Errorf("path %q is not contained by the project root", relativePath)
	}
	if cleaned != relativePath {
		return "", fmt.Errorf("path %q is not in canonical portable form", relativePath)
	}
	if filepath.VolumeName(cleaned) != "" || strings.Contains(strings.Split(cleaned, "/")[0], ":") {
		return "", fmt.Errorf("path %q contains a filesystem volume", relativePath)
	}
	target := filepath.Join(root.path, filepath.FromSlash(cleaned))
	resolved, err := resolveWithMissingTail(target)
	if err != nil {
		return "", fmt.Errorf("resolve path %q: %w", relativePath, err)
	}
	if !contains(root.path, resolved) {
		return "", fmt.Errorf("path %q escapes the project root through a symlink", relativePath)
	}
	return resolved, nil
}

func contains(rootPath, targetPath string) bool {
	relative, err := filepath.Rel(rootPath, targetPath)
	if err != nil {
		return false
	}
	return relative == "." || (relative != ".." && !strings.HasPrefix(relative, ".."+string(filepath.Separator)))
}

func resolveWithMissingTail(targetPath string) (string, error) {
	current := targetPath
	var missing []string
	for {
		_, err := os.Lstat(current)
		if err == nil {
			resolved, resolveErr := filepath.EvalSymlinks(current)
			if resolveErr != nil {
				return "", resolveErr
			}
			for index := len(missing) - 1; index >= 0; index-- {
				resolved = filepath.Join(resolved, missing[index])
			}
			return filepath.Clean(resolved), nil
		}
		if !errors.Is(err, os.ErrNotExist) {
			return "", err
		}
		parent := filepath.Dir(current)
		if parent == current {
			return "", fmt.Errorf("no existing ancestor for %q", targetPath)
		}
		missing = append(missing, filepath.Base(current))
		current = parent
	}
}
