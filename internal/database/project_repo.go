package database

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"t-deck/internal/config"
	"t-deck/internal/fsutil"
	"t-deck/internal/model"
)

// ProjectRepo handles persistence, local state updates, and FTS5 search.
type ProjectRepo struct {
	db            *sql.DB
	thumbnailsDir string

	thumbMu     sync.RWMutex
	thumbLoaded bool
	thumbCache  map[string]bool
}

// NewProjectRepo constructs a ProjectRepo bound to the given SQLite DB and thumbnails folder.
func NewProjectRepo(db *sql.DB, thumbnailsDir string) *ProjectRepo {
	return &ProjectRepo{
		db:            db,
		thumbnailsDir: thumbnailsDir,
		thumbCache:    make(map[string]bool),
	}
}

// ListProjects queries projects filtered by type, category, status, tag, and FTS5 search query.
func (r *ProjectRepo) ListProjects(filter model.ProjectFilter) ([]model.ProjectView, error) {
	var args []any
	var where []string

	trimmedQuery := strings.TrimSpace(filter.Query)
	ftsExpr := ""
	if trimmedQuery != "" {
		ftsExpr = buildFTSQuery(trimmedQuery)
	}
	useFTS := ftsExpr != ""

	baseSelect := `
		SELECT
			p.id, p.slug, p.name, p.type, COALESCE(p.category, ''), p.summary, p.tags, p.tech_stack,
			p.git_remote, p.bat_file, p.exe_name, p.latest_version, p.r2_release_key,
			p.release_notes, p.thumbnail_updated_at, p.updated_at,
			COALESCE(l.local_dir, ''), COALESCE(l.status, 'cloud_only'), COALESCE(l.last_opened_at, 0),
			COALESCE(l.has_env, 0), COALESCE(l.has_env_example, 0), COALESCE(l.has_bat, 0), COALESCE(l.flags_checked_at, 0),
			COALESCE(l.installed_version, ''), COALESCE(l.git_branch, ''), COALESCE(l.git_commit, ''), COALESCE(l.git_dirty, 0)
		FROM projects p
		LEFT JOIN local_state l ON l.project_id = p.id
	`
	where = append(where, `COALESCE(p.deleted_at, 0) = 0`)

	if useFTS {
		baseSelect += ` INNER JOIN projects_fts f ON f.project_id = p.id `
		where = append(where, `projects_fts MATCH ?`)
		args = append(args, ftsExpr)
	} else if trimmedQuery != "" {
		// Use instr() with a bound parameter instead of LIKE "%...%" to prevent LIKE wildcard injection.
		needle := strings.ToLower(trimmedQuery)
		where = append(where, `(instr(LOWER(p.name), ?) > 0 OR instr(LOWER(p.summary), ?) > 0 OR instr(LOWER(p.tags), ?) > 0 OR instr(LOWER(p.tech_stack), ?) > 0 OR instr(LOWER(COALESCE(p.category, '')), ?) > 0)`)
		args = append(args, needle, needle, needle, needle, needle)
	}

	if t := strings.TrimSpace(filter.Type); t != "" && t != "all" {
		where = append(where, `p.type = ?`)
		args = append(args, t)
	}

	if cat := strings.TrimSpace(filter.Category); cat != "" && cat != "all" {
		if cat == "__uncategorized__" {
			where = append(where, `TRIM(COALESCE(p.category, '')) = ''`)
		} else {
			where = append(where, `LOWER(COALESCE(p.category, '')) = LOWER(?)`)
			args = append(args, cat)
		}
	}

	if st := strings.TrimSpace(filter.Status); st != "" && st != "all" {
		where = append(where, `COALESCE(l.status, 'cloud_only') = ?`)
		args = append(args, st)
	}

	if tag := strings.TrimSpace(strings.TrimPrefix(filter.Tag, "#")); tag != "" {
		// Exact JSON array element match via json_each() with a bound parameter (no LIKE / string formatting).
		where = append(where, `json_valid(p.tags) AND EXISTS (SELECT 1 FROM json_each(p.tags) WHERE LOWER(value) = LOWER(?))`)
		args = append(args, tag)
	}

	querySQL := baseSelect
	if len(where) > 0 {
		querySQL += " WHERE " + strings.Join(where, " AND ")
	}
	if useFTS {
		querySQL += ` ORDER BY (CASE WHEN instr(LOWER(p.name), LOWER(?)) > 0 OR instr(LOWER(p.slug), LOWER(?)) > 0 THEN 0 ELSE 1 END) ASC, bm25(projects_fts, 10.0, 1.0, 5.0, 4.0) ASC, LOWER(p.name) ASC, LOWER(p.slug) ASC`
		args = append(args, trimmedQuery, trimmedQuery)
	} else {
		querySQL += " ORDER BY LOWER(p.name) ASC, LOWER(p.slug) ASC"
	}

	rows, err := r.db.Query(querySQL, args...)
	if err != nil {
		return nil, fmt.Errorf("list projects: %w", err)
	}
	defer rows.Close()

	type pendingFlags struct {
		projectID string
		hasEnv    bool
		hasEnvEx  bool
		hasBat    bool
		checkedAt int64
	}
	var toPersist []pendingFlags

	results := make([]model.ProjectView, 0)
	for rows.Next() {
		view, checkedAt, err := r.scanProjectRowWithFlags(rows, false)
		if err != nil {
			return nil, err
		}
		if view.LocalDir != "" && checkedAt == 0 {
			hasEnv, hasEnvEx, hasBat := computeLocalDirFlags(view.LocalDir, view.BatFile)
			view.HasEnv = hasEnv
			view.HasEnvExample = hasEnvEx
			view.HasBat = hasBat
			toPersist = append(toPersist, pendingFlags{
				projectID: view.ID,
				hasEnv:    hasEnv,
				hasEnvEx:  hasEnvEx,
				hasBat:    hasBat,
				checkedAt: time.Now().Unix(),
			})
		}
		results = append(results, view)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	_ = rows.Close()

	for _, pf := range toPersist {
		_ = r.UpdateLocalFlags(pf.projectID, pf.hasEnv, pf.hasEnvEx, pf.hasBat, pf.checkedAt)
	}

	targetsMap, _ := r.ListAllActiveTargetsMap()
	latestMap, _ := r.ListAllLatestPublishedReleasesMap()
	for i := range results {
		r.attachTargetsAndReleases(&results[i], targetsMap[results[i].ID], latestMap[results[i].ID])
	}
	return results, nil
}

// GetProject retrieves a single ProjectView by ID and refreshes live local file flags.
func (r *ProjectRepo) GetProject(id string) (model.ProjectView, error) {
	row := r.db.QueryRow(`
		SELECT
			p.id, p.slug, p.name, p.type, COALESCE(p.category, ''), p.summary, p.tags, p.tech_stack,
			p.git_remote, p.bat_file, p.exe_name, p.latest_version, p.r2_release_key,
			p.release_notes, p.thumbnail_updated_at, p.updated_at,
			COALESCE(l.local_dir, ''), COALESCE(l.status, 'cloud_only'), COALESCE(l.last_opened_at, 0),
			COALESCE(l.has_env, 0), COALESCE(l.has_env_example, 0), COALESCE(l.has_bat, 0), COALESCE(l.flags_checked_at, 0),
			COALESCE(l.installed_version, ''), COALESCE(l.git_branch, ''), COALESCE(l.git_commit, ''), COALESCE(l.git_dirty, 0)
		FROM projects p
		LEFT JOIN local_state l ON l.project_id = p.id
		WHERE p.id = ? AND COALESCE(p.deleted_at, 0) = 0
	`, id)

	view, _, err := r.scanProjectRowWithFlags(row, true)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return model.ProjectView{}, fmt.Errorf("project %q not found", id)
		}
		return model.ProjectView{}, err
	}
	targets, _ := r.ListProjectTargets(view.ID)
	latestAll, _ := r.ListAllLatestPublishedReleasesMap()
	r.attachTargetsAndReleases(&view, targets, latestAll[view.ID])
	return view, nil
}

// GetProjectModel retrieves raw Project metadata by ID (active non-deleted projects only).
func (r *ProjectRepo) GetProjectModel(id string) (model.Project, bool, error) {
	row := r.db.QueryRow(`
		SELECT
			id, slug, name, type, COALESCE(category, ''), summary, tags, tech_stack,
			git_remote, bat_file, exe_name, latest_version, r2_release_key,
			release_notes, thumbnail_updated_at, updated_at, COALESCE(deleted_at, 0)
		FROM projects WHERE id = ? AND COALESCE(deleted_at, 0) = 0
	`, id)

	var p model.Project
	var tagsJSON string
	err := row.Scan(
		&p.ID, &p.Slug, &p.Name, &p.Type, &p.Category, &p.Summary, &tagsJSON, &p.TechStack,
		&p.GitRemote, &p.BatFile, &p.ExeName, &p.LatestVersion, &p.R2ReleaseKey,
		&p.ReleaseNotes, &p.ThumbnailUpdatedAt, &p.UpdatedAt, &p.DeletedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return model.Project{}, false, nil
		}
		return model.Project{}, false, err
	}
	p.Tags = decodeTags(tagsJSON)
	return p, true, nil
}

// ListAllProjects returns all active project metadata records.
func (r *ProjectRepo) ListAllProjects() ([]model.Project, error) {
	return r.listProjectsRaw(false)
}

// ListAllProjectsIncludingDeleted returns all project records including soft-deleted tombstones for R2 sync.
func (r *ProjectRepo) ListAllProjectsIncludingDeleted() ([]model.Project, error) {
	return r.listProjectsRaw(true)
}

func (r *ProjectRepo) listProjectsRaw(includeDeleted bool) ([]model.Project, error) {
	query := `
		SELECT
			id, slug, name, type, COALESCE(category, ''), summary, tags, tech_stack,
			git_remote, bat_file, exe_name, latest_version, r2_release_key,
			release_notes, thumbnail_updated_at, updated_at, COALESCE(deleted_at, 0)
		FROM projects
	`
	if !includeDeleted {
		query += ` WHERE COALESCE(deleted_at, 0) = 0 `
	}
	query += ` ORDER BY updated_at DESC`

	rows, err := r.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var projects []model.Project
	for rows.Next() {
		var p model.Project
		var tagsJSON string
		if err := rows.Scan(
			&p.ID, &p.Slug, &p.Name, &p.Type, &p.Category, &p.Summary, &tagsJSON, &p.TechStack,
			&p.GitRemote, &p.BatFile, &p.ExeName, &p.LatestVersion, &p.R2ReleaseKey,
			&p.ReleaseNotes, &p.ThumbnailUpdatedAt, &p.UpdatedAt, &p.DeletedAt,
		); err != nil {
			return nil, err
		}
		p.Tags = decodeTags(tagsJSON)
		projects = append(projects, p)
	}
	return projects, rows.Err()
}

// ListAllLocalStates returns all local_state records keyed by project_id.
func (r *ProjectRepo) ListAllLocalStates() (map[string]model.LocalState, error) {
	rows, err := r.db.Query(`
		SELECT project_id, local_dir, status, last_opened_at,
		       COALESCE(has_env, 0), COALESCE(has_env_example, 0), COALESCE(has_bat, 0), COALESCE(flags_checked_at, 0)
		FROM local_state
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	states := make(map[string]model.LocalState)
	for rows.Next() {
		var st model.LocalState
		var hasEnv, hasEnvEx, hasBat int
		if err := rows.Scan(&st.ProjectID, &st.LocalDir, &st.Status, &st.LastOpenedAt, &hasEnv, &hasEnvEx, &hasBat, &st.FlagsCheckedAt); err != nil {
			return nil, err
		}
		st.HasEnv = hasEnv != 0
		st.HasEnvExample = hasEnvEx != 0
		st.HasBat = hasBat != 0
		states[st.ProjectID] = st
	}
	return states, rows.Err()
}

// SlugExists checks if a slug is already used by another active project ID.
func (r *ProjectRepo) SlugExists(slug string, excludeID string) (bool, error) {
	var count int
	err := r.db.QueryRow(`SELECT COUNT(1) FROM projects WHERE slug = ? AND id != ? AND COALESCE(deleted_at, 0) = 0`, slug, excludeID).Scan(&count)
	return count > 0, err
}

// UpsertProject inserts or updates a project and keeps projects_fts synchronized.
func (r *ProjectRepo) UpsertProject(p model.Project, local *model.LocalState) error {
	if p.ID == "" {
		return errors.New("project id is required")
	}
	if p.Type != config.ProjectTypeWeb && p.Type != config.ProjectTypeDesktop && p.Type != config.ProjectTypeHybrid {
		return fmt.Errorf("invalid project type %q", p.Type)
	}
	if p.BatFile == "" {
		p.BatFile = config.DefaultBatFile
	}
	if p.UpdatedAt == 0 {
		p.UpdatedAt = time.Now().Unix()
	}

	p.Category = strings.TrimSpace(p.Category)
	cleanTags := normalizeTags(p.Tags)
	tagsJSONBytes, err := json.Marshal(cleanTags)
	if err != nil {
		return err
	}
	tagsJSON := string(tagsJSONBytes)

	tx, err := r.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Evict any soft-deleted tombstone holding the same slug so active projects never collide on UNIQUE(slug).
	if p.DeletedAt == 0 {
		if _, err := tx.Exec(
			`UPDATE projects SET slug = slug || '__del_' || substr(id, 1, 8) WHERE slug = ? AND id != ? AND COALESCE(deleted_at, 0) > 0`,
			p.Slug, p.ID,
		); err != nil {
			return fmt.Errorf("evict tombstone slug: %w", err)
		}
	}

	_, err = tx.Exec(`
		INSERT INTO projects (
			id, slug, name, type, category, summary, tags, tech_stack,
			git_remote, bat_file, exe_name, latest_version, r2_release_key,
			release_notes, thumbnail_updated_at, updated_at, deleted_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			slug = excluded.slug,
			name = excluded.name,
			type = excluded.type,
			category = excluded.category,
			summary = excluded.summary,
			tags = excluded.tags,
			tech_stack = excluded.tech_stack,
			git_remote = excluded.git_remote,
			bat_file = excluded.bat_file,
			exe_name = excluded.exe_name,
			latest_version = excluded.latest_version,
			r2_release_key = excluded.r2_release_key,
			release_notes = excluded.release_notes,
			thumbnail_updated_at = excluded.thumbnail_updated_at,
			updated_at = excluded.updated_at,
			deleted_at = excluded.deleted_at
	`,
		p.ID, p.Slug, p.Name, p.Type, p.Category, p.Summary, tagsJSON, p.TechStack,
		p.GitRemote, p.BatFile, p.ExeName, p.LatestVersion, p.R2ReleaseKey,
		p.ReleaseNotes, p.ThumbnailUpdatedAt, p.UpdatedAt, p.DeletedAt,
	)
	if err != nil {
		return fmt.Errorf("upsert project: %w", err)
	}

	if local != nil {
		status := local.Status
		if status == "" {
			status = config.StatusReady
		}
		hasEnv, hasEnvEx, hasBat := local.HasEnv, local.HasEnvExample, local.HasBat
		checkedAt := local.FlagsCheckedAt
		if local.LocalDir != "" && checkedAt == 0 {
			hasEnv, hasEnvEx, hasBat = computeLocalDirFlags(local.LocalDir, p.BatFile)
			checkedAt = time.Now().Unix()
		}
		_, err = tx.Exec(`
			INSERT INTO local_state (project_id, local_dir, status, last_opened_at, has_env, has_env_example, has_bat, flags_checked_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?)
			ON CONFLICT(project_id) DO UPDATE SET
				local_dir = excluded.local_dir,
				status = excluded.status,
				last_opened_at = CASE
					WHEN excluded.last_opened_at > local_state.last_opened_at THEN excluded.last_opened_at
					ELSE local_state.last_opened_at
				END,
				has_env = excluded.has_env,
				has_env_example = excluded.has_env_example,
				has_bat = excluded.has_bat,
				flags_checked_at = excluded.flags_checked_at
		`, p.ID, local.LocalDir, status, local.LastOpenedAt, boolToInt(hasEnv), boolToInt(hasEnvEx), boolToInt(hasBat), checkedAt)
		if err != nil {
			return fmt.Errorf("upsert local_state: %w", err)
		}
	}

	if p.DeletedAt > 0 {
		if _, err := tx.Exec(`DELETE FROM projects_fts WHERE project_id = ?`, p.ID); err != nil {
			return err
		}
		if _, err := tx.Exec(`DELETE FROM local_state WHERE project_id = ?`, p.ID); err != nil {
			return err
		}
	} else if err := syncFTSTx(tx, p.ID, p.Name, p.Summary, strings.Join(cleanTags, " "), p.TechStack); err != nil {
		return err
	}

	return tx.Commit()
}

// UpsertLocalState updates or inserts a project's local_state row.
func (r *ProjectRepo) UpsertLocalState(st model.LocalState) error {
	if st.ProjectID == "" {
		return errors.New("project_id is required")
	}
	hasEnv, hasEnvEx, hasBat := st.HasEnv, st.HasEnvExample, st.HasBat
	checkedAt := st.FlagsCheckedAt
	if st.LocalDir != "" && checkedAt == 0 {
		batFile := config.DefaultBatFile
		_ = r.db.QueryRow(`SELECT COALESCE(NULLIF(bat_file, ''), ?) FROM projects WHERE id = ?`, config.DefaultBatFile, st.ProjectID).Scan(&batFile)
		hasEnv, hasEnvEx, hasBat = computeLocalDirFlags(st.LocalDir, batFile)
		checkedAt = time.Now().Unix()
	}
	_, err := r.db.Exec(`
		INSERT INTO local_state (project_id, local_dir, status, last_opened_at, has_env, has_env_example, has_bat, flags_checked_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(project_id) DO UPDATE SET
			local_dir = excluded.local_dir,
			status = excluded.status,
			last_opened_at = CASE
				WHEN excluded.last_opened_at > local_state.last_opened_at THEN excluded.last_opened_at
				ELSE local_state.last_opened_at
			END,
			has_env = excluded.has_env,
			has_env_example = excluded.has_env_example,
			has_bat = excluded.has_bat,
			flags_checked_at = excluded.flags_checked_at
	`, st.ProjectID, st.LocalDir, st.Status, st.LastOpenedAt, boolToInt(hasEnv), boolToInt(hasEnvEx), boolToInt(hasBat), checkedAt)
	return err
}

// UpdateLocalFlags stores cached file existence flags on local_state.
func (r *ProjectRepo) UpdateLocalFlags(projectID string, hasEnv, hasEnvEx, hasBat bool, checkedAt int64) error {
	if checkedAt == 0 {
		checkedAt = time.Now().Unix()
	}
	_, err := r.db.Exec(`
		UPDATE local_state
		SET has_env = ?, has_env_example = ?, has_bat = ?, flags_checked_at = ?
		WHERE project_id = ?
	`, boolToInt(hasEnv), boolToInt(hasEnvEx), boolToInt(hasBat), checkedAt, projectID)
	return err
}

// RefreshLocalFlags checks .env, .env.example, and start.bat on disk and updates local_state.
func (r *ProjectRepo) RefreshLocalFlags(projectID string) error {
	var localDir, batFile string
	err := r.db.QueryRow(`
		SELECT COALESCE(l.local_dir, ''), COALESCE(NULLIF(p.bat_file, ''), ?)
		FROM projects p
		LEFT JOIN local_state l ON l.project_id = p.id
		WHERE p.id = ?
	`, config.DefaultBatFile, projectID).Scan(&localDir, &batFile)
	if err != nil {
		return err
	}
	if strings.TrimSpace(localDir) == "" {
		return nil
	}
	hasEnv, hasEnvEx, hasBat := computeLocalDirFlags(localDir, batFile)
	return r.UpdateLocalFlags(projectID, hasEnv, hasEnvEx, hasBat, time.Now().Unix())
}

// UpdateLocalStatus updates only the status field for a project.
func (r *ProjectRepo) UpdateLocalStatus(projectID string, status string) error {
	_, err := r.db.Exec(`UPDATE local_state SET status = ? WHERE project_id = ?`, status, projectID)
	return err
}

// TouchLastOpened updates last_opened_at to current Unix timestamp.
func (r *ProjectRepo) TouchLastOpened(projectID string) error {
	now := time.Now().Unix()
	_, err := r.db.Exec(`UPDATE local_state SET last_opened_at = ? WHERE project_id = ?`, now, projectID)
	return err
}

// UpdateThumbnailTimestamp sets thumbnail_updated_at and updated_at on the project.
func (r *ProjectRepo) UpdateThumbnailTimestamp(projectID string, ts int64) error {
	_, err := r.db.Exec(
		`UPDATE projects SET thumbnail_updated_at = ?, updated_at = ? WHERE id = ?`,
		ts, ts, projectID,
	)
	if err == nil {
		r.MarkThumbnailExists(projectID, true)
	}
	return err
}

// IsProjectDeleted reports whether a project ID exists as a soft-deleted tombstone.
func (r *ProjectRepo) IsProjectDeleted(id string) (bool, error) {
	if strings.TrimSpace(id) == "" {
		return false, nil
	}
	var deletedAt int64
	err := r.db.QueryRow(`SELECT COALESCE(deleted_at, 0) FROM projects WHERE id = ?`, id).Scan(&deletedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, nil
		}
		return false, err
	}
	return deletedAt > 0, nil
}

// DeleteProject soft-deletes a project (setting deleted_at tombstone) and cleans local_state and FTS5.
func (r *ProjectRepo) DeleteProject(id string) error {
	now := time.Now().Unix()
	return r.SoftDeleteProjectAt(id, now)
}

// DeleteProjects soft-deletes multiple projects atomically in a single SQLite transaction.
func (r *ProjectRepo) DeleteProjects(ids []string) (int, error) {
	clean := make([]string, 0, len(ids))
	seen := make(map[string]struct{}, len(ids))
	for _, raw := range ids {
		id := strings.TrimSpace(raw)
		if id == "" {
			continue
		}
		if _, dup := seen[id]; dup {
			continue
		}
		seen[id] = struct{}{}
		clean = append(clean, id)
	}
	if len(clean) == 0 {
		return 0, nil
	}

	now := time.Now().Unix()
	tx, err := r.db.Begin()
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()

	deleted := 0
	for _, id := range clean {
		if _, err := tx.Exec(`DELETE FROM projects_fts WHERE project_id = ?`, id); err != nil {
			return 0, err
		}
		if _, err := tx.Exec(`DELETE FROM local_state WHERE project_id = ?`, id); err != nil {
			return 0, err
		}
		res, err := tx.Exec(`UPDATE projects SET deleted_at = ?, updated_at = ? WHERE id = ? AND COALESCE(deleted_at, 0) = 0`, now, now, id)
		if err != nil {
			return 0, err
		}
		if affected, _ := res.RowsAffected(); affected > 0 {
			deleted++
		}
	}
	if err := tx.Commit(); err != nil {
		return 0, err
	}
	for _, id := range clean {
		r.MarkThumbnailExists(id, false)
	}
	return deleted, nil
}

// SoftDeleteProjectAt marks a project deleted with an explicit timestamp (used by local delete and R2 sync).
func (r *ProjectRepo) SoftDeleteProjectAt(id string, deletedAt int64) error {
	if deletedAt <= 0 {
		deletedAt = time.Now().Unix()
	}
	tx, err := r.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.Exec(`DELETE FROM projects_fts WHERE project_id = ?`, id); err != nil {
		return err
	}
	if _, err := tx.Exec(`DELETE FROM local_state WHERE project_id = ?`, id); err != nil {
		return err
	}
	if _, err := tx.Exec(`UPDATE projects SET deleted_at = ?, updated_at = ? WHERE id = ?`, deletedAt, deletedAt, id); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return err
	}
	r.MarkThumbnailExists(id, false)
	return nil
}

// PurgeExpiredTombstones permanently removes soft-deleted rows older than cutoffUnix.
func (r *ProjectRepo) PurgeExpiredTombstones(cutoffUnix int64) error {
	tx, err := r.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.Exec(`DELETE FROM projects WHERE deleted_at > 0 AND deleted_at < ?`, cutoffUnix); err != nil {
		return err
	}
	if _, err := tx.Exec(`DELETE FROM categories WHERE deleted_at > 0 AND deleted_at < ?`, cutoffUnix); err != nil {
		return err
	}
	return tx.Commit()
}

// MarkThumbnailExists updates the in-memory thumbnail existence cache for projectID.
func (r *ProjectRepo) MarkThumbnailExists(projectID string, exists bool) {
	if strings.TrimSpace(projectID) == "" {
		return
	}
	r.thumbMu.Lock()
	defer r.thumbMu.Unlock()
	if r.thumbCache == nil {
		r.thumbCache = make(map[string]bool)
	}
	r.thumbCache[projectID] = exists
}

// InvalidateThumbnailCache forces a re-read of thumbnailsDir on the next lookup.
func (r *ProjectRepo) InvalidateThumbnailCache() {
	r.thumbMu.Lock()
	defer r.thumbMu.Unlock()
	r.thumbLoaded = false
	r.thumbCache = make(map[string]bool)
}

func (r *ProjectRepo) hasThumbnailCached(projectID string) bool {
	if r.thumbnailsDir == "" || projectID == "" {
		return false
	}
	r.thumbMu.RLock()
	loaded := r.thumbLoaded
	exists, known := r.thumbCache[projectID]
	r.thumbMu.RUnlock()
	if loaded {
		return exists
	}
	if known {
		return exists
	}

	r.thumbMu.Lock()
	defer r.thumbMu.Unlock()
	if !r.thumbLoaded {
		entries, err := os.ReadDir(r.thumbnailsDir)
		if err == nil {
			if r.thumbCache == nil {
				r.thumbCache = make(map[string]bool, len(entries))
			}
			for _, e := range entries {
				if e.IsDir() {
					continue
				}
				name := e.Name()
				if strings.HasSuffix(strings.ToLower(name), ".png") {
					id := strings.TrimSuffix(name, filepath.Ext(name))
					r.thumbCache[id] = true
				}
			}
		}
		r.thumbLoaded = true
	}
	return r.thumbCache[projectID]
}

type rowScanner interface {
	Scan(dest ...any) error
}

func (r *ProjectRepo) scanProjectRowWithFlags(row rowScanner, refreshLiveFlags bool) (model.ProjectView, int64, error) {
	var v model.ProjectView
	var tagsJSON string
	var hasEnvInt, hasEnvExInt, hasBatInt, gitDirtyInt int
	var flagsCheckedAt int64
	err := row.Scan(
		&v.ID, &v.Slug, &v.Name, &v.Type, &v.Category, &v.Summary, &tagsJSON, &v.TechStack,
		&v.GitRemote, &v.BatFile, &v.ExeName, &v.LatestVersion, &v.R2ReleaseKey,
		&v.ReleaseNotes, &v.ThumbnailUpdatedAt, &v.UpdatedAt,
		&v.LocalDir, &v.Status, &v.LastOpenedAt,
		&hasEnvInt, &hasEnvExInt, &hasBatInt, &flagsCheckedAt,
		&v.InstalledVersion, &v.GitBranch, &v.GitCommit, &gitDirtyInt,
	)
	if err != nil {
		return model.ProjectView{}, 0, err
	}
	v.Tags = decodeTags(tagsJSON)
	v.GitDirty = gitDirtyInt != 0
	if refreshLiveFlags && v.LocalDir != "" {
		v.HasEnv, v.HasEnvExample, v.HasBat = computeLocalDirFlags(v.LocalDir, v.BatFile)
		_ = r.UpdateLocalFlags(v.ID, v.HasEnv, v.HasEnvExample, v.HasBat, time.Now().Unix())
	} else {
		v.HasEnv = hasEnvInt != 0
		v.HasEnvExample = hasEnvExInt != 0
		v.HasBat = hasBatInt != 0
	}
	if r.hasThumbnailCached(v.ID) {
		v.ThumbnailURL = fmt.Sprintf("/thumbnails/%s.png?v=%d", v.ID, v.ThumbnailUpdatedAt)
	}
	return v, flagsCheckedAt, nil
}

func (r *ProjectRepo) attachTargetsAndReleases(v *model.ProjectView, targets []model.ProjectTarget, latest map[string]model.Release) {
	if targets == nil {
		targets = []model.ProjectTarget{}
	}
	if latest == nil {
		latest = make(map[string]model.Release)
	}
	v.Targets = targets
	v.LatestReleases = latest

	for _, t := range targets {
		if t.Kind == config.ProjectTypeWeb && v.LiveURL == "" && strings.TrimSpace(t.ProdURL) != "" {
			v.LiveURL = strings.TrimSpace(t.ProdURL)
		}
		if t.Kind == config.ProjectTypeDesktop {
			if rel, ok := latest[t.ID]; ok && rel.Version != "" && v.InstalledVersion != "" &&
				!strings.EqualFold(strings.TrimPrefix(v.InstalledVersion, "v"), strings.TrimPrefix(rel.Version, "v")) {
				v.HasUpdate = true
			}
		}
	}
	if !v.HasUpdate && v.InstalledVersion != "" && v.LatestVersion != "" &&
		!strings.EqualFold(strings.TrimPrefix(v.InstalledVersion, "v"), strings.TrimPrefix(v.LatestVersion, "v")) {
		v.HasUpdate = true
	}
}

func computeLocalDirFlags(localDir, batFile string) (hasEnv, hasEnvEx, hasBat bool) {
	if strings.TrimSpace(localDir) == "" {
		return false, false, false
	}
	hasEnv = fsutil.FileExists(filepath.Join(localDir, config.EnvFile))
	hasEnvEx = fsutil.FileExists(filepath.Join(localDir, config.EnvExampleFile))
	batName := strings.TrimSpace(batFile)
	if batName == "" {
		batName = config.DefaultBatFile
	}
	hasBat = fsutil.FileExists(filepath.Join(localDir, batName))
	return hasEnv, hasEnvEx, hasBat
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}
