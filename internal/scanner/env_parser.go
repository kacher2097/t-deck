package scanner

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"t-deck/internal/config"
	"t-deck/internal/model"
)

// GenerateEnvExample reads .env in projectDir, strips all secret values after '=',
// preserves comments and blank lines, and writes .env.example.
func GenerateEnvExample(projectID, projectDir string) (model.EnvSyncResult, error) {
	if strings.TrimSpace(projectDir) == "" {
		return model.EnvSyncResult{}, fmt.Errorf("project directory is empty")
	}

	envPath := filepath.Join(projectDir, config.EnvFile)
	examplePath := filepath.Join(projectDir, config.EnvExampleFile)

	raw, err := os.ReadFile(envPath)
	if err != nil {
		return model.EnvSyncResult{}, fmt.Errorf("read %s: %w", config.EnvFile, err)
	}

	sanitized, varsCount := SanitizeEnvContent(string(raw))
	if err := os.WriteFile(examplePath, []byte(sanitized), 0o644); err != nil {
		return model.EnvSyncResult{}, fmt.Errorf("write %s: %w", config.EnvExampleFile, err)
	}

	return model.EnvSyncResult{
		ProjectID:   projectID,
		EnvPath:     envPath,
		ExamplePath: examplePath,
		VarsCount:   varsCount,
	}, nil
}

// SanitizeEnvContent strips secret values after '=' while keeping keys, comments, and blank lines.
func SanitizeEnvContent(content string) (string, int) {
	normalized := strings.ReplaceAll(content, "\r\n", "\n")
	lines := strings.Split(normalized, "\n")
	out := make([]string, 0, len(lines))
	varsCount := 0

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			out = append(out, line)
			continue
		}

		if keyPart, _, ok := strings.Cut(line, "="); ok {
			out = append(out, strings.TrimRight(keyPart, " \t")+"=")
			varsCount++
		} else {
			out = append(out, line)
		}
	}

	return strings.Join(out, "\n"), varsCount
}

