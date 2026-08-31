package projectfs

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestResolveContainsPortablePath(t *testing.T) {
	t.Parallel()

	root, err := Open(t.TempDir())
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	target, err := root.Resolve("internal/example/file.go")
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}
	if !strings.HasPrefix(target, root.Path()+string(filepath.Separator)) {
		t.Fatalf("Resolve() target = %q, want path under %q", target, root.Path())
	}
}

func TestResolveRejectsTraversalAndWindowsSeparators(t *testing.T) {
	t.Parallel()

	root, err := Open(t.TempDir())
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	for _, candidate := range []string{"../outside", "/absolute", `nested\windows`, "C:/volume", "./file", "nested/../file", "nested//file", "nested/"} {
		if _, err := root.Resolve(candidate); err == nil {
			t.Errorf("Resolve(%q) error = nil, want containment error", candidate)
		}
	}
}

func TestResolveRejectsEscapingSymlink(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("creating symlinks may require elevated Windows privileges")
	}
	t.Parallel()

	rootPath := t.TempDir()
	outside := t.TempDir()
	if err := os.Symlink(outside, filepath.Join(rootPath, "escape")); err != nil {
		t.Fatalf("Symlink() error = %v", err)
	}
	root, err := Open(rootPath)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	if _, err := root.Resolve("escape/file.txt"); err == nil {
		t.Fatal("Resolve() error = nil, want symlink escape error")
	}
}

func TestOpenSupportsMissingProjectDirectory(t *testing.T) {
	t.Parallel()

	rootPath := filepath.Join(t.TempDir(), "new", "project")
	root, err := Open(rootPath)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	if root.Path() != filepath.Clean(rootPath) {
		t.Fatalf("Open() path = %q, want %q", root.Path(), filepath.Clean(rootPath))
	}
}
