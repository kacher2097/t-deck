package cloud

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
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

// UpdateDesktopRelease downloads, verifies SHA-256, checks exe lock, backs up old install, and atomically swaps in the new release.
func UpdateDesktopRelease(
	ctx context.Context,
	store ObjectStore,
	repo ProjectStore,
	settings config.Settings,
	projectID string,
	targetID string,
	releaseID string,
) (model.ProjectView, error) {
	p, found, err := repo.GetProjectModel(projectID)
	if err != nil {
		return model.ProjectView{}, err
	}
	if !found {
		return model.ProjectView{}, apperr.NotFound(fmt.Sprintf("không tìm thấy dự án %q", projectID))
	}
	relRepo, ok := repo.(releaseCatalogStore)
	if !ok {
		return model.ProjectView{}, apperr.NotReady("repository chưa hỗ trợ releases")
	}

	rel, err := resolveReleaseForUpdate(relRepo, p, projectID, targetID, releaseID)
	if err != nil {
		return model.ProjectView{}, err
	}

	view, _ := repo.GetProject(projectID)
	installDir := strings.TrimSpace(view.LocalDir)
	if installDir == "" {
		if strings.TrimSpace(settings.AppsDir) == "" {
			return model.ProjectView{}, apperr.Validation("chưa cấu hình thư mục Apps (apps_dir) trong Settings")
		}
		installDir = filepath.Join(settings.AppsDir, p.Slug)
	}

	if p.ExeName != "" && isWindowsProcessRunning(filepath.Base(p.ExeName)) {
		return model.ProjectView{}, apperr.Conflict(fmt.Sprintf("ứng dụng %s đang chạy; vui lòng đóng trước khi cập nhật", filepath.Base(p.ExeName)))
	}

	tmpArtifact := filepath.Join(os.TempDir(), fmt.Sprintf("tdeck-upd-%s-%d%s", p.Slug, time.Now().UnixNano(), filepath.Ext(rel.R2Key)))
	defer os.Remove(tmpArtifact)

	if err := store.DownloadFile(ctx, rel.R2Key, tmpArtifact); err != nil {
		return model.ProjectView{}, apperr.Cloud("tải release từ R2 thất bại", err)
	}

	if strings.TrimSpace(rel.SHA256) != "" {
		actualSHA, _ := computeFileSHA256(tmpArtifact)
		if !strings.EqualFold(actualSHA, rel.SHA256) {
			return model.ProjectView{}, apperr.Validation(fmt.Sprintf("checksum SHA-256 không khớp (expected %s, got %s)", rel.SHA256, actualSHA))
		}
	}

	if strings.EqualFold(filepath.Ext(rel.R2Key), ".exe") || rel.AssetKind == "installer_exe" {
		if err := os.MkdirAll(installDir, 0o755); err != nil {
			return model.ProjectView{}, apperr.IO("tạo thư mục cài đặt thất bại", err)
		}
		dstExe := filepath.Join(installDir, firstNonEmptyStr(rel.FileName, filepath.Base(rel.R2Key)))
		if err := copyFileSimple(tmpArtifact, dstExe); err != nil {
			return model.ProjectView{}, apperr.IO("sao chép file cài đặt thất bại", err)
		}
		_ = relRepo.UpdateInstalledRelease(p.ID, rel.Version, rel.ID)
		return repo.GetProject(p.ID)
	}

	if err := atomicSwapExtractedRelease(tmpArtifact, installDir, p, rel); err != nil {
		return model.ProjectView{}, err
	}

	if err := repo.UpsertLocalState(model.LocalState{
		ProjectID:          p.ID,
		LocalDir:           installDir,
		Status:             config.StatusReady,
		InstalledVersion:   rel.Version,
		InstalledReleaseID: rel.ID,
		InstalledAt:        time.Now().Unix(),
	}); err != nil {
		return model.ProjectView{}, err
	}
	_ = relRepo.UpdateInstalledRelease(p.ID, rel.Version, rel.ID)
	return repo.GetProject(p.ID)
}

// RollbackDesktopRelease restores the local .__bak directory swap or falls back to previous release.
func RollbackDesktopRelease(
	ctx context.Context,
	store ObjectStore,
	repo ProjectStore,
	settings config.Settings,
	projectID string,
	targetID string,
) (model.ProjectView, error) {
	view, err := repo.GetProject(projectID)
	if err != nil {
		return model.ProjectView{}, err
	}
	installDir := strings.TrimSpace(view.LocalDir)
	if installDir != "" && fsutil.DirExists(installDir+".__bak") {
		bakDir := installDir + ".__bak"
		tmpSwap := installDir + ".__swap"
		_ = os.RemoveAll(tmpSwap)
		if err := os.Rename(installDir, tmpSwap); err != nil {
			return model.ProjectView{}, apperr.IO("không thể rollback (thư mục đang bị khóa)", err)
		}
		if err := os.Rename(bakDir, installDir); err != nil {
			_ = os.Rename(tmpSwap, installDir)
			return model.ProjectView{}, apperr.IO("khôi phục bản backup thất bại", err)
		}
		_ = os.RemoveAll(tmpSwap)
		if meta, err := scanner.ReadMetaFile(installDir); err == nil && meta.InstalledVersion != "" {
			if relRepo, ok := repo.(releaseCatalogStore); ok {
				_ = relRepo.UpdateInstalledRelease(projectID, meta.InstalledVersion, "")
			}
		}
		return repo.GetProject(projectID)
	}

	relRepo, ok := repo.(releaseCatalogStore)
	if !ok || store == nil {
		return model.ProjectView{}, apperr.NotFound("không tìm thấy bản sao lưu .__bak cục bộ để rollback")
	}
	rels, err := relRepo.ListReleases(projectID, targetID)
	if err != nil || len(rels) < 2 {
		return model.ProjectView{}, apperr.NotFound("không có phiên bản cũ hơn để rollback")
	}
	return UpdateDesktopRelease(ctx, store, repo, settings, projectID, targetID, rels[1].ID)
}

func resolveReleaseForUpdate(relRepo releaseCatalogStore, p model.Project, projectID, targetID, releaseID string) (model.Release, error) {
	if strings.TrimSpace(releaseID) != "" {
		r, rFound, err := relRepo.GetRelease(releaseID)
		if err != nil {
			return model.Release{}, err
		}
		if !rFound {
			return model.Release{}, apperr.NotFound(fmt.Sprintf("không tìm thấy release %q", releaseID))
		}
		return r, nil
	}
	allRels, err := relRepo.ListReleases(projectID, targetID)
	if err != nil {
		return model.Release{}, err
	}
	for _, r := range allRels {
		if r.Status == "published" && r.R2Key != "" {
			return r, nil
		}
	}
	if p.R2ReleaseKey != "" {
		return model.Release{
			ProjectID: p.ID,
			Version:   p.LatestVersion,
			R2Key:     p.R2ReleaseKey,
			AssetKind: "portable_zip",
		}, nil
	}
	return model.Release{}, apperr.NotFound("không tìm thấy bản release trên R2 để cập nhật")
}

func atomicSwapExtractedRelease(zipPath, installDir string, p model.Project, rel model.Release) error {
	stagingDir := installDir + ".__staging"
	bakDir := installDir + ".__bak"
	_ = os.RemoveAll(stagingDir)
	if err := os.MkdirAll(stagingDir, 0o755); err != nil {
		return apperr.IO("tạo staging dir thất bại", err)
	}
	if err := unzipSecure(zipPath, stagingDir); err != nil {
		_ = os.RemoveAll(stagingDir)
		return apperr.IO("giải nén release thất bại", err)
	}

	_ = scanner.WriteMetaFile(stagingDir, model.ProjectMeta{
		ProjectID:        p.ID,
		Slug:             p.Slug,
		Type:             p.Type,
		InstalledVersion: rel.Version,
	})

	hadExisting := fsutil.DirExists(installDir)
	if hadExisting {
		_ = os.RemoveAll(bakDir)
		if err := os.Rename(installDir, bakDir); err != nil {
			_ = os.RemoveAll(stagingDir)
			return apperr.IO("sao lưu bản cũ thất bại (file có thể đang bị khóa)", err)
		}
	}
	if err := os.Rename(stagingDir, installDir); err != nil {
		if hadExisting {
			_ = os.Rename(bakDir, installDir)
		}
		_ = os.RemoveAll(stagingDir)
		return apperr.IO("hoán đổi thư mục cập nhật thất bại", err)
	}
	return nil
}

func computeFileSHA256(path string) (string, int64) {
	f, err := os.Open(path)
	if err != nil {
		return "", 0
	}
	defer f.Close()
	h := sha256.New()
	n, err := io.Copy(h, f)
	if err != nil {
		return "", 0
	}
	return hex.EncodeToString(h.Sum(nil)), n
}

func copyFileSimple(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	_, copyErr := io.Copy(out, in)
	closeErr := out.Close()
	if copyErr != nil {
		return copyErr
	}
	return closeErr
}

func isWindowsProcessRunning(exeName string) bool {
	if strings.TrimSpace(exeName) == "" {
		return false
	}
	cmd := exec.Command("tasklist", "/FI", "IMAGENAME eq "+exeName, "/FO", "CSV", "/NH")
	hideCloudCmdWindow(cmd)
	out, err := cmd.Output()
	if err != nil {
		return false
	}
	return strings.Contains(strings.ToLower(string(out)), strings.ToLower(exeName))
}

func firstNonEmptyStr(vals ...string) string {
	for _, v := range vals {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}
