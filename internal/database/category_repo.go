package database

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	"t-deck/internal/fsutil"
	"t-deck/internal/model"
)

// ListCategories returns all active user-defined categories ordered by sort_order and name.
func (r *ProjectRepo) ListCategories() ([]model.Category, error) {
	return r.listCategoriesRaw(false)
}

// ListAllCategoriesIncludingDeleted returns all categories including soft-deleted tombstones for R2 sync.
func (r *ProjectRepo) ListAllCategoriesIncludingDeleted() ([]model.Category, error) {
	return r.listCategoriesRaw(true)
}

func (r *ProjectRepo) listCategoriesRaw(includeDeleted bool) ([]model.Category, error) {
	query := `
		SELECT id, name, slug, sort_order, created_at, COALESCE(updated_at, created_at), COALESCE(deleted_at, 0)
		FROM categories
	`
	if !includeDeleted {
		query += ` WHERE COALESCE(deleted_at, 0) = 0 `
	}
	query += ` ORDER BY sort_order ASC, LOWER(name) ASC`

	rows, err := r.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("list categories: %w", err)
	}
	defer rows.Close()

	out := make([]model.Category, 0)
	for rows.Next() {
		var c model.Category
		if err := rows.Scan(&c.ID, &c.Name, &c.Slug, &c.SortOrder, &c.CreatedAt, &c.UpdatedAt, &c.DeletedAt); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

// CreateCategory inserts a new user-defined category (or revives a soft-deleted match).
func (r *ProjectRepo) CreateCategory(name string) (model.Category, error) {
	clean := strings.TrimSpace(name)
	if clean == "" {
		return model.Category{}, errors.New("tên phân loại không được để trống")
	}

	slug := fsutil.Slugify(clean)
	if slug == "" {
		slug = "group-" + uuid.NewString()[:8]
	}

	var existing model.Category
	err := r.db.QueryRow(`
		SELECT id, name, slug, sort_order, created_at, COALESCE(updated_at, created_at), COALESCE(deleted_at, 0)
		FROM categories
		WHERE LOWER(name) = LOWER(?) OR slug = ?
		LIMIT 1
	`, clean, slug).Scan(&existing.ID, &existing.Name, &existing.Slug, &existing.SortOrder, &existing.CreatedAt, &existing.UpdatedAt, &existing.DeletedAt)
	if err == nil {
		if existing.DeletedAt > 0 {
			now := time.Now().Unix()
			if _, updErr := r.db.Exec(
				`UPDATE categories SET deleted_at = 0, name = ?, slug = ?, updated_at = ? WHERE id = ?`,
				clean, slug, now, existing.ID,
			); updErr != nil {
				return model.Category{}, updErr
			}
			existing.Name = clean
			existing.Slug = slug
			existing.DeletedAt = 0
			existing.UpdatedAt = now
		}
		return existing, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return model.Category{}, err
	}

	var maxOrder int
	_ = r.db.QueryRow(`SELECT COALESCE(MAX(sort_order), 0) FROM categories WHERE COALESCE(deleted_at, 0) = 0`).Scan(&maxOrder)

	now := time.Now().Unix()
	cat := model.Category{
		ID:        uuid.NewString(),
		Name:      clean,
		Slug:      slug,
		SortOrder: maxOrder + 1,
		CreatedAt: now,
		UpdatedAt: now,
	}

	if _, err := r.db.Exec(`
		INSERT INTO categories (id, name, slug, sort_order, created_at, updated_at, deleted_at)
		VALUES (?, ?, ?, ?, ?, ?, 0)
	`, cat.ID, cat.Name, cat.Slug, cat.SortOrder, cat.CreatedAt, cat.UpdatedAt); err != nil {
		return model.Category{}, fmt.Errorf("create category: %w", err)
	}
	return cat, nil
}

// UpsertCategoryRecord applies a remote or merged category record (including deleted_at tombstones) during R2 sync.
func (r *ProjectRepo) UpsertCategoryRecord(cat model.Category) error {
	if strings.TrimSpace(cat.ID) == "" || strings.TrimSpace(cat.Name) == "" {
		return nil
	}
	if cat.Slug == "" {
		cat.Slug = fsutil.Slugify(cat.Name)
		if cat.Slug == "" {
			cat.Slug = "group-" + cat.ID[:minLen(len(cat.ID), 8)]
		}
	}
	if cat.CreatedAt == 0 {
		cat.CreatedAt = time.Now().Unix()
	}
	if cat.UpdatedAt == 0 {
		cat.UpdatedAt = cat.CreatedAt
	}

	tx, err := r.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Evict any conflicting soft-deleted row with same name/slug under a different ID.
	if cat.DeletedAt == 0 {
		_, _ = tx.Exec(
			`UPDATE categories SET name = name || '__del_' || substr(id, 1, 6), slug = slug || '__del_' || substr(id, 1, 6) WHERE id != ? AND (LOWER(name) = LOWER(?) OR slug = ?) AND COALESCE(deleted_at, 0) > 0`,
			cat.ID, cat.Name, cat.Slug,
		)
	}

	// If an active row exists with the same name/slug under another ID, update its timestamps/tombstone instead of crashing UNIQUE.
	var conflictID string
	err = tx.QueryRow(`SELECT id FROM categories WHERE id != ? AND (LOWER(name) = LOWER(?) OR slug = ?) LIMIT 1`, cat.ID, cat.Name, cat.Slug).Scan(&conflictID)
	if err == nil && conflictID != "" {
		if _, err := tx.Exec(
			`UPDATE categories SET name = ?, slug = ?, sort_order = ?, updated_at = ?, deleted_at = ? WHERE id = ?`,
			cat.Name, cat.Slug, cat.SortOrder, cat.UpdatedAt, cat.DeletedAt, conflictID,
		); err != nil {
			return err
		}
		return tx.Commit()
	}

	if _, err := tx.Exec(`
		INSERT INTO categories (id, name, slug, sort_order, created_at, updated_at, deleted_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			name = excluded.name,
			slug = excluded.slug,
			sort_order = excluded.sort_order,
			updated_at = excluded.updated_at,
			deleted_at = excluded.deleted_at
	`, cat.ID, cat.Name, cat.Slug, cat.SortOrder, cat.CreatedAt, cat.UpdatedAt, cat.DeletedAt); err != nil {
		return err
	}
	return tx.Commit()
}

// RenameCategory updates a category's name and updates matching projects atomically.
func (r *ProjectRepo) RenameCategory(id string, newName string) (model.Category, error) {
	clean := strings.TrimSpace(newName)
	if clean == "" {
		return model.Category{}, errors.New("tên phân loại không được để trống")
	}

	var cat model.Category
	err := r.db.QueryRow(`SELECT id, name, slug, sort_order, created_at, COALESCE(updated_at, created_at), COALESCE(deleted_at, 0) FROM categories WHERE id = ? AND COALESCE(deleted_at, 0) = 0`, id).
		Scan(&cat.ID, &cat.Name, &cat.Slug, &cat.SortOrder, &cat.CreatedAt, &cat.UpdatedAt, &cat.DeletedAt)
	if err != nil {
		return model.Category{}, fmt.Errorf("category not found: %w", err)
	}

	now := time.Now().Unix()
	oldName := cat.Name
	cat.Name = clean
	cat.UpdatedAt = now
	if newSlug := fsutil.Slugify(clean); newSlug != "" {
		cat.Slug = newSlug
	}

	tx, err := r.db.Begin()
	if err != nil {
		return model.Category{}, err
	}
	defer tx.Rollback()

	if _, err := tx.Exec(`UPDATE categories SET name = ?, slug = ?, updated_at = ? WHERE id = ?`, cat.Name, cat.Slug, now, id); err != nil {
		return model.Category{}, err
	}
	if _, err := tx.Exec(`UPDATE projects SET category = ?, updated_at = ? WHERE category = ? AND COALESCE(deleted_at, 0) = 0`, cat.Name, now, oldName); err != nil {
		return model.Category{}, err
	}
	if err := tx.Commit(); err != nil {
		return model.Category{}, err
	}
	return cat, nil
}

// DeleteCategory soft-deletes a category and clears it from any projects assigned to it.
func (r *ProjectRepo) DeleteCategory(id string) error {
	var name string
	err := r.db.QueryRow(`SELECT name FROM categories WHERE id = ? AND COALESCE(deleted_at, 0) = 0`, id).Scan(&name)
	if err != nil {
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

	if _, err := tx.Exec(`UPDATE projects SET category = '', updated_at = ? WHERE category = ? AND COALESCE(deleted_at, 0) = 0`, now, name); err != nil {
		return err
	}
	if _, err := tx.Exec(`UPDATE categories SET deleted_at = ?, updated_at = ? WHERE id = ?`, now, now, id); err != nil {
		return err
	}
	return tx.Commit()
}

// UpsertCategories ensures categories from remote catalog exist locally.
func (r *ProjectRepo) UpsertCategories(cats []model.Category) error {
	for _, c := range cats {
		if strings.TrimSpace(c.Name) == "" {
			continue
		}
		if strings.TrimSpace(c.ID) != "" {
			if err := r.UpsertCategoryRecord(c); err != nil {
				return err
			}
			continue
		}
		if _, err := r.CreateCategory(c.Name); err != nil {
			return err
		}
	}
	return nil
}

func minLen(a, b int) int {
	if a < b {
		return a
	}
	return b
}

