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

// ListProjectTargets returns all active targets belonging to projectID ordered by sort_order.
func (r *ProjectRepo) ListProjectTargets(projectID string) ([]model.ProjectTarget, error) {
	rows, err := r.db.Query(`
		SELECT id, project_id, kind, name, platform, distribution, subpath, build_cmd,
		       artifact_glob, launch_entry, prod_url, staging_url, health_url, version_url,
		       hosting_provider, sort_order, updated_at, COALESCE(deleted_at, 0)
		FROM project_targets
		WHERE project_id = ? AND COALESCE(deleted_at, 0) = 0
		ORDER BY sort_order ASC, LOWER(name) ASC
	`, projectID)
	if err != nil {
		return nil, fmt.Errorf("list project targets: %w", err)
	}
	defer rows.Close()
	return scanTargets(rows)
}

// ListAllTargetsIncludingDeleted returns all target rows including tombstones for R2 catalog sync.
func (r *ProjectRepo) ListAllTargetsIncludingDeleted() ([]model.ProjectTarget, error) {
	rows, err := r.db.Query(`
		SELECT id, project_id, kind, name, platform, distribution, subpath, build_cmd,
		       artifact_glob, launch_entry, prod_url, staging_url, health_url, version_url,
		       hosting_provider, sort_order, updated_at, COALESCE(deleted_at, 0)
		FROM project_targets
		ORDER BY project_id, sort_order ASC
	`)
	if err != nil {
		return nil, fmt.Errorf("list all targets: %w", err)
	}
	defer rows.Close()
	return scanTargets(rows)
}

// ListAllActiveTargetsMap loads all active project_targets keyed by project_id in 1 query.
func (r *ProjectRepo) ListAllActiveTargetsMap() (map[string][]model.ProjectTarget, error) {
	rows, err := r.db.Query(`
		SELECT id, project_id, kind, name, platform, distribution, subpath, build_cmd,
		       artifact_glob, launch_entry, prod_url, staging_url, health_url, version_url,
		       hosting_provider, sort_order, updated_at, COALESCE(deleted_at, 0)
		FROM project_targets
		WHERE COALESCE(deleted_at, 0) = 0
		ORDER BY sort_order ASC, LOWER(name) ASC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	list, err := scanTargets(rows)
	if err != nil {
		return nil, err
	}
	out := make(map[string][]model.ProjectTarget)
	for _, t := range list {
		out[t.ProjectID] = append(out[t.ProjectID], t)
	}
	return out, nil
}

// GetTarget retrieves a single active ProjectTarget by ID.
func (r *ProjectRepo) GetTarget(targetID string) (model.ProjectTarget, bool, error) {
	row := r.db.QueryRow(`
		SELECT id, project_id, kind, name, platform, distribution, subpath, build_cmd,
		       artifact_glob, launch_entry, prod_url, staging_url, health_url, version_url,
		       hosting_provider, sort_order, updated_at, COALESCE(deleted_at, 0)
		FROM project_targets
		WHERE id = ? AND COALESCE(deleted_at, 0) = 0
	`, targetID)

	var t model.ProjectTarget
	err := row.Scan(
		&t.ID, &t.ProjectID, &t.Kind, &t.Name, &t.Platform, &t.Distribution, &t.Subpath, &t.BuildCmd,
		&t.ArtifactGlob, &t.LaunchEntry, &t.ProdURL, &t.StagingURL, &t.HealthURL, &t.VersionURL,
		&t.HostingProvider, &t.SortOrder, &t.UpdatedAt, &t.DeletedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return model.ProjectTarget{}, false, nil
		}
		return model.ProjectTarget{}, false, err
	}
	return t, true, nil
}

// UpsertTarget creates or updates a ProjectTarget and keeps parent project type/entry synced.
func (r *ProjectRepo) UpsertTarget(t model.ProjectTarget) (model.ProjectTarget, error) {
	if strings.TrimSpace(t.ProjectID) == "" {
		return model.ProjectTarget{}, errors.New("project_id is required")
	}
	t.Kind = strings.ToLower(strings.TrimSpace(t.Kind))
	if t.Kind != config.ProjectTypeWeb && t.Kind != config.ProjectTypeDesktop {
		return model.ProjectTarget{}, fmt.Errorf("invalid target kind %q", t.Kind)
	}
	if strings.TrimSpace(t.ID) == "" {
		t.ID = uuid.NewString()
	}
	if strings.TrimSpace(t.Name) == "" {
		if t.Kind == config.ProjectTypeWeb {
			t.Name = "Web App"
		} else {
			t.Name = "Desktop App"
		}
	}
	if strings.TrimSpace(t.Platform) == "" {
		if t.Kind == config.ProjectTypeWeb {
			t.Platform = "web"
		} else {
			t.Platform = "windows"
		}
	}
	if strings.TrimSpace(t.Distribution) == "" {
		if t.Kind == config.ProjectTypeWeb {
			t.Distribution = "hosted"
		} else {
			t.Distribution = "portable"
		}
	}
	if t.UpdatedAt == 0 {
		t.UpdatedAt = time.Now().Unix()
	}

	tx, err := r.db.Begin()
	if err != nil {
		return model.ProjectTarget{}, err
	}
	defer tx.Rollback()

	_, err = tx.Exec(`
		INSERT INTO project_targets (
			id, project_id, kind, name, platform, distribution, subpath, build_cmd,
			artifact_glob, launch_entry, prod_url, staging_url, health_url, version_url,
			hosting_provider, sort_order, updated_at, deleted_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			kind = excluded.kind,
			name = excluded.name,
			platform = excluded.platform,
			distribution = excluded.distribution,
			subpath = excluded.subpath,
			build_cmd = excluded.build_cmd,
			artifact_glob = excluded.artifact_glob,
			launch_entry = excluded.launch_entry,
			prod_url = excluded.prod_url,
			staging_url = excluded.staging_url,
			health_url = excluded.health_url,
			version_url = excluded.version_url,
			hosting_provider = excluded.hosting_provider,
			sort_order = excluded.sort_order,
			updated_at = excluded.updated_at,
			deleted_at = excluded.deleted_at
	`,
		t.ID, t.ProjectID, t.Kind, strings.TrimSpace(t.Name), t.Platform, t.Distribution,
		strings.TrimSpace(t.Subpath), strings.TrimSpace(t.BuildCmd), strings.TrimSpace(t.ArtifactGlob),
		strings.TrimSpace(t.LaunchEntry), strings.TrimSpace(t.ProdURL), strings.TrimSpace(t.StagingURL),
		strings.TrimSpace(t.HealthURL), strings.TrimSpace(t.VersionURL), strings.TrimSpace(t.HostingProvider),
		t.SortOrder, t.UpdatedAt, t.DeletedAt,
	)
	if err != nil {
		return model.ProjectTarget{}, fmt.Errorf("upsert target: %w", err)
	}

	if err := syncProjectTypeFromTargetsTx(tx, t.ProjectID, t.UpdatedAt); err != nil {
		return model.ProjectTarget{}, err
	}

	if err := tx.Commit(); err != nil {
		return model.ProjectTarget{}, err
	}
	return t, nil
}

// DeleteTarget soft-deletes a target and recalculates parent project type.
func (r *ProjectRepo) DeleteTarget(targetID string) error {
	var projectID string
	if err := r.db.QueryRow(`SELECT project_id FROM project_targets WHERE id = ?`, targetID).Scan(&projectID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil
		}
		return err
	}
	now := time.Now().Unix()
	tx, err := r.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.Exec(`UPDATE project_targets SET deleted_at = ?, updated_at = ? WHERE id = ?`, now, now, targetID); err != nil {
		return err
	}
	if err := syncProjectTypeFromTargetsTx(tx, projectID, now); err != nil {
		return err
	}
	return tx.Commit()
}

func syncProjectTypeFromTargetsTx(tx *sql.Tx, projectID string, now int64) error {
	rows, err := tx.Query(`
		SELECT kind, launch_entry
		FROM project_targets
		WHERE project_id = ? AND COALESCE(deleted_at, 0) = 0
		ORDER BY sort_order ASC, updated_at DESC
	`, projectID)
	if err != nil {
		return err
	}
	defer rows.Close()

	var hasWeb, hasDesktop bool
	var webBat, deskExe string
	for rows.Next() {
		var kind, entry string
		if err := rows.Scan(&kind, &entry); err != nil {
			return err
		}
		if kind == config.ProjectTypeWeb {
			hasWeb = true
			if webBat == "" && entry != "" {
				webBat = entry
			}
		} else if kind == config.ProjectTypeDesktop {
			hasDesktop = true
			if deskExe == "" && entry != "" {
				deskExe = entry
			}
		}
	}
	if err := rows.Err(); err != nil {
		return err
	}

	if hasWeb && hasDesktop {
		if _, err := tx.Exec(`UPDATE projects SET type = ?, updated_at = ? WHERE id = ?`, config.ProjectTypeHybrid, now, projectID); err != nil {
			return err
		}
	} else if hasDesktop {
		if _, err := tx.Exec(`UPDATE projects SET type = ?, updated_at = ? WHERE id = ?`, config.ProjectTypeDesktop, now, projectID); err != nil {
			return err
		}
	} else if hasWeb {
		if _, err := tx.Exec(`UPDATE projects SET type = ?, updated_at = ? WHERE id = ?`, config.ProjectTypeWeb, now, projectID); err != nil {
			return err
		}
	}
	if webBat != "" {
		_, _ = tx.Exec(`UPDATE projects SET bat_file = ? WHERE id = ?`, webBat, projectID)
	}
	if deskExe != "" {
		_, _ = tx.Exec(`UPDATE projects SET exe_name = ? WHERE id = ?`, deskExe, projectID)
	}
	return nil
}

func scanTargets(rows *sql.Rows) ([]model.ProjectTarget, error) {
	out := make([]model.ProjectTarget, 0)
	for rows.Next() {
		var t model.ProjectTarget
		if err := rows.Scan(
			&t.ID, &t.ProjectID, &t.Kind, &t.Name, &t.Platform, &t.Distribution, &t.Subpath, &t.BuildCmd,
			&t.ArtifactGlob, &t.LaunchEntry, &t.ProdURL, &t.StagingURL, &t.HealthURL, &t.VersionURL,
			&t.HostingProvider, &t.SortOrder, &t.UpdatedAt, &t.DeletedAt,
		); err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, rows.Err()
}
