# Trung tâm Tài liệu Dự án T-Deck (Documentation Hub)

**T-Deck** là ứng dụng desktop điều phối danh mục phần mềm, khởi chạy không gian làm việc và quản lý phát hành đa nền tảng (**Web**, **Desktop App** và **Hybrid**) xây dựng trên **Wails v2 (Go 1.26 + WebView2)**, **React 19 / TypeScript / Tailwind CSS**, **SQLite (`modernc.org/sqlite` + FTS5)** và **Cloudflare R2 (S3 API)**.

Tài liệu trong thư mục này cùng các file `.md` ở thư mục gốc cung cấp bức tranh toàn diện từ hướng dẫn sử dụng cho người dùng cuối, tham chiếu cấu hình, hướng dẫn dòng lệnh CLI, đến kiến trúc kỹ thuật và đặc tả chi tiết của từng phân hệ.

---

## 1. Tài liệu Người dùng & Vận hành (Open-Source Guides)

| Tài liệu | Mục đích & Nội dung chính | Đối tượng đọc |
|---|---|---|
| **[../README.md](file:///d:/app/t-deck/README.md)** | Tổng quan dự án, điểm nổi bật, sơ đồ kiến trúc, hướng dẫn cài đặt nhanh (Quick Start) và bản đồ thư mục | Mọi người dùng & Developer |
| **[USER_GUIDE.md](file:///d:/app/t-deck/docs/USER_GUIDE.md)** | Hướng dẫn sử dụng chi tiết từ A–Z: cấu hình Workspace & R2, bấm `Scan Local`, nhận diện Brand Icon, lọc FTS5, chạy `.bat`/`.exe`, xử lý `.env`, phát hành R2 và hỏi đáp sự cố | Người dùng cuối, DevOps, QA |
| **[FEATURES.md](file:///d:/app/t-deck/docs/FEATURES.md)** | Phân tích chuyên sâu cơ chế hoạt động của 5 phân hệ cốt lõi: Bộ quét thông minh, Động cơ Brand Icon & Poster 16:9, SQLite FTS5, Trình khởi chạy & Giám sát PID/Port, và Phát hành R2 | Product Owner, Tech Lead, Developer |
| **[CONFIGURATION.md](file:///d:/app/t-deck/docs/CONFIGURATION.md)** | Tham chiếu đầy đủ về thư mục dữ liệu `<InstallDir>`, biến môi trường `.env`, file neo `.tdeck-meta.json`, chuẩn manifest `.tdeck.json` và cấu trúc bucket Cloudflare R2 | Developer, System Admin |
| **[CLI.md](file:///d:/app/t-deck/docs/CLI.md)** | Hướng dẫn sử dụng công cụ dòng lệnh `cmd/tdeck-sync` (tất cả flags & kịch bản mẫu) và script đóng gói `build.bat` | Developer, CI/CD Engineer |
| **[../CONTRIBUTING.md](file:///d:/app/t-deck/CONTRIBUTING.md)** | Quy chuẩn kiến trúc Go & React/TypeScript, quy trình kiểm thử (`go test`, `-race`, `npm run build`) và quy tắc đóng góp | Contributor, Maintainer |
| **[../SECURITY.md](file:///d:/app/t-deck/SECURITY.md)** | Kiến trúc bảo mật: tách biệt khóa R2 vào `.env`, làm sạch `.env.example`, chống Zip-Slip, kiểm tra `SHA-256` & Atomic Rollback | Security Reviewer, Maintainer |
| **[../CHANGELOG.md](file:///d:/app/t-deck/CHANGELOG.md)** | Lịch sử phát hành và chi tiết các tính năng/sửa lỗi qua từng phiên bản (`v1.0.0` → `v2.1.0`) | Mọi người dùng & Developer |

---

## 2. Tài liệu Kiến trúc & Đặc tả Kỹ thuật (Engineering Specs)

| Tài liệu | Mục đích | Đối tượng đọc |
|---|---|---|
| **[ARCHITECTURE.md](file:///d:/app/t-deck/docs/ARCHITECTURE.md)** | Kiến trúc tổng thể, bản đồ module Go & React, sơ đồ luồng dữ liệu và các quyết định kiến trúc cốt lõi (ADRs) | Tech Lead, Backend/Frontend Dev |
| **[PLAN.md](file:///d:/app/t-deck/docs/PLAN.md)** | Kế hoạch thực thi tổng thể (Master Plan), bảng 22 tickets (`P1-01`..`P3-07`), đồ thị phụ thuộc và cổng kiểm chứng | Người điều phối, Reviewer, Dev |
| **[specs/01-performance-and-sync.md](file:///d:/app/t-deck/docs/specs/01-performance-and-sync.md)** | Đặc tả Phase 1: Loại bỏ N+1 disk I/O, tối ưu `Scanner`, SQLite/HTTP cache, nén ảnh clipboard, sync `categories` và soft-delete tombstones | Backend Go & React Dev |
| **[specs/02-hybrid-release-manager.md](file:///d:/app/t-deck/docs/specs/02-hybrid-release-manager.md)** | Đặc tả Phase 2: Mô hình Hybrid (`project_targets` + `releases`), Web Release Center, Health/Version check, Desktop Update/Rollback nguyên tử & Presigned URLs | Fullstack Wails Dev |
| **[specs/03-automation-and-dashboard.md](file:///d:/app/t-deck/docs/specs/03-automation-and-dashboard.md)** | Đặc tả Phase 3: Manifest `.tdeck.json`, Multipart R2 upload + progress events, Build & Auto-Changelog drawer, `latest.json` Auto-Updater, Process/Port Monitor & Release Matrix View | Fullstack Wails Dev |
| **[specs/04-data-and-api-contracts.md](file:///d:/app/t-deck/docs/specs/04-data-and-api-contracts.md)** | Hợp đồng chuẩn về SQLite Schema & Migrations (`v1`→`v3`), cấu trúc object trên R2, schema `.tdeck.json`, danh sách Wails IPC methods & Runtime Events | Toàn bộ thành viên thực thi |

---

## 3. Trạng thái Thực thi các Giai đoạn (Phases Status)

| Giai đoạn | Tên giai đoạn | Trạng thái | Kết quả đạt được |
|---|---|---|---|
| **Baseline v1** | T-Deck Local Hub & R2 Sync | ✅ Hoàn thành | Scan workspace theo `.tdeck-meta.json`, FTS5 search, Launch `.bat`/`.exe`, kiểm tra `.env`, Category groups, Sync `catalog.json` & upload `.zip` desktop lên R2 |
| **Phase 1** | Hiệu năng & Toàn vẹn Đồng bộ | ✅ Hoàn thành | Xóa N+1 `os.Stat` và double IPC fetch, tối ưu `Scanner` về $O(N)$, bật HTTP cache thumbnail, resize ảnh 1280×720, thêm SQLite indexes, sync `categories` & tombstones `deleted_at` |
| **Phase 2** | Hybrid Web & Desktop Release Manager | ✅ Hoàn thành | Hỗ trợ dự án có đồng thời bản Web và Desktop (`project_targets`, `releases`), Web URLs + Health/Version Drift check, Static Bundle Archive, Desktop Release History, SHA-256 + Atomic Update/Rollback, Copy Public/Presigned Link |
| **Phase 3** | Tự động hóa & Release Matrix Dashboard | ✅ Hoàn thành | Chuẩn hóa `.tdeck.json`, Multipart S3 Upload + Live Progress/Build Log Drawer, tự sinh `latest.json` cho Auto-Updater, Process/Port Monitor, chế độ bảng **Release Matrix View** |
| **Phase 4 (v2.1)** | Nhận diện Thương hiệu Tự động & Mô-đun hóa UI | ✅ Hoàn thành | Tự động phát hiện bộ icon thương hiệu (`.png`/`.ico`) gắn vào poster 16:9 và huy hiệu App Icon trên UI khi bấm `Scan Local`; lọc triệt để phần mềm bên thứ 3; hiệu ứng Scan Skeleton UX; chia tách gọn `App.tsx` và các component |

---

## 4. Lệnh Thao tác Nhanh (Quick Commands)

```powershell
# Chạy chế độ phát triển (hot reload Vite + Wails Go bridge)
wails dev

# Chạy toàn bộ unit & integration tests Go
go test -v -count=1 ./...

# Chạy race detector cho backend Go
go test -race -count=1 ./...

# Kiểm tra TypeScript & build frontend
cd frontend; npm run build

# Đóng gói nhanh TDeck.exe (Frontend + Go backend)
.\build.bat full

# Chạy CLI quét thử nghiệm và kết xuất poster tại máy
go run ./cmd/tdeck-sync --dry-run
```
