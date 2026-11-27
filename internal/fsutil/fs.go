package fsutil

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

var reSlugInvalid = regexp.MustCompile(`[^a-z0-9-]+`)

// FileExists reports whether path exists and is a regular file.
func FileExists(path string) bool {
	if strings.TrimSpace(path) == "" {
		return false
	}
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

// DirExists reports whether path exists and is a directory.
func DirExists(path string) bool {
	if strings.TrimSpace(path) == "" {
		return false
	}
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}

// Slugify converts arbitrary text into a lowercase URL-safe slug.
func Slugify(input string) string {
	s := strings.ToLower(strings.TrimSpace(input))
	s = strings.ReplaceAll(s, "_", "-")
	s = strings.ReplaceAll(s, " ", "-")
	s = strings.ReplaceAll(s, ".", "-")
	s = reSlugInvalid.ReplaceAllString(s, "")
	s = strings.Trim(s, "-")
	for strings.Contains(s, "--") {
		s = strings.ReplaceAll(s, "--", "-")
	}
	return s
}

// UniqueStrings deduplicates non-empty trimmed strings case-insensitively while preserving order.
func UniqueStrings(items []string) []string {
	seen := make(map[string]struct{}, len(items))
	out := make([]string, 0, len(items))
	for _, item := range items {
		trimmed := strings.TrimSpace(item)
		if trimmed == "" {
			continue
		}
		key := strings.ToLower(trimmed)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, trimmed)
	}
	return out
}

// ResolveSafeSubpath resolves relPath inside baseDir and rejects any path escaping baseDir.
func ResolveSafeSubpath(baseDir, relPath string) (string, error) {
	baseTrimmed := strings.TrimSpace(baseDir)
	if baseTrimmed == "" || baseTrimmed == "." {
		return "", fmt.Errorf("thư mục gốc không hợp lệ")
	}
	baseClean := filepath.Clean(baseTrimmed)
	rel := strings.TrimSpace(relPath)
	if rel == "" {
		return baseClean, nil
	}
	if strings.ContainsRune(rel, '\x00') {
		return "", fmt.Errorf("đường dẫn chứa ký tự không hợp lệ")
	}
	var candidate string
	if filepath.IsAbs(rel) {
		candidate = filepath.Clean(rel)
	} else {
		candidate = filepath.Clean(filepath.Join(baseClean, rel))
	}
	if !strings.EqualFold(filepath.VolumeName(baseClean), filepath.VolumeName(candidate)) {
		return "", fmt.Errorf("đường dẫn vượt ra ngoài ổ đĩa gốc: %s", relPath)
	}
	relCheck, err := filepath.Rel(baseClean, candidate)
	if err != nil || relCheck == ".." || strings.HasPrefix(relCheck, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("đường dẫn vượt ra ngoài thư mục gốc: %s", relPath)
	}
	return candidate, nil
}

