package cloud

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"t-deck/internal/config"
	"t-deck/internal/database"
	"t-deck/internal/model"
)

type memoryStore struct {
	mu      sync.Mutex
	objects map[string][]byte
}

func newMemoryStore() *memoryStore {
	return &memoryStore{objects: make(map[string][]byte)}
}

func (m *memoryStore) UploadFile(_ context.Context, key string, localPath string, _ string) error {
	data, err := os.ReadFile(localPath)
	if err != nil {
		return err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.objects[key] = data
	return nil
}

func (m *memoryStore) DownloadFile(_ context.Context, key string, localPath string) error {
	m.mu.Lock()
	data, ok := m.objects[key]
	m.mu.Unlock()
	if !ok {
		return ErrObjectNotFound
	}
	if err := os.MkdirAll(filepath.Dir(localPath), 0o755); err != nil {
		return err
	}
	return os.WriteFile(localPath, data, 0o644)
}

func (m *memoryStore) DeleteObject(_ context.Context, key string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.objects, key)
	return nil
}

func (m *memoryStore) PresignGetURL(_ context.Context, key string, expiry time.Duration) (string, error) {
	return fmt.Sprintf("https://presigned.example.com/%s?expires=%d", key, int(expiry.Seconds())), nil
}

func (m *memoryStore) PutJSON(_ context.Context, key string, payload any) error {
	data, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.objects[key] = data
	return nil
}

func (m *memoryStore) GetJSON(_ context.Context, key string, out any) (bool, error) {
	m.mu.Lock()
	data, ok := m.objects[key]
	m.mu.Unlock()
	if !ok {
		return false, nil
	}
	if err := json.Unmarshal(data, out); err != nil {
		return false, err
	}
	return true, nil
}

func (m *memoryStore) ObjectExists(_ context.Context, key string) (bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	_, ok := m.objects[key]
	return ok, nil
}

func (m *memoryStore) TestConnection(_ context.Context) error {
	return nil
}

func TestPublishAndHydrateAcrossMachines(t *testing.T) {
	ctx := context.Background()
	store := newMemoryStore()

	// --- Machine A: Build & Publish Desktop App + Web App + Thumbnail ---
	tmpA := t.TempDir()
	dbA, err := database.Open(filepath.Join(tmpA, config.DatabaseFileName))
	if err != nil {
		t.Fatal(err)
	}
	defer dbA.Close()
	thumbsA := filepath.Join(tmpA, config.ThumbnailsDirName)
	repoA := database.NewProjectRepo(dbA, thumbsA)

	// Prepare desktop app build folder with exe + dll.
	buildDir := filepath.Join(tmpA, "dist-app")
	if err := os.MkdirAll(buildDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(buildDir, "AudioTranslator.exe"), []byte("MZ-APP"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(buildDir, "helper.dll"), []byte("DLL"), 0o644); err != nil {
		t.Fatal(err)
	}

	deskProj := model.Project{
		ID:                 "desk-1",
		Slug:               "audio-translator",
		Name:               "Audio Translator",
		Type:               config.ProjectTypeDesktop,
		Summary:            "Dịch audio đa ngôn ngữ",
		Tags:               []string{"audio", "ai"},
		ThumbnailUpdatedAt: 200,
		UpdatedAt:          200,
	}
	if err := repoA.UpsertProject(deskProj, &model.LocalState{
		ProjectID: deskProj.ID,
		LocalDir:  buildDir,
		Status:    config.StatusReady,
	}); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(thumbsA, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(thumbsA, deskProj.ID+".png"), []byte("PNGDATA"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Also add a web project on Machine A.
	webProj := model.Project{
		ID:        "web-1",
		Slug:      "t-hub-cockpit",
		Name:      "T-Hub Cockpit",
		Type:      config.ProjectTypeWeb,
		GitRemote: "https://github.com/example/t-hub.git",
		BatFile:   config.DefaultBatFile,
		UpdatedAt: 150,
	}
	if err := repoA.UpsertProject(webProj, &model.LocalState{
		ProjectID: webProj.ID,
		LocalDir:  filepath.Join(tmpA, "t-hub"),
		Status:    config.StatusReady,
	}); err != nil {
		t.Fatal(err)
	}

	published, err := PublishDesktopRelease(ctx, store, repoA, thumbsA, model.PublishRequest{
		ProjectID:    deskProj.ID,
		SourcePath:   buildDir,
		Version:      "v1.2.0",
		ReleaseNotes: "Initial release",
	})
	if err != nil {
		t.Fatalf("PublishDesktopRelease failed: %v", err)
	}
	if published.R2ReleaseKey != "releases/audio-translator/v1.2.0.zip" {
		t.Fatalf("unexpected release key: %s", published.R2ReleaseKey)
	}

	// --- Machine B: Fresh DB (New PC Hydration) ---
	tmpB := t.TempDir()
	dbB, err := database.Open(filepath.Join(tmpB, config.DatabaseFileName))
	if err != nil {
		t.Fatal(err)
	}
	defer dbB.Close()
	thumbsB := filepath.Join(tmpB, config.ThumbnailsDirName)
	repoB := database.NewProjectRepo(dbB, thumbsB)

	syncRep, err := SyncCatalogWithR2(ctx, store, repoB, thumbsB)
	if err != nil {
		t.Fatalf("SyncCatalogWithR2 on Machine B failed: %v", err)
	}
	if syncRep.Pulled != 2 || syncRep.ThumbnailsDown != 1 {
		t.Fatalf("expected 2 pulled and 1 thumbnail down, got %+v", syncRep)
	}

	// Verify cloud_only status on Machine B before hydration.
	deskB, err := repoB.GetProject(deskProj.ID)
	if err != nil {
		t.Fatal(err)
	}
	if deskB.Status != config.StatusCloudOnly || deskB.ThumbnailURL == "" {
		t.Fatalf("expected cloud_only + downloaded thumbnail, got %+v", deskB)
	}

	settingsB := config.Settings{
		WorkspaceDirs: []string{filepath.Join(tmpB, "workspace")},
		AppsDir:       filepath.Join(tmpB, "TDeckApps"),
	}

	// Hydrate Desktop App -> downloads zip, extracts, detects exe, marks ready.
	hydratedDesk, err := HydrateProjectOnNewPC(ctx, store, repoB, settingsB, deskProj.ID, nil)
	if err != nil {
		t.Fatalf("Hydrate desktop failed: %v", err)
	}
	if hydratedDesk.Status != config.StatusReady || hydratedDesk.ExeName != "AudioTranslator.exe" {
		t.Fatalf("unexpected hydrated desktop: %+v", hydratedDesk)
	}
	if _, err := os.Stat(filepath.Join(hydratedDesk.LocalDir, "helper.dll")); err != nil {
		t.Fatalf("expected helper.dll extracted: %v", err)
	}

	// Hydrate Web App with fake git clone containing .env.example -> marks missing_env.
	hydratedWeb, err := HydrateProjectOnNewPC(ctx, store, repoB, settingsB, webProj.ID, func(_ context.Context, _ string, targetDir string) error {
		if err := os.MkdirAll(targetDir, 0o755); err != nil {
			return err
		}
		return os.WriteFile(filepath.Join(targetDir, config.EnvExampleFile), []byte("API_KEY=\n"), 0o644)
	})
	if err != nil {
		t.Fatalf("Hydrate web failed: %v", err)
	}
	if hydratedWeb.Status != config.StatusMissingEnv {
		t.Fatalf("expected missing_env after cloning repo with .env.example, got %s", hydratedWeb.Status)
	}
}

func TestSyncCategoriesAndTombstonesPreventResurrection(t *testing.T) {
	ctx := context.Background()
	store := newMemoryStore()

	tmpA := t.TempDir()
	dbA, err := database.Open(filepath.Join(tmpA, config.DatabaseFileName))
	if err != nil {
		t.Fatal(err)
	}
	defer dbA.Close()
	thumbsA := filepath.Join(tmpA, config.ThumbnailsDirName)
	repoA := database.NewProjectRepo(dbA, thumbsA)

	cat, err := repoA.CreateCategory("Web Releases")
	if err != nil {
		t.Fatal(err)
	}
	p1 := model.Project{
		ID:        "p-del-1",
		Slug:      "temp-tool",
		Name:      "Temp Tool",
		Type:      config.ProjectTypeWeb,
		Category:  cat.Name,
		UpdatedAt: 100,
	}
	p2 := model.Project{
		ID:        "p-keep-2",
		Slug:      "keep-tool",
		Name:      "Keep Tool",
		Type:      config.ProjectTypeWeb,
		Category:  cat.Name,
		UpdatedAt: 100,
	}
	for _, p := range []model.Project{p1, p2} {
		if err := repoA.UpsertProject(p, &model.LocalState{ProjectID: p.ID, LocalDir: tmpA, Status: config.StatusReady}); err != nil {
			t.Fatal(err)
		}
	}

	if _, err := SyncCatalogWithR2(ctx, store, repoA, thumbsA); err != nil {
		t.Fatal(err)
	}

	// Machine B syncs initial state (receives category + 2 projects).
	tmpB := t.TempDir()
	dbB, err := database.Open(filepath.Join(tmpB, config.DatabaseFileName))
	if err != nil {
		t.Fatal(err)
	}
	defer dbB.Close()
	thumbsB := filepath.Join(tmpB, config.ThumbnailsDirName)
	repoB := database.NewProjectRepo(dbB, thumbsB)

	if _, err := SyncCatalogWithR2(ctx, store, repoB, thumbsB); err != nil {
		t.Fatal(err)
	}
	catsB, err := repoB.ListCategories()
	if err != nil || len(catsB) != 1 || catsB[0].Name != "Web Releases" {
		t.Fatalf("expected category synced to Machine B, got %+v (err=%v)", catsB, err)
	}

	// Machine A soft-deletes p1 and syncs.
	if err := repoA.SoftDeleteProjectAt(p1.ID, time.Now().Unix()+10); err != nil {
		t.Fatal(err)
	}
	if _, err := SyncCatalogWithR2(ctx, store, repoA, thumbsA); err != nil {
		t.Fatal(err)
	}

	// Machine B syncs -> p1 MUST NOT resurrect and MUST be deleted on Machine B too!
	if _, err := SyncCatalogWithR2(ctx, store, repoB, thumbsB); err != nil {
		t.Fatal(err)
	}
	activeB, err := repoB.ListAllProjects()
	if err != nil {
		t.Fatal(err)
	}
	if len(activeB) != 1 || activeB[0].ID != p2.ID {
		t.Fatalf("expected deleted project not to resurrect on Machine B, got %+v", activeB)
	}
}

func TestHybridPublishUpdateRollbackHealthAndLinks(t *testing.T) {
	ctx := context.Background()
	store := newMemoryStore()

	// 1. Web Health & Version Drift check against httptest.Server
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/version.json" {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"version":"v1.0.0"}`))
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("OK"))
	}))
	defer srv.Close()

	healthRes := CheckWebHealth(ctx, model.ProjectTarget{
		ID:         "tgt-web",
		ProjectID:  "proj-hyb",
		Kind:       config.ProjectTypeWeb,
		ProdURL:    srv.URL,
		HealthURL:  srv.URL + "/health",
		VersionURL: srv.URL + "/version.json",
	}, "v1.1.0")
	if !healthRes.Online || !healthRes.Drift || healthRes.LiveVersion != "v1.0.0" {
		t.Fatalf("expected online + version drift v1.0.0 vs v1.1.0, got %+v", healthRes)
	}

	// 2. Setup DB + Hybrid Project (Web static_zip + Desktop portable_zip)
	tmp := t.TempDir()
	db, err := database.Open(filepath.Join(tmp, config.DatabaseFileName))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	thumbs := filepath.Join(tmp, config.ThumbnailsDirName)
	repo := database.NewProjectRepo(db, thumbs)

	proj := model.Project{
		ID:        "proj-hyb",
		Slug:      "hyb-app",
		Name:      "Hybrid App",
		Type:      config.ProjectTypeHybrid,
		UpdatedAt: 100,
	}
	if err := repo.UpsertProject(proj, nil); err != nil {
		t.Fatal(err)
	}
	webTgt, _ := repo.UpsertTarget(model.ProjectTarget{ProjectID: proj.ID, Kind: config.ProjectTypeWeb, Name: "Web", ProdURL: srv.URL})
	deskTgt, _ := repo.UpsertTarget(model.ProjectTarget{ProjectID: proj.ID, Kind: config.ProjectTypeDesktop, Name: "Win"})

	// Publish Web static_zip
	webDist := filepath.Join(tmp, "dist")
	if err := os.MkdirAll(webDist, 0o755); err != nil {
		t.Fatal(err)
	}
	_ = os.WriteFile(filepath.Join(webDist, "index.html"), []byte("<h1>v1.1.0</h1>"), 0o644)
	settings := config.Settings{
		AppsDir: filepath.Join(tmp, "apps"),
		R2:      config.R2Config{PublicDomain: "https://cdn.example.com"},
	}

	webRel, err := PublishRelease(ctx, store, repo, settings, thumbs, model.PublishReleaseRequest{
		ProjectID:  proj.ID,
		TargetID:   webTgt.ID,
		SourcePath: webDist,
		Version:    "v1.1.0",
		Channel:    config.ReleaseChannelStable,
		AssetKind:  "static_zip",
	}, nil)
	if err != nil {
		t.Fatalf("PublishRelease web failed: %v", err)
	}
	if webRel.SHA256 == "" || !strings.Contains(webRel.R2Key, "/web/v1.1.0/") {
		t.Fatalf("unexpected web release: %+v", webRel)
	}

	// Test PublicDomain URL & Presigned fallback
	pubURL, err := GenerateReleaseDownloadURL(ctx, store, repo, settings, webRel.ID, 24)
	if err != nil || !strings.HasPrefix(pubURL, "https://cdn.example.com/") {
		t.Fatalf("expected public CDN URL, got %q err=%v", pubURL, err)
	}
	noCDNSettings := config.Settings{AppsDir: settings.AppsDir}
	presignedURL, err := GenerateReleaseDownloadURL(ctx, store, repo, noCDNSettings, webRel.ID, 12)
	if err != nil || !strings.Contains(presignedURL, "presigned.example.com") {
		t.Fatalf("expected presigned URL, got %q err=%v", presignedURL, err)
	}

	// Publish Desktop v1.0.0 and v1.1.0 -> Hydrate/Update -> Rollback
	buildV1 := filepath.Join(tmp, "build-v1")
	buildV2 := filepath.Join(tmp, "build-v2")
	_ = os.MkdirAll(buildV1, 0o755)
	_ = os.MkdirAll(buildV2, 0o755)
	_ = os.WriteFile(filepath.Join(buildV1, "Hyb.exe"), []byte("V1"), 0o644)
	_ = os.WriteFile(filepath.Join(buildV2, "Hyb.exe"), []byte("V2"), 0o644)

	relV1, err := PublishRelease(ctx, store, repo, settings, thumbs, model.PublishReleaseRequest{
		ProjectID:  proj.ID,
		TargetID:   deskTgt.ID,
		SourcePath: buildV1,
		Version:    "v1.0.0",
		Channel:    config.ReleaseChannelStable,
		AssetKind:  "portable_zip",
	}, nil)
	if err != nil {
		t.Fatal(err)
	}
	relV2, err := PublishRelease(ctx, store, repo, settings, thumbs, model.PublishReleaseRequest{
		ProjectID:  proj.ID,
		TargetID:   deskTgt.ID,
		SourcePath: buildV2,
		Version:    "v1.1.0",
		Channel:    config.ReleaseChannelStable,
		AssetKind:  "portable_zip",
	}, nil)
	if err != nil {
		t.Fatal(err)
	}

	// Verify latest.json auto-updater manifest generated
	var latestManifest model.AutoUpdateManifest
	okLatest, err := store.GetJSON(ctx, "releases/hyb-app/stable/latest.json", &latestManifest)
	if err != nil || !okLatest || latestManifest.Version != "v1.1.0" {
		t.Fatalf("expected latest.json v1.1.0, ok=%v err=%v manifest=%+v", okLatest, err, latestManifest)
	}

	// Install v1.0.0 then update to v1.1.0
	if _, err := UpdateDesktopRelease(ctx, store, repo, settings, proj.ID, deskTgt.ID, relV1.ID); err != nil {
		t.Fatalf("install v1 failed: %v", err)
	}
	v2View, err := UpdateDesktopRelease(ctx, store, repo, settings, proj.ID, deskTgt.ID, relV2.ID)
	if err != nil {
		t.Fatalf("update v2 failed: %v", err)
	}
	if v2View.InstalledVersion != "v1.1.0" {
		t.Fatalf("expected installed_version v1.1.0, got %s", v2View.InstalledVersion)
	}
	exeBytes, _ := os.ReadFile(filepath.Join(v2View.LocalDir, "Hyb.exe"))
	if string(exeBytes) != "V2" {
		t.Fatalf("expected V2 binary on disk, got %q", string(exeBytes))
	}

	// Rollback via local .__bak swap
	rbView, err := RollbackDesktopRelease(ctx, store, repo, settings, proj.ID, deskTgt.ID)
	if err != nil {
		t.Fatalf("rollback failed: %v", err)
	}
	exeRbBytes, _ := os.ReadFile(filepath.Join(rbView.LocalDir, "Hyb.exe"))
	if string(exeRbBytes) != "V1" || rbView.InstalledVersion != "v1.0.0" {
		t.Fatalf("expected V1 restored on rollback, got bytes=%q ver=%s", string(exeRbBytes), rbView.InstalledVersion)
	}

	// Tampered SHA-256 must abort before touching installed folder!
	tampered := relV2
	tampered.ID = "rel-bad-sha"
	tampered.SHA256 = "0000000000000000000000000000000000000000000000000000000000000000"
	_, _ = repo.UpsertRelease(tampered)
	if _, err := UpdateDesktopRelease(ctx, store, repo, settings, proj.ID, deskTgt.ID, tampered.ID); err == nil {
		t.Fatalf("expected SHA-256 mismatch error, got nil")
	}
	afterBadBytes, _ := os.ReadFile(filepath.Join(rbView.LocalDir, "Hyb.exe"))
	if string(afterBadBytes) != "V1" {
		t.Fatalf("expected V1 untouched after failed checksum, got %q", string(afterBadBytes))
	}
}

