# Hướng dẫn Đóng góp Mã nguồn (Contributing Guide)

Cảm ơn bạn đã quan tâm và đóng góp cho **T-Deck**! Tài liệu này quy định quy chuẩn kiến trúc, tổ chức module, phong cách lập trình và các bước kiểm thử bắt buộc trước khi gửi Pull Request.

---

## 1. Chuẩn bị Môi trường Phát triển

1. **Go**: Phiên bản `1.26` trở lên (`go version`).
   - Dự án sử dụng `modernc.org/sqlite` (Pure-Go SQLite), vì vậy bạn **không** cần cài đặt MinGW/GCC hay bật `CGO_ENABLED=1`.
2. **Node.js & npm**: Node.js `>= 20.x`.
3. **Wails CLI**: Phiên bản `v2.16.0+`:
   ```powershell
   go install github.com/wailsapp/wails/v2/cmd/wails@latest
   ```
4. **Cài đặt thư viện Frontend**:
   ```powershell
   cd frontend
   npm install
   ```

---

## 2. Nguyên tắc Kiến trúc & Tổ chức Mã nguồn

### 2.1. Tầng Backend Go (`app*.go` & `internal/*`)
- **Tách biệt rõ ràng theo miền trách nhiệm**:
  - `app.go`, `app_projects.go`, `app_releases.go`, `app_runtime.go`, `app_settings.go`: Chỉ đóng vai trò **Wails IPC Facade** điều phối giữa Frontend và các gói trong `internal/`. Không viết truy vấn SQL hay logic nghiệp vụ phức tạp trực tiếp trong `app*.go`.
  - `internal/scanner`: Mọi logic quét thư mục, nhận diện tech stack (`detector.go`), đọc `.tdeck.json` (`manifest.go`) và tạo `.env.example`.
  - `internal/media`: Toàn bộ xử lý hình ảnh thuần Go — kết xuất poster 16:9 (`generator.go`), dò tìm và giải mã bộ icon thương hiệu `.png`/`.ico` (`brand_icon.go`), và phục vụ HTTP `/thumbnails/*` (`thumbnail.go`).
  - `internal/database`: Quản lý SQLite WAL, schema migrations (`sqlite.go`), truy vấn FTS5 (`fts.go`) và repository (`project_repo.go`, `category_repo.go`, `release_repo.go`).
  - `internal/launcher`: Khởi chạy tiến trình tách biệt (`runner.go`), giám sát PID/Port (`process_monitor.go`) và kiểm tra sức khỏe Web (`health.go`).
  - `internal/cloud`: Giao tiếp S3/Cloudflare R2 (`r2_client.go`), đồng bộ `catalog.json` v2 (`catalog_sync.go`), đóng gói zip/unzip an toàn (`archive.go`) và phát hành đa phần (`release_manager.go`).
- **Chuẩn hóa mã lỗi (`internal/apperr`)**:
  - Sử dụng `apperr.Wrap(code, message, err)` để trả về lỗi có ngữ cảnh rõ ràng lên giao diện người dùng.

### 2.2. Tầng Frontend React 19 + TypeScript (`frontend/src/*`)
- **Chia tách Component & Custom Hooks theo đơn vị trách nhiệm**:
  - Giữ `App.tsx` gọn nhẹ, chỉ đảm nhận kết nối bố cục tổng thể và điều phối hộp thoại.
  - Tách logic trạng thái phức tạp vào `frontend/src/hooks/` (`useProjectFilters.ts`, `useBulkSelection.ts`, `useRuntimeEvents.ts`, `useReleaseBuildJob.ts`).
  - Chia nhỏ các thành phần giao diện theo thư mục miền: `catalog/`, `project/`, `sidebar/`, `inspector/`, `settings/`, `ui/`.
- **An toàn kiểu tĩnh (Strict TypeScript)**:
  - Mọi phương thức gọi xuống Go qua `window.go.main.App` phải được bọc kiểu tường minh trong `frontend/src/lib/api.ts` và khai báo cấu trúc dữ liệu khớp 1-1 với Go struct tại `frontend/src/types/project.ts`.
- **Thiết kế Giao diện (Design System)**:
  - Tuân thủ hệ màu tối (Carbon Dark Theme: `bg-canvas`, `bg-surface`, `bg-elevated`, `border-edge`, `text-primary`, `text-muted`, `text-accent`) và các token dùng chung trong `frontend/src/components/ui/uiTokens.ts`.

---

## 3. Quy trình Kiểm thử Bắt buộc (Verification Gates)

Trước khi hoàn tất bất kỳ thay đổi nào, hãy chạy đầy đủ bộ lệnh kiểm tra sau:

```powershell
# 1. Chạy toàn bộ Unit & Integration Tests của Go
go test -v -count=1 ./...

# 2. Kiểm tra xung đột bộ nhớ đa luồng (Race Detector)
go test -race -count=1 ./...

# 3. Kiểm tra kiểu TypeScript & biên dịch thử gói Frontend
cd frontend
npm run build
cd ..

# 4. Đóng gói thử bản thực thi Windows hoàn chỉnh
.\build.bat full
```

---

## 4. Quy chuẩn Commit & Pull Request

- Sử dụng định dạng **Conventional Commits** ngắn gọn, rõ nghĩa:
  - `feat(scanner): ignore third-party tools and discover deep containers`
  - `feat(media): auto-detect project brand icon suites for posters and UI badges`
  - `refactor(frontend): split App.tsx and inspector into focused hooks and components`
  - `fix(window): start maximised to prevent layout clipping on high-DPI displays`
  - `docs: add comprehensive open-source markdown documentation suite`
- Mỗi Pull Request nên tập trung vào một nhóm thay đổi cụ thể, đính kèm kết quả chạy `go test ./...` và ảnh chụp màn hình thực tế nếu có thay đổi về giao diện (UI/UX).

