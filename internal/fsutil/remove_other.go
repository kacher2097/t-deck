//go:build !windows

package fsutil

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// RemoveDirForce thoroughly deletes dir and all its contents, relaxing read-only permissions if needed.
func RemoveDirForce(dir string) error {
	trimmed := strings.TrimSpace(dir)
	if trimmed == "" {
		return nil
	}
	clean := filepath.Clean(trimmed)
	if _, err := os.Lstat(clean); os.IsNotExist(err) {
		return nil
	}
	if err := os.RemoveAll(clean); err == nil {
		return nil
	}
	_ = filepath.WalkDir(clean, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			_ = os.Chmod(path, 0o700)
		} else {
			_ = os.Chmod(path, 0o600)
		}
		return nil
	})
	if err := os.RemoveAll(clean); err != nil {
		return fmt.Errorf("không thể xóa triệt để %s: %w", clean, err)
	}
	return nil
}

