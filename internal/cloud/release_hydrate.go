package cloud

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"t-deck/internal/apperr"
	"t-deck/internal/config"
	"t-deck/internal/fsutil"
	"t-deck/internal/model"
	"t-deck/internal/scanner"
)

// GitCloner abstracts git clone for testability.
type GitCloner func(ctx context.Context, remoteURL string, targetDir string) error

// DefaultGitClone invokes `git clone` into targetDir without flashing a console window.
func DefaultGitClone(ctx context.Context, remoteURL string, targetDir string) error {
	cmd := exec.CommandContext(ctx, "git", "clone", remoteURL, targetDir)
	hideCloudCmdWindow(cmd)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("git clone failed: %w (%s)", err, strings.TrimSpace(string(out)))
	}
	return nil
}

// HydrateProjectOnNewPC installs a cloud_only desktop app from R2 or clones a web app via git.
func HydrateProjectOnNewPC(
	ctx context.Context,
	store ObjectStore,
	repo ProjectStore,
	settings config.Settings,
	projectID string,
	cloner GitCloner,
) (model.ProjectView, error) {
	p, found, err := repo.GetProjectModel(projectID)
	if err != nil {
		return model.ProjectView{}, err
	}
	if !found {
		return model.ProjectView{}, apperr.NotFound(fmt.Sprintf("không tìm thấy dự án %q", projectID))
	}

	if p.Type == config.ProjectTypeDesktop {
		return hydrateDesktop(ctx, store, repo, settings, p)
	}
	return hydrateWeb(ctx, repo, settings, p, cloner)
}

func hydrateDesktop(
	ctx context.Context,
	store ObjectStore,
	repo ProjectStore,
	settings config.Settings,
	p model.Project,
) (model.ProjectView, error) {
	if strings.TrimSpace(settings.AppsDir) == "" {
		return model.ProjectView{}, apperr.Validation("chưa cấu hình thư mục Apps (apps_dir) trong Settings")
	}
	if strings.TrimSpace(p.R2ReleaseKey) == "" {
		return model.ProjectView{}, apperr.NotFound("dự án chưa có bản release nào trên R2")
	}

	targetDir := filepath.Join(settings.AppsDir, p.Slug)
	tmpZipPath := filepath.Join(os.TempDir(), fmt.Sprintf("hydrate-%s-%d.zip", p.Slug, time.Now().UnixNano()))
	defer os.Remove(tmpZipPath)

	if err := store.DownloadFile(ctx, p.R2ReleaseKey, tmpZipPath); err != nil {
		return model.ProjectView{}, apperr.Cloud("tải release từ R2 thất bại", err)
	}

	if err := os.MkdirAll(targetDir, 0o755); err != nil {
		return model.ProjectView{}, apperr.IO("tạo thư mục cài đặt thất bại", err)
	}
	if err := unzipSecure(tmpZipPath, targetDir); err != nil {
		return model.ProjectView{}, apperr.IO("giải nén release thất bại", err)
	}

	if err := scanner.WriteMetaFile(targetDir, model.ProjectMeta{
		ProjectID:        p.ID,
		Slug:             p.Slug,
		Type:             p.Type,
		InstalledVersion: p.LatestVersion,
	}); err != nil {
		return model.ProjectView{}, apperr.IO("ghi file metadata thất bại", err)
	}

	if p.ExeName == "" || !fsutil.FileExists(filepath.Join(targetDir, p.ExeName)) {
		if detectedExe := detectExeRecursive(targetDir); detectedExe != "" {
			p.ExeName = detectedExe
			p.UpdatedAt = time.Now().Unix()
			_ = repo.UpsertProject(p, nil)
		}
	}

	if err := repo.UpsertLocalState(model.LocalState{
		ProjectID:        p.ID,
		LocalDir:         targetDir,
		Status:           config.StatusReady,
		InstalledVersion: p.LatestVersion,
		InstalledAt:      time.Now().Unix(),
	}); err != nil {
		return model.ProjectView{}, err
	}
	if relRepo, ok := repo.(interface {
		UpdateInstalledRelease(projectID, version, releaseID string) error
	}); ok && p.LatestVersion != "" {
		_ = relRepo.UpdateInstalledRelease(p.ID, p.LatestVersion, "")
	}

	return repo.GetProject(p.ID)
}

func hydrateWeb(
	ctx context.Context,
	repo ProjectStore,
	settings config.Settings,
	p model.Project,
	cloner GitCloner,
) (model.ProjectView, error) {
	if len(settings.WorkspaceDirs) == 0 {
		return model.ProjectView{}, apperr.Validation("chưa cấu hình thư mục Workspace (workspace_dirs) trong Settings")
	}
	if strings.TrimSpace(p.GitRemote) == "" {
		return model.ProjectView{}, apperr.Validation("dự án chưa có Git Remote URL để clone")
	}
	if cloner == nil {
		cloner = DefaultGitClone
	}

	targetDir := filepath.Join(settings.WorkspaceDirs[0], p.Slug)
	if _, err := os.Stat(targetDir); err == nil {
		return model.ProjectView{}, apperr.Conflict(fmt.Sprintf("thư mục đích đã tồn tại: %s", targetDir))
	}

	if err := os.MkdirAll(filepath.Dir(targetDir), 0o755); err != nil {
		return model.ProjectView{}, apperr.IO("tạo thư mục workspace thất bại", err)
	}
	if err := cloner(ctx, p.GitRemote, targetDir); err != nil {
		return model.ProjectView{}, err
	}

	if err := scanner.WriteMetaFile(targetDir, model.ProjectMeta{
		ProjectID: p.ID,
		Slug:      p.Slug,
		Type:      p.Type,
	}); err != nil {
		return model.ProjectView{}, apperr.IO("ghi file metadata thất bại", err)
	}

	if err := repo.UpsertLocalState(model.LocalState{
		ProjectID: p.ID,
		LocalDir:  targetDir,
		Status:    scanner.ComputeLocalStatus(targetDir, p.Type),
	}); err != nil {
		return model.ProjectView{}, err
	}

	return repo.GetProject(p.ID)
}

func detectExeRecursive(root string) string {
	var found string
	_ = filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		lower := strings.ToLower(d.Name())
		if strings.HasSuffix(lower, ".exe") && !strings.HasPrefix(lower, "unins") {
			if rel, err := filepath.Rel(root, path); err == nil {
				found = rel
				return filepath.SkipAll
			}
		}
		return nil
	})
	return found
}
