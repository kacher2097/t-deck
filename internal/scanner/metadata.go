package scanner

import (
	"bufio"
	"encoding/json"
	"html"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"unicode/utf8"

	"t-deck/internal/config"
	"t-deck/internal/fsutil"
	"t-deck/internal/model"
)

var (
	reMDBadge = regexp.MustCompile(`\[!\[[^\]]*\]\([^)]+\)\]\([^)]+\)`)
	reMDLink  = regexp.MustCompile(`!?\[([^\]]*)\]\([^)]+\)`)
	reHTMLTag = regexp.MustCompile(`<[^>]+>`)
	reMultiWS = regexp.MustCompile(`\s+`)
)

// ComputeLocalStatus determines whether a local project directory is ready or missing .env.
func ComputeLocalStatus(dir string, projectType string) string {
	if !fsutil.DirExists(dir) {
		return config.StatusMissingDir
	}
	if projectType == config.ProjectTypeWeb {
		hasEnvExample := fsutil.FileExists(filepath.Join(dir, config.EnvExampleFile))
		hasEnv := fsutil.FileExists(filepath.Join(dir, config.EnvFile))
		if hasEnvExample && !hasEnv {
			return config.StatusMissingEnv
		}
	}
	return config.StatusReady
}

// ReadMetaFile reads and parses .tdeck-meta.json from projectDir.
func ReadMetaFile(projectDir string) (model.ProjectMeta, error) {
	data, err := os.ReadFile(filepath.Join(projectDir, config.MetaFileName))
	if err != nil {
		return model.ProjectMeta{}, err
	}
	var meta model.ProjectMeta
	if err := json.Unmarshal(data, &meta); err != nil {
		return model.ProjectMeta{}, err
	}
	return meta, nil
}

// WriteMetaFile writes .tdeck-meta.json into projectDir.
func WriteMetaFile(projectDir string, meta model.ProjectMeta) error {
	if err := os.MkdirAll(projectDir, 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(meta, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(projectDir, config.MetaFileName), data, 0o644)
}

func extractReadmeSummary(dir string) string {
	candidates := []string{config.ReadmeFile, "readme.md", "README.txt", "README"}
	for _, name := range candidates {
		path := filepath.Join(dir, name)
		data, err := os.ReadFile(path)
		if err == nil {
			return CleanMarkdownSummary(string(data), config.MaxReadmeSummaryLen)
		}
	}
	return ""
}

// ExtractSummary extracts a human-readable summary from README or falls back to package/project manifests.
func ExtractSummary(dir string) string {
	if sum := extractReadmeSummary(dir); sum != "" {
		return sum
	}
	if pkg := readRootPackageJSON(dir); pkg != nil {
		if desc, ok := pkg["description"].(string); ok && strings.TrimSpace(desc) != "" {
			return truncateRunes(strings.TrimSpace(desc), config.MaxReadmeSummaryLen)
		}
	}
	if desc := readVersionFromJSONFile(filepath.Join(dir, config.WailsConfig), "info.comments"); desc != "" {
		return truncateRunes(desc, config.MaxReadmeSummaryLen)
	}
	for _, tomlName := range []string{"pyproject.toml", "Cargo.toml"} {
		if desc := extractTomlQuotedField(filepath.Join(dir, tomlName), "description"); desc != "" {
			return truncateRunes(desc, config.MaxReadmeSummaryLen)
		}
	}
	return ""
}

// CleanMarkdownSummary strips raw Markdown syntax and truncates to maxRunes runes.
func CleanMarkdownSummary(raw string, maxRunes int) string {
	if strings.TrimSpace(raw) == "" {
		return ""
	}
	var lines []string
	sc := bufio.NewScanner(strings.NewReader(raw))
	inCodeFence := false
	inHTMLComment := false
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if strings.HasPrefix(line, "```") {
			inCodeFence = !inCodeFence
			continue
		}
		if strings.Contains(line, "<!--") {
			if !strings.Contains(line, "-->") {
				inHTMLComment = true
			}
			continue
		}
		if inHTMLComment {
			if strings.Contains(line, "-->") {
				inHTMLComment = false
			}
			continue
		}
		if inCodeFence || line == "" {
			continue
		}
		if strings.HasPrefix(line, "---") || strings.HasPrefix(line, "===") || strings.HasPrefix(line, "|") {
			continue
		}
		line = reMDBadge.ReplaceAllString(line, "")
		line = strings.TrimLeft(line, "#>*-+ ")
		line = reMDLink.ReplaceAllString(line, "$1")
		line = reHTMLTag.ReplaceAllString(line, "")
		line = html.UnescapeString(line)
		line = strings.NewReplacer("**", "", "__", "", "`", "", "~~", "").Replace(line)
		line = strings.TrimSpace(line)
		// Skip bare URL lines without prose.
		if strings.HasPrefix(line, "http://") || strings.HasPrefix(line, "https://") {
			if !strings.Contains(line, " ") {
				continue
			}
		}
		if line != "" {
			lines = append(lines, line)
		}
		if len(strings.Join(lines, " ")) > maxRunes*2 {
			break
		}
	}

	joined := reMultiWS.ReplaceAllString(strings.Join(lines, " "), " ")
	joined = strings.TrimSpace(joined)
	return truncateRunes(joined, maxRunes)
}

func truncateRunes(s string, maxRunes int) string {
	if maxRunes <= 0 || utf8.RuneCountInString(s) <= maxRunes {
		return s
	}
	runes := []rune(s)
	return strings.TrimSpace(string(runes[:maxRunes]))
}

func parseGitRemote(dir string) string {
	cfgPath := filepath.Join(dir, filepath.FromSlash(config.GitConfigRel))
	if data, err := os.ReadFile(cfgPath); err == nil {
		sc := bufio.NewScanner(strings.NewReader(string(data)))
		inOrigin := false
		inAnyRemote := false
		firstRemoteURL := ""
		for sc.Scan() {
			line := strings.TrimSpace(sc.Text())
			if strings.HasPrefix(line, "[") {
				inOrigin = strings.EqualFold(line, `[remote "origin"]`)
				inAnyRemote = strings.HasPrefix(strings.ToLower(line), `[remote "`)
				continue
			}
			if (inOrigin || inAnyRemote) && strings.HasPrefix(strings.ToLower(line), "url") {
				if _, val, ok := strings.Cut(line, "="); ok {
					trimmed := strings.TrimSpace(val)
					if inOrigin && trimmed != "" {
						return trimmed
					}
					if firstRemoteURL == "" && trimmed != "" {
						firstRemoteURL = trimmed
					}
				}
			}
		}
		if firstRemoteURL != "" {
			return firstRemoteURL
		}
	}

	if pkg := readRootPackageJSON(dir); pkg != nil {
		switch repo := pkg["repository"].(type) {
		case string:
			return cleanRepoURL(repo)
		case map[string]any:
			if u, ok := repo["url"].(string); ok {
				return cleanRepoURL(u)
			}
		}
	}
	return ""
}

func cleanRepoURL(raw string) string {
	s := strings.TrimSpace(raw)
	s = strings.TrimPrefix(s, "git+")
	return strings.TrimSpace(s)
}

// ReadGitInfo reads the current branch and commit SHA from .git/HEAD if present.
func ReadGitInfo(dir string) (branch string, commit string, dirty bool) {
	headPath := filepath.Join(dir, ".git", "HEAD")
	data, err := os.ReadFile(headPath)
	if err != nil {
		return "", "", false
	}
	raw := strings.TrimSpace(string(data))
	if strings.HasPrefix(raw, "ref:") {
		ref := strings.TrimSpace(strings.TrimPrefix(raw, "ref:"))
		parts := strings.Split(ref, "/")
		branch = parts[len(parts)-1]
		refPath := filepath.Join(dir, ".git", filepath.FromSlash(ref))
		if shaBytes, err := os.ReadFile(refPath); err == nil {
			sha := strings.TrimSpace(string(shaBytes))
			if len(sha) > 7 {
				sha = sha[:7]
			}
			commit = sha
		}
	} else if len(raw) >= 7 {
		branch = "detached"
		commit = raw[:7]
	}
	return branch, commit, false
}

func inferTags(techStack, projectType string) []string {
	tags := []string{projectType}
	for _, part := range strings.Split(techStack, ",") {
		clean := fsutil.Slugify(part)
		if clean != "" {
			tags = append(tags, clean)
		}
	}
	return fsutil.UniqueStrings(tags)
}

func readRootPackageJSON(dir string) map[string]any {
	data, err := os.ReadFile(filepath.Join(dir, "package.json"))
	if err != nil {
		return nil
	}
	var root map[string]any
	if err := json.Unmarshal(data, &root); err != nil {
		return nil
	}
	return root
}

func extractTomlQuotedField(path string, key string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	sc := bufio.NewScanner(strings.NewReader(string(data)))
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, "[") {
			continue
		}
		k, val, ok := strings.Cut(line, "=")
		if !ok || !strings.EqualFold(strings.TrimSpace(k), key) {
			continue
		}
		v := strings.TrimSpace(val)
		v = strings.Trim(v, `"'`)
		if v != "" {
			return v
		}
	}
	return ""
}
