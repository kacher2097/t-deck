package scanner

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"t-deck/internal/config"
)

// ProjectManifest represents the declarative .tdeck.json config inside a project root.
type ProjectManifest struct {
	SchemaVersion int                    `json:"schema_version"`
	Slug          string                 `json:"slug"`
	Name          string                 `json:"name"`
	Category      string                 `json:"category"`
	Tags          []string               `json:"tags"`
	VersionSource *ManifestVersionSource `json:"version_source,omitempty"`
	Targets       []ManifestTarget       `json:"targets"`
}

// ManifestVersionSource configures how the project version is automatically read.
type ManifestVersionSource struct {
	Type     string `json:"type"` // "auto" | "json" | "git_tag"
	File     string `json:"file"`
	JSONPath string `json:"json_path"`
}

// ManifestTarget declares one Web or Desktop build/release target in .tdeck.json.
type ManifestTarget struct {
	Kind            string `json:"kind"` // "web" | "desktop"
	Name            string `json:"name"`
	Platform        string `json:"platform"`
	Distribution    string `json:"distribution"`
	Subpath         string `json:"subpath"`
	LaunchEntry     string `json:"launch_entry"`
	BuildCmd        string `json:"build_cmd"`
	ArtifactGlob    string `json:"artifact_glob"`
	InstallerGlob   string `json:"installer_glob"`
	ProdURL         string `json:"prod_url"`
	StagingURL      string `json:"staging_url"`
	HealthURL       string `json:"health_url"`
	VersionURL      string `json:"version_url"`
	HostingProvider string `json:"hosting_provider"`
}

// LoadProjectManifest reads and parses .tdeck.json from projectDir if it exists.
func LoadProjectManifest(projectDir string) (*ProjectManifest, bool, error) {
	path := filepath.Join(projectDir, config.ManifestFileName)
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, false, nil
		}
		return nil, false, err
	}
	var m ProjectManifest
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, true, err
	}
	return &m, true, nil
}

// ResolveProjectVersion detects a project's version from wails.json, package.json, git tags, or fallback "v0.1.0".
func ResolveProjectVersion(projectDir string, src *ManifestVersionSource) string {
	if src != nil && strings.TrimSpace(src.File) != "" {
		if v := readVersionFromJSONFile(filepath.Join(projectDir, src.File), src.JSONPath); v != "" {
			return ensureVPrefix(v)
		}
	}

	if v := readVersionFromJSONFile(filepath.Join(projectDir, config.WailsConfig), "info.productVersion"); v != "" {
		return ensureVPrefix(v)
	}
	if v := readVersionFromJSONFile(filepath.Join(projectDir, "package.json"), "version"); v != "" {
		return ensureVPrefix(v)
	}
	if out, err := exec.Command("git", "-C", projectDir, "describe", "--tags", "--abbrev=0").Output(); err == nil {
		tag := strings.TrimSpace(string(out))
		if tag != "" {
			return ensureVPrefix(tag)
		}
	}
	return "v0.1.0"
}

func readVersionFromJSONFile(path string, dotPath string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	var root map[string]any
	if err := json.Unmarshal(data, &root); err != nil {
		return ""
	}
	if strings.TrimSpace(dotPath) == "" {
		dotPath = "version"
	}
	parts := strings.Split(dotPath, ".")
	var curr any = root
	for _, p := range parts {
		m, ok := curr.(map[string]any)
		if !ok {
			return ""
		}
		curr = m[p]
	}
	if s, ok := curr.(string); ok {
		return strings.TrimSpace(s)
	}
	return ""
}

func ensureVPrefix(v string) string {
	trimmed := strings.TrimSpace(v)
	if trimmed == "" {
		return "v0.1.0"
	}
	if strings.HasPrefix(strings.ToLower(trimmed), "v") {
		return trimmed
	}
	return "v" + trimmed
}
