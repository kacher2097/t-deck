package cloud

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"

	"t-deck/internal/apperr"
	"t-deck/internal/config"
	"t-deck/internal/model"
	"t-deck/internal/scanner"
)

type releaseCatalogStore interface {
	ListProjectTargets(projectID string) ([]model.ProjectTarget, error)
	GetTarget(targetID string) (model.ProjectTarget, bool, error)
	UpsertTarget(t model.ProjectTarget) (model.ProjectTarget, error)
	ListReleases(projectID string, targetID string) ([]model.Release, error)
	GetRelease(releaseID string) (model.Release, bool, error)
	UpsertRelease(r model.Release) (model.Release, error)
	DeleteRelease(releaseID string) error
	UpdateInstalledRelease(projectID, version, releaseID string) error
}

// PublishDesktopRelease zips req.SourcePath, uploads it to R2, updates metadata, and syncs catalog.json.
func PublishDesktopRelease(
	ctx context.Context,
	store ObjectStore,
	repo ProjectStore,
	thumbnailsDir string,
	req model.PublishRequest,
) (model.ProjectView, error) {
	if strings.TrimSpace(req.ProjectID) == "" {
		return model.ProjectView{}, apperr.Validation("project_id không được để trống")
	}
	if strings.TrimSpace(req.SourcePath) == "" {
		return model.ProjectView{}, apperr.Validation("source_path không được để trống")
	}
	version := strings.TrimSpace(req.Version)
	if version == "" {
		return model.ProjectView{}, apperr.Validation("version không được để trống (VD: v1.0.0)")
	}
	if strings.Contains(version, "..") || strings.ContainsAny(version, `/\`) {
		return model.ProjectView{}, apperr.Validation(fmt.Sprintf("chuỗi version không hợp lệ %q", version))
	}

	p, found, err := repo.GetProjectModel(req.ProjectID)
	if err != nil {
		return model.ProjectView{}, err
	}
	if !found {
		return model.ProjectView{}, apperr.NotFound(fmt.Sprintf("không tìm thấy dự án %q", req.ProjectID))
	}
	if p.Type != config.ProjectTypeDesktop {
		return model.ProjectView{}, apperr.Validation("publish release kiểu cũ chỉ hỗ trợ dự án desktop")
	}

	srcInfo, err := os.Stat(req.SourcePath)
	if err != nil {
		return model.ProjectView{}, apperr.IO("không thể truy cập đường dẫn nguồn", err)
	}

	tmpZipPath := filepath.Join(os.TempDir(), fmt.Sprintf("%s-%s-%d.zip", p.Slug, version, time.Now().UnixNano()))
	defer os.Remove(tmpZipPath)

	if err := zipSource(req.SourcePath, tmpZipPath); err != nil {
		return model.ProjectView{}, apperr.IO("đóng gói file .zip thất bại", err)
	}

	r2Key := fmt.Sprintf("%s%s/%s.zip", config.ReleasesPrefix, p.Slug, version)
	if err := store.UploadFile(ctx, r2Key, tmpZipPath, "application/zip"); err != nil {
		return model.ProjectView{}, apperr.Cloud("upload bản phát hành lên R2 thất bại", err)
	}

	if !srcInfo.IsDir() && strings.EqualFold(filepath.Ext(req.SourcePath), ".exe") && p.ExeName == "" {
		p.ExeName = filepath.Base(req.SourcePath)
	}
	p.LatestVersion = version
	p.R2ReleaseKey = r2Key
	p.ReleaseNotes = strings.TrimSpace(req.ReleaseNotes)
	p.UpdatedAt = time.Now().Unix()

	if err := repo.UpsertProject(p, nil); err != nil {
		return model.ProjectView{}, fmt.Errorf("update project metadata: %w", err)
	}

	if relRepo, ok := repo.(releaseCatalogStore); ok {
		sha, size := computeFileSHA256(tmpZipPath)
		targetID := resolveOrEnsurePrimaryTarget(relRepo, p)
		if targetID != "" {
			rel := model.Release{
				ID:          uuid.NewString(),
				ProjectID:   p.ID,
				TargetID:    targetID,
				Version:     version,
				Channel:     config.ReleaseChannelStable,
				Changelog:   p.ReleaseNotes,
				R2Key:       r2Key,
				FileName:    filepath.Base(r2Key),
				FileSize:    size,
				SHA256:      sha,
				AssetKind:   "portable_zip",
				Status:      "published",
				PublishedAt: p.UpdatedAt,
			}
			_, _ = relRepo.UpsertRelease(rel)
			_ = syncReleaseIndexToR2(ctx, store, relRepo, p.Slug)
		}
	}

	if _, err := SyncCatalogWithR2(ctx, store, repo, thumbnailsDir); err != nil {
		return model.ProjectView{}, apperr.Cloud("đồng bộ catalog sau khi publish thất bại", err)
	}
	return repo.GetProject(p.ID)
}

// PublishRelease packages and uploads either a Web (static_zip / external_url) or Desktop (portable_zip / installer_exe) release.
func PublishRelease(
	ctx context.Context,
	store ObjectStore,
	repo ProjectStore,
	settings config.Settings,
	thumbsDir string,
	req model.PublishReleaseRequest,
	onProgress func(stage string, done, total int64),
) (model.Release, error) {
	if strings.TrimSpace(req.ProjectID) == "" {
		return model.Release{}, apperr.Validation("project_id không được để trống")
	}
	version := strings.TrimSpace(req.Version)
	if version == "" {
		return model.Release{}, apperr.Validation("version không được để trống")
	}
	if strings.Contains(version, "..") || strings.ContainsAny(version, `/\`) {
		return model.Release{}, apperr.Validation(fmt.Sprintf("chuỗi version không hợp lệ %q", version))
	}

	channel := strings.ToLower(strings.TrimSpace(req.Channel))
	if channel == "" {
		channel = config.ReleaseChannelStable
	}

	p, found, err := repo.GetProjectModel(req.ProjectID)
	if err != nil {
		return model.Release{}, err
	}
	if !found {
		return model.Release{}, apperr.NotFound(fmt.Sprintf("không tìm thấy dự án %q", req.ProjectID))
	}

	relRepo, ok := repo.(releaseCatalogStore)
	if !ok {
		return model.Release{}, apperr.NotReady("repository chưa hỗ trợ multi-target releases")
	}

	var target model.ProjectTarget
	if strings.TrimSpace(req.TargetID) != "" {
		t, tFound, err := relRepo.GetTarget(req.TargetID)
		if err != nil {
			return model.Release{}, err
		}
		if !tFound {
			return model.Release{}, apperr.NotFound(fmt.Sprintf("không tìm thấy target %q", req.TargetID))
		}
		target = t
	} else {
		tid := resolveOrEnsurePrimaryTarget(relRepo, p)
		target, _, _ = relRepo.GetTarget(tid)
	}

	assetKind := inferAssetKind(target, req)
	var r2Key, fileName, shaHex string
	var fileSize int64

	if assetKind != "external_url" {
		if strings.TrimSpace(req.SourcePath) == "" {
			return model.Release{}, apperr.Validation("vui lòng chọn file .exe hoặc thư mục build")
		}
		srcInfo, err := os.Stat(req.SourcePath)
		if err != nil {
			return model.Release{}, apperr.IO("không tìm thấy đường dẫn nguồn", err)
		}

		var uploadPath, contentType string
		if assetKind == "installer_exe" && !srcInfo.IsDir() {
			uploadPath = req.SourcePath
			fileName = filepath.Base(req.SourcePath)
			contentType = "application/octet-stream"
			r2Key = fmt.Sprintf("%s%s/%s/%s/%s", config.ReleasesPrefix, p.Slug, target.Kind, version, fileName)
		} else {
			extName := fmt.Sprintf("%s-%s-%s.zip", p.Slug, target.Kind, version)
			if target.Kind == config.ProjectTypeWeb {
				extName = fmt.Sprintf("%s-web-%s-dist.zip", p.Slug, version)
			}
			fileName = extName
			tmpZipPath := filepath.Join(os.TempDir(), fmt.Sprintf("%s-%d.zip", extName, time.Now().UnixNano()))
			defer os.Remove(tmpZipPath)

			if onProgress != nil {
				onProgress("zipping", 0, 1)
			}
			if err := zipSourceFast(req.SourcePath, tmpZipPath, func(done, total int64) {
				if onProgress != nil {
					onProgress("zipping", done, total)
				}
			}); err != nil {
				return model.Release{}, apperr.IO("nén file release thất bại", err)
			}
			uploadPath = tmpZipPath
			contentType = "application/zip"
			r2Key = fmt.Sprintf("%s%s/%s/%s/%s", config.ReleasesPrefix, p.Slug, target.Kind, version, fileName)
		}

		shaHex, fileSize = computeFileSHA256(uploadPath)
		if err := uploadArtifact(ctx, store, r2Key, uploadPath, contentType, onProgress); err != nil {
			return model.Release{}, apperr.Cloud("upload artifact lên R2 thất bại", err)
		}

		if !srcInfo.IsDir() && target.Kind == config.ProjectTypeDesktop && strings.EqualFold(filepath.Ext(req.SourcePath), ".exe") && p.ExeName == "" {
			p.ExeName = filepath.Base(req.SourcePath)
		}
	}

	view, _ := repo.GetProject(p.ID)
	var commitSHA string
	if view.LocalDir != "" {
		_, commitSHA, _ = scanner.ReadGitInfo(view.LocalDir)
	}
	deployURL := strings.TrimSpace(req.DeployURL)
	if deployURL == "" && target.Kind == config.ProjectTypeWeb {
		deployURL = target.ProdURL
	}

	now := time.Now().Unix()
	savedRel, err := relRepo.UpsertRelease(model.Release{
		ID:          uuid.NewString(),
		ProjectID:   p.ID,
		TargetID:    target.ID,
		Version:     version,
		Channel:     channel,
		CommitSHA:   commitSHA,
		Changelog:   strings.TrimSpace(req.Changelog),
		R2Key:       r2Key,
		FileName:    fileName,
		FileSize:    fileSize,
		SHA256:      shaHex,
		DeployURL:   deployURL,
		AssetKind:   assetKind,
		Status:      "published",
		PublishedAt: now,
	})
	if err != nil {
		return model.Release{}, err
	}

	if target.Kind == config.ProjectTypeDesktop && channel == config.ReleaseChannelStable && r2Key != "" {
		p.LatestVersion = version
		p.R2ReleaseKey = r2Key
		p.ReleaseNotes = savedRel.Changelog
		p.UpdatedAt = now
		if err := repo.UpsertProject(p, nil); err != nil {
			return model.Release{}, err
		}
	}

	if err := syncReleaseIndexToR2(ctx, store, relRepo, p.Slug); err != nil {
		return model.Release{}, apperr.Cloud("đồng bộ release index thất bại", err)
	}
	if target.Kind == config.ProjectTypeDesktop {
		_ = publishAutoUpdateManifest(ctx, store, relRepo, settings, p, target.ID, channel)
	}
	if _, err := SyncCatalogWithR2(ctx, store, repo, thumbsDir); err != nil {
		return model.Release{}, apperr.Cloud("đồng bộ catalog thất bại", err)
	}
	return savedRel, nil
}

// DeleteRelease marks a release deleted/yanked and optionally deletes its R2 artifact.
func DeleteRelease(ctx context.Context, store ObjectStore, repo ProjectStore, releaseID string, deleteR2Object bool) error {
	relRepo, ok := repo.(releaseCatalogStore)
	if !ok {
		return apperr.NotReady("repository chưa hỗ trợ releases")
	}
	rel, found, err := relRepo.GetRelease(releaseID)
	if err != nil {
		return err
	}
	if !found {
		return nil
	}
	if deleteR2Object && strings.TrimSpace(rel.R2Key) != "" && store != nil {
		if err := store.DeleteObject(ctx, rel.R2Key); err != nil {
			return apperr.Cloud("xóa file trên R2 thất bại", err)
		}
	}
	if err := relRepo.DeleteRelease(releaseID); err != nil {
		return err
	}
	if store != nil {
		if p, pFound, _ := repo.GetProjectModel(rel.ProjectID); pFound {
			_ = syncReleaseIndexToR2(ctx, store, relRepo, p.Slug)
		}
	}
	return nil
}

// GenerateReleaseDownloadURL returns a public CDN link if configured or an S3 presigned URL.
func GenerateReleaseDownloadURL(ctx context.Context, store ObjectStore, repo ProjectStore, settings config.Settings, releaseID string, expiryHours int) (string, error) {
	relRepo, ok := repo.(releaseCatalogStore)
	if !ok {
		return "", apperr.NotReady("repository chưa hỗ trợ releases")
	}
	rel, found, err := relRepo.GetRelease(releaseID)
	if err != nil {
		return "", err
	}
	if !found {
		return "", apperr.NotFound(fmt.Sprintf("không tìm thấy release %q", releaseID))
	}
	if strings.TrimSpace(rel.R2Key) == "" {
		if strings.TrimSpace(rel.DeployURL) != "" {
			return rel.DeployURL, nil
		}
		return "", apperr.NotFound("bản phát hành này không có artifact trên R2")
	}
	if pub := strings.TrimRight(strings.TrimSpace(settings.R2.PublicDomain), "/"); pub != "" {
		return pub + "/" + strings.TrimLeft(rel.R2Key, "/"), nil
	}
	if expiryHours <= 0 {
		expiryHours = 24
	}
	if expiryHours > 168 {
		expiryHours = 168
	}
	return store.PresignGetURL(ctx, rel.R2Key, time.Duration(expiryHours)*time.Hour)
}

func inferAssetKind(target model.ProjectTarget, req model.PublishReleaseRequest) string {
	if ak := strings.TrimSpace(req.AssetKind); ak != "" {
		return ak
	}
	if target.Kind == config.ProjectTypeWeb {
		if strings.TrimSpace(req.SourcePath) != "" {
			return "static_zip"
		}
		return "external_url"
	}
	if target.Distribution == "installer" {
		return "installer_exe"
	}
	return "portable_zip"
}

func uploadArtifact(ctx context.Context, store ObjectStore, r2Key, uploadPath, contentType string, onProgress func(stage string, done, total int64)) error {
	if progStore, hasProg := store.(interface {
		UploadFileWithProgress(ctx context.Context, key string, localPath string, contentType string, onProgress func(done, total int64)) error
	}); hasProg {
		return progStore.UploadFileWithProgress(ctx, r2Key, uploadPath, contentType, func(done, total int64) {
			if onProgress != nil {
				onProgress("uploading", done, total)
			}
		})
	}
	return store.UploadFile(ctx, r2Key, uploadPath, contentType)
}

func resolveOrEnsurePrimaryTarget(relRepo releaseCatalogStore, p model.Project) string {
	targets, err := relRepo.ListProjectTargets(p.ID)
	if err == nil {
		for _, t := range targets {
			if t.Kind == config.ProjectTypeDesktop || p.Type == config.ProjectTypeWeb {
				return t.ID
			}
		}
		if len(targets) > 0 {
			return targets[0].ID
		}
	}
	kind, platform, dist, entry, name := config.ProjectTypeDesktop, "windows", "portable", p.ExeName, "Desktop App"
	if p.Type == config.ProjectTypeWeb {
		kind, platform, dist, entry, name = config.ProjectTypeWeb, "web", "hosted", p.BatFile, "Web App"
	}
	created, err := relRepo.UpsertTarget(model.ProjectTarget{
		ProjectID:    p.ID,
		Kind:         kind,
		Name:         name,
		Platform:     platform,
		Distribution: dist,
		LaunchEntry:  entry,
	})
	if err != nil {
		return ""
	}
	return created.ID
}

func syncReleaseIndexToR2(ctx context.Context, store ObjectStore, relRepo releaseCatalogStore, slug string) error {
	getter, ok := relRepo.(interface {
		ListReleasesBySlug(slug string) ([]model.Release, error)
	})
	if !ok {
		return nil
	}
	rels, err := getter.ListReleasesBySlug(slug)
	if err != nil {
		return err
	}
	idx := model.ProjectReleaseIndex{
		Slug:      slug,
		UpdatedAt: time.Now().Unix(),
		Releases:  rels,
	}
	return store.PutJSON(ctx, fmt.Sprintf("%s%s/index.json", config.ReleasesPrefix, slug), idx)
}

func publishAutoUpdateManifest(ctx context.Context, store ObjectStore, relRepo releaseCatalogStore, settings config.Settings, p model.Project, targetID, channel string) error {
	rels, err := relRepo.ListReleases(p.ID, targetID)
	if err != nil || len(rels) == 0 {
		return nil
	}
	var latestPortable, latestInstaller *model.Release
	for i := range rels {
		r := &rels[i]
		if r.Status != "published" || r.Channel != channel || r.R2Key == "" {
			continue
		}
		if r.AssetKind == "installer_exe" && latestInstaller == nil {
			latestInstaller = r
		} else if r.AssetKind != "installer_exe" && latestPortable == nil {
			latestPortable = r
		}
	}
	primary := latestPortable
	if primary == nil {
		primary = latestInstaller
	}
	if primary == nil {
		return nil
	}
	makeURL := func(r2Key string) string {
		if pub := strings.TrimRight(strings.TrimSpace(settings.R2.PublicDomain), "/"); pub != "" {
			return pub + "/" + strings.TrimLeft(r2Key, "/")
		}
		return r2Key
	}
	plat := model.AutoUpdatePlatform{
		URL:    makeURL(primary.R2Key),
		SHA256: primary.SHA256,
		Size:   primary.FileSize,
	}
	if latestInstaller != nil {
		plat.InstallerURL = makeURL(latestInstaller.R2Key)
	}
	manifest := model.AutoUpdateManifest{
		Slug:      p.Slug,
		Version:   primary.Version,
		Channel:   channel,
		PubDate:   time.Unix(primary.PublishedAt, 0).UTC().Format(time.RFC3339),
		Notes:     primary.Changelog,
		Platforms: map[string]model.AutoUpdatePlatform{"windows-x86_64": plat},
	}
	return store.PutJSON(ctx, fmt.Sprintf("%s%s/%s/latest.json", config.ReleasesPrefix, p.Slug, channel), manifest)
}
