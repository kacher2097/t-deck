# Tài liệu Cấu hình, Biến môi trường & Chuẩn Manifest (Configuration Reference)

Tài liệu này quy định chi tiết toàn bộ hệ thống cấu hình của **T-Deck**, bao gồm: thư mục dữ liệu ngay tại thư mục cài đặt người dùng (`<InstallDir>`), danh sách biến môi trường `.env`, cấu trúc file neo danh tính `.tdeck-meta.json`, chuẩn manifest `.tdeck.json` tại từng dự án và cấu trúc lưu trữ đối tượng trên **Cloudflare R2**.

---

## 1. Cấu trúc Thư mục Dữ liệu Cục bộ (`<InstallDir>`)

Mọi dữ liệu vận hành của T-Deck được lưu trữ mặc định ngay trong **thư mục cài đặt của người dùng** (`<InstallDir>`, có thể ghi đè qua biến môi trường `TDECK_DATA_DIR`; chỉ fallback sang thư mục config hệ điều hành nếu thư mục cài đặt ở chế độ chỉ đọc). Nếu phát hiện dữ liệu cũ tại `%APPDATA%\TDeck`, T-Deck tự động di chuyển sang thư mục cài đặt mới ở lần chạy đầu tiên:

```text
<InstallDir>\
├── settings.json                 # Cấu hình chung (WorkspaceDirs, AppsDir, R2 non-secret settings)
├── .env                          # Lưu riêng khóa bí mật R2 (R2_ACCESS_KEY, R2_SECRET_KEY) với quyền 0600
├── tdeck.db                      # Cơ sở dữ liệu SQLite chính (chế độ WAL + FTS5)
├── tdeck.db-wal                  # Write-Ahead Log của SQLite
├── tdeck.db-shm                  # Shared Memory của SQLite
├── apps\                         # Thư mục mặc định giải nén các ứng dụng Desktop tải từ R2
└── thumbnails\
    ├── <project-uuid>.png        # Ảnh poster đại diện 16:9 (1280×720 PNG)
    └── <project-uuid>.icon.png   # Huy hiệu App Icon vuông (128×128 PNG, chỉ sinh khi dự án có bộ icon)
```

---

## 2. Biến môi trường (`.env`) & `settings.json`

T-Deck đọc cấu hình theo thứ tự ưu tiên: **Biến môi trường hệ thống / file `.env`** ghi đè lên giá trị trong **`settings.json`**.

| Biến môi trường | Khóa trong `settings.json` | Mặc định | Mô tả chi tiết |
|---|---|---|---|
| `TDECK_DATA_DIR` | *(Thư mục gốc dữ liệu)* | `<InstallDir>` | Tùy chọn ghi đè thư mục lưu `tdeck.db`, `settings.json`, `.env`, `thumbnails/`. |
| `TDECK_WORKSPACE_DIRS` | `workspace_dirs` | `[]` | Danh sách thư mục gốc để quét dự án (trong `.env` phân tách bằng `;` hoặc `,`). |
| `TDECK_APPS_DIR` | `apps_dir` | `<InstallDir>/apps` | Thư mục đích khi cài đặt các ứng dụng Desktop tải từ Cloudflare R2 về máy. |
| `TDECK_SCAN_RECENT_DAYS` | `scan_recent_days` | `120` | Chỉ quét các dự án có thay đổi trong $N$ ngày gần nhất (`0` = quét toàn bộ). |
| `TDECK_SCAN_EXCLUDE_DIRS` | `scan_exclude_dirs` | `[]` | Danh sách tên thư mục loại trừ bổ sung khi quét (phân tách bằng `,`). |
| `R2_ENABLED` | `r2.enabled` | `false` | Bật/tắt tính năng đồng bộ và phát hành qua Cloudflare R2 (`true` / `false`). |
| `R2_ACCOUNT_ID` | `r2.account_id` | `""` | Account ID của tài khoản Cloudflare. |
| `R2_ENDPOINT` | `r2.endpoint` | Tự suy từ Account ID | Endpoint S3 của R2 (`https://<account_id>.r2.cloudflarestorage.com`). |
| `R2_BUCKET` | `r2.bucket_name` | `my-storage` | Tên Bucket trên Cloudflare R2 dùng để lưu `catalog.json`, thumbnails và releases. |
| `R2_ACCESS_KEY` | *(Chỉ lưu ở `.env`)* | `""` | S3 Access Key ID của Cloudflare R2 API Token. |
| `R2_SECRET_KEY` | *(Chỉ lưu ở `.env`)* | `""` | S3 Secret Access Key của Cloudflare R2 API Token. |
| `R2_PUBLIC_DOMAIN` | `r2.public_domain` | `""` | Tên miền công khai gắn với Bucket (ví dụ: `https://cdn.example.com`) để tạo link tải nhanh. |

---

## 3. File Neo Danh tính Dự án (`.tdeck-meta.json`)

Khi T-Deck phát hiện hoặc khởi tạo một dự án, hệ thống tự động ghi file `.tdeck-meta.json` ở thư mục gốc của dự án đó:

```json
{
  "project_id": "4e132697-4120-415d-b779-a69f9004e5cb",
  "slug": "s3-desk",
  "type": "desktop"
}
```

- **`project_id`**: UUID v4 bất biến của dự án. Khi bạn di chuyển thư mục sang ổ đĩa khác và bấm **Scan Local**, T-Deck đọc `project_id` này để cập nhật đường dẫn mới mà không tạo dự án trùng.
- **`slug`**: Định danh ngắn gọn dạng kebab-case, dùng làm khóa đường dẫn khi đẩy bản phát hành lên R2 (`releases/<slug>/...`).
- **`type`**: Kiểu môi trường chính của dự án (`web`, `desktop`, hoặc `hybrid`).

---

## 4. Chuẩn Manifest Dự án (`.tdeck.json`)

Ngoài việc để T-Deck tự động suy luận cấu hình, bạn có thể đặt một file `.tdeck.json` tại thư mục gốc của bất kỳ dự án nào để khai báo tường minh thông tin dự án, cách đọc số phiên bản và danh sách các đích phát hành (**Targets**) cho cả giao diện GUI lẫn công cụ CLI `tdeck-sync`.

### 4.1. Ví dụ đầy đủ cho Dự án Hybrid (Có cả Web App lẫn Desktop App)

```json
{
  "schema_version": 2,
  "slug": "t-hub-suite",
  "name": "T-Hub Enterprise Suite",
  "category": "T-Hub Ecosystem",
  "summary": "Nền tảng quản trị trung tâm tích hợp cả cổng Web Dashboard và trạm Desktop Station.",
  "tech_stack": "Go, Wails, React, TypeScript, Tailwind CSS",
  "tags": ["desktop", "web", "go", "wails", "react", "typescript"],
  "version_source": {
    "strategy": "package_json",
    "file": "frontend/package.json"
  },
  "targets": [
    {
      "name": "Desktop Station (Portable)",
      "kind": "desktop",
      "distribution": "portable",
      "subpath": "apps/desktop-station",
      "build_cmd": ".\\build.bat full",
      "artifact_glob": "build/bin/*.exe",
      "launch_entry": "build/bin/THubStation.exe",
      "platform": "windows/amd64"
    },
    {
      "name": "Web Control Portal",
      "kind": "web",
      "distribution": "static_bundle",
      "subpath": "apps/web-portal",
      "build_cmd": "npm run build",
      "artifact_glob": "dist",
      "launch_entry": "start.bat",
      "web_url": "http://localhost:5173",
      "health_url": "http://localhost:5173/healthz",
      "version_url": "http://localhost:5173/version.json",
      "version_json_path": "version"
    }
  ]
}
```

### 4.2. Các chiến lược đọc phiên bản tự động (`version_source.strategy`)

| Chiến lược (`strategy`) | Ý nghĩa |
|---|---|
| `git_tag` | Đọc tag Git gần nhất của repo (`git describe --tags --abbrev=0`), nếu chưa có tag sẽ fallback về `0.1.0+<short-sha>`. |
| `package_json` | Đọc trường `"version"` từ file `package.json` (hoặc đường dẫn chỉ định trong `file`). |
| `wails_json` | Đọc trường `"info.productVersion"` từ file `wails.json`. |
| `file` | Đọc nội dung văn bản thuần từ một file phiên bản (ví dụ `VERSION`). |

---

## 5. Cấu trúc Lưu trữ trên Cloudflare R2

T-Deck tổ chức các đối tượng (Objects) trong Bucket Cloudflare R2 theo cấu trúc chuẩn sau:

```text
<r2-bucket>/
├── catalog.json                                          # Danh mục tổng hợp v2 (projects, categories, targets, latest releases)
├── thumbnails/
│   └── <project-uuid>.png                                # Ảnh preview 16:9 đồng bộ giữa các máy
└── releases/
    └── <project-slug>/
        ├── index.json                                    # Toàn bộ lịch sử phát hành của dự án <project-slug>
        ├── latest.json                                   # Feed JSON chuẩn cho Auto-Updater kiểm tra bản mới
        └── <target-id>/
            └── <version>/
                ├── <slug>-<version>.zip                  # Gói phát hành (Portable ZIP, Static Web Bundle, hoặc Installer EXE)
                └── sha256.txt                            # Mã kiểm tra toàn vẹn SHA-256
```

