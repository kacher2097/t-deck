package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"

	"t-deck/internal/apperr"
	"t-deck/internal/config"
	"t-deck/internal/fsutil"
	"t-deck/internal/model"
	"t-deck/internal/scanner"
)

// --- Project Management & Search ---

func (a *App) ListProjects(filter model.ProjectFilter) (res []model.ProjectView, err error) {
	defer apperr.Recover("ListProjects", &err, a.logPanic)
	if err := a.requireRepo(); err != nil {
		return nil, err
	}
	return a.repo.ListProjects(filter)
}

func (a *App) GetProjectDetail(id string) (res model.ProjectView, err error) {
	defer apperr.Recover("GetProjectDetail", &err, a.logPanic)
	if err := a.requireRepo(); err != nil {
		return model.ProjectView{}, err
	}
	return a.repo.GetProject(id)
}

func (a *App) UpsertProject(input model.ProjectInput) (res model.ProjectView, err error) {
	defer apperr.Recover("UpsertProject", &err, a.logPanic)
	if err := a.requireRepo(); err != nil {
		return model.ProjectView{}, err
	}

	name := strings.TrimSpace(input.Name)
	if name == "" {
		return model.ProjectView{}, apperr.Validation("tên dự án không được để trống")
	}
	pType := strings.TrimSpace(input.Type)
	if pType != config.ProjectTypeWeb && pType != config.ProjectTypeDesktop && pType != config.ProjectTypeHybrid {
		pType = config.ProjectTypeWeb
	}

	id := strings.TrimSpace(input.ID)
	var existing model.Project
	var found bool
	if id != "" {
		existing, found, _ = a.repo.GetProjectModel(id)
	} else {
		id = uuid.NewString()
	}

	slug := fsutil.Slugify(input.Slug)
	if slug == "" {
		slug = fsutil.Slugify(name)
	}
	if slug == "" {
		slug = "project-" + id[:8]
	}
	if exists, _ := a.repo.SlugExists(slug, id); exists {
		slug = fmt.Sprintf("%s-%s", slug, id[:6])
	}

	batFile := strings.TrimSpace(input.BatFile)
	if batFile == "" {
		batFile = config.DefaultBatFile
	}

	proj := model.Project{
		ID:                 id,
		Slug:               slug,
		Name:               name,
		Type:               pType,
		Category:           strings.TrimSpace(input.Category),
		Summary:            strings.TrimSpace(input.Summary),
		Tags:               input.Tags,
		TechStack:          strings.TrimSpace(input.TechStack),
		GitRemote:          strings.TrimSpace(input.GitRemote),
		BatFile:            batFile,
		ExeName:            strings.TrimSpace(input.ExeName),
		LatestVersion:      existing.LatestVersion,
		R2ReleaseKey:       existing.R2ReleaseKey,
		ReleaseNotes:       existing.ReleaseNotes,
		ThumbnailUpdatedAt: existing.ThumbnailUpdatedAt,
		UpdatedAt:          time.Now().Unix(),
	}
	if proj.Category != "" {
		_, _ = a.repo.CreateCategory(proj.Category)
	}
	if !found && strings.TrimSpace(input.LocalDir) != "" {
		if proj.Summary == "" {
			proj.Summary = scanner.ExtractSummary(input.LocalDir)
		}
		if fsutil.DirExists(input.LocalDir) && (proj.TechStack == "" || proj.GitRemote == "" || len(proj.Tags) == 0) {
			if probed, pErr := scanner.ProbeDirectory(input.LocalDir, nil); pErr == nil {
				if proj.TechStack == "" {
					proj.TechStack = probed.TechStack
				}
				if proj.GitRemote == "" {
					proj.GitRemote = probed.GitRemote
				}
				if len(proj.Tags) == 0 {
					proj.Tags = probed.Tags
				}
				if strings.TrimSpace(input.BatFile) == "" && probed.BatFile != "" {
					proj.BatFile = probed.BatFile
				}
				if strings.TrimSpace(input.ExeName) == "" && probed.ExeName != "" {
					proj.ExeName = probed.ExeName
				}
			}
		}
	}

	var local *model.LocalState
	if localDir := strings.TrimSpace(input.LocalDir); localDir != "" {
		cleanDir := filepath.Clean(localDir)
		local = &model.LocalState{
			ProjectID: id,
			LocalDir:  cleanDir,
			Status:    scanner.ComputeLocalStatus(cleanDir, pType),
		}
		if fsutil.DirExists(cleanDir) {
			_ = scanner.WriteMetaFile(cleanDir, model.ProjectMeta{
				ProjectID: id,
				Slug:      slug,
				Type:      pType,
			})
		}
	} else if !found {
		local = &model.LocalState{
			ProjectID: id,
			LocalDir:  "",
			Status:    config.StatusCloudOnly,
		}
	}

	if err := a.repo.UpsertProject(proj, local); err != nil {
		return model.ProjectView{}, err
	}
	if (!found || existing.ThumbnailUpdatedAt == 0) && a.thumbs != nil {
		_, _ = a.thumbs.GenerateProjectThumbnail(id)
	}
	if !found || strings.TrimSpace(input.ProdURL) != "" {
		a.ensureInitialTargets(proj, strings.TrimSpace(input.ProdURL))
	}
	return a.repo.GetProject(id)
}

// InspectLocalDirectory probes a local project folder and returns auto-detected metadata without mutating the database.
func (a *App) InspectLocalDirectory(dir string) (res model.ProjectProbe, err error) {
	defer apperr.Recover("InspectLocalDirectory", &err, a.logPanic)
	if err := a.requireRepo(); err != nil {
		return model.ProjectProbe{}, err
	}
	cats, _ := a.repo.ListCategories()
	catNames := make([]string, 0, len(cats))
	for _, c := range cats {
		if strings.TrimSpace(c.Name) != "" {
			catNames = append(catNames, c.Name)
		}
	}
	probe, probeErr := scanner.ProbeDirectory(dir, catNames)
	if probeErr != nil {
		return model.ProjectProbe{}, apperr.Validation(probeErr.Error())
	}
	if probe.ExistingID != "" {
		if _, found, _ := a.repo.GetProjectModel(probe.ExistingID); !found {
			probe.ExistingID = ""
		}
	}
	return probe, nil
}

func (a *App) ensureInitialTargets(proj model.Project, prodURL string) {
	if a.repo == nil {
		return
	}
	existing, _ := a.repo.ListProjectTargets(proj.ID)
	hasKind := func(kind string) bool {
		for _, t := range existing {
			if t.Kind == kind {
				return true
			}
		}
		return false
	}
	if (proj.Type == config.ProjectTypeWeb || proj.Type == config.ProjectTypeHybrid) && !hasKind(config.ProjectTypeWeb) {
		_, _ = a.repo.UpsertTarget(model.ProjectTarget{
			ProjectID:    proj.ID,
			Kind:         config.ProjectTypeWeb,
			Name:         "Web App",
			Platform:     "web",
			Distribution: "hosted",
			LaunchEntry:  proj.BatFile,
			ProdURL:      prodURL,
		})
	}
	if (proj.Type == config.ProjectTypeDesktop || proj.Type == config.ProjectTypeHybrid) && !hasKind(config.ProjectTypeDesktop) {
		_, _ = a.repo.UpsertTarget(model.ProjectTarget{
			ProjectID:    proj.ID,
			Kind:         config.ProjectTypeDesktop,
			Name:         "Desktop App",
			Platform:     "windows",
			Distribution: "portable",
			LaunchEntry:  proj.ExeName,
		})
	}
}

func (a *App) DeleteProject(id string, deleteLocalFiles bool) (err error) {
	defer apperr.Recover("DeleteProject", &err, a.logPanic)
	if err := a.requireRepo(); err != nil {
		return err
	}
	view, err := a.repo.GetProject(id)
	if err != nil {
		return err
	}

	if a.procMonitor != nil {
		_ = a.procMonitor.StopProject(id)
	}

	localDir := strings.TrimSpace(view.LocalDir)
	if deleteLocalFiles && localDir != "" && fsutil.DirExists(localDir) {
		if !a.isSafeProjectDeleteTarget(localDir) {
			return apperr.Validation("từ chối xóa thư mục gốc hệ thống hoặc thư mục workspace: " + localDir)
		}
		// Remove local files BEFORE committing DB soft-delete so a locked folder never ghosts out of T-Deck with leftover files.
		if err := fsutil.RemoveDirForce(localDir); err != nil {
			return apperr.IO("xóa thư mục cục bộ thất bại (dự án vẫn được giữ lại trong T-Deck để thử lại)", err)
		}
	}

	if err := a.repo.DeleteProject(id); err != nil {
		return err
	}
	_ = os.Remove(a.settingsMgr.ThumbnailPath(id))
	_ = os.Remove(filepath.Join(a.settingsMgr.ThumbnailsDir(), id+".icon.png"))

	if !deleteLocalFiles && localDir != "" && fsutil.DirExists(localDir) {
		// Preserve/ensure anchor so Scan Local honors the tombstone and never resurrects this project.
		if !fsutil.FileExists(filepath.Join(localDir, config.MetaFileName)) {
			_ = scanner.WriteMetaFile(localDir, model.ProjectMeta{
				ProjectID: view.ID,
				Slug:      view.Slug,
				Type:      view.Type,
			})
		}
	}

	go a.deleteRemoteThumbnailBestEffort(id)
	return nil
}

// DeleteProjects removes a batch of projects atomically from T-Deck and optionally deletes their local folders.
func (a *App) DeleteProjects(ids []string, deleteLocalFiles bool) (res model.BulkDeleteResult, err error) {
	defer apperr.Recover("DeleteProjects", &err, a.logPanic)
	res.Failed = make([]string, 0)
	if err := a.requireRepo(); err != nil {
		return res, err
	}
	if len(ids) == 0 {
		return res, apperr.Validation("chưa chọn dự án nào để xóa")
	}

	views := make([]model.ProjectView, 0, len(ids))
	for _, rawID := range ids {
		id := strings.TrimSpace(rawID)
		if id == "" {
			continue
		}
		v, getErr := a.repo.GetProject(id)
		if getErr != nil {
			res.Failed = append(res.Failed, fmt.Sprintf("%s: %v", id, getErr))
			continue
		}
		views = append(views, v)
	}
	if len(views) == 0 {
		return res, nil
	}

	readyIDs := make([]string, 0, len(views))
	readyViews := make([]model.ProjectView, 0, len(views))

	for _, v := range views {
		if a.procMonitor != nil {
			_ = a.procMonitor.StopProject(v.ID)
		}
		localDir := strings.TrimSpace(v.LocalDir)
		if deleteLocalFiles && localDir != "" && fsutil.DirExists(localDir) {
			if !a.isSafeProjectDeleteTarget(localDir) {
				res.Failed = append(res.Failed, fmt.Sprintf("%s: từ chối xóa thư mục gốc/workspace %s", v.Name, localDir))
				continue
			}
			if rmErr := fsutil.RemoveDirForce(localDir); rmErr != nil {
				res.Failed = append(res.Failed, fmt.Sprintf("%s: %v", v.Name, rmErr))
				continue
			}
		}
		readyIDs = append(readyIDs, v.ID)
		readyViews = append(readyViews, v)
	}
	if len(readyIDs) == 0 {
		return res, nil
	}

	deleted, err := a.repo.DeleteProjects(readyIDs)
	if err != nil {
		return res, apperr.Wrap(apperr.CodeInternal, "xóa hàng loạt dự án thất bại", err)
	}
	res.Deleted = deleted

	for _, v := range readyViews {
		_ = os.Remove(a.settingsMgr.ThumbnailPath(v.ID))
		_ = os.Remove(filepath.Join(a.settingsMgr.ThumbnailsDir(), v.ID+".icon.png"))
		localDir := strings.TrimSpace(v.LocalDir)
		if !deleteLocalFiles && localDir != "" && fsutil.DirExists(localDir) {
			if !fsutil.FileExists(filepath.Join(localDir, config.MetaFileName)) {
				_ = scanner.WriteMetaFile(localDir, model.ProjectMeta{
					ProjectID: v.ID,
					Slug:      v.Slug,
					Type:      v.Type,
				})
			}
		}
		go a.deleteRemoteThumbnailBestEffort(v.ID)
	}
	return res, nil
}

func (a *App) deleteRemoteThumbnailBestEffort(projectID string) {
	defer apperr.Recover("delete-remote-thumbnail", nil, a.logPanic)
	if a.settingsMgr == nil || strings.TrimSpace(projectID) == "" {
		return
	}
	settings, store, err := a.loadR2Store()
	if err != nil || !settings.R2.Enabled || store == nil {
		return
	}
	_ = store.DeleteObject(a.opContext(), config.ThumbnailsPrefix+projectID+".png")
}

func (a *App) isSafeProjectDeleteTarget(dir string) bool {
	if !isSafeDeleteTarget(dir) {
		return false
	}
	if a.settingsMgr == nil {
		return true
	}
	settings, err := a.settingsMgr.Load()
	if err != nil {
		return true
	}
	cleaned := filepath.Clean(strings.TrimSpace(dir))
	for _, ws := range settings.WorkspaceDirs {
		if wsClean := filepath.Clean(strings.TrimSpace(ws)); wsClean != "" && strings.EqualFold(cleaned, wsClean) {
			return false
		}
	}
	if appsClean := filepath.Clean(strings.TrimSpace(settings.AppsDir)); appsClean != "" && strings.EqualFold(cleaned, appsClean) {
		return false
	}
	return true
}

func isSafeDeleteTarget(dir string) bool {
	cleaned := filepath.Clean(strings.TrimSpace(dir))
	if len(cleaned) < 4 {
		return false
	}
	vol := filepath.VolumeName(cleaned)
	if vol != "" && strings.EqualFold(cleaned, vol+string(filepath.Separator)) {
		return false
	}
	return cleaned != string(filepath.Separator) && cleaned != "." && cleaned != ".."
}

// --- Category Management ---

func (a *App) ListCategories() ([]model.Category, error) {
	if err := a.requireRepo(); err != nil {
		return nil, err
	}
	return a.repo.ListCategories()
}

func (a *App) CreateCategory(name string) (model.Category, error) {
	if err := a.requireRepo(); err != nil {
		return model.Category{}, err
	}
	return a.repo.CreateCategory(name)
}

func (a *App) RenameCategory(id string, newName string) (model.Category, error) {
	if err := a.requireRepo(); err != nil {
		return model.Category{}, err
	}
	return a.repo.RenameCategory(id, newName)
}

func (a *App) DeleteCategory(id string) error {
	if err := a.requireRepo(); err != nil {
		return err
	}
	return a.repo.DeleteCategory(id)
}

// --- Declarative Manifest Import ---

func (a *App) ImportProjectManifest(projectID string) (res model.ProjectView, err error) {
	defer apperr.Recover("ImportProjectManifest", &err, a.logPanic)
	if err := a.requireRepo(); err != nil {
		return model.ProjectView{}, err
	}
	view, err := a.repo.GetProject(projectID)
	if err != nil {
		return model.ProjectView{}, err
	}
	if strings.TrimSpace(view.LocalDir) == "" {
		return model.ProjectView{}, apperr.Validation("dự án chưa có thư mục cục bộ")
	}
	manifest, found, err := scanner.LoadProjectManifest(view.LocalDir)
	if err != nil {
		return model.ProjectView{}, apperr.IO("đọc .tdeck.json thất bại", err)
	}
	if !found || manifest == nil {
		return model.ProjectView{}, apperr.NotFound("không tìm thấy file .tdeck.json trong thư mục dự án")
	}
	if err := scanner.ApplyProjectManifestToStore(a.repo, projectID, manifest); err != nil {
		return model.ProjectView{}, err
	}
	return a.repo.GetProject(projectID)
}
