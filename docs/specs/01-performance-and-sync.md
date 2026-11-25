# Đặc tả Phase 1: Tối ưu Hiệu năng & Toàn vẹn Đồng bộ (`01-performance-and-sync.md`)

**Mục tiêu Phase 1:** Loại bỏ hoàn toàn các nút thắt I/O đồng bộ (N+1 `os.Stat`, double IPC fetch, `Scanner` $O(N^2)$), tăng tốc đồng bộ R2 & hiển thị ảnh, đồng thời vá 2 lỗ hổng dữ liệu khi sync nhiều máy (mất `categories` và hồi sinh project đã xóa).

---

## 1. Bảng Kiểm toán Điểm nghẽn Baseline (Codebase v1)

| Hạng mục | Vị trí chính xác trong code v1 | Vấn đề gây chậm / lỗi | Giải pháp Phase 1 |
|---|---|---|---|
| **N+1 Disk Stat khi đọc DB** | `internal/database/project_repo.go:392-408` (`enrichLocalFlags`) | Mỗi dòng trả về từ `ListProjects` gọi 4 `os.Stat` đồng bộ (`.env`, `.env.example`, `.bat`, `thumbnails/<id>.png`). 50 project = 200 syscalls/lần gọi. | Lưu `has_env`, `has_env_example`, `has_bat` vào `local_state`; dựng `ThumbnailURL` từ `thumbnail_updated_at > 0` + in-memory cache danh sách file ảnh. |
| **Double IPC Fetch** | `frontend/src/App.tsx:53-92` (`loadQueryMatched` + `loadFiltered`) | Mỗi lần đổi filter ở Sidebar hoặc `refreshAll()` đều gọi `api.listProjects` 2 lần qua Wails bridge. | Frontend chỉ gọi `api.listProjects({ query: debouncedQuery })` **1 lần**, sau đó dùng `useMemo` + `applyFacets` ở client (0ms, 0 IPC khi đổi filter). |
| **`Scanner` $O(N^2)$ DB reads** | `internal/scanner/scanner.go:256` (`relinkExistingProject`) | Trong vòng lặp từng project lại gọi `s.repo.ListAllLocalStates()` đọc toàn bộ bảng `local_state`. | Gọi `ListAllLocalStates()` **1 lần** ở đầu `ScanWithOptions` và truyền `map[string]model.LocalState` vào hàm xử lý. |
| **Full-Tree Walk trong Scanner** | `internal/scanner/scanner.go:364-382` (`projectLatestModTime`) & `internal/scanner/detector.go:99-101` (`findBuiltExe`) | Walk toàn bộ cây thư mục để tìm `latest ModTime` thay vì dừng sớm; gọi lại `findBuiltExe()` ngay cả khi `ExeName` đã có và file vẫn tồn tại. | `projectLatestModTime` thoát ngay (`filepath.SkipAll`) khi gặp file có `ModTime >= cutoff`; bỏ `findBuiltExe()` nếu `existing.ExeName` hợp lệ. |
| **R2 Client & Sync tuần tự** | `app.go:423-433` (`loadR2Store`) & `internal/cloud/catalog_sync.go:95-123` | Tạo `s3.Client` mới mỗi thao tác (mất HTTP/2 keep-alive); tải/up thumbnail tuần tự và gọi `HeadObject` trước `GetObject` (2 RTT/ảnh). | Cache `*cloud.R2Client` theo cấu hình; sync thumbnail bằng worker pool (6–8 goroutines) và bỏ `HeadObject` thừa trước `DownloadFile`. |
| **HTTP Cache & Ảnh 4K qua IPC** | `internal/media/thumbnail.go:100` & `frontend/src/lib/clipboard.ts:57-66` | Header trả `Cache-Control: no-store` dù URL đã có `?v=<ts>`; clipboard/drop gửi nguyên ảnh gốc 4K base64 (3–8MB) qua IPC. | Trả `Cache-Control: public, max-age=31536000, immutable` khi có `?v=`; resize `<canvas>` tối đa `1280×720` ở frontend trước khi gửi PNG. |
| **Đồng bộ thiếu `categories` & `deleted_at`** | `internal/cloud/catalog_sync.go:26-148` & `internal/database/project_repo.go:355-369` | `Catalog.Categories` không được đọc/ghi khi sync; `DeleteProject` xóa cứng nên Sync lần sau kéo ngược lại project đã xóa. | Áp dụng Schema v2 (`deleted_at`), merge `categories` và `tombstones` khi sync `catalog.json` v2 (vẫn đọc được v1). |
| **Re-render & Font bloat** | `frontend/src/components/ProjectCard.tsx`, `SidebarNav.tsx:162-214`, `main.tsx:1-8` | `ProjectCard` chưa `React.memo`; `SidebarNav` chạy 7 lần `.filter()`; `main.tsx` import CSS chứa cả `.woff` cũ lẫn `.woff2`. | Bọc `React.memo` + callbacks ổn định; gộp pass đếm Sidebar còn 1 vòng lặp; tối ưu font bundle chỉ dùng `.woff2` / subset cần thiết. |

---

## 2. Chi tiết Kỹ thuật Từng Ticket

### `P1-01` — DB Migration v2, Pragmas & Cached Local Flags
- **Files sửa**: `internal/database/sqlite.go`, `internal/database/project_repo.go`, `internal/model/project.go`, `internal/scanner/scanner.go`, `internal/launcher/runner.go`.
- **Chi tiết thực hiện**:
  1. Cập nhật `sqlite.go` theo [04-data-and-api-contracts.md](04-data-and-api-contracts.md#12-các-phiên-bản-schema-pragma-user_version): thêm các pragma `synchronous = NORMAL`, `temp_store = MEMORY`, `cache_size = -16000` và migration lên `user_version = 2` (cột `deleted_at`, `has_env`, `has_env_example`, `has_bat`, `flags_checked_at`, và 5 indexes).
  2. Thêm hàm `scanner.ComputeLocalFlags(dir, batFile string) (hasEnv, hasEnvExample, hasBat bool)` và ghi vào `local_state` tại các điểm mutate:
     - `Scanner` (`inspectDir`, `handleExistingMeta`, `relinkExistingProject`)
     - `App.UpsertProject`
     - `Runner.launchWeb` (khi tự tạo `.env` hoặc `start.bat`)
     - `App.SyncEnvExample` và `App.GenerateStartBat`
     - Backfill 1 lần ngay sau migration v2 cho các row có `flags_checked_at = 0`.
  3. Sửa `ProjectRepo.ListProjects` chỉ select `WHERE p.deleted_at = 0`, đọc thẳng `has_env`, `has_env_example`, `has_bat` từ cột DB (0 `os.Stat`).
  4. Sửa `ProjectRepo.GetProject(id)` vẫn giữ kiểm tra file thực tế (refresh flags vào DB) khi người dùng mở chi tiết 1 project để đảm bảo số liệu trong modal luôn chính xác 100% dù người dùng tự sửa tay bên ngoài.
  5. Thêm `ThumbnailExistsCache` (map + `sync.RWMutex`) trong `ProjectRepo` hoặc `ThumbnailService`, invalidate ngay khi `SaveClipboardPNG`, `GenerateProjectThumbnail`, `DeleteProject`, `SyncCatalogWithR2`.

### `P1-02` — Client-side Faceted Filtering & React Render Optimization
- **Files sửa**: `frontend/src/App.tsx`, `frontend/src/components/SidebarNav.tsx`, `frontend/src/components/ProjectCard.tsx`.
- **Chi tiết thực hiện**:
  1. Tách `applyFacets` thành helper dùng chung (ví dụ `frontend/src/lib/facets.ts`).
  2. Trong `App.tsx`: xóa `loadFiltered` và state thừa; giữ duy nhất `loadQueryMatched` (`api.listProjects({ type: "", category: "", status: "", tag: "", query: debouncedQuery })`) và tính `filteredProjects = useMemo(() => applyFacets(queryMatchedProjects, ...), [...])`.
  3. Ổn định hóa các handler truyền xuống `ProjectCard` (`useCallback`) nhận `project: ProjectView` hoặc `id` để `React.memo(ProjectCard)` không re-render toàn grid khi đang gõ tìm kiếm hoặc đổi trạng thái 1 card.
  4. Trong `SidebarNav.tsx`: thay 7 lệnh `.filter(...)` tách biệt bằng 1 vòng lặp `for (const p of forTypeFacet)` và `for (const p of forStatusFacet)` trong `useMemo`.

### `P1-03` — Tối ưu `Scanner` ($O(N)$ DB + Early Exit + Parallel Walk)
- **Files sửa**: `internal/scanner/scanner.go`, `internal/scanner/detector.go`, `internal/scanner/metadata.go`.
- **Chi tiết thực hiện**:
  1. Trong `ScanWithOptions`: gọi `existingStates, err := s.repo.ListAllLocalStates()` **1 lần duy nhất** trước vòng lặp `roots`. Truyền `existingStates` xuống `relinkExistingProject`.
  2. Trong `projectLatestModTime(root, cutoff)`:
     - Kiểm tra nhanh `os.Stat` các file neo nhẹ (`.git/index`, `.git/HEAD`, `package.json`, `go.mod`, `wails.json`) trước. Nếu bất kỳ file nào có `ModTime >= cutoff`, trả `true` ngay lập tức (không cần `WalkDir`).
     - Trong `WalkDir`, ngay khi gặp 1 file có `!ModTime().Before(cutoff)`, trả `filepath.SkipAll`.
  3. Trong `relinkExistingProject`: chỉ gọi `detectProjectMarkers` / `findBuiltExe` khi `existing.ExeName == ""` hoặc `!fsutil.FileExists(filepath.Join(dir, existing.ExeName))` hoặc `existing.TechStack == ""`.
  4. Nếu có nhiều `WorkspaceDirs`, quét các root song song bằng worker pool (tối đa `min(len(roots), 4)`) và dùng mutex bảo vệ `seenProjectIDs` + `report`; hoặc gom các lệnh `UpsertProject`/`UpsertLocalState` vào batch helper để giảm số transaction SQLite WAL commits.

### `P1-04` — Thumbnail HTTP Cache & Client-side Image Resize (`1280×720`)
- **Files sửa**: `internal/media/thumbnail.go`, `frontend/src/lib/clipboard.ts`.
- **Chi tiết thực hiện**:
  1. Trong `ThumbnailService.ServeHTTP`: nếu `r.URL.Query().Get("v") != ""`, đặt `w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")`; ngược lại giữ `no-store`.
  2. Trong `clipboard.ts`:
     - Mọi file ảnh (kể cả `image/png` kích thước lớn) đều kiểm tra kích thước qua `Image`; nếu `naturalWidth > 1280 || naturalHeight > 720`, vẽ xuống `<canvas>` với tỷ lệ giữ nguyên sao cho `width <= 1280` và `height <= 720` rồi xuất `canvas.toDataURL("image/png")`.
     - Nếu `image/png` đã nhỏ hơn hoặc bằng `1280×720` và `< 500 KB`, giữ nguyên blob gốc không vẽ lại để tránh tốn CPU.

### `P1-05` — R2 Connection Pooling & Concurrent Thumbnail Sync
- **Files sửa**: `app.go`, `internal/cloud/r2_client.go`, `internal/cloud/catalog_sync.go`.
- **Chi tiết thực hiện**:
  1. Cache `*cloud.R2Client` và signature cấu hình (`AccountID + AccessKeyID + SecretAccessKey + BucketName`) trong `App` với `sync.RWMutex`; chỉ khởi tạo lại khi `SaveSettings` thay đổi thông số R2.
  2. Trong `R2Client.DownloadFile`: trả sentinel error `ErrObjectNotFound` khi `isNotFoundErr(err)` là `true`.
  3. Trong `SyncCatalogWithR2`: bỏ lời gọi `store.ObjectExists` trước `DownloadFile`; dùng bounded worker pool (`errgroup` hoặc semaphore channel kích thước 6) để tải/upload thumbnails song song, dùng `atomic.Int64` để đếm `ThumbnailsDown` / `ThumbnailsUp`.

### `P1-06` — Đồng bộ 2 chiều `Categories` & Soft-Delete Tombstones (`deleted_at`)
- **Files sửa**: `internal/model/project.go`, `internal/database/project_repo.go`, `internal/database/category_repo.go`, `internal/cloud/catalog_sync.go`, `app.go`.
- **Chi tiết thực hiện**:
  1. `DeleteProject(id, deleteLocalFiles)`: chuyển sang soft-delete (`UPDATE projects SET deleted_at = ?, updated_at = ? WHERE id = ?`, xóa `projects_fts` và xóa `local_state`, xóa file thumbnail cục bộ).
  2. `DeleteCategory(id)`: tương tự, đặt `deleted_at = now, updated_at = now` thay vì xóa hàng ngay, đồng thời clear `projects.category = ''` của các project thuộc nhóm đó.
  3. Thêm `ListAllProjectsIncludingDeleted()` và `ListAllCategoriesIncludingDeleted()` cho `SyncCatalogWithR2`.
  4. `SyncCatalogWithR2`:
     - Merge `Categories` 2 chiều theo `updated_at` (giữ bản ghi có `updated_at` mới hơn, bao gồm `deleted_at`).
     - Merge `Projects` 2 chiều theo `updated_at`: nếu bên thắng có `deleted_at > 0` thì đánh dấu soft-delete ở bên còn lại và xóa `local_state` / `projects_fts`.
     - Dọn các tombstone có `deleted_at > 0` và quá 90 ngày trước khi ghi `catalog.json` v2.
     - Vẫn đọc mượt `catalog.json` v1 (khi `remoteCatalog.Version <= 1`, gán `DeletedAt = 0`).

### `P1-07` — Tối ưu Asset Font Bundled trong Binary
- **Files sửa**: `frontend/src/main.tsx`, `frontend/vite.config.ts`.
- **Chi tiết thực hiện**:
  - Thay import CSS mặc định của `@fontsource/*` (vốn kéo theo toàn bộ `cyrillic`, `greek`, `vietnamese`, `latin` và cả file `.woff` legacy) bằng import chỉ các subset cần thiết cho tiếng Việt/Anh (`latin`, `latin-ext`, `vietnamese`) hoặc dùng `vite.config.ts` / `@fontsource-variable` để loại bỏ `.woff` cũ (WebView2 trên Windows dùng Chromium hiện đại hỗ trợ 100% `.woff2`).

---

## 3. Tiêu chí Nghiệm thu & Benchmarks (Acceptance Criteria)

1. **Tốc độ lọc & tìm kiếm**:
   - Đổi Category / Type / Status / Tag trong `SidebarNav` phát sinh **0 lệnh gọi IPC** (`api.listProjects` không chạy lại).
   - `ProjectRepo.ListProjects` thực hiện **0 syscall `os.Stat`** cho `.env`, `.env.example`, `.bat`.
2. **Tốc độ `Scanner`**:
   - Số query `ListAllLocalStates()` trong `ScanWithOptions` giảm từ $O(N)$ xuống cố định **1 lần**.
   - Re-scan thư mục `h:\app` không thay đổi có thời gian giảm ≥ 50% so với baseline.
3. **Đồng bộ đa máy (Test case bắt buộc)**:
   - Tạo Category trên máy A → Sync R2 → Sync trên máy B → Category xuất hiện trên máy B.
   - Xóa Project X trên máy A → Sync R2 → Sync trên máy B → Project X biến mất trên máy B (không bị hồi sinh).
   - Sync `catalog.json` format v1 cũ từ R2 không lỗi, tự nâng cấp lên v2.
4. **Hồi quy**: `go test -v -count=1 ./...`, `cd frontend && npm run build`, và `wails build -clean -platform windows/amd64` PASS 100%.
