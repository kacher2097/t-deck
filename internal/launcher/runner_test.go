package launcher

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"t-deck/internal/config"
	"t-deck/internal/database"
	"t-deck/internal/model"
)

type fakeStarter struct {
	calls []string
}

func (f *fakeStarter) Start(dir string, name string, args ...string) error {
	f.calls = append(f.calls, dir+"|"+name+"|"+strings.Join(args, " "))
	return nil
}

func (f *fakeStarter) LookPath(file string) (string, error) {
	return file, nil
}

func TestLaunchWebPreflightAndStartBat(t *testing.T) {
	tmp := t.TempDir()
	db, err := database.Open(filepath.Join(tmp, config.DatabaseFileName))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	repo := database.NewProjectRepo(db, filepath.Join(tmp, config.ThumbnailsDirName))
	projDir := filepath.Join(tmp, "my-web")
	if err := os.MkdirAll(projDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(projDir, config.EnvExampleFile), []byte("PORT=\nSECRET=\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	p := model.Project{
		ID:        "web-1",
		Slug:      "my-web",
		Name:      "My Web",
		Type:      config.ProjectTypeWeb,
		BatFile:   config.DefaultBatFile,
		UpdatedAt: 100,
	}
	if err := repo.UpsertProject(p, &model.LocalState{
		ProjectID: p.ID,
		LocalDir:  projDir,
		Status:    config.StatusMissingEnv,
	}); err != nil {
		t.Fatal(err)
	}

	starter := &fakeStarter{}
	runner := NewRunnerWithStarter(repo, starter)

	// 1. Missing .env with .env.example -> creates .env, opens notepad, does NOT run start.bat.
	res1, err := runner.LaunchProject(p.ID)
	if err != nil {
		t.Fatalf("Launch 1 failed: %v", err)
	}
	if res1.Status != config.LaunchStatusEnvCreatedNeedsConfig {
		t.Fatalf("expected %s, got %s", config.LaunchStatusEnvCreatedNeedsConfig, res1.Status)
	}
	if _, err := os.Stat(filepath.Join(projDir, config.EnvFile)); err != nil {
		t.Fatalf("expected .env to be created: %v", err)
	}
	if len(starter.calls) != 1 || !strings.Contains(starter.calls[0], "notepad.exe") {
		t.Fatalf("expected notepad launch only, got %v", starter.calls)
	}

	// 2. Generate start.bat template, now .env exists -> runs start.bat in projDir.
	if _, err := WriteStartBat(projDir, config.DefaultBatFile, p.Name, "node"); err != nil {
		t.Fatalf("WriteStartBat failed: %v", err)
	}
	starter.calls = nil
	res2, err := runner.LaunchProject(p.ID)
	if err != nil {
		t.Fatalf("Launch 2 failed: %v", err)
	}
	if res2.Status != config.LaunchStatusLaunched {
		t.Fatalf("expected LAUNCHED, got %+v", res2)
	}
	if len(starter.calls) != 1 || !strings.Contains(starter.calls[0], "cmd.exe /K start.bat") {
		t.Fatalf("unexpected start call: %v", starter.calls)
	}

	updated, err := repo.GetProject(p.ID)
	if err != nil {
		t.Fatal(err)
	}
	if updated.Status != config.StatusReady || updated.LastOpenedAt == 0 {
		t.Fatalf("expected status ready and last_opened_at > 0, got %+v", updated)
	}
}

func TestLaunchDesktopResolvesNestedExe(t *testing.T) {
	tmp := t.TempDir()
	db, err := database.Open(filepath.Join(tmp, config.DatabaseFileName))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	repo := database.NewProjectRepo(db, filepath.Join(tmp, config.ThumbnailsDirName))
	projDir := filepath.Join(tmp, "s3-desk")
	binDir := filepath.Join(projDir, "bin")
	if err := os.MkdirAll(binDir, 0o755); err != nil {
		t.Fatal(err)
	}
	exePath := filepath.Join(binDir, "S3Desk.exe")
	if err := os.WriteFile(exePath, []byte("MZ"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Even when ExeName only stores "S3Desk.exe" without "bin\", runner resolves bin\S3Desk.exe.
	p := model.Project{
		ID:        "desk-1",
		Slug:      "s3-desk",
		Name:      "S3 Desk",
		Type:      config.ProjectTypeDesktop,
		ExeName:   "S3Desk.exe",
		UpdatedAt: 100,
	}
	if err := repo.UpsertProject(p, &model.LocalState{
		ProjectID: p.ID,
		LocalDir:  projDir,
		Status:    config.StatusReady,
	}); err != nil {
		t.Fatal(err)
	}

	starter := &fakeStarter{}
	runner := NewRunnerWithStarter(repo, starter)
	res, err := runner.LaunchProject(p.ID)
	if err != nil {
		t.Fatalf("LaunchProject desktop failed: %v", err)
	}
	if res.Status != config.LaunchStatusLaunched || res.OpenedFile != exePath {
		t.Fatalf("expected launched %s, got %+v", exePath, res)
	}
}

func TestParseNetstatListeningPortsAndResolveGlob(t *testing.T) {
	sample := `
Active Connections

  Proto  Local Address          Foreign Address        State           PID
  TCP    0.0.0.0:5173           0.0.0.0:0              LISTENING       1234
  TCP    127.0.0.1:8080         0.0.0.0:0              LISTENING       1234
  TCP    [::1]:3000             [::]:0                 LISTENING       5678
  TCP    127.0.0.1:9999         127.0.0.1:50000        ESTABLISHED     1234
`
	parsed := ParseNetstatListeningPorts(sample)
	if len(parsed[1234]) != 2 || parsed[1234][0] != 5173 || parsed[1234][1] != 8080 {
		t.Fatalf("unexpected ports for PID 1234: %+v", parsed[1234])
	}
	if len(parsed[5678]) != 1 || parsed[5678][0] != 3000 {
		t.Fatalf("unexpected ports for PID 5678: %+v", parsed[5678])
	}

	tmp := t.TempDir()
	sub := filepath.Join(tmp, "release")
	_ = os.MkdirAll(sub, 0o755)
	_ = os.WriteFile(filepath.Join(sub, "App-1.0.exe"), []byte("1"), 0o644)
	resolved, err := ResolveArtifactGlob(tmp, filepath.Join("release", "*.exe"))
	if err != nil || !strings.HasSuffix(resolved, "App-1.0.exe") {
		t.Fatalf("expected resolved App-1.0.exe, got %q err=%v", resolved, err)
	}
}
