# Đặc tả Hợp đồng Dữ liệu, Storage & API (`04-data-and-api-contracts.md`)

Tài liệu này là **Single Source of Truth** cho cấu trúc bảng SQLite, lịch sử migration (`PRAGMA user_version`), cấu trúc object trên Cloudflare R2, schema file `.tdeck.json` và hợp đồng giao tiếp Wails IPC / Events giữa Go và TypeScript cho toàn bộ Phases 1–3.

---

## 1. SQLite Schema & Kế hoạch Migration (`tdeck.db`)

### 1.1. Pragmas chuẩn kết nối (`internal/database/sqlite.go`)
Áp dụng ngay khi mở DB ở `database.Open()`:
```sql
PRAGMA journal_mode = WAL;
PRAGMA foreign_keys = ON;
PRAGMA busy_timeout = 5000;
PRAGMA synchronous = NORMAL;    -- Phase 1: giảm fsync khi ghi WAL
PRAGMA temp_store = MEMORY;     -- Phase 1: tăng tốc FTS5 sort & ORDER BY
PRAGMA cache_size = -16000;     -- Phase 1: ~16 MB page cache trong RAM
```

### 1.2. Các phiên bản Schema (`PRAGMA user_version`)

#### Version 1 (Hiện tại — Baseline)
- `projects(id, slug, name, type, category, summary, tags, tech_stack, git_remote, bat_file, exe_name, latest_version, r2_release_key, release_notes, thumbnail_updated_at, updated_at)`
- `categories(id, name, slug, sort_order, created_at)`
- `local_state(project_id, local_dir, status, last_opened_at)`
- `projects_fts` (`USING fts5(project_id UNINDEXED, name, summary, tags, tech_stack)`)

#### Version 2 (Phase 1 — Hiệu năng, Sync Categories & Tombstone)
Thực hiện tuần tự trong transaction khi `user_version < 2`:
```sql
-- 1. Soft-delete tombstones để đồng bộ xóa giữa nhiều máy
ALTER TABLE projects ADD COLUMN deleted_at INTEGER DEFAULT 0;
ALTER TABLE categories ADD COLUMN deleted_at INTEGER DEFAULT 0;
ALTER TABLE categories ADD COLUMN updated_at INTEGER DEFAULT 0;
UPDATE categories SET updated_at = created_at WHERE updated_at = 0;

-- 2. Cache cờ hệ thống file vào local_state (loại bỏ N+1 os.Stat lúc ListProjects)
ALTER TABLE local_state ADD COLUMN has_env INTEGER DEFAULT 0;
ALTER TABLE local_state ADD COLUMN has_env_example INTEGER DEFAULT 0;
ALTER TABLE local_state ADD COLUMN has_bat INTEGER DEFAULT 0;
ALTER TABLE local_state ADD COLUMN flags_checked_at INTEGER DEFAULT 0;

-- 3. Indexes cho bộ lọc & sắp xếp
CREATE INDEX IF NOT EXISTS idx_projects_type ON projects(type);
CREATE INDEX IF NOT EXISTS idx_projects_category ON projects(category);
CREATE INDEX IF NOT EXISTS idx_projects_updated_at ON projects(updated_at DESC);
CREATE INDEX IF NOT EXISTS idx_projects_deleted_at ON projects(deleted_at);
CREATE INDEX IF NOT EXISTS idx_local_state_status ON local_state(status);

PRAGMA user_version = 2;
```

#### Version 3 (Phase 2 — Multi-Target Hybrid Web/Desktop & Release History)
> [!IMPORTANT]
> Trước khi chạy migration lên `user_version = 3`, hàm `database.Open()` **bắt buộc** copy `tdeck.db` sang `tdeck.db.v2.bak` (nếu tồn tại) trước khi mở transaction.

```sql
-- 1. Mở rộng CHECK constraint của projects.type sang ('web', 'desktop', 'hybrid')
-- Do SQLite không hỗ trợ ALTER CHECK trực tiếp, dùng table rebuild an toàn trong transaction:
PRAGMA foreign_keys = OFF;
BEGIN TRANSACTION;

CREATE TABLE projects_new (
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

INSERT INTO projects_new SELECT
    id, slug, name, type, category, summary, tags, tech_stack,
    git_remote, bat_file, exe_name, latest_version, r2_release_key,
    release_notes, thumbnail_updated_at, updated_at, deleted_at
FROM projects;

DROP TABLE projects;
ALTER TABLE projects_new RENAME TO projects;

CREATE INDEX IF NOT EXISTS idx_projects_type ON projects(type);
CREATE INDEX IF NOT EXISTS idx_projects_category ON projects(category);
CREATE INDEX IF NOT EXISTS idx_projects_updated_at ON projects(updated_at DESC);
CREATE INDEX IF NOT EXISTS idx_projects_deleted_at ON projects(deleted_at);

-- 2. Bảng project_targets: 1 project có thể có 1..N target (Web + Desktop Windows/macOS/Linux)
CREATE TABLE IF NOT EXISTS project_targets (
    id TEXT PRIMARY KEY,
    project_id TEXT NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    kind TEXT NOT NULL CHECK(kind IN ('web', 'desktop')),
    name TEXT NOT NULL DEFAULT '',               -- VD: "Web Portal", "Desktop Client"
    platform TEXT NOT NULL DEFAULT 'windows',    -- web | windows | macos | linux
    distribution TEXT NOT NULL DEFAULT 'portable', -- hosted | static_zip | portable | installer
    subpath TEXT DEFAULT '',                     -- thư mục con trong monorepo (nếu có)
    build_cmd TEXT DEFAULT '',                   -- VD: "wails build -clean" hoặc "npm run build"
    artifact_glob TEXT DEFAULT '',               -- VD: "build/bin/*.exe" hoặc "dist"
    launch_entry TEXT DEFAULT '',                -- start.bat hoặc đường dẫn tương đối .exe
    prod_url TEXT DEFAULT '',
    staging_url TEXT DEFAULT '',
    health_url TEXT DEFAULT '',
    version_url TEXT DEFAULT '',                 -- endpoint trả JSON {"version": "..."}
    hosting_provider TEXT DEFAULT '',            -- Cloudflare Pages | Vercel | VPS | R2 Static
    sort_order INTEGER DEFAULT 0,
    updated_at INTEGER NOT NULL,
    deleted_at INTEGER DEFAULT 0
);
CREATE INDEX IF NOT EXISTS idx_targets_project ON project_targets(project_id, sort_order);

-- 3. Bảng releases: lịch sử phát hành đầy đủ cho cả Web & Desktop
CREATE TABLE IF NOT EXISTS releases (
    id TEXT PRIMARY KEY,
    project_id TEXT NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    target_id TEXT NOT NULL REFERENCES project_targets(id) ON DELETE CASCADE,
    version TEXT NOT NULL,                       -- SemVer: v2.5.0
    channel TEXT NOT NULL DEFAULT 'stable',      -- stable | beta | canary
    git_tag TEXT DEFAULT '',
    commit_sha TEXT DEFAULT '',
    changelog TEXT DEFAULT '',
    r2_key TEXT DEFAULT '',                      -- releases/<slug>/<kind>/<version>/<file>
    file_name TEXT DEFAULT '',
    file_size INTEGER DEFAULT 0,
    sha256 TEXT DEFAULT '',
    deploy_url TEXT DEFAULT '',                  -- link web deployment hoặc R2 static preview
    asset_kind TEXT DEFAULT 'portable_zip',      -- portable_zip | installer_exe | static_zip | external_url
    status TEXT NOT NULL DEFAULT 'published' CHECK(status IN ('published', 'draft', 'yanked')),
    published_at INTEGER NOT NULL,
    deleted_at INTEGER DEFAULT 0
);
CREATE INDEX IF NOT EXISTS idx_releases_lookup ON releases(project_id, target_id, published_at DESC);
CREATE INDEX IF NOT EXISTS idx_releases_channel ON releases(project_id, channel, status);

-- 4. Mở rộng local_state để phát hiện lệch phiên bản & trạng thái Git cục bộ
ALTER TABLE local_state ADD COLUMN installed_version TEXT DEFAULT '';
ALTER TABLE local_state ADD COLUMN installed_release_id TEXT DEFAULT '';
ALTER TABLE local_state ADD COLUMN installed_at INTEGER DEFAULT 0;
ALTER TABLE local_state ADD COLUMN git_branch TEXT DEFAULT '';
ALTER TABLE local_state ADD COLUMN git_commit TEXT DEFAULT '';
ALTER TABLE local_state ADD COLUMN git_dirty INTEGER DEFAULT 0;

-- 5. Backfill dữ liệu từ v1 sang project_targets & releases (không mất dữ liệu cũ)
INSERT INTO project_targets (
    id, project_id, kind, name, platform, distribution, launch_entry, updated_at
)
SELECT
    lower(hex(randomblob(16))),
    id,
    type,
    CASE WHEN type = 'desktop' THEN 'Desktop App' ELSE 'Web App' END,
    CASE WHEN type = 'desktop' THEN 'windows' ELSE 'web' END,
    CASE WHEN type = 'desktop' THEN 'portable' ELSE 'hosted' END,
    CASE WHEN type = 'desktop' THEN exe_name ELSE bat_file END,
    updated_at
FROM projects
WHERE deleted_at = 0
  AND NOT EXISTS (SELECT 1 FROM project_targets pt WHERE pt.project_id = projects.id);

INSERT INTO releases (
    id, project_id, target_id, version, channel, changelog, r2_key, file_name, asset_kind, status, published_at
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
    p.updated_at
FROM projects p
JOIN project_targets pt ON pt.project_id = p.id AND pt.kind = p.type
WHERE p.latest_version != '' AND p.r2_release_key != '';

COMMIT;
PRAGMA foreign_keys = ON;
PRAGMA user_version = 3;
```

> [!NOTE]
> Các cột v1 (`bat_file`, `exe_name`, `latest_version`, `r2_release_key`, `release_notes`) vẫn được giữ đồng bộ với target chính (primary target) và release mới nhất để bảo đảm tương thích ngược với các client T-Deck cũ hơn chưa nâng cấp.

---

## 2. Cấu trúc Object trên Cloudflare R2 Bucket

```text
<bucket>/
├── catalog.json                                 # Danh mục gọn nhẹ (Schema v2)
├── thumbnails/
│   └── <project_id>.png                         # Ảnh preview 1280×720
└── releases/
    └── <slug>/
        ├── index.json                           # Toàn bộ lịch sử releases của project <slug>
        ├── stable/
        │   └── latest.json                      # Manifest cho Auto-Updater
        ├── beta/
        │   └── latest.json                      # Manifest cho kênh Beta (nếu có)
        ├── desktop/
        │   └── <version>/
        │       ├── <slug>-<version>-portable.zip
        │       └── <slug>-Setup-<version>.exe
        └── web/
            └── <version>/
                └── <slug>-web-<version>-dist.zip
```

### 2.1. `catalog.json` (Schema v2 — Backward Compatible với v1)
- Khi đọc `catalog.json`: nếu `version == 1` hoặc `version == 0`, parser tự hiểu là dữ liệu v1 (không có `targets`, `categories`, `tombstones`).
- Khi ghi `catalog.json`: luôn xuất `version: 2`:
```json
{
  "version": 2,
  "updated_at": 1795314000,
  "categories": [
    {
      "id": "cat-uuid",
      "name": "T-Hub Ecosystem",
      "slug": "t-hub-ecosystem",
      "sort_order": 1,
      "created_at": 1795000000,
      "updated_at": 1795310000,
      "deleted_at": 0
    }
  ],
  "projects": [
    {
      "id": "proj-uuid",
      "slug": "t-hub",
      "name": "T-Hub Cockpit",
      "type": "hybrid",
      "category": "T-Hub Ecosystem",
      "summary": "...",
      "tags": ["go", "react", "desktop"],
      "tech_stack": "Go, Wails, React",
      "git_remote": "https://github.com/...",
      "bat_file": "start.bat",
      "exe_name": "build/bin/T-Hub.exe",
      "latest_version": "v0.3.4",
      "r2_release_key": "releases/t-hub/desktop/v0.3.4/win-unpacked.zip",
      "release_notes": "...",
      "thumbnail_updated_at": 1795310000,
      "updated_at": 1795314000,
      "deleted_at": 0
    }
  ],
  "targets": [
    {
      "id": "tgt-web-uuid",
      "project_id": "proj-uuid",
      "kind": "web",
      "name": "Web Dashboard",
      "platform": "web",
      "distribution": "hosted",
      "prod_url": "https://hub.example.com",
      "health_url": "https://hub.example.com/health",
      "version_url": "https://hub.example.com/version.json",
      "hosting_provider": "Cloudflare Pages",
      "updated_at": 1795314000,
      "deleted_at": 0
    }
  ]
}
```
- **Quy tắc Tombstone Retention**: bản ghi có `deleted_at > 0` được giữ trong `catalog.json` tối đa **90 ngày** để mọi máy phụ kịp nhận tín hiệu xóa trước khi dọn sạch vĩnh viễn.

### 2.2. `releases/<slug>/<channel>/latest.json` (Auto-Updater Manifest)
```json
{
  "slug": "t-hub",
  "version": "v0.3.4",
  "channel": "stable",
  "pub_date": "2026-11-22T03:30:00Z",
  "notes": "- Fix plugin sync\n- Optimize SQLite WAL",
  "platforms": {
    "windows-x86_64": {
      "url": "https://cdn.example.com/releases/t-hub/desktop/v0.3.4/win-unpacked.zip",
      "installer_url": "https://cdn.example.com/releases/t-hub/desktop/v0.3.4/T-Hub-Setup-0.3.4.exe",
      "sha256": "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
      "size": 45120480
    }
  }
}
```

---

## 3. Đặc tả File Manifest `.tdeck.json` tại Từng Repo (Phase 3)

Đặt tại thư mục gốc dự án (bên cạnh `.tdeck-meta.json`). Cho phép khai báo cấu trúc đa nền tảng thay thế map `desktopReleases` hardcode trong `cmd/tdeck-sync/main.go`.

```json
{
  "schema_version": 1,
  "slug": "t-hub",
  "name": "T-Hub Cockpit",
  "category": "T-Hub Ecosystem",
  "tags": ["go", "wails", "react", "cockpit"],
  "version_source": {
    "type": "auto",
    "file": "package.json",
    "json_path": "version"
  },
  "targets": [
    {
      "kind": "web",
      "name": "Web Portal",
      "subpath": "web",
      "distribution": "static_zip",
      "launch_entry": "start.bat",
      "build_cmd": "npm run build",
      "artifact_glob": "web/dist",
      "prod_url": "https://hub.example.com",
      "staging_url": "https://staging-hub.example.com",
      "health_url": "https://hub.example.com/health",
      "version_url": "https://hub.example.com/version.json",
      "hosting_provider": "Cloudflare Pages"
    },
    {
      "kind": "desktop",
      "name": "Desktop Client",
      "platform": "windows",
      "distribution": "portable",
      "subpath": "desktop-app",
      "build_cmd": "wails build -clean -platform windows/amd64",
      "artifact_glob": "build/bin/T-Hub.exe",
      "launch_entry": "build/bin/T-Hub.exe",
      "installer_glob": "release/*/T-Hub-Setup-*.exe"
    }
  ]
}
```

### Thứ tự ưu tiên đọc version (`version_source.type = "auto"`):
1. `wails.json` → `info.productVersion` (nếu có)
2. `package.json` → `version`
3. `git describe --tags --abbrev=0`
4. Fallback `v0.1.0`

---

## 4. Hợp đồng Wails Go ↔ TypeScript IPC & Runtime Events

### 4.1. Danh sách Methods mới trên `App` (`app.go` & `frontend/src/lib/api.ts`)

| Phase | Method Signature (Go) | Mục đích |
|---|---|---|
| **P1** | `ListProjects(filter model.ProjectFilter) ([]model.ProjectView, error)` | Giữ nguyên signature; tối ưu bỏ `os.Stat` khi đọc DB và hỗ trợ filter `deleted_at = 0`. |
| **P2** | `UpsertTarget(target model.ProjectTarget) (model.ProjectTarget, error)` | Tạo/cập nhật target Web hoặc Desktop của 1 project; tự đồng bộ `projects.type = "hybrid"` nếu project có cả 2 loại. |
| **P2** | `DeleteTarget(targetID string) error` | Soft-delete target (`deleted_at = now`). |
| **P2** | `ListProjectTargets(projectID string) ([]model.ProjectTarget, error)` | Lấy danh sách targets của project. |
| **P2** | `ListReleases(projectID string, targetID string) ([]model.Release, error)` | Trả lịch sử releases (mới nhất trước). |
| **P2** | `PublishRelease(req model.PublishReleaseRequest) (model.Release, error)` | Nén + tính SHA-256 + upload lên R2 (Web `dist.zip` hoặc Desktop `.zip`/`.exe`) + ghi bảng `releases` + sync `releases/<slug>/index.json`. |
| **P2** | `DeleteRelease(releaseID string, deleteR2Object bool) error` | Đánh dấu `deleted_at` / `yanked` và tùy chọn xóa file trên R2. |
| **P2** | `UpdateDesktopRelease(projectID string, targetID string, releaseID string) (model.ProjectView, error)` | Tải bản chỉ định (hoặc latest), verify SHA-256, backup `.bak`, swap nguyên tử, cập nhật `installed_version`. |
| **P2** | `RollbackDesktopRelease(projectID string, targetID string) (model.ProjectView, error)` | Phục hồi ngay bản `.bak` gần nhất hoặc bản release trước đó. |
| **P2** | `GenerateReleaseDownloadURL(releaseID string, expiryHours int) (string, error)` | Trả `PublicDomain` URL hoặc S3 Presigned GetObject URL. |
| **P2** | `CheckWebHealth(targetID string) (model.WebHealthResult, error)` | Kiểm tra HTTP status, latency ms và đọc `version_url` (so sánh version drift). |
| **P2** | `CheckAllWebHealth() ([]model.WebHealthResult, error)` | Ping song song toàn bộ Web targets đang hoạt động. |
| **P2** | `OpenTargetURL(targetID string, env string) error` | Mở `prod_url` / `staging_url` qua `wailsruntime.BrowserOpenURL`. |
| **P3** | `SuggestReleaseChangelog(projectID string, targetID string) (string, error)` | Chạy `git log <last_tag>..HEAD` tạo draft Release Notes. |
| **P3** | `BuildAndPublishTarget(req model.BuildPublishRequest) (string, error)` | Chạy `build_cmd` bất đồng bộ, trả `jobID` và stream logs qua Wails events. |
| **P3** | `CancelActiveJob(jobID string) error` | Hủy tiến trình build/upload đang chạy qua `context.CancelFunc`. |
| **P3** | `ListRunningProcesses() ([]model.RunningProcess, error)` | Danh sách PID, project_id, cổng TCP đang lắng nghe cục bộ. |
| **P3** | `StopProjectProcess(projectID string) error` | Dừng process tree đang chạy (Windows `taskkill /T /F /PID`). |
| **P3** | `ImportProjectManifest(projectID string) (model.ProjectView, error)` | Đọc `.tdeck.json` từ `local_dir` và nạp/cập nhật targets tương ứng. |

### 4.2. Wails Runtime Events (`wailsruntime.EventsEmit`)

| Event Name | Payload TypeScript | Tần suất / Điều kiện phát |
|---|---|---|
| `release:progress` | `{ job_id: string; project_id: string; stage: "zipping" \| "uploading" \| "downloading" \| "extracting"; bytes_done: number; bytes_total: number; percent: number; speed_bps: number }` | Tối đa mỗi `150ms` hoặc khi đổi `stage` |
| `build:log` | `{ job_id: string; project_id: string; stream: "stdout" \| "stderr"; line: string; ts: number }` | Theo dòng output (gộp batch mỗi `80ms` để tránh nghẽn IPC) |
| `build:status` | `{ job_id: string; project_id: string; status: "running" \| "success" \| "failed" \| "cancelled"; error?: string }` | Khi chuyển trạng thái job |
| `health:result` | `{ target_id: string; project_id: string; online: boolean; status_code: number; latency_ms: number; live_version: string; drift: boolean; checked_at: number; error?: string }` | Sau mỗi lần `CheckWebHealth` / `CheckAllWebHealth` |
| `process:changed` | `{ project_id: string; running: boolean; pid: number; ports: number[] }` | Khi launch, thoát process hoặc phát hiện port mới |
