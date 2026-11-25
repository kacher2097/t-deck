# Đặc tả Phase 3: Tự động hóa Pipeline, Auto-Updater & Matrix Dashboard (`03-automation-and-dashboard.md`)

**Mục tiêu Phase 3:** Xóa bỏ toàn bộ cấu hình release hardcode trong `cmd/tdeck-sync`, chuyển việc build–changelog–nén–upload thành pipeline 1-click có thanh tiến độ thực (multipart S3), sinh manifest `latest.json` cho cơ chế auto-update của các app con, theo dõi tiến trình/cổng đang chạy tại máy và cung cấp **Release Matrix Dashboard** cho phép kiểm soát hàng loạt dự án trong 1 màn hình.

---

## 1. Chi tiết Kỹ thuật Từng Ticket

### `P3-01` — Engine Đọc Manifest `.tdeck.json` & Tái cấu trúc `cmd/tdeck-sync`
- **Files mới / sửa**: `internal/scanner/manifest.go` (mới), `internal/scanner/scanner.go`, `cmd/tdeck-sync/main.go`, `app.go`.
- **Chi tiết thực hiện**:
  1. Xây dựng `scanner.LoadProjectManifest(dir string)` đọc và validate `.tdeck.json` theo schema tại [04-data-and-api-contracts.md](04-data-and-api-contracts.md#3-đặc-tả-file-manifest-tdeckjson-tại-từng-repo-phase-3).
  2. Xây dựng `scanner.ResolveVersion(dir string, src ManifestVersionSource)` tự động lấy version theo thứ tự ưu tiên: `wails.json` (`info.productVersion`) → `package.json` (`version`) → `git describe --tags --abbrev=0` → fallback `"v0.1.0"`.
  3. Trong `Scanner.inspectDir` / `relinkExistingProject`: nếu thư mục chứa `.tdeck.json`, tự động đồng bộ `targets` vào bảng `project_targets` (giữ nguyên ID nếu trùng `kind + name`) và cập nhật `category`, `tags`.
  4. **Xóa bỏ hoàn toàn map `desktopReleases`** khỏi `cmd/tdeck-sync/main.go`. CLI mới sẽ:
     - Đọc cấu hình target từ `.tdeck.json` hoặc `project_targets` trong SQLite.
     - Hỗ trợ flags: `--dry-run`, `--slug <project>`, `--target <web|desktop>`, `--channel <stable|beta>`, `--bump <patch|minor|major>`, `--build` (chạy `build_cmd` trước khi publish).

### `P3-02` — Multipart R2 Upload/Download & Tối ưu Nén Zip
- **Files sửa**: `go.mod`, `internal/cloud/r2_client.go`, `internal/cloud/archive.go`, `internal/cloud/release_manager.go`.
- **Chi tiết thực hiện**:
  1. Thêm `github.com/aws/aws-sdk-go-v2/feature/s3/manager` (`go get github.com/aws/aws-sdk-go-v2/feature/s3/manager`).
  2. Dùng `manager.NewUploader(c.client, func(u *manager.Uploader) { u.PartSize = 10 * 1024 * 1024; u.Concurrency = 4 })` cho file lớn (`>= 15 MB`) và `manager.NewDownloader` để tải song song theo part.
  3. Bọc `io.Reader` nguồn bằng `progressReader` phát event `release:progress` (giới hạn tần suất throttle mỗi `150ms`) gửi `bytes_done`, `bytes_total`, `percent`, `speed_bps` về frontend qua Wails runtime.
  4. Tối ưu `internal/cloud/archive.go`:
     - Các file đã nén sẵn (`.exe`, `.msi`, `.zip`, `.asar`, `.dll`, `.png`, `.woff2`, `.mp4`) dùng `zip.Store` hoặc `flate.BestSpeed` (mức 1) — giảm 60–75% thời gian đóng gói release mà không làm tăng dung lượng đáng kể.
     - Bổ sung tham số bỏ qua thư mục tạm (`node_modules`, `.git`, `*.pdb`, `*.log`) khi nén thư mục build.

### `P3-03` — 1-Click Build → Auto Changelog → Publish & Live Build Log Drawer
- **Files mới / sửa**: `internal/launcher/builder.go` (mới), `app.go`, `frontend/src/components/inspector/BuildLogDrawer.tsx` (mới), `frontend/src/components/inspector/InspectorReleaseTab.tsx`.
- **Chi tiết thực hiện**:
  1. `SuggestReleaseChangelog(projectID, targetID)`: chạy `git -C <dir> log <last_tag>..HEAD --pretty=format:"- %s (%h)"` (tối đa 30 commits gần nhất) trả về draft markdown để điền sẵn vào ô Release Notes.
  2. `BuildAndPublishTarget(req)`:
     - Tạo job bất đồng bộ quản lý bởi `JobRegistry` (`map[string]context.CancelFunc` trong `App`), trả về `jobID` ngay cho UI.
     - Chạy `target.build_cmd` trong `filepath.Join(localDir, target.subpath)` qua `cmd.exe /C <build_cmd>` (chỉ chạy lệnh đã lưu trong DB hoặc `.tdeck.json`, không nhận arbitrary command từ bên ngoài).
     - Stream `stdout` và `stderr` theo dòng (buffer batch mỗi `80ms`) qua event `build:log`; theo dõi nút **Hủy Job** (`CancelActiveJob`).
     - Sau khi build exit code `0`, tự phân giải artifact khớp `artifact_glob` → tự động chuyển tiếp sang bước nén + tính SHA-256 + upload R2 (`release:progress`).
  3. Frontend `BuildLogDrawer.tsx`: panel console nền `#0D0E11` chữ `JetBrains Mono` có auto-scroll, nút Copy Log, thanh tiến độ upload (%) và nút `Hủy`.

### `P3-04` — Auto-Updater Manifest (`latest.json`) Publisher
- **Files sửa**: `internal/cloud/release_manager.go`, `internal/model/project.go`.
- **Chi tiết thực hiện**:
  1. Mỗi khi publish thành công một Desktop release (`status == "published"`), backend tự động tổng hợp các asset mới nhất thuộc cùng `(project_id, channel)` (bao gồm `portable_zip` và `installer_exe` nếu có) và ghi file `releases/<slug>/<channel>/latest.json` lên R2.
  2. Dùng `R2Config.PublicDomain` làm URL tải mặc định trong `latest.json` (nếu có); ngược lại ghi `r2_key` để client tự fetch hoặc qua CDN proxy.
  3. Hiển thị URL `latest.json` trong `InspectorReleaseTab.tsx` kèm đoạn snippet JSON/Go tham chiếu giúp các app Wails/Electron khác của bạn kết nối auto-update trực tiếp mà không tốn server trung gian.

### `P3-05` — Local Process & Port Monitor
- **Files mới / sửa**: `internal/launcher/process_monitor.go` (mới), `internal/launcher/runner.go`, `app.go`, `frontend/src/components/ProjectCard.tsx`.
- **Chi tiết thực hiện**:
  1. Khi `Runner.launchWeb` hoặc `Runner.launchDesktop` khởi chạy ứng dụng, lưu `project_id ↔ PID` vào `ProcessMonitor` (theo dõi trạng thái tiến trình định kỳ 3–5 giây và kiểm tra socket TCP đang mở thuộc process tree của PID đó).
  2. Nếu Web app mở qua `start.bat` spawn tiến trình con (`node.exe`, `vite`, `go run`, `python`), duyệt process tree Windows (qua Win32 Toolhelp32 / `netstat -ano` được cache theo chu kỳ) để lấy chính xác cổng localhost đang lắng nghe (ví dụ `:5173`, `:3000`, `:8080`).
  3. Bắn event `process:changed` khi trạng thái đổi.
  4. UI hiển thị badge xanh `RUNNING :5173` trên `ProjectCard` và trong Matrix View, kèm nút:
     - `Open Localhost` (mở `http://localhost:5173`)
     - `Stop` (gọi `StopProjectProcess` → `taskkill /T /F /PID <pid>`).

### `P3-06` — Chế độ Xem Bảng `Release Matrix View` & Bộ lọc Nâng cao
- **Files mới / sửa**: `frontend/src/components/ReleaseMatrixTable.tsx` (mới), `frontend/src/components/TopCommandBar.tsx`, `frontend/src/components/SidebarNav.tsx`, `frontend/src/App.tsx`.
- **Chi tiết thực hiện**:
  1. Thêm nút chuyển đổi chế độ xem **Grid Cards ↔ Matrix Table** ở `TopCommandBar.tsx` (lưu lựa chọn vào `localStorage`) và nút **Kiểm tra Health Web hàng loạt** (`CheckAllWebHealth`).
  2. Cấu trúc cột của `ReleaseMatrixTable.tsx`:
     - **Project & Nhóm**: Thumbnail nhỏ + Tên + Category pill + Loại (`WEB` / `EXE` / `HYBRID`).
     - **Web Live**: Version đang live + chấm màu Health (`ONLINE 42ms` / `OFFLINE` / `DRIFT`) + nút mở nhanh.
     - **Desktop Release**: Version mới nhất trên R2 (`v0.3.4`) vs Version đang cài ở máy (`v0.3.3` → hiện nút `Update`), dung lượng.
     - **Local & Git**: Trạng thái (`READY`, `RUNNING :5173`, `MISSING .ENV`) + Git branch & badge `dirty` nếu có.
     - **Cập nhật gần nhất**: Thời gian release/mở gần nhất.
     - **Quick Actions**: `Run / Launch`, `Open Web`, `Update`, `Copy Download Link`, `Chi tiết`.
  3. Bổ sung nhóm **Bộ lọc nhanh Release** trong `SidebarNav.tsx`:
     - `Có bản Release` (`has_release`)
     - `Cần cập nhật` (`has_update` hoặc `web_drift`)
     - `Đang chạy (Local)` (`is_running`)
     - `Kênh Beta` (`channel == 'beta'`)

### `P3-07` — Bộ Kiểm thử Đầu-Cuối (E2E Verification) & Đóng gói
- **Nội dung**:
  - Viết unit/integration tests cho `manifest.go`, `health_checker.go`, `process_monitor.go`, `target_repo.go`, `release_repo.go`, và `UpdateDesktopRelease` (dùng mock `ObjectStore` + temp dirs giả lập rollback).
  - Chạy `go test -race -count=1 ./...` đảm bảo không có data race ở scanner pool, health checker, process monitor và thumbnail cache.
  - Chạy `cd frontend && npm run build` và `wails build -clean -platform windows/amd64`.
  - Lập checklist kiểm thử thực tế trên 2 máy Windows với R2 bucket thật (Publish → Sync → Hydrate → Update → Rollback → Check Web Health).

---

## 2. Tiêu chí Nghiệm thu Phase 3

1. `cmd/tdeck-sync/main.go` không còn chứa map `desktopReleases` hardcode; đọc toàn bộ cấu hình từ `.tdeck.json` / DB.
2. Upload file `.zip`/`.exe` > 50 MB dùng Multipart S3 và hiển thị thanh tiến độ `%` + tốc độ `MB/s` mượt trên UI mà không đơ giao diện.
3. Nút **Build & Publish** chạy `build_cmd`, stream log live vào `BuildLogDrawer` và hủy được giữa chừng bằng nút `Hủy`.
4. Sau khi publish Desktop release, file `releases/<slug>/stable/latest.json` xuất hiện trên R2 với đúng `version`, `sha256`, `size`, `url`.
5. Chế độ **Release Matrix View** hiển thị toàn bộ danh sách dự án dưới dạng bảng gọn, phản ánh đúng Web Health, Desktop Version Drift và Local Running Ports.
