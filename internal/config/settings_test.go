package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSettingsManagerLoadSave(t *testing.T) {
	dir := t.TempDir()
	mgr, err := NewSettingsManager(dir)
	if err != nil {
		t.Fatalf("NewSettingsManager failed: %v", err)
	}

	// First load returns defaults without file on disk.
	def, err := mgr.Load()
	if err != nil {
		t.Fatalf("Load default failed: %v", err)
	}
	if def.R2.BucketName != DefaultBucketName {
		t.Fatalf("expected default bucket %q, got %q", DefaultBucketName, def.R2.BucketName)
	}

	saved, err := mgr.Save(Settings{
		WorkspaceDirs:  []string{`D:\Projects`, `D:\Projects\`, ` C:\Dev `},
		AppsDir:        `C:\TDeckApps\`,
		ScanRecentDays: 120,
		R2: R2Config{
			Enabled:         true,
			AccountID:       " acc-123 ",
			AccessKeyID:     " key-1 ",
			SecretAccessKey: " sec-1 ",
			BucketName:      "",
			PublicDomain:    "https://cdn.example.com/",
		},
	})
	if err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	if len(saved.WorkspaceDirs) != 2 {
		t.Fatalf("expected 2 deduplicated workspace dirs, got %v", saved.WorkspaceDirs)
	}
	if saved.R2.BucketName != DefaultBucketName {
		t.Fatalf("expected fallback bucket name %q, got %q", DefaultBucketName, saved.R2.BucketName)
	}
	if saved.R2.PublicDomain != "https://cdn.example.com" {
		t.Fatalf("expected trimmed public domain, got %q", saved.R2.PublicDomain)
	}

	if _, err := os.Stat(filepath.Join(dir, SettingsFileName)); err != nil {
		t.Fatalf("settings.json missing: %v", err)
	}

	reloaded, err := mgr.Load()
	if err != nil {
		t.Fatalf("Reload failed: %v", err)
	}
	if reloaded.R2.AccountID != "acc-123" || reloaded.AppsDir != filepath.Clean(`C:\TDeckApps\`) {
		t.Fatalf("unexpected reloaded settings: %+v", reloaded)
	}
}

func TestEnvSecretsNeverWrittenToSettingsJSON(t *testing.T) {
	dir := t.TempDir()
	envPath := filepath.Join(dir, EnvFile)
	mgr := &SettingsManager{
		baseDir:    dir,
		useEnvFile: true,
		envPath:    envPath,
	}

	t.Cleanup(func() {
		for _, k := range []string{
			EnvKeyWorkspaceDirs, EnvKeyAppsDir, EnvKeyScanRecentDays, EnvKeyScanExcludeDirs,
			EnvKeyR2Enabled, EnvKeyR2AccountID, EnvKeyR2Endpoint, EnvKeyR2AccessKey,
			EnvKeyR2SecretKey, EnvKeyR2Bucket, EnvKeyR2PublicDomain,
		} {
			_ = os.Unsetenv(k)
		}
	})

	_, err := mgr.Save(Settings{
		WorkspaceDirs:   []string{`H:\app`},
		AppsDir:         `C:\TDeckApps`,
		ScanRecentDays:  120,
		ScanExcludeDirs: []string{"agent-skills-master"},
		R2: R2Config{
			Enabled:         true,
			AccountID:       "acct-abc",
			AccessKeyID:     "my-access-key",
			SecretAccessKey: "my-super-secret-key",
			BucketName:      "my-storage",
		},
	})
	if err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	jsonBytes, err := os.ReadFile(mgr.SettingsPath())
	if err != nil {
		t.Fatal(err)
	}
	jsonText := string(jsonBytes)
	if strings.Contains(jsonText, "my-access-key") || strings.Contains(jsonText, "my-super-secret-key") {
		t.Fatalf("secrets leaked into settings.json:\n%s", jsonText)
	}

	envBytes, err := os.ReadFile(envPath)
	if err != nil {
		t.Fatal(err)
	}
	envText := string(envBytes)
	if !strings.Contains(envText, "R2_ACCESS_KEY=my-access-key") || !strings.Contains(envText, "R2_SECRET_KEY=my-super-secret-key") {
		t.Fatalf("expected secrets persisted in .env, got:\n%s", envText)
	}

	loaded, err := mgr.Load()
	if err != nil {
		t.Fatal(err)
	}
	if loaded.R2.AccessKeyID != "my-access-key" || loaded.R2.SecretAccessKey != "my-super-secret-key" {
		t.Fatalf("expected secrets loaded back from .env, got: %+v", loaded.R2)
	}
}
