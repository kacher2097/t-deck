package main

import (
	"encoding/base64"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"t-deck/internal/config"
	"t-deck/internal/model"
)

func TestAppEndToEndLocalLifecycle(t *testing.T) {
	tmp := t.TempDir()
	mgr, err := config.NewSettingsManager(tmp)
	if err != nil {
		t.Fatal(err)
	}

	app := &App{settingsMgr: mgr}
	if err := app.initServices(); err != nil {
		t.Fatalf("initServices failed: %v", err)
	}
	defer app.shutdown(nil)

	projDir := filepath.Join(tmp, "ws", "demo-web")
	if err := os.MkdirAll(projDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(projDir, config.EnvFile), []byte("SECRET_KEY=my-secret\nPORT=3000\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	created, err := app.UpsertProject(model.ProjectInput{
		Name:      "Demo Web",
		Type:      config.ProjectTypeWeb,
		Summary:   "Demo service",
		Tags:      []string{"web", "demo"},
		TechStack: "React, Vite",
		LocalDir:  projDir,
	})
	if err != nil {
		t.Fatalf("UpsertProject failed: %v", err)
	}

	// Sync .env -> .env.example
	envRes, err := app.SyncEnvExample(created.ID)
	if err != nil {
		t.Fatalf("SyncEnvExample failed: %v", err)
	}
	if envRes.VarsCount != 2 {
		t.Fatalf("expected 2 vars synced, got %d", envRes.VarsCount)
	}

	// Generate start.bat
	if err := app.GenerateStartBat(created.ID, "node"); err != nil {
		t.Fatalf("GenerateStartBat failed: %v", err)
	}

	// Save PNG thumbnail
	pngBytes := []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A, 0x00, 0x00}
	b64 := "data:image/png;base64," + base64.StdEncoding.EncodeToString(pngBytes)
	thumbURL, err := app.SaveClipboardThumbnail(created.ID, b64)
	if err != nil {
		t.Fatalf("SaveClipboardThumbnail failed: %v", err)
	}
	if !strings.HasPrefix(thumbURL, "/thumbnails/"+created.ID+".png") {
		t.Fatalf("unexpected thumbnail URL: %s", thumbURL)
	}

	// Serve thumbnail via HTTP handler (with and without cache-bust query param)
	req := httptest.NewRequest(http.MethodGet, "/thumbnails/"+created.ID+".png", nil)
	rec := httptest.NewRecorder()
	app.serveThumbnail(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 serving thumbnail, got %d", rec.Code)
	}
	if cc := rec.Header().Get("Cache-Control"); cc != "no-store" {
		t.Fatalf("expected no-store without ?v=, got %q", cc)
	}

	reqCached := httptest.NewRequest(http.MethodGet, "/thumbnails/"+created.ID+".png?v=123", nil)
	recCached := httptest.NewRecorder()
	app.serveThumbnail(recCached, reqCached)
	if recCached.Code != http.StatusOK || !strings.Contains(recCached.Header().Get("Cache-Control"), "immutable") {
		t.Fatalf("expected 200 + immutable Cache-Control with ?v=, got code=%d cc=%q", recCached.Code, recCached.Header().Get("Cache-Control"))
	}

	// Verify directory traversal blocked
	badReq := httptest.NewRequest(http.MethodGet, "/thumbnails/../settings.json", nil)
	badRec := httptest.NewRecorder()
	app.serveThumbnail(badRec, badReq)
	if badRec.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for traversal attempt, got %d", badRec.Code)
	}
}

func TestScanTwelveMixedProjectsIgnoresNodeModules(t *testing.T) {
	tmp := t.TempDir()
	mgr, err := config.NewSettingsManager(filepath.Join(tmp, "appdata"))
	if err != nil {
		t.Fatal(err)
	}
	ws := filepath.Join(tmp, "workspace")
	apps := filepath.Join(tmp, "apps")

	// Create 7 web projects (with node_modules traps) and 5 desktop projects.
	for i := 1; i <= 7; i++ {
		pDir := filepath.Join(ws, fmt.Sprintf("web-app-%d", i))
		if err := os.MkdirAll(filepath.Join(pDir, "node_modules", "dep"), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(pDir, "package.json"), []byte(`{"name":"app"}`), 0o644); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(pDir, "node_modules", "dep", "package.json"), []byte(`{"name":"dep"}`), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	for i := 1; i <= 5; i++ {
		dDir := filepath.Join(apps, fmt.Sprintf("desk-app-%d", i))
		if err := os.MkdirAll(dDir, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dDir, fmt.Sprintf("Desk%d.exe", i)), []byte("MZ"), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	if _, err := mgr.Save(config.Settings{
		WorkspaceDirs: []string{ws},
		AppsDir:       apps,
	}); err != nil {
		t.Fatal(err)
	}

	app := &App{settingsMgr: mgr}
	if err := app.initServices(); err != nil {
		t.Fatal(err)
	}
	defer app.shutdown(nil)

	rep, err := app.ScanLocalWorkspaces()
	if err != nil {
		t.Fatalf("ScanLocalWorkspaces failed: %v", err)
	}
	if rep.Added != 12 || rep.Discovered != 12 {
		t.Fatalf("expected 12 mixed projects discovered/added without node_modules, got %+v", rep)
	}

	webs, err := app.ListProjects(model.ProjectFilter{Type: config.ProjectTypeWeb})
	if err != nil {
		t.Fatal(err)
	}
	desks, err := app.ListProjects(model.ProjectFilter{Type: config.ProjectTypeDesktop})
	if err != nil {
		t.Fatal(err)
	}
	if len(webs) != 7 || len(desks) != 5 {
		t.Fatalf("expected 7 web and 5 desktop projects, got web=%d desktop=%d", len(webs), len(desks))
	}
}

func TestInspectLocalDirectoryAndAutoFillUpsert(t *testing.T) {
	tmp := t.TempDir()
	mgr, err := config.NewSettingsManager(filepath.Join(tmp, "appdata"))
	if err != nil {
		t.Fatal(err)
	}
	app := &App{settingsMgr: mgr}
	if err := app.initServices(); err != nil {
		t.Fatal(err)
	}
	defer app.shutdown(nil)

	if _, err := app.CreateCategory("DevTools & Automation"); err != nil {
		t.Fatal(err)
	}

	projDir := filepath.Join(tmp, "cli-r2-tool")
	if err := os.MkdirAll(filepath.Join(projDir, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(projDir, "package.json"), []byte(`{"name":"cli-r2-tool","version":"0.3.0","description":"Sync files to R2 bucket","keywords":["r2","cli"],"homepage":"https://r2tool.example.com","dependencies":{"vite":"6","typescript":"5"}}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(projDir, ".git", "config"), []byte("[remote \"origin\"]\n\turl = https://github.com/user/cli-r2-tool.git\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	probe, err := app.InspectLocalDirectory(projDir)
	if err != nil {
		t.Fatalf("InspectLocalDirectory failed: %v", err)
	}
	if probe.Category != "DevTools & Automation" {
		t.Fatalf("expected matched category DevTools & Automation, got %q", probe.Category)
	}
	if probe.ProdURL != "https://r2tool.example.com" {
		t.Fatalf("expected prod_url from homepage, got %q", probe.ProdURL)
	}
	if probe.Summary != "Sync files to R2 bucket" {
		t.Fatalf("expected summary from package.json description, got %q", probe.Summary)
	}

	// Upserting with minimal fields should still auto-fill stack/git/tags and generate thumbnail.
	created, err := app.UpsertProject(model.ProjectInput{
		Name:     probe.Name,
		Type:     probe.Type,
		LocalDir: projDir,
	})
	if err != nil {
		t.Fatalf("UpsertProject failed: %v", err)
	}
	if created.TechStack == "" || created.GitRemote != "https://github.com/user/cli-r2-tool.git" {
		t.Fatalf("expected auto-filled tech_stack & git_remote, got %+v", created)
	}
	if created.ThumbnailUpdatedAt == 0 || created.ThumbnailURL == "" {
		t.Fatalf("expected auto-generated thumbnail on new project, got ts=%d url=%q", created.ThumbnailUpdatedAt, created.ThumbnailURL)
	}
}

func TestDeleteProjectThoroughCleanupAndLockSafety(t *testing.T) {
	tmp := t.TempDir()
	mgr, err := config.NewSettingsManager(filepath.Join(tmp, "appdata"))
	if err != nil {
		t.Fatal(err)
	}
	ws := filepath.Join(tmp, "workspace")
	if _, err := mgr.Save(config.Settings{WorkspaceDirs: []string{ws}}); err != nil {
		t.Fatal(err)
	}

	app := &App{settingsMgr: mgr}
	if err := app.initServices(); err != nil {
		t.Fatal(err)
	}
	defer app.shutdown(nil)

	projDir := filepath.Join(ws, "heavy-app")
	gitPack := filepath.Join(projDir, ".git", "objects", "pack")
	if err := os.MkdirAll(gitPack, 0o755); err != nil {
		t.Fatal(err)
	}
	roFile := filepath.Join(gitPack, "pack-abc.idx")
	if err := os.WriteFile(roFile, []byte("idx"), 0o644); err != nil {
		t.Fatal(err)
	}
	_ = os.Chmod(roFile, 0o444)

	created, err := app.UpsertProject(model.ProjectInput{
		Name:     "Heavy App",
		Type:     config.ProjectTypeWeb,
		LocalDir: projDir,
		ProdURL:  "https://heavy.example.com",
	})
	if err != nil {
		t.Fatalf("UpsertProject failed: %v", err)
	}

	// Add a release row to verify cascading soft-delete of targets & releases.
	targets, err := app.repo.ListProjectTargets(created.ID)
	if err != nil || len(targets) == 0 {
		t.Fatalf("expected initial target, got %v err=%v", targets, err)
	}
	if _, err := app.repo.UpsertRelease(model.Release{
		ProjectID: created.ID,
		TargetID:  targets[0].ID,
		Version:   "v1.0.0",
		Status:    "published",
	}); err != nil {
		t.Fatalf("UpsertRelease failed: %v", err)
	}

	// Delete with deleteLocalFiles=true must remove dir, thumbnail, targets, and releases thoroughly.
	if err := app.DeleteProject(created.ID, true); err != nil {
		t.Fatalf("DeleteProject failed: %v", err)
	}
	if _, statErr := os.Lstat(projDir); !os.IsNotExist(statErr) {
		t.Fatalf("expected project directory %s completely deleted, err=%v", projDir, statErr)
	}
	if _, statErr := os.Stat(mgr.ThumbnailPath(created.ID)); !os.IsNotExist(statErr) {
		t.Fatalf("expected local thumbnail removed, err=%v", statErr)
	}
	if _, getErr := app.GetProjectDetail(created.ID); getErr == nil {
		t.Fatalf("expected deleted project to be hidden from GetProjectDetail")
	}
	if isDel, delErr := app.repo.IsProjectDeleted(created.ID); delErr != nil || !isDel {
		t.Fatalf("expected tombstone IsProjectDeleted=true, got %v err=%v", isDel, delErr)
	}
	if remTargets, err := app.repo.ListProjectTargets(created.ID); err != nil || len(remTargets) != 0 {
		t.Fatalf("expected 0 active targets after delete, got %d err=%v", len(remTargets), err)
	}
	if remRels, err := app.repo.ListReleases(created.ID, ""); err != nil || len(remRels) != 0 {
		t.Fatalf("expected 0 active releases after delete, got %d err=%v", len(remRels), err)
	}

	// Workspace root itself must be protected from deletion.
	if app.isSafeProjectDeleteTarget(ws) {
		t.Fatalf("expected workspace root %s to be rejected by isSafeProjectDeleteTarget", ws)
	}
}
