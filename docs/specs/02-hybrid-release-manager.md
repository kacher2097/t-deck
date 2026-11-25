# Đặc tả Phase 2: Quản lý Release Web & Desktop App Đa Nền Tảng (`02-hybrid-release-manager.md`)

**Mục tiêu Phase 2:** Chuyển đổi T-Deck từ launcher 1 chiều thành hệ thống quản lý phát hành hoàn chỉnh cho cả dự án **Web**, **Desktop App** và **Hybrid (Web + Desktop chung 1 project)** — bao gồm lịch sử release, kiểm tra sức khỏe & lệch phiên bản Web live, đóng gói static bundle Web, cập nhật/rollback nguyên tử cho Desktop App và chia sẻ link tải qua R2.

---

## 1. Chi tiết Kỹ thuật Từng Ticket

### `P2-01` — Database Migration v3 & Target/Release Repositories
- **Files mới / sửa**: `internal/database/sqlite.go`, `internal/database/target_repo.go` (mới), `internal/database/release_repo.go` (mới), `internal/model/project.go`.
- **Chi tiết thực hiện**:
  1. Thực hiện migration `user_version = 3` như đặc tả tại [04-data-and-api-contracts.md](04-data-and-api-contracts.md#version-3-phase-2--multi-target-hybrid-webdesktop--release-history), có bước backup `tdeck.db` → `tdeck.db.v2.bak` trước khi chạy transaction.
  2. Bổ sung Go structs:
     - `model.ProjectTarget`
     - `model.Release`
     - `model.WebHealthResult`
     - Mở rộng `model.ProjectView` thêm `Targets []model.ProjectTarget`, `LatestReleases map[string]model.Release` (key là `target_id`), `InstalledVersion string`, `GitBranch string`, `GitDirty bool`, `HasUpdate bool`.
  3. Viết `TargetRepo` & `ReleaseRepo` với full unit tests cho CRUD, soft-delete, query lịch sử theo `project_id`/`target_id`, và tự động đồng bộ trạng thái `projects.type = "hybrid"` khi 1 project có cả target `web` và `desktop` chưa xóa.

### `P2-02` — Quản lý Targets Đa Nền Tảng trong UI (`Config` & `AddProjectModal`)
- **Files sửa**: `app.go`, `frontend/src/types/project.ts`, `frontend/src/lib/api.ts`, `frontend/src/components/inspector/InspectorConfigTab.tsx`, `frontend/src/components/AddProjectModal.tsx`.
- **Chi tiết thực hiện**:
  1. Cho phép chọn loại dự án `Web App`, `Desktop App`, hoặc `Hybrid (Web + Desktop)`.
  2. Trong `InspectorConfigTab.tsx`, bổ sung khối **"Targets Phát hành & Triển khai"**:
     - Hiển thị danh sách target hiện có (chip `WEB` / `DESKTOP` + tên + platform/distribution).
     - Nút `+ Thêm Target Web` hoặc `+ Thêm Target Desktop` cho phép khai báo thêm target vào cùng 1 project mà không phải tạo card mới.
     - Form cấu hình cho từng Target:
       - **Web target**: `prod_url`, `staging_url`, `health_url`, `version_url`, `hosting_provider`, `subpath`, `launch_entry` (`start.bat`), `build_cmd`, `artifact_glob` (`dist`).
       - **Desktop target**: `platform` (`windows`), `distribution` (`portable` | `installer`), `subpath`, `launch_entry` (`.exe`), `build_cmd`, `artifact_glob`.

### `P2-03` — Web Release Center & Quick Browser Launch
- **Files mới / sửa**: `app.go`, `internal/launcher/runner.go`, `frontend/src/components/ProjectCard.tsx`, `frontend/src/components/inspector/InspectorReleaseTab.tsx`.
- **Chi tiết thực hiện**:
  1. Expose `OpenTargetURL(targetID string, env string)` gọi `wailsruntime.BrowserOpenURL(a.ctx, url)` (validate scheme chỉ cho phép `http://` hoặc `https://` để tránh command/file injection).
  2. Trên `ProjectCard.tsx`:
     - Nếu project có Web target với `prod_url` → hiển thị nút icon `Globe` (`Mở Live Web`) ở footer bên cạnh Explorer / VS Code.
     - Nếu project là `hybrid` → hiển thị badge `HYBRID` trên góc trên bên trái ảnh preview; nút action chính giữ Launch Desktop hoặc sổ menu nhanh (`Run Web .bat` / `Launch EXE` / `Open Live`).
  3. Trong `InspectorReleaseTab.tsx`:
     - Chia rõ sub-section hoặc sub-pill theo từng Target của project (ví dụ pill chuyển giữa `[Web: Portal]` và `[Desktop: Windows Client]`).

### `P2-04` — Web Health & Version Drift Checker
- **Files mới / sửa**: `internal/cloud/health_checker.go` (mới), `app.go`, `frontend/src/components/inspector/InspectorReleaseTab.tsx`, `frontend/src/components/ProjectCard.tsx`.
- **Chi tiết thực hiện**:
  1. Xây dựng `CheckWebHealth(ctx, target)`:
     - Dùng `http.Client` timeout `5s` (tái sử dụng `http.Transport`).
     - Gọi `health_url` (hoặc fallback `prod_url` bằng `HEAD`/`GET`), đo `latency_ms` và HTTP status (`200..399` = online).
     - Nếu `version_url` được cấu hình: GET JSON và parse linh hoạt các key phổ biến (`version`, `app_version`, `tag`, `build`). Nếu không có `version_url`, kiểm tra tùy chọn meta `<meta name="app-version" content="...">` khi `prod_url` trả HTML.
     - So sánh `live_version` với bản release mới nhất của target đó trong SQLite → đặt cờ `drift = true` nếu khác nhau.
  2. `CheckAllWebHealth()`: chạy concurrent worker pool (tối đa 8 goroutines), bắn event `health:result` về UI để cập nhật badge `ONLINE 42ms · v1.4.2` hoặc `OFFLINE` / `DRIFT`.

### `P2-05` — Web Static Bundle Archive Release lên R2
- **Files sửa**: `internal/cloud/release_manager.go`, `app.go`, `frontend/src/components/inspector/InspectorReleaseTab.tsx`.
- **Chi tiết thực hiện**:
  1. Cho phép target Web thực hiện **"Đóng gói & Lưu Release Web"**:
     - Chọn thư mục build (`dist/`, `out/`, `build/`) hoặc tự lấy theo `target.subpath + target.artifact_glob`.
     - Nén thành `releases/<slug>/web/<version>/<slug>-web-<version>-dist.zip`, tính SHA-256 & kích thước file.
     - Upload lên R2, ghi bản ghi vào bảng `releases` (`asset_kind = 'static_zip'`, lưu kèm `deploy_url = prod_url` và `commit_sha` từ `.git/HEAD`), đồng bộ `releases/<slug>/index.json`.
  2. Cho phép tạo bản ghi **"External Deploy Release"** (chỉ ghi `version`, `deploy_url`, `commit_sha`, `changelog` mà không cần upload file nếu web deploy trực tiếp qua Git/Vercel/Cloudflare Pages) để lưu thống nhất 1 timeline release cho bản Web.

### `P2-06` — Desktop Release History, Multi-Asset & Integrity (SHA-256)
- **Files sửa**: `internal/cloud/release_manager.go`, `internal/cloud/catalog_sync.go`, `internal/database/release_repo.go`, `frontend/src/components/inspector/InspectorReleaseTab.tsx`.
- **Chi tiết thực hiện**:
  1. Nâng cấp `PublishDesktopRelease` → `PublishRelease`:
     - Hỗ trợ đồng thời `asset_kind`: `portable_zip` (dùng cho Hydrate/Update nội bộ) hoặc `installer_exe` (bộ cài gửi user).
     - Tính SHA-256 checksum (`crypto/sha256`) của artifact ngay sau khi đóng gói, lưu `file_name`, `file_size`, `sha256`, `channel` (`stable` | `beta`), `git_tag`, `commit_sha` vào `releases`.
     - Ghi/cập nhật file `releases/<slug>/index.json` trên R2 để máy khác tải về toàn bộ lịch sử release mà không làm phình `catalog.json`.
  2. Giao diện **Timeline Lịch sử Release** trong `InspectorReleaseTab.tsx`:
     - Bảng/danh sách các phiên bản: `Version + Channel pill` | `Asset & Dung lượng` | `SHA-256 (rút gọn + copy)` | `Ngày phát hành` | `Hành động` (`Cài đặt / Rollback`, `Copy Link`, `Xóa`).
  3. Hỗ trợ **Retention / Cleanup**: xóa một bản release cũ khỏi R2 (`DeleteObject`) và đánh dấu `deleted_at` trong DB + `index.json`.

### `P2-07` — Safe Atomic Desktop Update & Rollback
- **Files sửa**: `internal/cloud/release_manager.go`, `internal/scanner/metadata.go`, `app.go`, `frontend/src/components/ProjectCard.tsx`, `frontend/src/components/inspector/InspectorReleaseTab.tsx`.
- **Chi tiết thực hiện**:
  1. Ghi nhận `installed_version` và `installed_release_id` vào bảng `local_state` + file `.tdeck-meta.json` mỗi khi `HydrateProjectOnNewPC` hoặc `UpdateDesktopRelease` thành công.
  2. Tính cờ `has_update` khi `installed_version != ""` và khác với bản `stable` mới nhất trên `releases` (ưu tiên so sánh SemVer chuẩn; fallback so sánh khác chuỗi nếu không theo SemVer).
  3. Quy trình `UpdateDesktopRelease(projectID, targetID, releaseID)` an toàn trên Windows:
     - **Bước 1**: Kiểm tra process `.exe` mục tiêu có đang chạy không (tránh lỗi lock file trên Windows); nếu đang chạy → báo lỗi rõ ràng yêu cầu đóng app (hoặc nút `Dừng app & Cập nhật`).
     - **Bước 2**: Tải artifact từ R2 về file tạm và **verify SHA-256** khớp với `release.sha256`. Sai hash → hủy ngay, không đụng thư mục đang cài.
     - **Bước 3**: Giải nén vào thư mục staging `<apps_dir>/<slug>.__staging` (hoặc copy installer nếu `asset_kind == 'installer_exe'`).
     - **Bước 4**: Di chuyển thư mục hiện tại sang `<apps_dir>/<slug>.__bak` (xóa `__bak` cũ hơn nếu có), đổi tên `__staging` → `<apps_dir>/<slug>`, cập nhật `.tdeck-meta.json` và `local_state`.
     - **Bước 5**: Nếu bước swap lỗi → tự động khôi phục `__bak` về lại vị trí gốc.
  4. `RollbackDesktopRelease`: nếu có bản `__bak` cục bộ thì swap ngược ngay (0ms tải mạng); nếu không có thì tải bản release đã chọn trong lịch sử và chạy lại quy trình swap trên.

### `P2-08` — Shareable Download Links (`PublicDomain` & S3 Presigned URLs)
- **Files sửa**: `internal/cloud/r2_client.go`, `app.go`, `frontend/src/components/settings/R2SettingsSection.tsx`, `frontend/src/components/inspector/InspectorReleaseTab.tsx`.
- **Chi tiết thực hiện**:
  1. Bổ sung ô nhập `Public Domain (CDN / Custom Domain)` trong `R2SettingsSection.tsx` (hiện `R2Config.PublicDomain` đã có trong backend nhưng chưa có UI nhập liệu).
  2. Thêm method `PresignGetURL(ctx, key, expiry)` vào `ObjectStore` sử dụng `s3.NewPresignClient(c.client)`.
  3. `GenerateReleaseDownloadURL(releaseID, expiryHours)`:
     - Nếu `R2Config.PublicDomain != ""` → trả `https://<public_domain>/<r2_key>`.
     - Nếu không có `PublicDomain` → sinh Presigned S3 URL hết hạn theo `expiryHours` (mặc định 24h, tùy chọn 1h / 24h / 7 ngày).
  4. Trên UI `InspectorReleaseTab.tsx`: nút `Copy Link` sao chép URL vào clipboard + toast thông báo loại link (`Public CDN` hoặc `Presigned 24h`).

---

## 2. Thiết kế Bố cục Giao diện Nâng cấp (Phase 2)

Tuân thủ tuyệt đối palette & tokens hiện có trong `frontend/src/components/ui/uiTokens.ts` (`bg-canvas`, `bg-surface`, `bg-elevated`, `border-edge`, font `Inter` / `Plus Jakarta Sans` / `JetBrains Mono`).

### 2.1. `InspectorReleaseTab.tsx` (Layout 2 cột trong Modal `980px`)
- **Thanh chọn Target (Target Switcher Bar)** ở đỉnh tab:
  - `[Web: Portal (Live)]` `[Desktop: Windows Client (v0.3.4)]` `[+ Thêm Target]`
- **Cột trái (5/12)**:
  - Card **Trạng thái hiện tại**:
    - Với **Web**: URL Production + Staging (nút Mở trình duyệt + Copy), trạng thái Health (`ONLINE · 48ms` / `OFFLINE`), Live Version vs Latest Release, nút `Kiểm tra ngay`.
    - Với **Desktop**: Version mới nhất trên R2, Version đang cài ở máy này (`installed_version` + cảnh báo nếu lệch), nút `Update ngay` / `Rollback` / `Copy Download Link`.
- **Cột phải (7/12)**:
  - Form **Phát hành phiên bản mới** (Version, Channel `stable`/`beta`, nguồn build hoặc thư mục `dist`/`.exe`, Release Notes).
  - Bảng **Lịch sử Release (Release History)** cuộn nội bộ với badge Channel, kích thước, SHA-256 và nút thao tác nhanh.

### 2.2. `ProjectCard.tsx`
- **Top-left badges**: `WEB` / `EXE` / `HYBRID`, kèm badge version (`v0.3.4`), và nếu `has_update == true` hiện pill nhỏ màu `status-warn` (`Update v0.3.5`).
- **Footer actions**: giữ tối đa 4 nút icon gọn gàng — Primary action (`Run .bat` / `Launch` / `Install` / `Update`), nút `Globe` (nếu có `prod_url`), `FolderOpen`, `Code2`, `SlidersHorizontal`.

---

## 3. Tiêu chí Nghiệm thu Phase 2

1. Một project kiểu `hybrid` chứa được đồng thời ít nhất 1 target `web` và 1 target `desktop` trong cùng 1 card và 1 modal chi tiết.
2. Nút **Mở Live Web** mở đúng `prod_url`; **Check Health** trả về HTTP code, latency và phát hiện được lệch version khi `/version.json` khác bản release mới nhất.
3. Publish Desktop hoặc Web `dist.zip` ghi đầy đủ `releases` + `sha256` + `file_size` và cập nhật `releases/<slug>/index.json` trên R2.
4. `UpdateDesktopRelease` từ chối nếu sai SHA-256 hoặc file `.exe` đang khóa, và rollback thành công khi test lỗi giả lập hoặc bấm Rollback.
5. `GenerateReleaseDownloadURL` trả về Public CDN link khi có `PublicDomain`, và trả về Presigned URL hoạt động được khi `PublicDomain` rỗng.
