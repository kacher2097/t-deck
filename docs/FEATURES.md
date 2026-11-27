# Danh mục Tính năng Chuyên sâu (T-Deck Feature Reference)

Tài liệu này mô tả chi tiết từng phân hệ tính năng của **T-Deck**, giải thích cơ chế kỹ thuật bên dưới và cách các mô-đun phối hợp với nhau.

---

## 1. Bộ Quét Không gian làm việc Thông minh (`internal/scanner`)

### 1.1. Khám phá đa tầng & Nhận diện Thư mục Container
- **Duyệt linh hoạt từ ổ đĩa hoặc thư mục gốc**: Khi người dùng cấu hình `D:\` hoặc `D:\app`, bộ quét tự động nhận biết các thư mục gom nhóm (workspace containers như `app`, `projects`, `repos`, `workspaces`, `tool-github`, `git vnpay`...) để đi sâu xuống từng dự án con bên trong mà không gộp nhầm thư mục cha thành một dự án đơn lẻ.
- **Nhận diện hơn 25 dấu hiệu công nghệ (`detector.go`)**:
  - **Desktop / Native**: Wails (`wails.json`), Electron (`electron-builder.yml`, `package.json`), Tauri (`src-tauri/tauri.conf.json`), Go Native GUI, Python Desktop (`.exe` đóng gói), Rust (`Cargo.toml`).
  - **Web / Backend / Fullstack**: React, Vite, Next.js, Vue, Nuxt, Angular, Node.js, TypeScript, Tailwind CSS, Go API, Python (`pyproject.toml`, `requirements.txt`), Java / Spring Boot / Armeria (`pom.xml`, `build.gradle`), PHP/WordPress, Docker Compose (`docker-compose.yml`).
- **Bộ lọc loại trừ phần mềm bên thứ ba**:
  - Tự động loại bỏ các thư mục chứa trình cài đặt hoặc phần mềm công cụ không phải dự án mã nguồn (như `TablePlus`, `PostgreSQL`, `Proxifier`, `Xshell`, `Xftp`, `Go` SDK, `Browser`, `uv`, các bản giải nén tạm `*-acceptance`, `*-upstream-*`).
  - Tránh quét trùng lặp các bản build artifact (`.exe`) nằm bên trong cây thư mục của một dự án đã được nhận diện.

### 1.2. Neo Danh tính Bất biến (`.tdeck-meta.json`)
- Mỗi dự án được gắn một file `.tdeck-meta.json` tại thư mục gốc chứa `project_id` (UUID v4), `slug`, và `type`.
- **Tự động Relink $O(N)$**: Khi thư mục dự án được di chuyển sang đường dẫn mới hoặc đổi tên thư mục cha, T-Deck nạp bảng ánh xạ `local_state` vào bộ nhớ và tự động cập nhật đường dẫn mới cho đúng `project_id` đó, giữ nguyên toàn bộ lịch sử phát hành, nhóm phân loại và ảnh đại diện.

---

## 2. Động cơ Nhận diện Thương hiệu & Kết xuất Poster 16:9 (`internal/media`)

### 2.1. Tự động Dò tìm Bộ Icon Thương hiệu (`brand_icon.go`)
- **Không cần cấu hình thủ công**: Khi chạy `Scan Local` (hoặc tạo lại thumbnail), T-Deck tự động quét các đường dẫn icon chuẩn trong thư mục gốc và tối đa 2 cấp thư mục con (`apps/*`, `*-desktop/*`, `frontend/*`, `web/*`, `extension/*`):
  - `build/appicon.png`, `build/windows/icon.png`, `build/windows/icon.ico`, `build/icon.ico`
  - `public/apple-touch-icon.png`, `public/favicon/apple-touch-icon.png`, `public/favicon/favicon-96x96.png`
  - `web/public/icon-512.png`, `web/public/icon-192.png`
  - `frontend/public/apple-touch-icon.png`, `frontend/assest/favicon/apple-touch-icon.png`
  - `assets/icon.png`, `extension/icons/icon128.png`, `icons/icon128.png`
- **Bộ giải mã Đa định dạng Thuần Go**:
  - Hỗ trợ giải mã trực tiếp cả `.png` lẫn `.ico` (tự động chọn frame có độ phân giải cao nhất từ cả frame PNG nhúng lẫn frame **32-bit BGRA BMP/DIB** kèm mặt nạ alpha).
  - Kiểm tra nghiêm ngặt tỷ lệ khung hình vuông (`0.78 <= width/height <= 1.28`) và kích thước (`48px <= size <= 2048px`), tự động loại bỏ các ảnh banner ngang hoặc icon SVG mặc định của Vite template.

### 2.2. Hai Chế độ Kết xuất (Có Brand Icon vs Không có Brand Icon)
- **Khi dự án có Brand Icon**:
  1. Xuất file riêng `<InstallDir>\thumbnails\<id>.icon.png` (`128×128` sắc nét với nội suy Catmull-Rom) và cấp trường `icon_url` cho giao diện hiển thị cạnh tên dự án trên Card, bảng Matrix và tiêu đề Modal.
  2. Kết xuất poster `1280×720` có khóa nhận diện thương hiệu: gắn khung icon bo góc `124×124` kèm huy hiệu `APP ICON` ở góc trên trái và đặt tấm kính tối `288×288` chứa logo dự án ở tâm khung schematic bên phải.
- **Khi dự án không có Brand Icon**:
  - Giữ nguyên 100% thiết kế poster kỹ thuật `1280×720` tiêu chuẩn (gồm thanh màu chủ đề, nhãn phân vùng kiến trúc, tiêu đề dự án, các huy hiệu công nghệ và sơ đồ kiến trúc đặc trưng theo từng miền nghiệp vụ).

---

## 3. Cơ sở dữ liệu Local-First & Tìm kiếm FTS5 (`internal/database`)

- **Pure-Go SQLite (`modernc.org/sqlite`) ở chế độ `WAL`**: Cho phép đọc/ghi đồng thời tốc độ cao mà không cần cài đặt trình biên dịch C (CGO).
- **Bảng chỉ mục toàn văn `projects_fts` (FTS5)**:
  - Đánh chỉ mục tự động trên 5 trường: `name`, `slug`, `summary`, `tech_stack`, và `tags`.
  - Đồng bộ tức thì trong cùng một transaction mỗi khi thêm, sửa hoặc xóa dự án.
- **Bộ nhớ đệm trạng thái ổ đĩa (`enrichLocalFlags`)**:
  - Tránh lỗi N+1 disk I/O bằng cách lưu sẵn các cờ `has_local_dir`, `has_env`, `has_env_example`, `has_start_bat` và bộ đệm kiểm tra file ảnh thumbnail/icon trong bộ nhớ.

---

## 4. Trình Khởi chạy, Bảo vệ `.env` & Giám sát Thời gian thực (`internal/launcher`)

### 4.1. Pre-flight `.env` Safety Guard (`runner.go`)
- Trước khi thực thi bất kỳ Web App hoặc script `.bat` nào, `Runner` kiểm tra sự tồn tại của `.env`.
- Nếu thư mục có `.env.example` nhưng thiếu `.env`, T-Deck **tự động khởi tạo `.env` từ `.env.example`**, chuyển trạng thái dự án sang `missing_env`, mở ngay file `.env` trong Notepad và dừng lệnh chạy để bảo vệ ứng dụng khỏi crash do thiếu biến môi trường.

### 4.2. Trình tạo `start.bat` & Đồng bộ `.env.example` (`templates.go`, `env_parser.go`)
- Cung cấp 5 mẫu `start.bat` chuẩn hóa sẵn mã trang UTF-8 (`chcp 65001`) cho các stack phổ biến.
- Tính năng **Sync `.env.example`** phân tích cú pháp từng dòng trong `.env`, giữ nguyên cấu trúc comment và tên biến nhưng xóa sạch giá trị bí mật sau dấu `=`.

### 4.3. Giám sát Tiến trình, Cổng TCP & Sức khỏe Web (`process_monitor.go`, `health.go`)
- **Process & Port Monitor**: Theo dõi cây tiến trình con được khởi chạy từ T-Deck, tự động dò cổng TCP (`LISTENING`) của tiến trình đó để hiển thị trực tiếp lên nút **Dừng (:PORT)** trên thẻ dự án.
- **Concurrent Web Health & Version Drift Checker**: Kiểm tra song song toàn bộ Web targets, đo thời gian phản hồi (`latency_ms`), mã HTTP và đối chiếu phiên bản thực tế trên server (`remote_version`) với phiên bản mới nhất đã phát hành (`version_drift`).

---

## 5. Quản lý Phát hành Hybrid & Đồng bộ Cloudflare R2 (`internal/cloud`)

- **Mô hình Đa Đích (`project_targets` & `releases`)**:
  - Cho phép một dự án duy nhất quản lý song song cả bản Web, bản Desktop Portable (`.zip`) và bản Desktop Installer (`.exe`).
- **Tải lên Đa phần (Multipart Upload) & Luồng Sự kiện Thời gian thực**:
  - Tự động chia nhỏ gói phát hành lớn khi đẩy lên Cloudflare R2, phát sự kiện `release:progress` và `build:log` qua Wails Runtime Events tới ngăn kéo **ReleaseBuildDrawer** trên giao diện.
- **Cập nhật Nguyên tử (Atomic Update) & Rollback (`.bak`)**:
  - Kiểm tra mã băm `SHA-256` của gói tải về trước khi giải nén vào thư mục tạm (`staging`), đổi tên bản hiện tại sang `.bak` rồi mới kích hoạt bản mới. Nếu giải nén lỗi hoặc người dùng bấm **Rollback**, hệ thống khôi phục bản `.bak` tức thì.
- **Đồng bộ Danh mục 2 Chiều với Tombstones (`catalog_sync.go`)**:
  - Hợp nhất `catalog.json` v2 giữa máy cục bộ và Cloudflare R2 dựa trên `updated_at` và cờ xóa mềm `deleted_at` (ngăn hiện tượng dự án đã xóa bị kéo ngược trở lại khi sync).

