package config

import (
	"encoding/json"
	"errors"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// R2Config stores Cloudflare R2 S3-compatible connection parameters.
// AccessKeyID and SecretAccessKey are sourced from .env / environment variables and never persisted to settings.json.
type R2Config struct {
	Enabled         bool   `json:"enabled"`
	AccountID       string `json:"account_id"`
	AccessKeyID     string `json:"access_key_id"`
	SecretAccessKey string `json:"secret_access_key"`
	BucketName      string `json:"bucket_name"`
	PublicDomain    string `json:"public_domain"`
}

// Settings holds local machine configuration stored in <InstallDir>\settings.json and .env.
type Settings struct {
	WorkspaceDirs   []string `json:"workspace_dirs"`
	AppsDir         string   `json:"apps_dir"`
	ScanRecentDays  int      `json:"scan_recent_days"`
	ScanExcludeDirs []string `json:"scan_exclude_dirs"`
	R2              R2Config `json:"r2"`
}

// SettingsManager manages reading and writing T-Deck local settings and paths.
type SettingsManager struct {
	baseDir    string
	useEnvFile bool
	envPath    string
}

// ResolveDefaultDataDir resolves the user-chosen installation directory for storing T-Deck data
// (tdeck.db, settings.json, .env, thumbnails/, apps/).
// Priority: TDECK_DATA_DIR env -> <repoRoot> (dev build/bin) -> <exeDir> (installed app) -> os.UserConfigDir/TDeck (if read-only).
func ResolveDefaultDataDir() (string, error) {
	if envDir, ok := lookupNonEmptyEnv(EnvKeyDataDir); ok {
		return filepath.Clean(envDir), nil
	}

	candidate := ""
	if exe, err := os.Executable(); err == nil {
		if resolved, rErr := filepath.EvalSymlinks(exe); rErr == nil {
			exe = resolved
		}
		exeDir := filepath.Dir(exe)
		if isTempOrGoBuildDir(exeDir) {
			if wd, wdErr := os.Getwd(); wdErr == nil {
				candidate = wd
			}
		} else if devRoot := detectDevRepoRoot(exeDir); devRoot != "" {
			candidate = devRoot
		} else {
			candidate = exeDir
		}
	}

	if candidate == "" {
		if wd, err := os.Getwd(); err == nil {
			candidate = wd
		}
	}

	if candidate != "" && isDirWritable(candidate) {
		return filepath.Clean(candidate), nil
	}

	cfgDir, err := os.UserConfigDir()
	if err != nil {
		if candidate != "" {
			return filepath.Clean(candidate), nil
		}
		return "", err
	}
	return filepath.Join(cfgDir, AppName), nil
}

func isTempOrGoBuildDir(dir string) bool {
	norm := strings.ToLower(filepath.Clean(dir))
	tmp := strings.ToLower(filepath.Clean(os.TempDir()))
	if tmp != "" && (norm == tmp || strings.HasPrefix(norm, tmp+string(filepath.Separator))) {
		return true
	}
	return strings.Contains(norm, string(filepath.Separator)+"go-build")
}

func detectDevRepoRoot(exeDir string) string {
	clean := filepath.Clean(exeDir)
	if strings.EqualFold(filepath.Base(clean), "bin") {
		parent := filepath.Dir(clean)
		if strings.EqualFold(filepath.Base(parent), "build") {
			repoRoot := filepath.Dir(parent)
			if info, err := os.Stat(filepath.Join(repoRoot, WailsConfig)); err == nil && !info.IsDir() {
				return repoRoot
			}
		}
	}
	return ""
}

func isDirWritable(dir string) bool {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return false
	}
	f, err := os.CreateTemp(dir, ".tdeck-write-check-*")
	if err != nil {
		return false
	}
	name := f.Name()
	_ = f.Close()
	_ = os.Remove(name)
	return true
}

func migrateLegacyAppDataIfNeeded(baseDir string) {
	if strings.TrimSpace(baseDir) == "" {
		return
	}
	targetDB := filepath.Join(baseDir, DatabaseFileName)
	if _, err := os.Stat(targetDB); err == nil {
		return
	}
	cfgDir, err := os.UserConfigDir()
	if err != nil || strings.TrimSpace(cfgDir) == "" {
		return
	}
	legacyDir := filepath.Join(cfgDir, AppName)
	if strings.EqualFold(filepath.Clean(legacyDir), filepath.Clean(baseDir)) {
		return
	}
	legacyDB := filepath.Join(legacyDir, DatabaseFileName)
	if _, err := os.Stat(legacyDB); err != nil {
		return
	}

	for _, name := range []string{DatabaseFileName, DatabaseFileName + "-wal", DatabaseFileName + "-shm", SettingsFileName, EnvFile} {
		src := filepath.Join(legacyDir, name)
		dst := filepath.Join(baseDir, name)
		if _, err := os.Stat(dst); err == nil {
			continue
		}
		if info, err := os.Stat(src); err == nil && !info.IsDir() {
			perm := os.FileMode(0o644)
			if name == EnvFile || name == SettingsFileName {
				perm = 0o600
			}
			_ = copyFileForMigration(src, dst, perm)
		}
	}

	legacyThumbs := filepath.Join(legacyDir, ThumbnailsDirName)
	targetThumbs := filepath.Join(baseDir, ThumbnailsDirName)
	if entries, err := os.ReadDir(legacyThumbs); err == nil {
		_ = os.MkdirAll(targetThumbs, 0o755)
		for _, entry := range entries {
			if entry.IsDir() {
				continue
			}
			src := filepath.Join(legacyThumbs, entry.Name())
			dst := filepath.Join(targetThumbs, entry.Name())
			if _, err := os.Stat(dst); err == nil {
				continue
			}
			_ = copyFileForMigration(src, dst, 0o644)
		}
	}
}

func copyFileForMigration(src, dst string, perm os.FileMode) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, perm)
	if err != nil {
		return err
	}
	_, copyErr := io.Copy(out, in)
	closeErr := out.Close()
	if copyErr != nil {
		_ = os.Remove(dst)
		return copyErr
	}
	return closeErr
}

// NewSettingsManager creates a SettingsManager rooted at the installation directory (or customDir if non-empty).
func NewSettingsManager(customDir string) (*SettingsManager, error) {
	isCustom := strings.TrimSpace(customDir) != ""
	base := strings.TrimSpace(customDir)
	if !isCustom {
		resolved, err := ResolveDefaultDataDir()
		if err != nil {
			return nil, err
		}
		base = resolved
	}
	if err := os.MkdirAll(base, 0o755); err != nil {
		return nil, err
	}
	if !isCustom {
		migrateLegacyAppDataIfNeeded(base)
	}
	thumbDir := filepath.Join(base, ThumbnailsDirName)
	if err := os.MkdirAll(thumbDir, 0o755); err != nil {
		return nil, err
	}

	mgr := &SettingsManager{
		baseDir:    base,
		useEnvFile: !isCustom,
		envPath:    filepath.Join(base, EnvFile),
	}
	if mgr.useEnvFile {
		_ = LoadDotEnvFiles(CandidateDotEnvPaths(base)...)
		mgr.envPath = ResolveWritableDotEnvPath(base)
	}
	return mgr, nil
}

// BaseDir returns the TDeck application data directory.
func (m *SettingsManager) BaseDir() string {
	return m.baseDir
}

// SettingsPath returns the full path to settings.json.
func (m *SettingsManager) SettingsPath() string {
	return filepath.Join(m.baseDir, SettingsFileName)
}

// EnvPath returns the active .env path managed for secrets/config.
func (m *SettingsManager) EnvPath() string {
	return m.envPath
}

// DBPath returns the full path to tdeck.db.
func (m *SettingsManager) DBPath() string {
	return filepath.Join(m.baseDir, DatabaseFileName)
}

// ThumbnailsDir returns the full path to the local thumbnails directory.
func (m *SettingsManager) ThumbnailsDir() string {
	return filepath.Join(m.baseDir, ThumbnailsDirName)
}

// DefaultAppsDir returns the default directory inside the installation folder for downloaded desktop apps.
func (m *SettingsManager) DefaultAppsDir() string {
	return filepath.Join(m.baseDir, DefaultAppsDirName)
}

// ThumbnailPath returns the local PNG file path for a project's thumbnail.
func (m *SettingsManager) ThumbnailPath(projectID string) string {
	return filepath.Join(m.ThumbnailsDir(), projectID+".png")
}

// DefaultSettings returns a clean initial configuration.
func DefaultSettings() Settings {
	return Settings{
		WorkspaceDirs:   []string{},
		AppsDir:         "",
		ScanRecentDays:  DefaultScanRecentDays,
		ScanExcludeDirs: []string{},
		R2: R2Config{
			Enabled:    false,
			BucketName: DefaultBucketName,
		},
	}
}

// Load reads settings.json and overlays externalized configuration & secrets from .env/env vars.
func (m *SettingsManager) Load() (Settings, error) {
	if m.useEnvFile {
		_ = LoadDotEnvFiles(CandidateDotEnvPaths(m.baseDir)...)
	}

	settings := DefaultSettings()
	data, err := os.ReadFile(m.SettingsPath())
	if err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			return Settings{}, err
		}
	} else {
		if err := json.Unmarshal(data, &settings); err != nil {
			return Settings{}, err
		}
	}

	if m.useEnvFile {
		applyEnvOverrides(&settings)
	}
	normalized := normalizeSettings(settings)
	if strings.TrimSpace(normalized.AppsDir) == "" {
		normalized.AppsDir = m.DefaultAppsDir()
	} else if !filepath.IsAbs(normalized.AppsDir) {
		normalized.AppsDir = filepath.Clean(filepath.Join(m.baseDir, normalized.AppsDir))
	}
	return normalized, nil
}

// Save writes non-secret configuration to settings.json and persists secrets/config to .env.
func (m *SettingsManager) Save(s Settings) (Settings, error) {
	normalized := normalizeSettings(s)
	if strings.TrimSpace(normalized.AppsDir) == "" {
		normalized.AppsDir = m.DefaultAppsDir()
	} else if !filepath.IsAbs(normalized.AppsDir) {
		normalized.AppsDir = filepath.Clean(filepath.Join(m.baseDir, normalized.AppsDir))
	}
	if err := os.MkdirAll(m.baseDir, 0o755); err != nil {
		return Settings{}, err
	}

	if m.useEnvFile {
		envUpdates := map[string]string{
			EnvKeyWorkspaceDirs:   strings.Join(normalized.WorkspaceDirs, ","),
			EnvKeyAppsDir:         normalized.AppsDir,
			EnvKeyScanRecentDays:  strconv.Itoa(normalized.ScanRecentDays),
			EnvKeyScanExcludeDirs: strings.Join(normalized.ScanExcludeDirs, ","),
			EnvKeyR2Enabled:       strconv.FormatBool(normalized.R2.Enabled),
			EnvKeyR2AccountID:     normalized.R2.AccountID,
			EnvKeyR2Bucket:        normalized.R2.BucketName,
			EnvKeyR2PublicDomain:  normalized.R2.PublicDomain,
			EnvKeyR2AccessKey:     normalized.R2.AccessKeyID,
			EnvKeyR2SecretKey:     normalized.R2.SecretAccessKey,
		}
		if normalized.R2.AccountID != "" {
			envUpdates[EnvKeyR2Endpoint] = "https://" + normalized.R2.AccountID + ".r2.cloudflarestorage.com"
		}
		if err := UpsertDotEnv(m.envPath, envUpdates); err != nil {
			return Settings{}, err
		}
	}

	// Never write plaintext API keys / secret keys into settings.json when env storage is active.
	diskCopy := normalized
	if m.useEnvFile {
		diskCopy.R2.AccessKeyID = ""
		diskCopy.R2.SecretAccessKey = ""
	}

	payload, err := json.MarshalIndent(diskCopy, "", "  ")
	if err != nil {
		return Settings{}, err
	}
	if err := os.WriteFile(m.SettingsPath(), payload, 0o600); err != nil {
		return Settings{}, err
	}
	return normalized, nil
}

func applyEnvOverrides(s *Settings) {
	if raw, ok := lookupNonEmptyEnv(EnvKeyWorkspaceDirs); ok {
		s.WorkspaceDirs = splitListEnv(raw)
	}
	if raw, ok := lookupNonEmptyEnv(EnvKeyAppsDir); ok {
		s.AppsDir = raw
	}
	if raw, ok := lookupNonEmptyEnv(EnvKeyScanRecentDays); ok {
		if days, err := strconv.Atoi(raw); err == nil && days >= 0 {
			s.ScanRecentDays = days
		}
	}
	if raw, ok := lookupNonEmptyEnv(EnvKeyScanExcludeDirs); ok {
		s.ScanExcludeDirs = splitListEnv(raw)
	}

	if raw, ok := lookupNonEmptyEnv(EnvKeyR2Enabled); ok {
		if b, err := strconv.ParseBool(raw); err == nil {
			s.R2.Enabled = b
		}
	}
	if raw, ok := lookupNonEmptyEnv(EnvKeyR2AccountID); ok {
		s.R2.AccountID = raw
	}
	if s.R2.AccountID == "" {
		if endpoint, ok := lookupNonEmptyEnv(EnvKeyR2Endpoint); ok {
			s.R2.AccountID = extractAccountIDFromEndpoint(endpoint)
		}
	}
	if raw, ok := lookupNonEmptyEnv(EnvKeyR2AccessKey); ok {
		s.R2.AccessKeyID = raw
	}
	if raw, ok := lookupNonEmptyEnv(EnvKeyR2SecretKey); ok {
		s.R2.SecretAccessKey = raw
	}
	if raw, ok := lookupNonEmptyEnv(EnvKeyR2Bucket); ok {
		s.R2.BucketName = raw
	}
	if raw, ok := lookupNonEmptyEnv(EnvKeyR2PublicDomain); ok {
		s.R2.PublicDomain = raw
	}
	if s.R2.AccessKeyID != "" && s.R2.SecretAccessKey != "" && s.R2.AccountID != "" {
		if _, explicit := os.LookupEnv(EnvKeyR2Enabled); !explicit {
			s.R2.Enabled = true
		}
	}
}

func lookupNonEmptyEnv(key string) (string, bool) {
	val, ok := os.LookupEnv(key)
	if !ok {
		return "", false
	}
	trimmed := strings.TrimSpace(val)
	if trimmed == "" {
		return "", false
	}
	return trimmed, true
}

func splitListEnv(raw string) []string {
	parts := strings.FieldsFunc(raw, func(r rune) bool {
		return r == ',' || r == ';'
	})
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if t := strings.TrimSpace(p); t != "" {
			out = append(out, t)
		}
	}
	return out
}

func extractAccountIDFromEndpoint(endpoint string) string {
	u, err := url.Parse(strings.TrimSpace(endpoint))
	if err != nil || u.Host == "" {
		return ""
	}
	host := strings.Split(u.Host, ":")[0]
	if sub, _, ok := strings.Cut(host, "."); ok {
		return sub
	}
	return ""
}

func normalizeSettings(s Settings) Settings {
	seen := make(map[string]struct{}, len(s.WorkspaceDirs))
	cleanDirs := make([]string, 0, len(s.WorkspaceDirs))
	for _, dir := range s.WorkspaceDirs {
		trimmed := strings.TrimSpace(dir)
		if trimmed == "" {
			continue
		}
		cleaned := filepath.Clean(trimmed)
		key := strings.ToLower(cleaned)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		cleanDirs = append(cleanDirs, cleaned)
	}
	s.WorkspaceDirs = cleanDirs

	if trimmedApps := strings.TrimSpace(s.AppsDir); trimmedApps != "" {
		s.AppsDir = filepath.Clean(trimmedApps)
	} else {
		s.AppsDir = ""
	}

	if s.ScanRecentDays < 0 {
		s.ScanRecentDays = 0
	}

	seenExcl := make(map[string]struct{}, len(s.ScanExcludeDirs))
	cleanExcl := make([]string, 0, len(s.ScanExcludeDirs))
	for _, dir := range s.ScanExcludeDirs {
		trimmed := strings.TrimSpace(dir)
		if trimmed == "" {
			continue
		}
		key := strings.ToLower(trimmed)
		if _, ok := seenExcl[key]; ok {
			continue
		}
		seenExcl[key] = struct{}{}
		cleanExcl = append(cleanExcl, trimmed)
	}
	s.ScanExcludeDirs = cleanExcl

	s.R2.AccountID = strings.TrimSpace(s.R2.AccountID)
	s.R2.AccessKeyID = strings.TrimSpace(s.R2.AccessKeyID)
	s.R2.SecretAccessKey = strings.TrimSpace(s.R2.SecretAccessKey)
	s.R2.BucketName = strings.TrimSpace(s.R2.BucketName)
	if s.R2.BucketName == "" {
		s.R2.BucketName = DefaultBucketName
	}
	s.R2.PublicDomain = strings.TrimRight(strings.TrimSpace(s.R2.PublicDomain), "/")
	return s
}
