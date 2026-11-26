//go:build windows

package fsutil

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"time"
)

const (
	winErrSharingViolation syscall.Errno = 32
	winErrLockViolation    syscall.Errno = 33
	winErrDirNotEmpty      syscall.Errno = 145
)

// RemoveDirForce thoroughly deletes dir and all its contents on Windows,
// handling paths > 260 chars (\\?\ extended prefix), ReadOnly/Hidden/System
// attributes (.git packfiles, caches), NTFS junctions/reparse points, and
// transient antivirus/indexer/async-delete locks via bounded retries.
func RemoveDirForce(dir string) error {
	trimmed := strings.TrimSpace(dir)
	if trimmed == "" {
		return nil
	}
	abs, err := filepath.Abs(filepath.Clean(trimmed))
	if err != nil {
		return fmt.Errorf("resolve path %s: %w", trimmed, err)
	}
	extRoot := toExtendedWindowsPath(abs)

	// Fast check: already gone.
	if _, err := os.Lstat(extRoot); os.IsNotExist(err) {
		return nil
	}

	var lastErr error
	const maxPasses = 4
	for pass := 0; pass < maxPasses; pass++ {
		if pass > 0 {
			time.Sleep(time.Duration(80*(1<<(pass-1))) * time.Millisecond)
		}
		lastErr = removeTreeWindows(extRoot)
		if _, statErr := os.Lstat(extRoot); os.IsNotExist(statErr) {
			return nil
		}
		if lastErr == nil {
			lastErr = fmt.Errorf("thư mục vẫn còn tồn tại sau khi xóa: %s", abs)
		}
	}

	// Report a concrete leftover path if any remains to aid diagnosis.
	if leftover := findFirstLeftover(extRoot); leftover != "" {
		return fmt.Errorf("không thể xóa triệt để %s (còn tệp/thư mục bị khóa: %s): %w", abs, stripExtendedPrefix(leftover), lastErr)
	}
	return fmt.Errorf("không thể xóa triệt để %s: %w", abs, lastErr)
}

func removeTreeWindows(extPath string) error {
	p16, err := syscall.UTF16PtrFromString(extPath)
	if err != nil {
		return err
	}
	attrs, err := syscall.GetFileAttributes(p16)
	if err != nil {
		if errors.Is(err, syscall.ENOENT) || errors.Is(err, syscall.ERROR_FILE_NOT_FOUND) || errors.Is(err, syscall.ERROR_PATH_NOT_FOUND) {
			return nil
		}
		return err
	}

	isDir := attrs&syscall.FILE_ATTRIBUTE_DIRECTORY != 0
	isReparse := attrs&syscall.FILE_ATTRIBUTE_REPARSE_POINT != 0

	// Clear ReadOnly / Hidden / System flags so deletion is never blocked by attributes.
	if attrs&(syscall.FILE_ATTRIBUTE_READONLY|syscall.FILE_ATTRIBUTE_HIDDEN|syscall.FILE_ATTRIBUTE_SYSTEM) != 0 {
		_ = syscall.SetFileAttributes(p16, syscall.FILE_ATTRIBUTE_NORMAL)
	}

	// Non-directory files or reparse points (symlinks / NTFS junctions):
	// unlink directly WITHOUT recursing into reparse targets.
	if !isDir {
		return removeFileWithRetry(extPath, p16)
	}
	if isReparse {
		return removeDirWithRetry(extPath, p16)
	}

	entries, readErr := os.ReadDir(extPath)
	var firstErr error
	if readErr != nil && !os.IsNotExist(readErr) {
		firstErr = readErr
	}

	// Best-effort sweep through ALL entries instead of aborting on the first error,
	// so transient locks on one child don't prevent cleaning the rest of the tree.
	for _, entry := range entries {
		childPath := extPath + `\` + entry.Name()
		if !entry.IsDir() && entry.Type()&os.ModeSymlink == 0 {
			if err := removeFileFast(childPath); err != nil && firstErr == nil {
				firstErr = err
			}
			continue
		}
		if err := removeTreeWindows(childPath); err != nil && firstErr == nil {
			firstErr = err
		}
	}

	// Re-clear directory attributes in case they were preserved/restored, then remove directory.
	_ = syscall.SetFileAttributes(p16, syscall.FILE_ATTRIBUTE_NORMAL)
	if err := removeDirWithRetry(extPath, p16); err != nil {
		if firstErr == nil {
			firstErr = err
		}
		return firstErr
	}
	return firstErr
}

func removeFileFast(extPath string) error {
	p16, err := syscall.UTF16PtrFromString(extPath)
	if err != nil {
		return err
	}
	return removeFileWithRetry(extPath, p16)
}

func removeFileWithRetry(extPath string, p16 *uint16) error {
	var err error
	for attempt := 0; attempt < 5; attempt++ {
		if attempt > 0 {
			_ = syscall.SetFileAttributes(p16, syscall.FILE_ATTRIBUTE_NORMAL)
			time.Sleep(time.Duration(25*(1<<(attempt-1))) * time.Millisecond)
		}
		err = syscall.DeleteFile(p16)
		if err == nil || errors.Is(err, syscall.ERROR_FILE_NOT_FOUND) || errors.Is(err, syscall.ERROR_PATH_NOT_FOUND) {
			return nil
		}
		if !isRetryableWinDeleteErr(err) {
			break
		}
	}
	return fmt.Errorf("delete file %s: %w", stripExtendedPrefix(extPath), err)
}

func removeDirWithRetry(extPath string, p16 *uint16) error {
	var err error
	for attempt := 0; attempt < 6; attempt++ {
		if attempt > 0 {
			_ = syscall.SetFileAttributes(p16, syscall.FILE_ATTRIBUTE_NORMAL)
			time.Sleep(time.Duration(30*(1<<(attempt-1))) * time.Millisecond)
		}
		err = syscall.RemoveDirectory(p16)
		if err == nil || errors.Is(err, syscall.ERROR_FILE_NOT_FOUND) || errors.Is(err, syscall.ERROR_PATH_NOT_FOUND) {
			return nil
		}
		if !isRetryableWinDeleteErr(err) {
			break
		}
	}
	return fmt.Errorf("remove dir %s: %w", stripExtendedPrefix(extPath), err)
}

func isRetryableWinDeleteErr(err error) bool {
	return errors.Is(err, syscall.ERROR_ACCESS_DENIED) ||
		errors.Is(err, winErrSharingViolation) ||
		errors.Is(err, winErrLockViolation) ||
		errors.Is(err, winErrDirNotEmpty)
}

func toExtendedWindowsPath(abs string) string {
	clean := filepath.Clean(abs)
	if strings.HasPrefix(clean, `\\?\`) {
		return clean
	}
	if strings.HasPrefix(clean, `\\`) {
		// UNC path: \\server\share -> \\?\UNC\server\share
		return `\\?\UNC\` + strings.TrimPrefix(clean, `\\`)
	}
	return `\\?\` + clean
}

func stripExtendedPrefix(p string) string {
	if strings.HasPrefix(p, `\\?\UNC\`) {
		return `\\` + strings.TrimPrefix(p, `\\?\UNC\`)
	}
	return strings.TrimPrefix(p, `\\?\`)
}

func findFirstLeftover(extRoot string) string {
	entries, err := os.ReadDir(extRoot)
	if err != nil || len(entries) == 0 {
		return ""
	}
	first := extRoot + `\` + entries[0].Name()
	if entries[0].IsDir() {
		if deeper := findFirstLeftover(first); deeper != "" {
			return deeper
		}
	}
	return first
}
