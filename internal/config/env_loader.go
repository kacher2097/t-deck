package config

import (
	"bufio"
	"errors"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// CandidateDotEnvPaths returns .env file locations ordered from highest to lowest priority.
func CandidateDotEnvPaths(appDataDir string) []string {
	var paths []string
	if exe, err := os.Executable(); err == nil {
		paths = append(paths, filepath.Join(filepath.Dir(exe), EnvFile))
	}
	if wd, err := os.Getwd(); err == nil {
		paths = append(paths, filepath.Join(wd, EnvFile))
	}
	if strings.TrimSpace(appDataDir) != "" {
		paths = append(paths, filepath.Join(appDataDir, EnvFile))
	}
	return uniquePaths(paths)
}

// ResolveWritableDotEnvPath picks an existing .env file or falls back to appDataDir/.env.
func ResolveWritableDotEnvPath(appDataDir string) string {
	for _, p := range CandidateDotEnvPaths(appDataDir) {
		if info, err := os.Stat(p); err == nil && !info.IsDir() {
			return p
		}
	}
	if strings.TrimSpace(appDataDir) != "" {
		return filepath.Join(appDataDir, EnvFile)
	}
	return EnvFile
}

// LoadDotEnvFiles loads environment variables from the given .env paths.
// Earlier files take precedence; already-set OS environment variables are preserved.
func LoadDotEnvFiles(paths ...string) error {
	for _, path := range paths {
		if strings.TrimSpace(path) == "" {
			continue
		}
		if err := loadSingleDotEnv(path); err != nil {
			return err
		}
	}
	return nil
}

func loadSingleDotEnv(path string) error {
	f, err := os.Open(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return err
	}
	defer f.Close()

	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		line = strings.TrimPrefix(line, "export ")
		key, val, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		key = strings.TrimSpace(key)
		val = stripQuotes(strings.TrimSpace(val))
		if key == "" {
			continue
		}
		if existing, present := os.LookupEnv(key); !present || strings.TrimSpace(existing) == "" {
			_ = os.Setenv(key, val)
		}
	}
	return sc.Err()
}

// UpsertDotEnv updates or appends key=value pairs in envPath and sets them in os.Environ.
func UpsertDotEnv(envPath string, updates map[string]string) error {
	if err := os.MkdirAll(filepath.Dir(envPath), 0o755); err != nil {
		return err
	}

	var existingLines []string
	if raw, err := os.ReadFile(envPath); err == nil {
		normalized := strings.ReplaceAll(string(raw), "\r\n", "\n")
		existingLines = strings.Split(strings.TrimRight(normalized, "\n"), "\n")
	}

	applied := make(map[string]bool, len(updates))
	outLines := make([]string, 0, len(existingLines)+len(updates))
	for _, line := range existingLines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			outLines = append(outLines, line)
			continue
		}
		clean := strings.TrimPrefix(trimmed, "export ")
		k, _, ok := strings.Cut(clean, "=")
		k = strings.TrimSpace(k)
		if ok {
			if newVal, found := updates[k]; found {
				outLines = append(outLines, k+"="+formatEnvValue(newVal))
				applied[k] = true
				continue
			}
		}
		outLines = append(outLines, line)
	}

	var missingKeys []string
	for k := range updates {
		if !applied[k] {
			missingKeys = append(missingKeys, k)
		}
	}
	sort.Strings(missingKeys)
	for _, k := range missingKeys {
		outLines = append(outLines, k+"="+formatEnvValue(updates[k]))
	}

	for k, v := range updates {
		_ = os.Setenv(k, v)
	}

	content := strings.Join(outLines, "\n") + "\n"
	return os.WriteFile(envPath, []byte(content), 0o600)
}

func formatEnvValue(val string) string {
	if strings.ContainsAny(val, " \t#") {
		escaped := strings.ReplaceAll(val, `"`, `\"`)
		return `"` + escaped + `"`
	}
	return val
}

func stripQuotes(val string) string {
	if len(val) >= 2 {
		if (val[0] == '"' && val[len(val)-1] == '"') ||
			(val[0] == '\'' && val[len(val)-1] == '\'') {
			return val[1 : len(val)-1]
		}
	}
	return val
}

func uniquePaths(paths []string) []string {
	seen := make(map[string]struct{}, len(paths))
	out := make([]string, 0, len(paths))
	for _, p := range paths {
		cleaned := filepath.Clean(p)
		key := strings.ToLower(cleaned)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, cleaned)
	}
	return out
}

