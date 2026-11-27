package main

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	wailsruntime "github.com/wailsapp/wails/v2/pkg/runtime"

	"t-deck/internal/apperr"
	"t-deck/internal/cloud"
	"t-deck/internal/config"
	"t-deck/internal/model"
)

// --- Cloudflare R2 Sync, Releases, Targets & Web Health ---

func (a *App) SyncCatalogWithR2() (rep model.SyncReport, err error) {
	defer apperr.Recover("SyncCatalogWithR2", &err, a.logPanic)
	_, store, err := a.loadR2Store()
	if err != nil {
		return model.SyncReport{}, err
	}
	return cloud.SyncCatalogWithR2(a.opContext(), store, a.repo, a.settingsMgr.ThumbnailsDir())
}

func (a *App) PublishDesktopRelease(req model.PublishRequest) (res model.ProjectView, err error) {
	defer apperr.Recover("PublishDesktopRelease", &err, a.logPanic)
	_, store, err := a.loadR2Store()
	if err != nil {
		return model.ProjectView{}, err
	}
	return cloud.PublishDesktopRelease(a.opContext(), store, a.repo, a.settingsMgr.ThumbnailsDir(), req)
}

func (a *App) HydrateProjectOnNewPC(projectID string) (res model.ProjectView, err error) {
	defer apperr.Recover("HydrateProjectOnNewPC", &err, a.logPanic)
	settings, err := a.settingsMgr.Load()
	if err != nil {
		return model.ProjectView{}, err
	}
	proj, found, err := a.repo.GetProjectModel(projectID)
	if err != nil {
		return model.ProjectView{}, err
	}
	if !found {
		return model.ProjectView{}, apperr.NotFound(fmt.Sprintf("không tìm thấy dự án %q", projectID))
	}

	var store cloud.ObjectStore
	if proj.Type == config.ProjectTypeDesktop {
		_, store, err = a.loadR2Store()
		if err != nil {
			return model.ProjectView{}, err
		}
	}
	return cloud.HydrateProjectOnNewPC(a.opContext(), store, a.repo, settings, projectID, cloud.DefaultGitClone)
}

func (a *App) ListProjectTargets(projectID string) ([]model.ProjectTarget, error) {
	if err := a.requireRepo(); err != nil {
		return nil, err
	}
	return a.repo.ListProjectTargets(projectID)
}

func (a *App) UpsertTarget(target model.ProjectTarget) (model.ProjectTarget, error) {
	if err := a.requireRepo(); err != nil {
		return model.ProjectTarget{}, err
	}
	return a.repo.UpsertTarget(target)
}

func (a *App) DeleteTarget(targetID string) error {
	if err := a.requireRepo(); err != nil {
		return err
	}
	return a.repo.DeleteTarget(targetID)
}

func (a *App) ListReleases(projectID string, targetID string) ([]model.Release, error) {
	if err := a.requireRepo(); err != nil {
		return nil, err
	}
	return a.repo.ListReleases(projectID, targetID)
}

func (a *App) PublishRelease(req model.PublishReleaseRequest) (res model.Release, err error) {
	defer apperr.Recover("PublishRelease", &err, a.logPanic)
	settings, store, err := a.loadR2Store()
	if err != nil {
		return model.Release{}, err
	}
	jobID := uuid.NewString()
	onProgress := a.makeProgressEmitter(jobID, req.ProjectID)
	return cloud.PublishRelease(a.opContext(), store, a.repo, settings, a.settingsMgr.ThumbnailsDir(), req, onProgress)
}

func (a *App) DeleteRelease(releaseID string, deleteR2Object bool) (err error) {
	defer apperr.Recover("DeleteRelease", &err, a.logPanic)
	var store cloud.ObjectStore
	if deleteR2Object {
		_, store, err = a.loadR2Store()
		if err != nil {
			return err
		}
	} else {
		_, store, _ = a.loadR2Store()
	}
	return cloud.DeleteRelease(a.opContext(), store, a.repo, releaseID, deleteR2Object)
}

func (a *App) UpdateDesktopRelease(projectID string, targetID string, releaseID string) (res model.ProjectView, err error) {
	defer apperr.Recover("UpdateDesktopRelease", &err, a.logPanic)
	settings, store, err := a.loadR2Store()
	if err != nil {
		return model.ProjectView{}, err
	}
	return cloud.UpdateDesktopRelease(a.opContext(), store, a.repo, settings, projectID, targetID, releaseID)
}

func (a *App) RollbackDesktopRelease(projectID string, targetID string) (res model.ProjectView, err error) {
	defer apperr.Recover("RollbackDesktopRelease", &err, a.logPanic)
	settings, store, err := a.loadR2Store()
	if err != nil {
		settings, _ = a.settingsMgr.Load()
		return cloud.RollbackDesktopRelease(a.opContext(), nil, a.repo, settings, projectID, targetID)
	}
	return cloud.RollbackDesktopRelease(a.opContext(), store, a.repo, settings, projectID, targetID)
}

func (a *App) GenerateReleaseDownloadURL(releaseID string, expiryHours int) (string, error) {
	settings, store, err := a.loadR2Store()
	if err != nil {
		return "", err
	}
	return cloud.GenerateReleaseDownloadURL(a.opContext(), store, a.repo, settings, releaseID, expiryHours)
}

func (a *App) CheckWebHealth(targetID string) (res model.WebHealthResult, err error) {
	defer apperr.Recover("CheckWebHealth", &err, a.logPanic)
	if err := a.requireRepo(); err != nil {
		return model.WebHealthResult{}, err
	}
	tgt, found, err := a.repo.GetTarget(targetID)
	if err != nil {
		return model.WebHealthResult{}, err
	}
	if !found {
		return model.WebHealthResult{}, apperr.NotFound(fmt.Sprintf("không tìm thấy target %q", targetID))
	}
	latestVer := ""
	if rels, _ := a.repo.ListReleases(tgt.ProjectID, tgt.ID); len(rels) > 0 {
		latestVer = rels[0].Version
	}
	res = cloud.CheckWebHealth(a.opContext(), tgt, latestVer)
	if a.ctx != nil {
		wailsruntime.EventsEmit(a.ctx, "health:result", res)
	}
	return res, nil
}

func (a *App) CheckAllWebHealth() (res []model.WebHealthResult, err error) {
	defer apperr.Recover("CheckAllWebHealth", &err, a.logPanic)
	if err := a.requireRepo(); err != nil {
		return nil, err
	}
	targetsMap, err := a.repo.ListAllActiveTargetsMap()
	if err != nil {
		return nil, err
	}
	latestMap, _ := a.repo.ListAllLatestPublishedReleasesMap()

	var allTargets []model.ProjectTarget
	latestVerByTarget := make(map[string]string)
	for pid, list := range targetsMap {
		for _, t := range list {
			allTargets = append(allTargets, t)
			if rel, ok := latestMap[pid][t.ID]; ok {
				latestVerByTarget[t.ID] = rel.Version
			}
		}
	}
	return cloud.CheckAllWebHealth(a.opContext(), allTargets, latestVerByTarget, func(r model.WebHealthResult) {
		if a.ctx != nil {
			wailsruntime.EventsEmit(a.ctx, "health:result", r)
		}
	}), nil
}

func (a *App) OpenTargetURL(targetID string, env string) error {
	if err := a.requireRepo(); err != nil {
		return err
	}
	tgt, found, err := a.repo.GetTarget(targetID)
	if err != nil {
		return err
	}
	if !found {
		return apperr.NotFound(fmt.Sprintf("không tìm thấy target %q", targetID))
	}
	raw := tgt.ProdURL
	if strings.EqualFold(strings.TrimSpace(env), "staging") && strings.TrimSpace(tgt.StagingURL) != "" {
		raw = tgt.StagingURL
	}
	return a.openSafeURL(raw)
}

func (a *App) makeProgressEmitter(jobID, projectID string) func(stage string, done, total int64) {
	var mu sync.Mutex
	lastEmit := time.Time{}
	return func(stage string, done, total int64) {
		if a.ctx == nil {
			return
		}
		mu.Lock()
		if time.Since(lastEmit) < 150*time.Millisecond && done < total {
			mu.Unlock()
			return
		}
		lastEmit = time.Now()
		mu.Unlock()
		pct := 0.0
		if total > 0 {
			pct = float64(done) * 100.0 / float64(total)
		}
		wailsruntime.EventsEmit(a.ctx, "release:progress", map[string]any{
			"job_id":      jobID,
			"project_id":  projectID,
			"stage":       stage,
			"bytes_done":  done,
			"bytes_total": total,
			"percent":     pct,
		})
	}
}
