# Hướng dẫn Sử dụng CLI (`tdeck-sync` & `build.bat`)

Bên cạnh giao diện đồ họa `TDeck.exe`, dự án cung cấp 2 công cụ dòng lệnh giúp tự động hóa hoàn toàn quy trình đóng gói, quét dự án hàng loạt, kết xuất poster 16:9 và đẩy bản phát hành lên **Cloudflare R2** trong môi trường CI/CD hoặc script định kỳ.

---

## 1. Công cụ Đồng bộ & Phát hành Hàng loạt (`cmd/tdeck-sync`)

`tdeck-sync` sử dụng chung cơ sở dữ liệu SQLite (`<InstallDir>\tdeck.db`), cấu hình `settings.json` / `.env` và bộ quét `internal/scanner` với ứng dụng GUI.

### 1.1. Cú pháp & Danh sách Cờ (Flags)

```powershell
go run ./cmd/tdeck-sync [flags]
```

| Cờ (Flag) | Kiểu | Mặc định | Ý nghĩa |
|---|---|---|---|
| `--dry-run` | `bool` | `false` | Chỉ chạy quét ổ đĩa, tự động điền mô tả/nhóm, tạo poster 16:9 ở máy cục bộ và in bảng tổng kết mà **không** kết nối hay tải lên Cloudflare R2. |
| `--slug` | `string` | `""` *(tất cả)* | Chỉ xử lý một dự án duy nhất có `slug` khớp với giá trị truyền vào (ví dụ: `--slug s3-desk`). |
| `--target` | `string` | `"all"` | Lọc loại đích phát hành cần đẩy lên R2: `all`, `desktop`, hoặc `web`. |
| `--channel` | `string` | `"stable"` | Kênh phát hành cho bản build lần này: `stable`, `beta`, hoặc `canary`. |
| `--publish-desktop` | `bool` | `true` | Tự động đóng gói và đẩy các Desktop Targets lên Cloudflare R2. |
| `--publish-web` | `bool` | `false` | Tự động đóng gói và đẩy các Web Static Bundle Targets lên Cloudflare R2. |
| `--force-publish` | `bool` | `false` | Bắt buộc đóng gói và đẩy lại lên R2 ngay cả khi số phiên bản (`version`) trùng với bản mới nhất đã phát hành. |
| `--build` | `bool` | `false` | Thực thi lệnh `build_cmd` của từng Target trước khi đóng gói và tải lên R2. |
| `--gen-thumbs` | `bool` | `true` | Tự động kết xuất ảnh poster `1280×720` cho các dự án chưa có ảnh đại diện. |
| `--force-thumbs` | `bool` | `false` | Vẽ lại toàn bộ ảnh poster `1280×720` cho tất cả dự án khớp bộ lọc. |

### 1.2. Các Kịch bản Sử dụng Thực tế (Recipes)

#### Kịch bản 1: Quét thử nghiệm và xem bảng tổng kết danh mục tại máy (Không đẩy R2)
```powershell
go run ./cmd/tdeck-sync --dry-run
```

#### Kịch bản 2: Vẽ lại poster cho một dự án cụ thể mà không tải lên đám mây
```powershell
go run ./cmd/tdeck-sync --dry-run --slug bizmastery --force-thumbs
```

#### Kịch bản 3: Tự động chạy Build, đóng gói và phát hành một dự án lên kênh `beta`
```powershell
go run ./cmd/tdeck-sync --slug s3-desk --build --channel beta --force-publish
```

#### Kịch bản 4: Đồng bộ toàn bộ danh mục và phát hành tất cả ứng dụng Desktop có phiên bản mới lên R2
```powershell
go run ./cmd/tdeck-sync --publish-desktop=true --publish-web=false
```

---

## 2. Script Đóng gói Nhanh (`build.bat`)

Script [build.bat](file:///d:/app/t-deck/build.bat) ở thư mục gốc giúp biên dịch Frontend (TypeScript + Vite) và đóng gói ứng dụng Wails (`TDeck.exe` hoặc bộ cài NSIS) chỉ với một lệnh duy nhất. Tự động kiểm tra và đóng tiến trình `TDeck.exe` đang chạy trước khi ghi đè file thực thi mới.

### 2.1. Cú pháp

```powershell
.\build.bat [fast | full | clean | fe | installer] [run]
```

### 2.2. Chi tiết các Chế độ Build

| Lệnh | Mô tả quy trình thực hiện |
|---|---|
| `.\build.bat` | Mở menu tương tác bằng tiếng Việt trên cửa sổ dòng lệnh để chọn chế độ `1..5`. |
| `.\build.bat fast` | **Build nhanh Backend Go**: Bỏ qua bước build Frontend (nếu đã có `frontend\dist\index.html`), tắt `TDeck.exe` đang chạy và đóng gói lại `build\bin\TDeck.exe`. |
| `.\build.bat full` | **Build đầy đủ (Khuyên dùng)**: Chạy `tsc && vite build` trong `frontend/`, tắt `TDeck.exe` đang chạy và đóng gói `build\bin\TDeck.exe`. |
| `.\build.bat full run` | Thực hiện `full` build và **tự động mở ngay `TDeck.exe`** sau khi đóng gói thành công. |
| `.\build.bat clean` | **Clean Build toàn diện**: Chạy lại `npm install`, build mới Frontend và gọi `wails build -clean` để xóa sạch bộ đệm build cũ. |
| `.\build.bat fe` | **Chỉ kiểm tra & build Frontend**: Chạy `tsc && vite build` để kiểm tra lỗi kiểu TypeScript và xuất gói tĩnh ra `frontend\dist`. |
| `.\build.bat installer` | **Đóng gói Bộ cài đặt Windows (NSIS)**: Build Frontend + Go và gọi `makensis.exe` để xuất bộ cài `build\bin\TDeck-amd64-installer.exe`. |
| `.\build.bat release` | **Đóng gói Phát hành Đa nền tảng**: Xuất trọn bộ Windows Installer (`.exe`), Windows Portable (`.zip`), Linux Installer (`.run`), Linux Tarball (`.tar.gz`) và `SHA256SUMS.txt` vào `build\release\`. |

