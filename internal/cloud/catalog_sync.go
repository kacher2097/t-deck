package cloud

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"sync/atomic"
	"time"

	"t-deck/internal/config"
	"t-deck/internal/fsutil"
	"t-deck/internal/model"
)

// ProjectStore defines the persistence operations required by cloud sync and release management (DIP).
type ProjectStore interface {
	GetProject(id string) (model.ProjectView, error)
	GetProjectModel(id string) (model.Project, bool, error)
	ListAllProjects() ([]model.Project, error)
	UpsertProject(p model.Project, local *model.LocalState) error
	UpsertLocalState(st model.LocalState) error
}

// SyncCatalogWithR2 performs a two-way merge of catalog.json (categories, projects, tombstones) and thumbnails concurrently.
func SyncCatalogWithR2(ctx context.Context, store ObjectStore, repo ProjectStore, thumbnailsDir string) (model.SyncReport, error) {
	var report model.SyncReport

	var remoteCatalog model.Catalog
	exists, err := store.GetJSON(ctx, config.CatalogKey, &remoteCatalog)
	if err != nil {
		return report, fmt.Errorf("fetch remote %s: %w", config.CatalogKey, err)
	}
	if !exists {
		remoteCatalog = model.Catalog{Version: 2, Projects: []model.Project{}}
	}

	if err := syncCategoriesWithR2(repo, &remoteCatalog); err != nil {
		return report, err
	}

	var localProjects []model.Project
	if fullRepo, ok := repo.(interface {
		ListAllProjectsIncludingDeleted() ([]model.Project, error)
	}); ok {
		localProjects, err = fullRepo.ListAllProjectsIncludingDeleted()
	} else {
		localProjects, err = repo.ListAllProjects()
	}
	if err != nil {
		return report, fmt.Errorf("list local projects: %w", err)
	}

	localMap := make(map[string]model.Project, len(localProjects))
	for _, p := range localProjects {
		localMap[p.ID] = p
	}

	remoteMap := make(map[string]model.Project, len(remoteCatalog.Projects))
	for _, p := range remoteCatalog.Projects {
		remoteMap[p.ID] = p
	}

	mergedMap := make(map[string]model.Project, len(localMap)+len(remoteMap))
	softDeleter, hasSoftDelete := repo.(interface {
		SoftDeleteProjectAt(id string, deletedAt int64) error
	})

	for id, remoteProj := range remoteMap {
		localProj, hasLocal := localMap[id]
		if !hasLocal {
			if remoteProj.DeletedAt > 0 {
				if err := repo.UpsertProject(remoteProj, nil); err != nil {
					return report, fmt.Errorf("record remote tombstone %s: %w", remoteProj.Slug, err)
				}
				mergedMap[id] = remoteProj
				continue
			}
			if err := repo.UpsertProject(remoteProj, &model.LocalState{
				ProjectID: remoteProj.ID,
				LocalDir:  "",
				Status:    config.StatusCloudOnly,
			}); err != nil {
				return report, fmt.Errorf("insert cloud project %s: %w", remoteProj.Slug, err)
			}
			mergedMap[id] = remoteProj
			report.Pulled++
			continue
		}

		if remoteProj.UpdatedAt > localProj.UpdatedAt {
			if remoteProj.DeletedAt > 0 {
				if hasSoftDelete {
					if err := softDeleter.SoftDeleteProjectAt(remoteProj.ID, remoteProj.DeletedAt); err != nil {
						return report, fmt.Errorf("apply remote delete %s: %w", remoteProj.Slug, err)
					}
				} else if err := repo.UpsertProject(remoteProj, nil); err != nil {
					return report, err
				}
				_ = os.Remove(filepath.Join(thumbnailsDir, remoteProj.ID+".png"))
				_ = os.Remove(filepath.Join(thumbnailsDir, remoteProj.ID+".icon.png"))
				mergedMap[id] = remoteProj
				report.UpdatedLocal++
				continue
			}
			if err := repo.UpsertProject(remoteProj, nil); err != nil {
				return report, fmt.Errorf("update local project %s: %w", remoteProj.Slug, err)
			}
			mergedMap[id] = remoteProj
			report.UpdatedLocal++
		} else if localProj.UpdatedAt > remoteProj.UpdatedAt {
			mergedMap[id] = localProj
			report.UpdatedRemote++
		} else {
			mergedMap[id] = localProj
		}
	}

	for id, localProj := range localMap {
		if _, hasRemote := remoteMap[id]; !hasRemote {
			mergedMap[id] = localProj
			if localProj.DeletedAt == 0 {
				report.Pushed++
			}
		}
	}

	if err := syncTargetsWithR2(repo, &remoteCatalog); err != nil {
		return report, err
	}

	if err := os.MkdirAll(thumbnailsDir, 0o755); err != nil {
		return report, fmt.Errorf("create thumbnails dir: %w", err)
	}

	if err := syncThumbnailsConcurrent(ctx, store, thumbnailsDir, mergedMap, localMap, remoteMap, &report); err != nil {
		return report, err
	}
	if marker, ok := repo.(interface{ InvalidateThumbnailCache() }); ok {
		marker.InvalidateThumbnailCache()
	}

	now := time.Now().Unix()
	tombstoneCutoff := now - int64(config.TombstoneRetentionDays)*24*3600
	if purger, ok := repo.(interface{ PurgeExpiredTombstones(int64) error }); ok {
		_ = purger.PurgeExpiredTombstones(tombstoneCutoff)
	}

	mergedSlice := make([]model.Project, 0, len(mergedMap))
	for _, p := range mergedMap {
		if p.DeletedAt > 0 && p.DeletedAt < tombstoneCutoff {
			continue
		}
		mergedSlice = append(mergedSlice, p)
	}
	sort.Slice(mergedSlice, func(i, j int) bool {
		return mergedSlice[i].Slug < mergedSlice[j].Slug
	})

	mergedCats := collectMergedCategories(repo, tombstoneCutoff)
	mergedTargets := collectMergedTargets(repo, tombstoneCutoff)
	outCatalog := model.Catalog{
		Version:    2,
		UpdatedAt:  now,
		Categories: mergedCats,
		Projects:   mergedSlice,
		Targets:    mergedTargets,
	}
	if err := store.PutJSON(ctx, config.CatalogKey, outCatalog); err != nil {
		return report, fmt.Errorf("upload %s: %w", config.CatalogKey, err)
	}

	syncRemoteReleaseIndexes(ctx, store, repo, mergedSlice)

	report.SyncedAt = now
	report.Message = fmt.Sprintf(
		"Đồng bộ thành công: +%d từ Cloud, +%d đẩy lên, cập nhật %d cục bộ / %d từ xa.",
		report.Pulled, report.Pushed, report.UpdatedLocal, report.UpdatedRemote,
	)
	return report, nil
}

func syncCategoriesWithR2(repo ProjectStore, remoteCatalog *model.Catalog) error {
	catRepo, ok := repo.(interface {
		ListAllCategoriesIncludingDeleted() ([]model.Category, error)
		UpsertCategoryRecord(model.Category) error
	})
	if !ok {
		return nil
	}
	localCats, err := catRepo.ListAllCategoriesIncludingDeleted()
	if err != nil {
		return fmt.Errorf("list local categories: %w", err)
	}

	localByKey := make(map[string]model.Category, len(localCats))
	for _, c := range localCats {
		localByKey[c.ID] = c
	}
	for _, rc := range remoteCatalog.Categories {
		if rc.UpdatedAt == 0 {
			rc.UpdatedAt = rc.CreatedAt
		}
		lc, hasLocal := localByKey[rc.ID]
		if !hasLocal || rc.UpdatedAt > lc.UpdatedAt {
			if err := catRepo.UpsertCategoryRecord(rc); err != nil {
				return fmt.Errorf("sync category %s: %w", rc.Name, err)
			}
		}
	}
	return nil
}

func collectMergedCategories(repo ProjectStore, tombstoneCutoff int64) []model.Category {
	catRepo, ok := repo.(interface {
		ListAllCategoriesIncludingDeleted() ([]model.Category, error)
	})
	if !ok {
		return nil
	}
	all, err := catRepo.ListAllCategoriesIncludingDeleted()
	if err != nil {
		return nil
	}
	out := make([]model.Category, 0, len(all))
	for _, c := range all {
		if c.DeletedAt > 0 && c.DeletedAt < tombstoneCutoff {
			continue
		}
		out = append(out, c)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].SortOrder != out[j].SortOrder {
			return out[i].SortOrder < out[j].SortOrder
		}
		return out[i].Slug < out[j].Slug
	})
	return out
}

func syncTargetsWithR2(repo ProjectStore, remoteCatalog *model.Catalog) error {
	if len(remoteCatalog.Targets) == 0 {
		return nil
	}
	tRepo, ok := repo.(interface {
		ListAllTargetsIncludingDeleted() ([]model.ProjectTarget, error)
		UpsertTarget(model.ProjectTarget) (model.ProjectTarget, error)
	})
	if !ok {
		return nil
	}
	localTargets, err := tRepo.ListAllTargetsIncludingDeleted()
	if err != nil {
		return fmt.Errorf("list local targets: %w", err)
	}
	localMap := make(map[string]model.ProjectTarget, len(localTargets))
	for _, t := range localTargets {
		localMap[t.ID] = t
	}
	for _, rt := range remoteCatalog.Targets {
		lt, hasLocal := localMap[rt.ID]
		if !hasLocal || rt.UpdatedAt > lt.UpdatedAt {
			if _, err := tRepo.UpsertTarget(rt); err != nil {
				return fmt.Errorf("sync target %s: %w", rt.Name, err)
			}
		}
	}
	return nil
}

func collectMergedTargets(repo ProjectStore, tombstoneCutoff int64) []model.ProjectTarget {
	tRepo, ok := repo.(interface {
		ListAllTargetsIncludingDeleted() ([]model.ProjectTarget, error)
	})
	if !ok {
		return nil
	}
	all, err := tRepo.ListAllTargetsIncludingDeleted()
	if err != nil {
		return nil
	}
	out := make([]model.ProjectTarget, 0, len(all))
	for _, t := range all {
		if t.DeletedAt > 0 && t.DeletedAt < tombstoneCutoff {
			continue
		}
		out = append(out, t)
	}
	return out
}

func syncRemoteReleaseIndexes(ctx context.Context, store ObjectStore, repo ProjectStore, projects []model.Project) {
	relRepo, ok := repo.(releaseCatalogStore)
	if !ok {
		return
	}
	for _, p := range projects {
		if p.DeletedAt > 0 || p.LatestVersion == "" {
			continue
		}
		existing, err := relRepo.ListReleases(p.ID, "")
		if err == nil && len(existing) > 0 {
			continue
		}
		var idx model.ProjectReleaseIndex
		key := fmt.Sprintf("%s%s/index.json", config.ReleasesPrefix, p.Slug)
		okGet, err := store.GetJSON(ctx, key, &idx)
		if err != nil || !okGet {
			continue
		}
		for _, r := range idx.Releases {
			_, _ = relRepo.UpsertRelease(r)
		}
	}
}

func syncThumbnailsConcurrent(
	ctx context.Context,
	store ObjectStore,
	thumbnailsDir string,
	mergedMap, localMap, remoteMap map[string]model.Project,
	report *model.SyncReport,
) error {
	type thumbTask struct {
		id         string
		localPath  string
		remoteKey  string
		isDownload bool
	}
	var tasks []thumbTask
	for id, mergedProj := range mergedMap {
		if mergedProj.DeletedAt > 0 || mergedProj.ThumbnailUpdatedAt <= 0 {
			continue
		}
		localThumbPath := filepath.Join(thumbnailsDir, id+".png")
		remoteKey := config.ThumbnailsPrefix + id + ".png"

		localProj, hasLocal := localMap[id]
		remoteProj, hasRemote := remoteMap[id]

		localThumbExists := fsutil.FileExists(localThumbPath)
		shouldDownload := !localThumbExists || (hasRemote && remoteProj.ThumbnailUpdatedAt > localProj.ThumbnailUpdatedAt)
		shouldUpload := localThumbExists && !shouldDownload && (!hasRemote || (hasLocal && localProj.ThumbnailUpdatedAt > remoteProj.ThumbnailUpdatedAt))

		if shouldDownload {
			tasks = append(tasks, thumbTask{id: id, localPath: localThumbPath, remoteKey: remoteKey, isDownload: true})
		} else if shouldUpload {
			tasks = append(tasks, thumbTask{id: id, localPath: localThumbPath, remoteKey: remoteKey, isDownload: false})
		}
	}
	if len(tasks) == 0 {
		return nil
	}

	var downCount, upCount atomic.Int64
	var errOnce sync.Once
	var firstErr error
	sem := make(chan struct{}, 6)
	var wg sync.WaitGroup

	for _, t := range tasks {
		t := t
		wg.Add(1)
		go func() {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			if t.isDownload {
				if err := store.DownloadFile(ctx, t.remoteKey, t.localPath); err != nil {
					if errors.Is(err, ErrObjectNotFound) || errors.Is(err, os.ErrNotExist) {
						return
					}
					errOnce.Do(func() { firstErr = fmt.Errorf("download thumbnail %s: %w", t.id, err) })
					return
				}
				downCount.Add(1)
			} else {
				if err := store.UploadFile(ctx, t.remoteKey, t.localPath, "image/png"); err != nil {
					errOnce.Do(func() { firstErr = fmt.Errorf("upload thumbnail %s: %w", t.id, err) })
					return
				}
				upCount.Add(1)
			}
		}()
	}
	wg.Wait()

	report.ThumbnailsDown = int(downCount.Load())
	report.ThumbnailsUp = int(upCount.Load())
	return firstErr
}
