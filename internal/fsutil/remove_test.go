package fsutil

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestRemoveDirForceHandlesReadOnlyAndLongPaths(t *testing.T) {
	base := t.TempDir()
	projRoot := filepath.Join(base, "ai-company-studio-office-integration")

	// 1. Read-only .git packfile (classic Windows os.RemoveAll failure)
	gitPackDir := filepath.Join(projRoot, ".git", "objects", "pack")
	if err := os.MkdirAll(gitPackDir, 0o755); err != nil {
		t.Fatal(err)
	}
	packFile := filepath.Join(gitPackDir, "pack-deadbeef.idx")
	if err := os.WriteFile(packFile, []byte("git-pack-data"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(packFile, 0o444); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(gitPackDir, 0o555); err != nil {
		t.Fatal(err)
	}

	// 2. Deep >280-char .pnpm path exceeding Windows MAX_PATH (260 chars)
	deepRel := filepath.Join(
		"data",
		"omniroute-runtime",
		"node_modules",
		"fumadocs-ui",
		"dist",
		"node_modules",
		".pnpm",
		"react-medium-image-zoom@5.4.9_react-dom@19.3.0_react@19.3.0__react@19.3.0",
		"node_modules",
		"react-medium-image-zoom",
		"dist",
		"utils",
	)
	deepDir := filepath.Join(projRoot, deepRel)
	deepFile := filepath.Join(deepDir, "compute-positioned-style.js")
	if len(deepFile) <= 260 {
		// Pad path to ensure >270 chars even on short temp roots.
		deepDir = filepath.Join(deepDir, strings.Repeat("nested-segment-", 6))
		deepFile = filepath.Join(deepDir, "compute-positioned-style.js")
	}

	createTargetDir := deepDir
	createTargetFile := deepFile
	if runtime.GOOS == "windows" {
		createTargetDir = `\\?\` + deepDir
		createTargetFile = `\\?\` + deepFile
	}
	if err := os.MkdirAll(createTargetDir, 0o755); err != nil {
		t.Fatalf("mkdir deep path (len=%d): %v", len(deepFile), err)
	}
	if err := os.WriteFile(createTargetFile, []byte("export const x = 1;\n"), 0o644); err != nil {
		t.Fatalf("write deep file (len=%d): %v", len(deepFile), err)
	}

	// 3. Root marker file
	if err := os.WriteFile(filepath.Join(projRoot, ".tdeck-meta.json"), []byte(`{"project_id":"test"}`), 0o444); err != nil {
		t.Fatal(err)
	}

	if err := RemoveDirForce(projRoot); err != nil {
		t.Fatalf("RemoveDirForce failed: %v", err)
	}
	if _, err := os.Lstat(projRoot); !os.IsNotExist(err) {
		t.Fatalf("expected %s to be completely removed, lstat err=%v", projRoot, err)
	}
}

func TestRemoveDirForceDetectsLockedFileAndCleansAfterRelease(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("mandatory file lock test applies to Windows")
	}
	projRoot := filepath.Join(t.TempDir(), "locked-proj")
	if err := os.MkdirAll(projRoot, 0o755); err != nil {
		t.Fatal(err)
	}
	lockedPath := filepath.Join(projRoot, "held.lock")
	f, err := os.OpenFile(lockedPath, os.O_CREATE|os.O_RDWR, 0o644)
	if err != nil {
		t.Fatal(err)
	}

	if err := RemoveDirForce(projRoot); err == nil {
		_ = f.Close()
		t.Fatalf("expected error while file handle is held open")
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}
	if err := RemoveDirForce(projRoot); err != nil {
		t.Fatalf("RemoveDirForce after releasing lock failed: %v", err)
	}
	if DirExists(projRoot) {
		t.Fatalf("expected %s to be gone after second RemoveDirForce", projRoot)
	}
}

