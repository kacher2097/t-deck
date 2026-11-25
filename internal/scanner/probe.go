package scanner

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"t-deck/internal/config"
	"t-deck/internal/fsutil"
	"t-deck/internal/model"
)

// ProbeDirectory inspects a single local project directory and returns auto-detected
// metadata suitable for pre-filling Add Project or refreshing project configuration.
func ProbeDirectory(dir string, availableCategories []string) (model.ProjectProbe, error) {
	cleaned := filepath.Clean(strings.TrimSpace(dir))
	if cleaned == "" || cleaned == "." {
		return model.ProjectProbe{}, fmt.Errorf("đường dẫn thư mục không hợp lệ")
	}
	if !fsutil.DirExists(cleaned) {
		return model.ProjectProbe{}, fmt.Errorf("không tìm thấy thư mục: %s", cleaned)
	}

	probe := model.ProjectProbe{
		LocalDir:  cleaned,
		Name:      filepath.Base(cleaned),
		GitRemote: parseGitRemote(cleaned),
		Summary:   ExtractSummary(cleaned),
	}

	// Read .tdeck-meta.json if already tracked on disk.
	var metaSlug, metaType string
	if meta, err := ReadMetaFile(cleaned); err == nil && strings.TrimSpace(meta.ProjectID) != "" {
		probe.HasMeta = true
		probe.ExistingID = strings.TrimSpace(meta.ProjectID)
		metaSlug = strings.TrimSpace(meta.Slug)
		metaType = strings.TrimSpace(meta.Type)
	}

	// Detect code/executable markers (root + standard subdirs).
	_, markerInfo := detectProjectMarkers(cleaned, false)
	probe.Type = markerInfo.projectType
	if probe.Type == "" {
		probe.Type = config.ProjectTypeWeb
	}
	if metaType == config.ProjectTypeWeb || metaType == config.ProjectTypeDesktop || metaType == config.ProjectTypeHybrid {
		probe.Type = metaType
	}
	probe.TechStack = markerInfo.techStack
	probe.BatFile = markerInfo.batFile
	if probe.BatFile == "" {
		probe.BatFile = config.DefaultBatFile
	}
	probe.ExeName = markerInfo.exeName

	// Refine name from wails.json if present (keeps folder name otherwise, which is cleaner than scoped npm names).
	if prodName := readVersionFromJSONFile(filepath.Join(cleaned, config.WailsConfig), "info.productName"); prodName != "" {
		probe.Name = prodName
	}

	// Check .tdeck.json declarative manifest for explicit overrides and web prod URL.
	var manifestSrc *ManifestVersionSource
	var manifestTags []string
	if manifest, found, err := LoadProjectManifest(cleaned); err == nil && found && manifest != nil {
		manifestSrc = manifest.VersionSource
		if strings.TrimSpace(manifest.Name) != "" {
			probe.Name = strings.TrimSpace(manifest.Name)
		}
		if strings.TrimSpace(manifest.Slug) != "" {
			metaSlug = strings.TrimSpace(manifest.Slug)
		}
		if strings.TrimSpace(manifest.Category) != "" {
			probe.Category = strings.TrimSpace(manifest.Category)
		}
		manifestTags = manifest.Tags

		hasWebTarget, hasDeskTarget := false, false
		for _, t := range manifest.Targets {
			switch strings.ToLower(strings.TrimSpace(t.Kind)) {
			case config.ProjectTypeWeb:
				hasWebTarget = true
				if probe.ProdURL == "" && strings.TrimSpace(t.ProdURL) != "" {
					probe.ProdURL = strings.TrimSpace(t.ProdURL)
				}
				if strings.TrimSpace(t.LaunchEntry) != "" {
					probe.BatFile = strings.TrimSpace(t.LaunchEntry)
				}
			case config.ProjectTypeDesktop:
				hasDeskTarget = true
				if strings.TrimSpace(t.LaunchEntry) != "" {
					probe.ExeName = strings.TrimSpace(t.LaunchEntry)
				}
			}
		}
		if hasWebTarget && hasDeskTarget {
			probe.Type = config.ProjectTypeHybrid
		} else if hasDeskTarget && !hasWebTarget {
			probe.Type = config.ProjectTypeDesktop
		} else if hasWebTarget && !hasDeskTarget {
			probe.Type = config.ProjectTypeWeb
		}
	}

	// Fallback ProdURL from CNAME or package.json homepage (excluding git repo URLs).
	if probe.ProdURL == "" {
		probe.ProdURL = extractProdURL(cleaned, probe.GitRemote)
	}

	if metaSlug != "" {
		probe.Slug = fsutil.Slugify(metaSlug)
	}
	if probe.Slug == "" {
		probe.Slug = fsutil.Slugify(probe.Name)
	}

	// Merge inferred tags + package.json keywords + manifest tags.
	tags := inferTags(probe.TechStack, probe.Type)
	tags = append(tags, extractPackageKeywords(cleaned)...)
	for _, t := range manifestTags {
		if clean := fsutil.Slugify(t); clean != "" {
			tags = append(tags, clean)
		}
	}
	probe.Tags = fsutil.UniqueStrings(tags)
	if len(probe.Tags) > 8 {
		probe.Tags = probe.Tags[:8]
	}

	if probe.Category == "" {
		probe.Category = suggestCategory(probe, availableCategories)
	}

	probe.Version = ResolveProjectVersion(cleaned, manifestSrc)
	probe.HasEnv, _, probe.HasBat = computeDirFlags(cleaned, probe.BatFile)

	return probe, nil
}

func extractPackageKeywords(dir string) []string {
	pkg := readRootPackageJSON(dir)
	if pkg == nil {
		return nil
	}
	rawList, ok := pkg["keywords"].([]any)
	if !ok {
		return nil
	}
	out := make([]string, 0, len(rawList))
	for _, item := range rawList {
		if s, ok := item.(string); ok {
			if clean := fsutil.Slugify(s); clean != "" {
				out = append(out, clean)
			}
		}
	}
	return out
}

func extractProdURL(dir string, gitRemote string) string {
	if cnameData, err := os.ReadFile(filepath.Join(dir, "CNAME")); err == nil {
		domain := strings.TrimSpace(string(cnameData))
		if domain != "" && !strings.ContainsAny(domain, " \t\r\n") {
			if strings.HasPrefix(domain, "http://") || strings.HasPrefix(domain, "https://") {
				return domain
			}
			return "https://" + domain
		}
	}
	if pkg := readRootPackageJSON(dir); pkg != nil {
		if hp, ok := pkg["homepage"].(string); ok {
			hp = strings.TrimSpace(hp)
			lower := strings.ToLower(hp)
			if (strings.HasPrefix(lower, "http://") || strings.HasPrefix(lower, "https://")) &&
				!strings.Contains(lower, "localhost") &&
				!strings.Contains(lower, "github.com/") &&
				!strings.Contains(lower, "gitlab.com/") &&
				!strings.EqualFold(hp, gitRemote) {
				return hp
			}
		}
	}
	return ""
}

func suggestCategory(p model.ProjectProbe, available []string) string {
	if len(available) == 0 {
		return ""
	}
	findMatch := func(needles ...string) string {
		for _, cat := range available {
			lowerCat := strings.ToLower(cat)
			for _, n := range needles {
				if strings.Contains(lowerCat, n) {
					return cat
				}
			}
		}
		return ""
	}

	haystack := strings.ToLower(p.Name + " " + p.Slug + " " + p.Summary + " " + p.TechStack + " " + p.LocalDir)

	switch {
	case strings.Contains(haystack, "t-hub") || strings.Contains(haystack, "noxus") || strings.Contains(haystack, "desktop-station"):
		if m := findMatch("t-hub", "ecosystem"); m != "" {
			return m
		}
	case strings.Contains(haystack, "vnpay") || strings.Contains(haystack, "java") || strings.Contains(haystack, "spring boot") || strings.Contains(haystack, "maven"):
		if m := findMatch("java", "vnpay", "enterprise"); m != "" {
			return m
		}
	case strings.Contains(haystack, "video") || strings.Contains(haystack, "media") || strings.Contains(haystack, "tts") ||
		strings.Contains(haystack, "audio") || strings.Contains(haystack, "ffmpeg") || strings.Contains(haystack, "movie"):
		if m := findMatch("video", "media"); m != "" {
			return m
		}
	case strings.Contains(haystack, "tool") || strings.Contains(haystack, "bot") || strings.Contains(haystack, "crawl") ||
		strings.Contains(haystack, "automation") || strings.Contains(haystack, "cli") || strings.Contains(haystack, "r2"):
		if m := findMatch("devtool", "automation", "tool"); m != "" {
			return m
		}
	}

	if p.Type == config.ProjectTypeDesktop {
		if m := findMatch("desktop"); m != "" {
			return m
		}
	}

	// Token overlap between project haystack and existing category names.
	for _, cat := range available {
		for _, tok := range strings.Fields(strings.ToLower(cat)) {
			clean := strings.Trim(tok, "()&,-/")
			if len(clean) >= 3 && strings.Contains(haystack, clean) {
				return cat
			}
		}
	}

	if m := findMatch("business", "web app"); m != "" {
		return m
	}
	return ""
}

