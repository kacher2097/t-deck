package database

import (
	"database/sql"
	"fmt"
	"io"
	"os"
	"path/filepath"

	_ "modernc.org/sqlite"
)

const schemaSQL = `
PRAGMA journal_mode = WAL;
PRAGMA foreign_keys = ON;
PRAGMA busy_timeout = 5000;
PRAGMA synchronous = NORMAL;
PRAGMA temp_store = MEMORY;
PRAGMA cache_size = -16000;

CREATE TABLE IF NOT EXISTS projects (
    id TEXT PRIMARY KEY,
    slug TEXT UNIQUE NOT NULL,
    name TEXT NOT NULL,
    type TEXT NOT NULL CHECK(type IN ('web', 'desktop')),
    category TEXT DEFAULT '',
    summary TEXT DEFAULT '',
    tags TEXT DEFAULT '[]',
    tech_stack TEXT DEFAULT '',
    git_remote TEXT DEFAULT '',
    bat_file TEXT DEFAULT 'start.bat',
    exe_name TEXT DEFAULT '',
    latest_version TEXT DEFAULT '',
    r2_release_key TEXT DEFAULT '',
    release_notes TEXT DEFAULT '',
    thumbnail_updated_at INTEGER DEFAULT 0,
    updated_at INTEGER NOT NULL,
    deleted_at INTEGER DEFAULT 0
);

CREATE TABLE IF NOT EXISTS categories (
    id TEXT PRIMARY KEY,
    name TEXT UNIQUE NOT NULL,
    slug TEXT UNIQUE NOT NULL,
    sort_order INTEGER DEFAULT 0,
    created_at INTEGER NOT NULL,
    updated_at INTEGER DEFAULT 0,
    deleted_at INTEGER DEFAULT 0
);

CREATE TABLE IF NOT EXISTS local_state (
    project_id TEXT PRIMARY KEY,
    local_dir TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'ready' CHECK(status IN ('ready', 'cloud_only', 'missing_env', 'missing_dir')),
    last_opened_at INTEGER DEFAULT 0,
    has_env INTEGER DEFAULT 0,
    has_env_example INTEGER DEFAULT 0,
    has_bat INTEGER DEFAULT 0,
    flags_checked_at INTEGER DEFAULT 0,
    FOREIGN KEY (project_id) REFERENCES projects(id) ON DELETE CASCADE
);

CREATE VIRTUAL TABLE IF NOT EXISTS projects_fts USING fts5(
    project_id UNINDEXED,
    name,
    summary,
    tags,
    tech_stack
);
`

// Open initializes a SQLite connection with WAL mode, foreign keys, and schema migrations.
func Open(dbPath string) (*sql.DB, error) {
	if err := os.MkdirAll(filepath.Dir(dbPath), 0o755); err != nil {
		return nil, fmt.Errorf("create db dir: %w", err)
	}

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}

	db.SetMaxOpenConns(1)

	if err := db.Ping(); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("ping sqlite: %w", err)
	}

	if _, err := db.Exec(schemaSQL); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("migrate sqlite schema: %w", err)
	}

	if err := migrateColumns(db, dbPath); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("migrate columns: %w", err)
	}

	return db, nil
}

func migrateColumns(db *sql.DB, dbPath string) error {
	var userVer int
	if err := db.QueryRow(`PRAGMA user_version`).Scan(&userVer); err != nil {
		return err
	}

	projCols, err := tableColumns(db, "projects")
	if err != nil {
		return err
	}
	if !projCols["category"] {
		if _, err := db.Exec(`ALTER TABLE projects ADD COLUMN category TEXT DEFAULT ''`); err != nil {
			return err
		}
	}
	if !projCols["deleted_at"] {
		if _, err := db.Exec(`ALTER TABLE projects ADD COLUMN deleted_at INTEGER DEFAULT 0`); err != nil {
			return err
		}
	}

	catCols, err := tableColumns(db, "categories")
	if err != nil {
		return err
	}
	if !catCols["deleted_at"] {
		if _, err := db.Exec(`ALTER TABLE categories ADD COLUMN deleted_at INTEGER DEFAULT 0`); err != nil {
			return err
		}
	}
	if !catCols["updated_at"] {
		if _, err := db.Exec(`ALTER TABLE categories ADD COLUMN updated_at INTEGER DEFAULT 0`); err != nil {
			return err
		}
		if _, err := db.Exec(`UPDATE categories SET updated_at = created_at WHERE updated_at = 0`); err != nil {
			return err
		}
	}

	localCols, err := tableColumns(db, "local_state")
	if err != nil {
		return err
	}
	for _, col := range []struct {
		name string
		ddl  string
	}{
		{"has_env", `ALTER TABLE local_state ADD COLUMN has_env INTEGER DEFAULT 0`},
		{"has_env_example", `ALTER TABLE local_state ADD COLUMN has_env_example INTEGER DEFAULT 0`},
		{"has_bat", `ALTER TABLE local_state ADD COLUMN has_bat INTEGER DEFAULT 0`},
		{"flags_checked_at", `ALTER TABLE local_state ADD COLUMN flags_checked_at INTEGER DEFAULT 0`},
	} {
		if !localCols[col.name] {
			if _, err := db.Exec(col.ddl); err != nil {
				return err
			}
		}
	}

	indexSQL := `
CREATE INDEX IF NOT EXISTS idx_projects_type ON projects(type);
CREATE INDEX IF NOT EXISTS idx_projects_category ON projects(category);
CREATE INDEX IF NOT EXISTS idx_projects_updated_at ON projects(updated_at DESC);
CREATE INDEX IF NOT EXISTS idx_projects_deleted_at ON projects(deleted_at);
CREATE INDEX IF NOT EXISTS idx_local_state_status ON local_state(status);
`
	if _, err := db.Exec(indexSQL); err != nil {
		return err
	}

	if userVer < 2 {
		if _, err := db.Exec(`PRAGMA user_version = 2`); err != nil {
			return err
		}
		userVer = 2
	}

	if userVer < 3 {
		if err := migrateToV3(db, dbPath); err != nil {
			return err
		}
	}

	return nil
}

func migrateToV3(db *sql.DB, dbPath string) error {
	if dbPath != "" && dbPath != ":memory:" {
		if info, err := os.Stat(dbPath); err == nil && info.Size() > 0 {
			bakPath := dbPath + ".v2.bak"
			if _, bakErr := os.Stat(bakPath); os.IsNotExist(bakErr) {
				_ = copyFileBackup(dbPath, bakPath)
			}
		}
	}

	if _, err := db.Exec(`PRAGMA foreign_keys = OFF`); err != nil {
		return err
	}
	defer func() {
		_, _ = db.Exec(`PRAGMA foreign_keys = ON`)
	}()

	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Rebuild projects table to allow 'hybrid' in type CHECK constraint.
	if _, err := tx.Exec(`
		CREATE TABLE IF NOT EXISTS projects_new (
			id TEXT PRIMARY KEY,
			slug TEXT UNIQUE NOT NULL,
			name TEXT NOT NULL,
			type TEXT NOT NULL CHECK(type IN ('web', 'desktop', 'hybrid')),
			category TEXT DEFAULT '',
			summary TEXT DEFAULT '',
			tags TEXT DEFAULT '[]',
			tech_stack TEXT DEFAULT '',
			git_remote TEXT DEFAULT '',
			bat_file TEXT DEFAULT 'start.bat',
			exe_name TEXT DEFAULT '',
			latest_version TEXT DEFAULT '',
			r2_release_key TEXT DEFAULT '',
			release_notes TEXT DEFAULT '',
			thumbnail_updated_at INTEGER DEFAULT 0,
			updated_at INTEGER NOT NULL,
			deleted_at INTEGER DEFAULT 0
		);
		INSERT INTO projects_new (
			id, slug, name, type, category, summary, tags, tech_stack,
			git_remote, bat_file, exe_name, latest_version, r2_release_key,
			release_notes, thumbnail_updated_at, updated_at, deleted_at
		)
		SELECT
			id, slug, name, type, category, summary, tags, tech_stack,
			git_remote, bat_file, exe_name, latest_version, r2_release_key,
			release_notes, thumbnail_updated_at, updated_at, COALESCE(deleted_at, 0)
		FROM projects;
		DROP TABLE projects;
		ALTER TABLE projects_new RENAME TO projects;
		CREATE INDEX IF NOT EXISTS idx_projects_type ON projects(type);
		CREATE INDEX IF NOT EXISTS idx_projects_category ON projects(category);
		CREATE INDEX IF NOT EXISTS idx_projects_updated_at ON projects(updated_at DESC);
		CREATE INDEX IF NOT EXISTS idx_projects_deleted_at ON projects(deleted_at);
	`); err != nil {
		return fmt.Errorf("rebuild projects table for v3: %w", err)
	}

	if _, err := tx.Exec(`
		CREATE TABLE IF NOT EXISTS project_targets (
			id TEXT PRIMARY KEY,
			project_id TEXT NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
			kind TEXT NOT NULL CHECK(kind IN ('web', 'desktop')),
			name TEXT NOT NULL DEFAULT '',
			platform TEXT NOT NULL DEFAULT 'windows',
			distribution TEXT NOT NULL DEFAULT 'portable',
			subpath TEXT DEFAULT '',
			build_cmd TEXT DEFAULT '',
			artifact_glob TEXT DEFAULT '',
			launch_entry TEXT DEFAULT '',
			prod_url TEXT DEFAULT '',
			staging_url TEXT DEFAULT '',
			health_url TEXT DEFAULT '',
			version_url TEXT DEFAULT '',
			hosting_provider TEXT DEFAULT '',
			sort_order INTEGER DEFAULT 0,
			updated_at INTEGER NOT NULL,
			deleted_at INTEGER DEFAULT 0
		);
		CREATE INDEX IF NOT EXISTS idx_targets_project ON project_targets(project_id, sort_order);

		CREATE TABLE IF NOT EXISTS releases (
			id TEXT PRIMARY KEY,
			project_id TEXT NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
			target_id TEXT NOT NULL REFERENCES project_targets(id) ON DELETE CASCADE,
			version TEXT NOT NULL,
			channel TEXT NOT NULL DEFAULT 'stable',
			git_tag TEXT DEFAULT '',
			commit_sha TEXT DEFAULT '',
			changelog TEXT DEFAULT '',
			r2_key TEXT DEFAULT '',
			file_name TEXT DEFAULT '',
			file_size INTEGER DEFAULT 0,
			sha256 TEXT DEFAULT '',
			deploy_url TEXT DEFAULT '',
			asset_kind TEXT DEFAULT 'portable_zip',
			status TEXT NOT NULL DEFAULT 'published' CHECK(status IN ('published', 'draft', 'yanked')),
			published_at INTEGER NOT NULL,
			deleted_at INTEGER DEFAULT 0
		);
		CREATE INDEX IF NOT EXISTS idx_releases_lookup ON releases(project_id, target_id, published_at DESC);
		CREATE INDEX IF NOT EXISTS idx_releases_channel ON releases(project_id, channel, status);
	`); err != nil {
		return fmt.Errorf("create v3 tables: %w", err)
	}

	localCols, err := tableColumnsTx(tx, "local_state")
	if err != nil {
		return err
	}
	for _, col := range []struct {
		name string
		ddl  string
	}{
		{"installed_version", `ALTER TABLE local_state ADD COLUMN installed_version TEXT DEFAULT ''`},
		{"installed_release_id", `ALTER TABLE local_state ADD COLUMN installed_release_id TEXT DEFAULT ''`},
		{"installed_at", `ALTER TABLE local_state ADD COLUMN installed_at INTEGER DEFAULT 0`},
		{"git_branch", `ALTER TABLE local_state ADD COLUMN git_branch TEXT DEFAULT ''`},
		{"git_commit", `ALTER TABLE local_state ADD COLUMN git_commit TEXT DEFAULT ''`},
		{"git_dirty", `ALTER TABLE local_state ADD COLUMN git_dirty INTEGER DEFAULT 0`},
	} {
		if !localCols[col.name] {
			if _, err := tx.Exec(col.ddl); err != nil {
				return err
			}
		}
	}

	// Backfill targets and release history from v1 project columns.
	if _, err := tx.Exec(`
		INSERT INTO project_targets (
			id, project_id, kind, name, platform, distribution, launch_entry, updated_at, deleted_at
		)
		SELECT
			lower(hex(randomblob(16))),
			id,
			CASE WHEN type = 'desktop' THEN 'desktop' ELSE 'web' END,
			CASE WHEN type = 'desktop' THEN 'Desktop App' ELSE 'Web App' END,
			CASE WHEN type = 'desktop' THEN 'windows' ELSE 'web' END,
			CASE WHEN type = 'desktop' THEN 'portable' ELSE 'hosted' END,
			CASE WHEN type = 'desktop' THEN exe_name ELSE bat_file END,
			updated_at,
			COALESCE(deleted_at, 0)
		FROM projects
		WHERE COALESCE(deleted_at, 0) = 0
		  AND NOT EXISTS (SELECT 1 FROM project_targets pt WHERE pt.project_id = projects.id);

		INSERT INTO releases (
			id, project_id, target_id, version, channel, changelog, r2_key, file_name, asset_kind, status, published_at, deleted_at
		)
		SELECT
			lower(hex(randomblob(16))),
			p.id,
			pt.id,
			p.latest_version,
			'stable',
			p.release_notes,
			p.r2_release_key,
			p.exe_name,
			'portable_zip',
			'published',
			p.updated_at,
			0
		FROM projects p
		JOIN project_targets pt ON pt.project_id = p.id AND pt.kind = p.type
		WHERE p.latest_version != '' AND p.r2_release_key != ''
		  AND NOT EXISTS (SELECT 1 FROM releases r WHERE r.project_id = p.id AND r.version = p.latest_version);
	`); err != nil {
		return fmt.Errorf("backfill v3 targets/releases: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return err
	}
	_, err = db.Exec(`PRAGMA user_version = 3`)
	return err
}

var allowedSchemaTables = map[string]bool{
	"projects":        true,
	"categories":      true,
	"local_state":     true,
	"project_targets": true,
	"releases":        true,
}

func tableColumnsTx(tx *sql.Tx, tableName string) (map[string]bool, error) {
	if !allowedSchemaTables[tableName] {
		return nil, fmt.Errorf("invalid schema table name %q", tableName)
	}
	rows, err := tx.Query(`SELECT name FROM pragma_table_info(?)`, tableName)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	cols := make(map[string]bool)
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, err
		}
		cols[name] = true
	}
	return cols, rows.Err()
}

func copyFileBackup(src, dst string) error {
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

func tableColumns(db *sql.DB, tableName string) (map[string]bool, error) {
	if !allowedSchemaTables[tableName] {
		return nil, fmt.Errorf("invalid schema table name %q", tableName)
	}
	rows, err := db.Query(`SELECT name FROM pragma_table_info(?)`, tableName)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	cols := make(map[string]bool)
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, err
		}
		cols[name] = true
	}
	return cols, rows.Err()
}
