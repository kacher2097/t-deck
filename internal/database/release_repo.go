package database

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	"t-deck/internal/config"
	"t-deck/internal/model"
)

// ListReleases returns active releases for a project (optionally filtered by targetID), newest first.
func (r *ProjectRepo) ListReleases(projectID string, targetID string) ([]model.Release, error) {
	query := `
		SELECT id, project_id, target_id, version, channel, git_tag, commit_sha, changelog,
		       r2_key, file_name, file_size, sha256, deploy_url, asset_kind, status, published_at, COALESCE(deleted_at, 0)
		FROM releases
		WHERE project_id = ? AND COALESCE(deleted_at, 0) = 0
	`
	args := []any{projectID}
	if strings.TrimSpace(targetID) != "" {
		query += ` AND target_id = ?`
		args = append(args, targetID)
	}
	query += ` ORDER BY published_at DESC, rowid DESC`

	rows, err := r.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("list releases: %w", err)
	}
	defer rows.Close()
	return scanReleases(rows)
}

// ListReleasesBySlug returns active releases for a project matched by slug, newest first.
func (r *ProjectRepo) ListReleasesBySlug(slug string) ([]model.Release, error) {
	rows, err := r.db.Query(`
		SELECT r.id, r.project_id, r.target_id, r.version, r.channel, r.git_tag, r.commit_sha, r.changelog,
		       r.r2_key, r.file_name, r.file_size, r.sha256, r.deploy_url, r.asset_kind, r.status, r.published_at, COALESCE(r.deleted_at, 0)
		FROM releases r
		JOIN projects p ON p.id = r.project_id
		WHERE p.slug = ? AND COALESCE(r.deleted_at, 0) = 0
		ORDER BY r.published_at DESC, r.rowid DESC
	`, slug)
	if err != nil {
		return nil, fmt.Errorf("list releases by slug: %w", err)
	}
	defer rows.Close()
	return scanReleases(rows)
}

// GetRelease retrieves a single Release record by ID.
func (r *ProjectRepo) GetRelease(releaseID string) (model.Release, bool, error) {
	row := r.db.QueryRow(`
		SELECT id, project_id, target_id, version, channel, git_tag, commit_sha, changelog,
		       r2_key, file_name, file_size, sha256, deploy_url, asset_kind, status, published_at, COALESCE(deleted_at, 0)
		FROM releases
		WHERE id = ? AND COALESCE(deleted_at, 0) = 0
	`, releaseID)

	var rel model.Release
	err := row.Scan(
		&rel.ID, &rel.ProjectID, &rel.TargetID, &rel.Version, &rel.Channel, &rel.GitTag, &rel.CommitSHA,
		&rel.Changelog, &rel.R2Key, &rel.FileName, &rel.FileSize, &rel.SHA256, &rel.DeployURL,
		&rel.AssetKind, &rel.Status, &rel.PublishedAt, &rel.DeletedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return model.Release{}, false, nil
		}
		return model.Release{}, false, err
	}
	return rel, true, nil
}

// ListAllLatestPublishedReleasesMap returns the newest published release per target_id across all projects.
func (r *ProjectRepo) ListAllLatestPublishedReleasesMap() (map[string]map[string]model.Release, error) {
	rows, err := r.db.Query(`
		SELECT id, project_id, target_id, version, channel, git_tag, commit_sha, changelog,
		       r2_key, file_name, file_size, sha256, deploy_url, asset_kind, status, published_at, COALESCE(deleted_at, 0)
		FROM releases
		WHERE COALESCE(deleted_at, 0) = 0 AND status = 'published'
		ORDER BY published_at DESC, rowid DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	list, err := scanReleases(rows)
	if err != nil {
		return nil, err
	}

	out := make(map[string]map[string]model.Release)
	for _, rel := range list {
		byTarget, ok := out[rel.ProjectID]
		if !ok {
			byTarget = make(map[string]model.Release)
			out[rel.ProjectID] = byTarget
		}
		// Ordered by published_at DESC, so first seen per target_id is newest.
		if _, exists := byTarget[rel.TargetID]; !exists {
			byTarget[rel.TargetID] = rel
		}
	}
	return out, nil
}

// UpsertRelease inserts or updates a Release row.
func (r *ProjectRepo) UpsertRelease(rel model.Release) (model.Release, error) {
	if strings.TrimSpace(rel.ProjectID) == "" {
		return model.Release{}, errors.New("project_id is required")
	}
	if strings.TrimSpace(rel.TargetID) == "" {
		return model.Release{}, errors.New("target_id is required")
	}
	if strings.TrimSpace(rel.Version) == "" {
		return model.Release{}, errors.New("version is required")
	}
	if strings.TrimSpace(rel.ID) == "" {
		rel.ID = uuid.NewString()
	}
	if strings.TrimSpace(rel.Channel) == "" {
		rel.Channel = config.ReleaseChannelStable
	}
	if strings.TrimSpace(rel.AssetKind) == "" {
		rel.AssetKind = "portable_zip"
	}
	if strings.TrimSpace(rel.Status) == "" {
		rel.Status = "published"
	}
	if rel.PublishedAt == 0 {
		rel.PublishedAt = time.Now().Unix()
	}

	_, err := r.db.Exec(`
		INSERT INTO releases (
			id, project_id, target_id, version, channel, git_tag, commit_sha, changelog,
			r2_key, file_name, file_size, sha256, deploy_url, asset_kind, status, published_at, deleted_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			version = excluded.version,
			channel = excluded.channel,
			git_tag = excluded.git_tag,
			commit_sha = excluded.commit_sha,
			changelog = excluded.changelog,
			r2_key = excluded.r2_key,
			file_name = excluded.file_name,
			file_size = excluded.file_size,
			sha256 = excluded.sha256,
			deploy_url = excluded.deploy_url,
			asset_kind = excluded.asset_kind,
			status = excluded.status,
			published_at = excluded.published_at,
			deleted_at = excluded.deleted_at
	`,
		rel.ID, rel.ProjectID, rel.TargetID, rel.Version, rel.Channel, rel.GitTag, rel.CommitSHA,
		rel.Changelog, rel.R2Key, rel.FileName, rel.FileSize, rel.SHA256, rel.DeployURL,
		rel.AssetKind, rel.Status, rel.PublishedAt, rel.DeletedAt,
	)
	if err != nil {
		return model.Release{}, fmt.Errorf("upsert release: %w", err)
	}
	return rel, nil
}

// DeleteRelease soft-deletes and yanks a release record.
func (r *ProjectRepo) DeleteRelease(releaseID string) error {
	now := time.Now().Unix()
	_, err := r.db.Exec(`UPDATE releases SET deleted_at = ?, status = 'yanked' WHERE id = ?`, now, releaseID)
	return err
}

// UpdateInstalledRelease records the installed desktop version and release ID on local_state.
func (r *ProjectRepo) UpdateInstalledRelease(projectID, version, releaseID string) error {
	now := time.Now().Unix()
	_, err := r.db.Exec(`
		UPDATE local_state
		SET installed_version = ?, installed_release_id = ?, installed_at = ?
		WHERE project_id = ?
	`, version, releaseID, now, projectID)
	return err
}

// UpdateGitState records git branch, commit, and dirty flag into local_state.
func (r *ProjectRepo) UpdateGitState(projectID, branch, commit string, dirty bool) error {
	_, err := r.db.Exec(`
		UPDATE local_state
		SET git_branch = ?, git_commit = ?, git_dirty = ?
		WHERE project_id = ?
	`, branch, commit, boolToInt(dirty), projectID)
	return err
}

func scanReleases(rows *sql.Rows) ([]model.Release, error) {
	out := make([]model.Release, 0)
	for rows.Next() {
		var rel model.Release
		if err := rows.Scan(
			&rel.ID, &rel.ProjectID, &rel.TargetID, &rel.Version, &rel.Channel, &rel.GitTag, &rel.CommitSHA,
			&rel.Changelog, &rel.R2Key, &rel.FileName, &rel.FileSize, &rel.SHA256, &rel.DeployURL,
			&rel.AssetKind, &rel.Status, &rel.PublishedAt, &rel.DeletedAt,
		); err != nil {
			return nil, err
		}
		out = append(out, rel)
	}
	return out, rows.Err()
}
