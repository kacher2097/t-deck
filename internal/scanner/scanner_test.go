package scanner

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"t-deck/internal/config"
	"t-deck/internal/database"
	"t-deck/internal/model"
)

func TestScannerDiscoveryAndAutoRelink(t *testing.T) {
	tmp := t.TempDir()
	wsDir := filepath.Join(tmp, "workspace")
	appsDir := filepath.Join(tmp, "apps")
	if err := os.MkdirAll(wsDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(appsDir, 0o755); err != nil {
		t.Fatal(err)
	}

	// 1. Web app with package.json, README.md, and nested node_modules (must be ignored).
	webDir := filepath.Join(wsDir, "my-web-app")
	if err := os.MkdirAll(filepath.Join(webDir, "node_modules", "fake-pkg"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(webDir, "package.json"), []byte(`{"name":"my-web","dependencies":{"react":"19"}}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(webDir, "node_modules", "fake-pkg", "package.json"), []byte(`{"name":"ignored"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(webDir, "README.md"), []byte("# My Web App\n\nỨng dụng **quản lý** kho hàng."), 0o644); err != nil {
		t.Fatal(err)
	}

	// 2. Desktop app in appsDir with .exe.
	deskDir := filepath.Join(appsDir, "audio-tool")
	if err := os.MkdirAll(deskDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(deskDir, "AudioTool.exe"), []byte("MZ"), 0o644); err != nil {
		t.Fatal(err)
	}

	db, err := database.Open(filepath.Join(tmp, config.DatabaseFileName))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	repo := database.NewProjectRepo(db, filepath.Join(tmp, config.ThumbnailsDirName))
	sc := NewScanner(repo)

	rep1, err := sc.Scan([]string{wsDir}, appsDir)
	if err != nil {
		t.Fatalf("Scan 1 failed: %v", err)
	}
	if rep1.Added != 2 {
		t.Fatalf("expected 2 added projects, got %+v", rep1)
	}

	// Verify .tdeck-meta.json created and summary cleaned.
	meta, err := ReadMetaFile(webDir)
	if err != nil || meta.ProjectID == "" {
		t.Fatalf("expected .tdeck-meta.json in webDir: %v", err)
	}
	pView, err := repo.GetProject(meta.ProjectID)
	if err != nil {
		t.Fatal(err)
	}
	if pView.Type != config.ProjectTypeWeb || pView.Summary != "My Web App Ứng dụng quản lý kho hàng." {
		t.Fatalf("unexpected project view: %+v", pView)
	}

	// Move webDir to another path and re-scan -> must Auto-Relink without duplicating!
	movedWebDir := filepath.Join(wsDir, "archived", "my-web-app-v2")
	if err := os.MkdirAll(filepath.Dir(movedWebDir), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(webDir, movedWebDir); err != nil {
		t.Fatal(err)
	}

	rep2, err := sc.Scan([]string{wsDir}, appsDir)
	if err != nil {
		t.Fatalf("Scan 2 failed: %v", err)
	}
	if rep2.Relinked != 1 || rep2.Added != 0 {
		t.Fatalf("expected 1 relinked and 0 added, got %+v", rep2)
	}

	all, err := repo.ListProjects(model.ProjectFilter{})
	if err != nil {
		t.Fatal(err)
	}
	if len(all) != 2 {
		t.Fatalf("expected still 2 total projects after move, got %d", len(all))
	}
	relinkedView, err := repo.GetProject(meta.ProjectID)
	if err != nil {
		t.Fatal(err)
	}
	if relinkedView.LocalDir != movedWebDir || relinkedView.Status != config.StatusReady {
		t.Fatalf("expected relinked local_dir %q and status ready, got %+v", movedWebDir, relinkedView)
	}
}


func TestScannerRecentDaysAndExtraExclude(t *testing.T) {
	tmp := t.TempDir()
	ws := filepath.Join(tmp, "ws")
	recentDir := filepath.Join(ws, "recent-app")
	oldDir := filepath.Join(ws, "old-app")
	excludedDir := filepath.Join(ws, "agent-skills-master")

	for _, d := range []string{recentDir, oldDir, excludedDir} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(d, "package.json"), []byte(`{"name":"x"}`), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	// Age old-app package.json by 180 days (> 120 days = 4 months).
	oldTime := time.Now().Add(-180 * 24 * time.Hour)
	if err := os.Chtimes(filepath.Join(oldDir, "package.json"), oldTime, oldTime); err != nil {
		t.Fatal(err)
	}

	db, err := database.Open(filepath.Join(tmp, config.DatabaseFileName))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	repo := database.NewProjectRepo(db, filepath.Join(tmp, config.ThumbnailsDirName))
	sc := NewScanner(repo)

	rep, err := sc.ScanWithOptions(ScanOptions{
		WorkspaceDirs:    []string{ws},
		RecentDays:       120,
		ExtraExcludeDirs: []string{"agent-skills-master"},
	})
	if err != nil {
		t.Fatalf("ScanWithOptions failed: %v", err)
	}
	if rep.Added != 1 {
		t.Fatalf("expected only 1 recent non-excluded project added, got %+v", rep)
	}
	all, err := repo.ListProjects(model.ProjectFilter{})
	if err != nil {
		t.Fatal(err)
	}
	if len(all) != 1 || all[0].Name != "recent-app" {
		t.Fatalf("expected only recent-app, got %+v", all)
	}
}

func TestManifestLoadingAndVersionResolution(t *testing.T) {
	tmp := t.TempDir()
	projDir := filepath.Join(tmp, "ws", "hyb-manifest")
	if err := os.MkdirAll(projDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(projDir, "package.json"), []byte(`{"name":"hyb","version":"2.4.1"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	manifestJSON := `{
  "schema_version": 1,
  "slug": "hyb-manifest",
  "name": "Hybrid Manifest",
  "targets": [
    { "kind": "web", "name": "Web Portal", "prod_url": "https://hyb.example.com" },
    { "kind": "desktop", "name": "Desktop Win", "launch_entry": "Hyb.exe" }
  ]
}`
	if err := os.WriteFile(filepath.Join(projDir, config.ManifestFileName), []byte(manifestJSON), 0o644); err != nil {
		t.Fatal(err)
	}

	ver := ResolveProjectVersion(projDir, nil)
	if ver != "v2.4.1" {
		t.Fatalf("expected resolved version v2.4.1 from package.json, got %s", ver)
	}

	db, err := database.Open(filepath.Join(tmp, config.DatabaseFileName))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	repo := database.NewProjectRepo(db, filepath.Join(tmp, config.ThumbnailsDirName))
	sc := NewScanner(repo)
	if _, err := sc.Scan([]string{filepath.Join(tmp, "ws")}, ""); err != nil {
		t.Fatal(err)
	}

	all, err := repo.ListProjects(model.ProjectFilter{})
	if err != nil || len(all) != 1 {
		t.Fatalf("expected 1 project, got %+v (err=%v)", all, err)
	}
	if all[0].Type != config.ProjectTypeHybrid || len(all[0].Targets) != 2 || all[0].LiveURL != "https://hyb.example.com" {
		t.Fatalf("expected hybrid project with 2 manifest targets, got %+v", all[0])
	}
}

func TestScannerSkipsDeletedTombstoneAndOverlappingRoots(t *testing.T) {
	tmp := t.TempDir()
	parent := filepath.Join(tmp, "drive")
	child := filepath.Join(parent, "AllProjects")
	projDir := filepath.Join(child, "my-py-app")
	if err := os.MkdirAll(projDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(projDir, "pyproject.toml"), []byte("[project]\nname='my-py-app'\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	db, err := database.Open(filepath.Join(tmp, config.DatabaseFileName))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	repo := database.NewProjectRepo(db, filepath.Join(tmp, config.ThumbnailsDirName))
	sc := NewScanner(repo)

	// Overlapping parent + child roots must discover the project exactly once.
	if _, err := sc.Scan([]string{parent, child}, ""); err != nil {
		t.Fatal(err)
	}
	all, err := repo.ListProjects(model.ProjectFilter{})
	if err != nil || len(all) != 1 {
		t.Fatalf("expected 1 project from overlapping roots, got %d (err=%v)", len(all), err)
	}

	// Soft-delete the project; rescanning must NOT resurrect it.
	if err := repo.DeleteProject(all[0].ID); err != nil {
		t.Fatal(err)
	}
	if _, err := sc.Scan([]string{parent, child}, ""); err != nil {
		t.Fatal(err)
	}
	after, err := repo.ListProjects(model.ProjectFilter{})
	if err != nil || len(after) != 0 {
		t.Fatalf("expected deleted project NOT to resurrect on rescan, got %d (err=%v)", len(after), err)
	}
}

func TestCleanMarkdownSummaryUnescapesHTML(t *testing.T) {
	raw := "# Admin One &mdash; Free Vue 3\n\n<p>Dashboard &amp; UI kit &quot;dark&quot;</p>"
	got := CleanMarkdownSummary(raw, 120)
	want := `Admin One — Free Vue 3 Dashboard & UI kit "dark"`
	if got != want {
		t.Fatalf("expected %q, got %q", want, got)
	}
}

func TestProbeDirectoryExtractsNestedStacksGitRemoteAndMetadata(t *testing.T) {
	tmp := t.TempDir()
	projDir := filepath.Join(tmp, "sample-wails-app")
	if err := os.MkdirAll(filepath.Join(projDir, "frontend"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(projDir, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(filepath.Join(projDir, "go.mod"), []byte("module example.com/sample\n\nrequire github.com/wailsapp/wails/v2 v2.10.1\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(projDir, config.WailsConfig), []byte(`{"name":"sample-wails-app","info":{"productName":"Sample Wails App","productVersion":"1.4.0"}}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(projDir, "frontend", "package.json"), []byte(`{"name":"frontend","dependencies":{"react":"19","vite":"6","tailwindcss":"4","typescript":"5"}}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(projDir, ".git", "config"), []byte("[remote \"upstream\"]\n\turl = https://github.com/acme/sample-wails.git\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(projDir, "README.md"), []byte("[![Build](https://img.shields.io/badge/build-pass-green)](https://ci.example.com)\nhttps://ci.example.com\n# Sample Wails App\nQuản lý media và video tự động."), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(projDir, "start.bat"), []byte("@echo off\nwails dev\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	probe, err := ProbeDirectory(projDir, []string{"Business & Web Apps", "Video & Media AI"})
	if err != nil {
		t.Fatalf("ProbeDirectory failed: %v", err)
	}
	if probe.Name != "Sample Wails App" {
		t.Fatalf("expected name Sample Wails App, got %q", probe.Name)
	}
	if probe.Type != config.ProjectTypeDesktop {
		t.Fatalf("expected desktop type for wails project, got %q", probe.Type)
	}
	if probe.GitRemote != "https://github.com/acme/sample-wails.git" {
		t.Fatalf("expected fallback remote url, got %q", probe.GitRemote)
	}
	for _, wantStack := range []string{"Go", "Wails", "React", "Vite", "Tailwind CSS", "TypeScript"} {
		if !containsWord(probe.TechStack, wantStack) {
			t.Fatalf("expected tech_stack %q to include %q", probe.TechStack, wantStack)
		}
	}
	if probe.Summary != "Sample Wails App Quản lý media và video tự động." {
		t.Fatalf("unexpected cleaned summary: %q", probe.Summary)
	}
	if probe.Category != "Video & Media AI" {
		t.Fatalf("expected category Video & Media AI, got %q", probe.Category)
	}
	if probe.Version != "v1.4.0" || !probe.HasBat || probe.BatFile != "start.bat" {
		t.Fatalf("unexpected probe version/bat flags: %+v", probe)
	}
}

func containsWord(haystack, needle string) bool {
	return len(needle) > 0 && (haystack == needle ||
		len(haystack) > len(needle) && (haystack[:len(needle)] == needle ||
			haystack[len(haystack)-len(needle):] == needle ||
			findSubstr(haystack, needle)))
}

func findSubstr(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
