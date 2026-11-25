# Tài liệu Dự án & Kế hoạch Nâng cấp T-Deck

T-Deck là ứng dụng desktop đa nền tảng xây dựng trên **Wails v2 (Go 1.26 + WebView2)** và **React 19 / TypeScript / Tailwind CSS**, dùng **SQLite (`modernc.org/sqlite` + FTS5)** làm kho lưu trữ cục bộ (local-first) và **Cloudflare R2 (S3 API)** để đồng bộ danh mục dự án, ảnh preview và bản phát hành (release artifacts) giữa nhiều máy tính mà không cần vận hành server backend riêng.

Tài liệu này mô tả đầy đủ kiến trúc hiện tại, phân tích nút thắt hiệu năng, các hợp đồng dữ liệu/API và bản đặc tả chi tiết cho 3 giai đoạn (Phases 1–3) nhằm nâng cấp T-Deck thành **Release & Workspace Control Plane** cho cả ứng dụng **Web**, **Desktop App** và dự án **Hybrid (có cả Web lẫn Desktop)**.

---

## Danh mục tài liệu

| Tài liệu | Mục đích | Đối tượng đọc |
|---|---|---|
| [ARCHITECTURE.md](ARCHITECTURE.md) | Kiến trúc hiện tại, sơ đồ module Go & React, luồng dữ liệu cốt lõi, lỗ hổng nền tảng, kiến trúc mục tiêu V2 và ADRs | Tech Lead, Backend/Frontend Dev |
| [PLAN.md](PLAN.md) | Kế hoạch thực thi tổng thể (Master Plan), bảng 22 tickets (`P1-01`..`P3-07`), đồ thị phụ thuộc, verification gates và rollback plan | Người điều phối, Reviewer, Dev |
| [specs/01-performance-and-sync.md](specs/01-performance-and-sync.md) | Đặc tả Phase 1: Loại bỏ N+1 disk I/O, tối ưu `Scanner`, SQLite/HTTP cache, nén ảnh clipboard, sync `categories` và soft-delete tombstones | Backend Go & React Dev |
| [specs/02-hybrid-release-manager.md](specs/02-hybrid-release-manager.md) | Đặc tả Phase 2: Mô hình Hybrid (`project_targets` + `releases`), Web Release Center, Health/Version check, Desktop Update/Rollback nguyên tử & Presigned URLs | Fullstack Wails Dev |
| [specs/03-automation-and-dashboard.md](specs/03-automation-and-dashboard.md) | Đặc tả Phase 3: Manifest `.tdeck.json`, Multipart R2 upload + progress events, Build & Auto-Changelog drawer, `latest.json` Auto-Updater, Process/Port Monitor & Release Matrix View | Fullstack Wails Dev |
| [specs/04-data-and-api-contracts.md](specs/04-data-and-api-contracts.md) | Hợp đồng chuẩn về SQLite Schema & Migrations (`v1`→`v3`), cấu trúc object trên R2, schema `.tdeck.json`, danh sách Wails IPC methods & Runtime Events | Toàn bộ thành viên thực thi |

---

## Lộ trình các Giai đoạn (Phases)

| Giai đoạn | Tên giai đoạn | Trạng thái | Mục tiêu trọng tâm |
|---|---|---|---|
| **Baseline v1** | T-Deck Local Hub & R2 Sync | Hoàn thành | Scan workspace theo `.tdeck-meta.json`, FTS5 search, Launch `.bat`/`.exe`, kiểm tra `.env`, Category groups, Sync `catalog.json` & upload 1 file `.zip` desktop lên R2 |
| **Phase 1** | Hiệu năng & Toàn vẹn Đồng bộ | Đã đặc tả (`P1-01`..`P1-07`) | Xóa N+1 `os.Stat` và double IPC fetch, tối ưu `Scanner` từ $O(N^2)$ về $O(N)$, bật HTTP cache thumbnail, resize ảnh 1280×720, thêm SQLite indexes, sync `categories` & tombstones `deleted_at` |
| **Phase 2** | Hybrid Web & Desktop Release Manager | Đã đặc tả (`P2-01`..`P2-08`) | Hỗ trợ dự án có đồng thời bản Web và Desktop (`project_targets`, `releases`), Web URLs + Health/Version Drift check, Static Bundle Archive, Desktop Release History, SHA-256 + Atomic Update/Rollback, Copy Public/Presigned Link |
| **Phase 3** | Tự động hóa & Release Matrix Dashboard | Đã đặc tả (`P3-01`..`P3-07`) | Bỏ hardcode `tdeck-sync` bằng `.tdeck.json`, Multipart S3 Upload + Live Progress/Build Log Drawer, tự sinh `latest.json` cho Auto-Updater, Process/Port Monitor, chế độ bảng **Release Matrix View** |

---

## Lệnh thao tác nhanh (Quick Commands)

```powershell
# Chạy chế độ dev (hot reload Vite + Wails Go bridge)
wails dev

# Chạy toàn bộ unit tests Go
go test -v -count=1 ./...

# Chạy race detector cho backend Go (bắt buộc ở Phase 2-3 khi có worker pool & process monitor)
go test -race -count=1 ./...

# Kiểm tra TypeScript & build frontend
cd frontend; npm run build

# Build file thực thi Windows production
wails build -clean -platform windows/amd64

# Chạy CLI đồng bộ & publish hàng loạt (hiện tại)
go run ./cmd/tdeck-sync --dry-run
```
