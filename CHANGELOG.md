# Nhật ký Thay đổi (Changelog)

Mọi thay đổi đáng chú ý của dự án **T-Deck** đều được ghi nhận tại tài liệu này theo chuẩn [Keep a Changelog](https://keepachangelog.com/en/1.1.0/).

---

## [v2.2.0] - 2026-11-27

### Thêm mới (Added)
- **Lưu dữ liệu mặc định tại Thư mục Cài đặt của Người dùng (`<InstallDir>`)**:
  - Chuyển toàn bộ vị trí lưu mặc định của `tdeck.db`, `settings.json`, `.env`, `thumbnails/` và `apps/` từ ổ `C:` (`%APPDATA%\TDeck` & `C:\TDeckApps`) sang ngay trong thư mục cài đặt mà người dùng chọn (`ResolveDefaultDataDir` trong `internal/config/settings.go`).
  - Tự động di chuyển dữ liệu cũ từ `%APPDATA%\TDeck` sang thư mục cài đặt mới ở lần khởi chạy đầu tiên nếu chưa có `tdeck.db`, đồng thời hỗ trợ ghi đè qua biến `TDECK_DATA_DIR`.
- **Bộ cài đặt Chuẩn cho Windows & Linux + Gói Phát hành Đa Hệ điều hành**:
  - **Windows NSIS Installer (`build/windows/installer/project.nsi`)**: Lưu chuẩn UTF-8 BOM + `MUI_LANGUAGE "Vietnamese"` không lỗi font tiếng Việt, đầy đủ checkbox lựa chọn (Desktop Shortcut, Start Menu, Khởi động cùng Windows, Thêm vào PATH, Khởi chạy ngay / Mở thư mục cài đặt) và hỏi giữ lại dữ liệu khi gỡ cài đặt.
  - **Linux Installer (`build/linux/install.sh`, `uninstall.sh`, `t-deck.desktop`)**: Hỗ trợ hộp thoại đồ họa (`zenity` / `whiptail`) và CLI UTF-8 tiếng Việt với đầy đủ checkbox tạo shortcut Desktop, Menu Ứng dụng, `~/.local/bin`, Autostart.
  - **Release Packaging (`scripts/package-releases.ps1`, `.github/workflows/release.yml`, `build.bat release`)**: Xuất đồng thời `TDeck-windows-amd64-installer.exe`, `TDeck-windows-amd64-portable.zip`, `TDeck-linux-amd64-installer.run`, `TDeck-linux-amd64.tar.gz` kèm `SHA256SUMS.txt`.
- **Hỗ trợ Đa nền tảng trong Launcher & Process Monitor**:
  - Bổ sung nhánh chạy lệnh build (`sh -c`), mở thư mục/IDE/Terminal (`xdg-open`, `gnome-terminal`/`konsole`/`xterm`), chạy script `.sh` và dừng tiến trình trên Linux.

---

## [v2.1.0] - 2026-11-27

### Thêm mới (Added)
- **Tự động Nhận diện Bộ Icon Thương hiệu (`internal/media/brand_icon.go`)**:
  - Tự động dò tìm bộ icon vuông chuẩn (`48px–2048px`) ở định dạng `.png` và `.ico` (hỗ trợ cả frame PNG nhúng và frame 32-bit BGRA BMP/DIB) tại thư mục gốc và các thư mục con của dự án khi bấm **Scan Local**.
  - Tự động kết xuất file `<id>.icon.png` (`128×128`) và cấp trường `icon_url` để hiển thị huy hiệu **App Icon** trực tiếp cạnh tên dự án trên thẻ Card (`ProjectCard`), dòng bảng ma trận (`ReleaseMatrixRow`) và tiêu đề cửa sổ chi tiết (`ProjectDetailModal`).
  - Tích hợp logo dự án vào poster 16:9 (`1280×720`): gắn khung icon bo góc kèm nhãn `APP ICON` ở góc trên bên trái và đặt tấm kính tối (Hero Emblem `288×288`) ở trung tâm bảng schematic bên phải. Giữ nguyên 100% thiết kế gốc đối với các dự án chưa có bộ icon.
- **Trải nghiệm Quét Dự án (`ScanLoadingOverlay` & Staggered Skeleton UX)**:
  - Bổ sung hộp thoại tiến trình quét nổi bật ở chính giữa màn hình khi bấm **Scan Local**.
  - Hiển thị lưới thẻ khung xương (Skeleton Cards) và hiệu ứng xuất hiện tuần tự từng thẻ dự án sau khi quét xong.
- **Hệ thống Tài liệu Open-Source Hoàn chỉnh**:
  - Bổ sung đầy đủ [README.md](file:///d:/app/t-deck/README.md), [docs/USER_GUIDE.md](file:///d:/app/t-deck/docs/USER_GUIDE.md), [docs/FEATURES.md](file:///d:/app/t-deck/docs/FEATURES.md), [docs/CONFIGURATION.md](file:///d:/app/t-deck/docs/CONFIGURATION.md), [docs/CLI.md](file:///d:/app/t-deck/docs/CLI.md), [CONTRIBUTING.md](file:///d:/app/t-deck/CONTRIBUTING.md), [SECURITY.md](file:///d:/app/t-deck/SECURITY.md) và [CHANGELOG.md](file:///d:/app/t-deck/CHANGELOG.md).

### Cải tiến & Sửa lỗi (Changed & Fixed)
- **Nâng cấp Độ chính xác của Bộ Quét (`internal/scanner`)**:
  - Mở rộng khả năng khám phá sâu bên trong các thư mục gom nhóm (`D:\app`, `D:\tool-github`, `D:\projects`...) để không bỏ sót dự án thực tế.
  - Bổ sung bộ lọc đa tầng loại trừ triệt để các phần mềm cài sẵn của bên thứ ba (`TablePlus`, `PostgreSQL`, `Proxifier`, `Xshell`, `Xftp`, `Go` SDK, `Browser`, `uv`...) và các thư mục bản build tạm.
- **Tái cấu trúc & Mô-đun hóa Giao diện Frontend (`frontend/src`)**:
  - Rút gọn hơn 55% dung lượng `App.tsx` bằng cách tách các custom hooks chuyên biệt (`useProjectFilters`, `useBulkSelection`, `useRuntimeEvents`, `useReleaseBuildJob`) và các component theo miền (`CatalogToolbar`, `CatalogEmptyState`, `ProjectCardPreview`, `ProjectPrimaryActionButton`, `ReleaseMatrixRow`, `SidebarCategorySection`, `InspectorMetadataForm`, `TargetEditorForm`).
- **Tối ưu Hiển thị Cửa sổ Khi Khởi động (`main.go` & `SidebarNav.tsx`)**:
  - Thiết lập `WindowStartState: options.Maximised` và kích thước khôi phục an toàn (`1280×760`) giúp cửa sổ mới mở không bị tràn hoặc lấp thanh tác vụ trên màn hình Windows đặt tỷ lệ phóng 125%–150% DPI.

---

## [v2.0.0] - 2026-11-26

### Thêm mới (Added)
- **Quản lý Phát hành Hybrid (Web + Desktop Targets)**:
  - Bổ sung bảng `project_targets` và `releases` trong SQLite (`schema v3`), cho phép một dự án quản lý đồng thời cả bản Web, bản Desktop Portable (`.zip`) và bản Desktop Installer (`.exe`).
  - Hỗ trợ kiểm tra toàn vẹn `SHA-256`, cập nhật nguyên tử (Atomic Update), khôi phục bản cũ (`.bak` Rollback), tạo Presigned Download URL và xuất feed `latest.json` cho Auto-Updater.
- **Chế độ Bảng Ma trận Phát hành (`Release Matrix View`) & Ngăn kéo Build trực tiếp (`ReleaseBuildDrawer`)**:
  - Chuyển đổi tức thì giữa chế độ Lưới Card và Bảng Ma trận Phát hành.
  - Tích hợp chạy lệnh build có luồng log thời gian thực (`build:log`), thanh tiến trình tải lên đa phần (`release:progress`) và tự động gợi ý Changelog từ lịch sử Git.
- **Giám sát Tiến trình, Cổng TCP & Sức khỏe Web (`Process & Health Monitor`)**:
  - Theo dõi PID và cổng TCP đang mở của các dự án đang chạy, hỗ trợ nút dừng trực tiếp trên thẻ dự án.
  - Kiểm tra đồng thời trạng thái HTTP và độ lệch phiên bản (`version_drift`) của toàn bộ Web targets.
- **Chuẩn Manifest `.tdeck.json`**:
  - Đồng nhất cấu hình giữa giao diện GUI và công cụ CLI `cmd/tdeck-sync`.

---

## [v1.0.0] - 2026-11-20

### Thêm mới (Added)
- Phiên bản nền tảng của **T-Deck** trên Wails v2 + React 19 + Pure-Go SQLite (`WAL` + `FTS5`).
- Quét thư mục dự án cục bộ, neo danh tính bằng `.tdeck-meta.json`, kiểm tra an toàn `.env` trước khi chạy `.bat`/`.exe`, kết xuất poster 16:9 thuần Go và đồng bộ danh mục cơ bản với Cloudflare R2.

