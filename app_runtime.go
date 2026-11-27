package main

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	wailsruntime "github.com/wailsapp/wails/v2/pkg/runtime"

	"t-deck/internal/apperr"
	"t-deck/internal/cloud"
	"t-deck/internal/launcher"
	"t-deck/internal/media"
	"t-deck/internal/model"
	"t-deck/internal/scanner"
)

// --- Local Scanner, Thumbnails, Launch & Build Jobs ---

func (a *App) ScanLocalWorkspaces() (rep model.ScanReport, err error) {
	defer apperr.Recover("ScanLocalWorkspaces", &err, a.logPanic)
	if a.scanner == nil {
		return model.ScanReport{}, apperr.NotReady("scanner chưa sẵn sàng")
	}
	settings, err := a.settingsMgr.Load()
	if err != nil {
		return model.ScanReport{}, err
	}
	rep, err = a.scanner.ScanWithOptions(scanner.ScanOptions{
		WorkspaceDirs:    settings.WorkspaceDirs,
		AppsDir:          settings.AppsDir,
		RecentDays:       settings.ScanRecentDays,
		ExtraExcludeDirs: settings.ScanExcludeDirs,
	})
	if err == nil {
		a.generateMissingThumbnails()
	}
	return rep, err
}

func (a *App) generateMissingThumbnails() {
	if a.repo == nil || a.thumbs == nil {
		return
	}
	views, listErr := a.repo.ListProjects(model.ProjectFilter{})
	if listErr != nil {
		return
	}
	var ids []string
	for _, v := range views {
		if v.ThumbnailURL == "" || (v.IconURL == "" && v.LocalDir != "" && media.HasProjectBrandLogo(v.LocalDir)) {
			ids = append(ids, v.ID)
		}
	}
	if len(ids) == 0 {
		return
	}
	var wg sync.WaitGroup
	sem := make(chan struct{}, 6)
	for _, id := range ids {
		id := id
		wg.Add(1)
		go func() {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			_, _ = a.thumbs.GenerateProjectThumbnail(id)
		}()
	}
	wg.Wait()
	a.repo.InvalidateThumbnailCache()
}

func (a *App) SaveClipboardThumbnail(projectID string, base64PNG string) (string, error) {
	if a.thumbs == nil {
		return "", apperr.NotReady("thumbnail service chưa sẵn sàng")
	}
	return a.thumbs.SaveClipboardPNG(projectID, base64PNG)
}

func (a *App) GenerateProjectThumbnail(projectID string) (string, error) {
	if a.thumbs == nil {
		return "", apperr.NotReady("thumbnail service chưa sẵn sàng")
	}
	return a.thumbs.GenerateProjectThumbnail(projectID)
}

func (a *App) SyncEnvExample(projectID string) (model.EnvSyncResult, error) {
	if err := a.requireRepo(); err != nil {
		return model.EnvSyncResult{}, err
	}
	view, err := a.repo.GetProject(projectID)
	if err != nil {
		return model.EnvSyncResult{}, err
	}
	res, err := scanner.GenerateEnvExample(projectID, view.LocalDir)
	if err == nil {
		_ = a.repo.RefreshLocalFlags(projectID)
	}
	return res, err
}

func (a *App) GenerateStartBat(projectID string, template string) error {
	if err := a.requireRepo(); err != nil {
		return err
	}
	view, err := a.repo.GetProject(projectID)
	if err != nil {
		return err
	}
	if _, err = launcher.WriteStartBat(view.LocalDir, view.BatFile, view.Name, template); err != nil {
		return err
	}
	_ = a.repo.RefreshLocalFlags(projectID)
	return nil
}

func (a *App) GetBatTemplates() []launcher.BatTemplate {
	return launcher.AvailableBatTemplates()
}

func (a *App) LaunchProject(projectID string) (res model.LaunchResult, err error) {
	defer apperr.Recover("LaunchProject", &err, a.logPanic)
	if a.runner == nil {
		return model.LaunchResult{}, apperr.NotReady("runner chưa sẵn sàng")
	}
	return a.runner.LaunchProject(projectID)
}

func (a *App) OpenInExplorer(projectID string) error {
	if a.runner == nil {
		return apperr.NotReady("runner chưa sẵn sàng")
	}
	return a.runner.OpenInExplorer(projectID)
}

func (a *App) OpenInEditor(projectID string, editor string) error {
	if a.runner == nil {
		return apperr.NotReady("runner chưa sẵn sàng")
	}
	return a.runner.OpenInEditor(projectID, editor)
}

func (a *App) SuggestReleaseChangelog(projectID string, _ string) (string, error) {
	if err := a.requireRepo(); err != nil {
		return "", err
	}
	view, err := a.repo.GetProject(projectID)
	if err != nil {
		return "", err
	}
	return launcher.SuggestChangelog(view.LocalDir)
}

func (a *App) BuildAndPublishTarget(req model.BuildPublishRequest) (string, error) {
	if err := a.requireRepo(); err != nil {
		return "", err
	}
	settings, store, err := a.loadR2Store()
	if err != nil {
		return "", err
	}
	view, err := a.repo.GetProject(req.ProjectID)
	if err != nil {
		return "", err
	}
	tgt, found, err := a.repo.GetTarget(req.TargetID)
	if err != nil {
		return "", err
	}
	if !found {
		return "", apperr.NotFound(fmt.Sprintf("không tìm thấy target %q", req.TargetID))
	}
	if strings.TrimSpace(view.LocalDir) == "" {
		return "", apperr.Validation("dự án chưa có thư mục cục bộ để build")
	}

	jobID := uuid.NewString()
	jobCtx, cancel := context.WithCancel(a.opContext())
	a.jobsMu.Lock()
	if a.activeJobs == nil {
		a.activeJobs = make(map[string]context.CancelFunc)
	}
	a.activeJobs[jobID] = cancel
	a.jobsMu.Unlock()

	emitStatus := func(status, errMsg string) {
		if a.ctx == nil {
			return
		}
		wailsruntime.EventsEmit(a.ctx, "build:status", map[string]any{
			"job_id":     jobID,
			"project_id": req.ProjectID,
			"status":     status,
			"error":      errMsg,
		})
	}
	emitLog := func(stream, line string) {
		if a.ctx == nil {
			return
		}
		wailsruntime.EventsEmit(a.ctx, "build:log", map[string]any{
			"job_id":     jobID,
			"project_id": req.ProjectID,
			"stream":     stream,
			"line":       line,
			"ts":         time.Now().UnixMilli(),
		})
	}

	go func() {
		defer func() {
			var panicErr error
			apperr.Recover("BuildAndPublishTarget", &panicErr, a.logPanic)
			if panicErr != nil {
				emitStatus("failed", panicErr.Error())
			}
			cancel()
			a.jobsMu.Lock()
			delete(a.activeJobs, jobID)
			a.jobsMu.Unlock()
		}()

		emitStatus("running", "")
		workDir := view.LocalDir
		if strings.TrimSpace(tgt.Subpath) != "" {
			workDir = filepath.Join(view.LocalDir, tgt.Subpath)
		}
		if strings.TrimSpace(tgt.BuildCmd) != "" {
			emitLog("stdout", "$ "+tgt.BuildCmd)
			if err := launcher.RunBuildCommand(jobCtx, workDir, tgt.BuildCmd, emitLog); err != nil {
				if errors.Is(jobCtx.Err(), context.Canceled) {
					emitStatus("cancelled", "Đã hủy build theo yêu cầu")
					return
				}
				emitStatus("failed", err.Error())
				return
			}
		}

		pattern := strings.TrimSpace(tgt.ArtifactGlob)
		if pattern == "" {
			pattern = tgt.LaunchEntry
		}
		artifactPath, err := launcher.ResolveArtifactGlob(workDir, pattern)
		if err != nil {
			emitStatus("failed", err.Error())
			return
		}
		emitLog("stdout", "Resolved artifact: "+artifactPath)

		version := strings.TrimSpace(req.Version)
		if version == "" {
			version = scanner.ResolveProjectVersion(view.LocalDir, nil)
		}

		rel, err := cloud.PublishRelease(jobCtx, store, a.repo, settings, a.settingsMgr.ThumbnailsDir(), model.PublishReleaseRequest{
			ProjectID:  req.ProjectID,
			TargetID:   req.TargetID,
			SourcePath: artifactPath,
			Version:    version,
			Channel:    req.Channel,
			AssetKind:  req.AssetKind,
			Changelog:  req.Changelog,
		}, a.makeProgressEmitter(jobID, req.ProjectID))
		if err != nil {
			if errors.Is(jobCtx.Err(), context.Canceled) {
				emitStatus("cancelled", "Đã hủy upload")
				return
			}
			emitStatus("failed", err.Error())
			return
		}
		emitLog("stdout", fmt.Sprintf("Published %s (%s)", rel.Version, rel.R2Key))
		emitStatus("success", "")
	}()

	return jobID, nil
}

func (a *App) CancelActiveJob(jobID string) error {
	a.jobsMu.Lock()
	cancel, ok := a.activeJobs[jobID]
	a.jobsMu.Unlock()
	if ok && cancel != nil {
		cancel()
	}
	return nil
}

func (a *App) ListRunningProcesses() ([]model.RunningProcess, error) {
	if a.procMonitor == nil {
		return []model.RunningProcess{}, nil
	}
	return a.procMonitor.ListRunning(), nil
}

func (a *App) StopProjectProcess(projectID string) error {
	if a.procMonitor == nil {
		return nil
	}
	return a.procMonitor.StopProject(projectID)
}

func (a *App) OpenLocalhostPort(port int) error {
	if port <= 0 || port > 65535 {
		return apperr.Validation(fmt.Sprintf("cổng không hợp lệ: %d", port))
	}
	return a.openSafeURL(fmt.Sprintf("http://localhost:%d", port))
}
