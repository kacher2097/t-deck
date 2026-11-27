# Hướng dẫn Sử dụng T-Deck (User Guide)

Tài liệu này hướng dẫn chi tiết cách thiết lập, vận hành và khai thác toàn bộ tính năng của **T-Deck** trong công việc hằng ngày: từ quét thư mục dự án trên máy, tìm kiếm, khởi chạy ứng dụng, quản lý biến môi trường `.env`, đến phát hành và đồng bộ phần mềm qua **Cloudflare R2**.

---

## Mục lục

1. [Khởi chạy lần đầu & Cấu hình hệ thống](#1-khởi-chạy-lần-đầu--cấu-hình-hệ-thống)
2. [Quét dự án cục bộ (Scan Local)](#2-quét-dự-án-cục-bộ-scan-local)
3. [Nhận diện Thương hiệu & Ảnh đại diện (Brand Icons & Posters)](#3-nhận-diện-thương-hiệu--ảnh-đại-diện-brand-icons--posters)
4. [Tìm kiếm, Lọc đa chiều & Chế độ hiển thị](#4-tìm-kiếm-lọc-đa-chiều--chế-độ-hiển-thị)
5. [Khởi chạy Dự án & Quản lý Tiến trình](#5-khởi-chạy-dự-án--quản-lý-tiến-trình)
6. [Quản lý `.env`, Tạo `start.bat` & Tiện ích Máy trạm](#6-quản-lý-env-tạo-startbat--tiện-ích-máy-trạm)
7. [Phát hành & Quản lý Phiên bản qua Cloudflare R2](#7-phát-hành--quản-lý-phiên-bản-qua-cloudflare-r2)
8. [Đồng bộ Đa máy & Cài đặt Dự án sang Máy mới](#8-đồng-bộ-đa-máy--cài-đặt-dự-án-sang-máy-mới)
9. [Bảng phím tắt & Mẹo thao tác nhanh](#9-bảng-phím-tắt--mẹo-thao-tác-nhanh)
10. [Xử lý sự cố thường gặp (Troubleshooting)](#10-xử-lý-sự-cố-thường-gặp-troubleshooting)

---

## 1. Khởi chạy lần đầu & Cấu hình hệ thống

Khi mở `TDeck.exe`, cửa sổ ứng dụng tự động khởi tạo ở chế độ **Toàn màn hình (Maximised)** để hiển thị trọn vẹn thanh lọc bên trái và lưới thẻ dự án trên mọi độ phân giải màn hình (kể cả màn hình laptop đặt tỷ lệ phóng 125%–150% DPI).

Bấm vào biểu tượng **Cài đặt (⚙️)** ở góc trên cùng bên phải để mở hộp thoại thiết lập:

### 1.1. Cấu hình Không gian làm việc (Workspace Settings)
- **Thư mục Workspace (`WorkspaceDirs`)**: Nhập một hoặc nhiều đường dẫn gốc chứa mã nguồn của bạn (ví dụ: `D:\`, `D:\app`, `E:\Workspaces`). Bạn có thể bấm nút **Chọn thư mục** để duyệt trực tiếp từ Windows Explorer.
- **Thư mục cài đặt Apps (`AppsDir`)**: Đường dẫn mặc định nơi T-Deck giải nén các ứng dụng Desktop tải từ Cloudflare R2 về máy (mặc định: `<InstallDir>\apps` ngay trong thư mục cài đặt).
- **Giới hạn ngày hoạt động (`ScanRecentDays`)**: Chỉ quét các dự án mới có thay đổi trong $N$ ngày gần đây (đặt `0` nếu muốn quét toàn bộ không giới hạn thời gian).
- **Thư mục loại trừ bổ sung (`ScanExcludeDirs`)**: Danh sách tên thư mục bạn muốn bộ quét bỏ qua (phân tách bằng dấu phẩy).

### 1.2. Cấu hình Cloudflare R2 (Tùy chọn)
Nếu bạn muốn đồng bộ danh mục và lưu trữ bản cài đặt giữa nhiều máy tính:
1. Bật công tắc **Kích hoạt đồng bộ Cloudflare R2**.
2. Điền `Account ID`, `Bucket Name`, `Access Key ID`, `Secret Access Key` và `Public Domain` (nếu bucket có gắn custom domain công khai).
3. Bấm **Kiểm tra kết nối R2** để xác nhận quyền đọc/ghi vào bucket trước khi bấm **Lưu cài đặt**.

> [!IMPORTANT]
> Khi lưu cài đặt, T-Deck tự động tách riêng `Access Key ID` và `Secret Access Key` vào file `<InstallDir>\.env` (quyền truy cập riêng tư) và xóa trắng hai trường này trong `settings.json` để tránh lộ khóa bí mật.

---

## 2. Quét dự án cục bộ (Scan Local)

Để đưa các dự án đang có trên ổ cứng vào T-Deck:

1. Bấm nút **Scan Local** trên thanh điều khiển trên cùng.
2. Một lớp phủ tiến trình (**Đang quét Workspace cục bộ...**) xuất hiện ở giữa màn hình, đồng thời lưới dự án chuyển sang hiệu ứng khung xương (**Skeleton Cards**) và hiển thị dần từng dự án ngay khi quét xong.
3. Trong một lần quét, T-Deck thực hiện tự động toàn bộ chuỗi tác vụ sau:
   - **Khám phá thư mục dự án**: Duyệt các thư mục con trong `WorkspaceDirs` (bao gồm cả các thư mục gom nhóm như `D:\app`, `D:\projects`, `D:\tool-github`...).
   - **Lọc bỏ phần mềm bên thứ ba**: Tự động loại trừ các ứng dụng cài sẵn hoặc runtime không phải dự án của bạn (như `Go`, `PostgreSQL`, `TablePlus`, `Xshell`, `Xftp`, `Proxifier`, `Browser`, `uv`...).
   - **Phân tích công nghệ & siêu dữ liệu**: Tự nhận diện loại dự án (`Web`, `Desktop`, `Hybrid`), ngôn ngữ, framework, lệnh chạy (`.bat` hoặc `.exe`), nhánh Git, remote URL và tóm tắt nội dung từ `README.md`.
   - **Ghi neo danh tính `.tdeck-meta.json`**: Lưu một file nhỏ `.tdeck-meta.json` chứa UUID của dự án tại thư mục gốc để lần quét sau nhận diện tức thì dù thư mục bị đổi chỗ.
   - **Kết xuất bộ nhận diện hình ảnh**: Tự động tạo poster 16:9 và trích xuất icon thương hiệu cho từng dự án.

---

## 3. Nhận diện Thương hiệu & Ảnh đại diện (Brand Icons & Posters)

### 3.1. Tự động trích xuất bộ Icon thương hiệu khi bấm `Scan Local`
Khi quét dự án, T-Deck tự động tìm kiếm bộ icon vuông chuẩn (`48px` đến `2048px`) ở định dạng `.png` hoặc `.ico` tại các vị trí tiêu chuẩn của dự án (`build/appicon.png`, `build/windows/icon.ico`, `public/apple-touch-icon.png`, `public/favicon/apple-touch-icon.png`, `web/public/icon-512.png`, `assets/icon.png`, `extension/icons/icon128.png`, `icons/icon128.png`... kể cả trong các thư mục con `apps/*/` hoặc `*-desktop/`).

- **Đối với dự án có sẵn bộ icon nhận diện**:
  - **Huy hiệu App Icon (`icon_url`)**: Hiển thị trực tiếp biểu tượng ứng dụng bo góc cạnh tên dự án trên thẻ Card (`20×20`), trên dòng bảng Release Matrix (`16×16`) và trên thanh tiêu đề cửa sổ Chi tiết (`24×24`).
  - **Poster 16:9 có gắn thương hiệu**: Kết xuất biểu tượng dự án kèm nhãn `APP ICON` ở góc trên bên trái poster và đặt một khung kính tối (Hero Emblem `288×288`) mang logo dự án ở chính giữa bảng schematic bên phải.
- **Đối với dự án không có bộ icon**:
  - Giữ nguyên hoàn toàn giao diện tiêu chuẩn và poster kỹ thuật 16:9 được vẽ tự động theo nhóm ngành nghề (AI Agent, Video/Media, FinOps, Cloud Storage, DevTools...) mà không làm xáo trộn bố cục.

### 3.2. Thay đổi hoặc tùy chỉnh ảnh đại diện thủ công
Bạn có thể thay ảnh preview 16:9 của bất kỳ dự án nào theo 4 cách:
1. **Kéo & Thả (Drag & Drop)**: Kéo một file ảnh (`.png`, `.jpg`, `.webp`) từ máy tính thả trực tiếp vào khung ảnh trên thẻ dự án.
2. **Dán từ Clipboard (`Ctrl+V`)**: Chụp màn hình ứng dụng (`Win+Shift+S`), click vào khung ảnh của thẻ dự án và nhấn `Ctrl+V`.
3. **Chọn file từ máy**: Rê chuột vào góc dưới phải của khung ảnh và bấm nút **Tải ảnh** (hoặc nút **Tải ảnh lên** trong cửa sổ Chi tiết).
4. **Tạo lại poster tự động**: Mở cửa sổ Chi tiết dự án → bấm nút **✨ Tạo theo mô tả** để T-Deck vẽ lại poster 16:9 dựa trên mô tả, tags và bộ icon hiện có trong thư mục dự án.

---

## 4. Tìm kiếm, Lọc đa chiều & Chế độ hiển thị

### 4.1. Tìm kiếm toàn văn FTS5 (`Ctrl+K`)
- Nhấn `Ctrl+K` để đưa con trỏ vào thanh tìm kiếm.
- Nhập từ khóa bất kỳ: tên dự án (`bizmastery`), chức năng (`dịch video`, `telegram`), công nghệ (`wails`, `spring-boot`) hoặc `#tag`.
- Cơ chế FTS5 của SQLite hỗ trợ khớp tiền tố (prefix matching) theo thời gian thực ngay khi bạn gõ.

### 4.2. Bộ lọc đa chiều ở thanh bên trái (Sidebar)
Các bộ lọc ở thanh bên trái có thể **kết hợp đồng thời** với nhau:
- **Phân loại (Nhóm)**: Lọc theo nhóm nghiệp vụ (`Business & Web Apps`, `Video & Media AI`, `DevTools & Automation`, `Java & Enterprise`, `T-Hub Ecosystem`...) hoặc bấm **+ Thêm** để tạo nhóm mới. Rê chuột vào tên nhóm để hiện nút xóa nhóm (kèm hộp xác nhận an toàn).
- **Môi trường**: Lọc nhanh `Tất cả`, `Web Apps`, `Desktop Apps`, hoặc `Hybrid (Web + EXE)`.
- **Trạng thái cục bộ**:
  - 🟢 **Sẵn sàng (`ready`)**: Đã có đầy đủ mã nguồn/file chạy và cấu hình `.env`.
  - ⚪ **Chưa tải (`cloud_only`)**: Dự án được đồng bộ từ R2 về nhưng chưa tải xuống máy này.
  - 🟠 **Thiếu `.env` (`missing_env`)**: Có file `.env.example` nhưng chưa tạo file `.env`.
  - 🔴 **Thiếu thư mục (`missing_dir`)**: Thư mục từng quét đã bị xóa hoặc ổ đĩa rời chưa cắm vào máy.
- **Release & Chạy**: Lọc các dự án `Có Release` trên R2, `Cần cập nhật` (bản trên máy cũ hơn bản trên R2), hoặc `Đang chạy` (có tiến trình đang hoạt động).
- **Tags**: Click vào một hoặc nhiều `#tag` để lọc giao nhau (AND filter).

### 4.3. Sắp xếp & Chuyển đổi chế độ xem
- **8 chế độ sắp xếp**: Tên dự án (`A → Z`, `Z → A`), Cập nhật gần nhất, Mở chạy gần đây, Sẵn sàng chạy trước, Theo phân loại (Nhóm), Theo môi trường (`Web / EXE`), hoặc Độ khớp tìm kiếm.
- **2 chế độ hiển thị** (chuyển đổi bằng 2 nút biểu tượng Lưới / Bảng bên trái nút `Health`):
  - **Lưới Card (Grid View)**: Hiển thị trực quan poster 16:9, mô tả, tags và các nút thao tác nhanh.
  - **Bảng Ma trận Phát hành (Release Matrix View)**: Hiển thị dạng bảng cô đọng gồm Tên dự án (kèm App Icon), Môi trường, Trạng thái, Phiên bản đang cài vs Phiên bản R2 mới nhất, Trạng thái Web Live/Offline + cổng đang mở, và cụm nút hành động.

### 4.4. Chọn nhiều & Xóa danh mục hàng loạt
- Bấm nút **Chọn nhiều** trên thanh công cụ danh sách → tích chọn các thẻ dự án (hoặc bấm **Chọn tất cả**) → bấm **Xóa đã chọn**.
- **An toàn dữ liệu tuyệt đối**: Việc xóa dự án trên giao diện T-Deck **chỉ gỡ dự án khỏi danh mục quản lý của T-Deck**, tuyệt đối **không** xóa thư mục mã nguồn hay file gốc trên ổ cứng của bạn.

---

## 5. Khởi chạy Dự án & Quản lý Tiến trình

### 5.1. Nút hành động thông minh trên từng thẻ
Tùy theo trạng thái thực tế của dự án trên máy, nút chính ở góc dưới bên trái mỗi thẻ sẽ tự động chuyển đổi:

| Trạng thái dự án | Nhãn nút chính | Hành vi khi bấm |
|---|---|---|
| Desktop App đã sẵn sàng | **▶ Launch** | Khởi chạy file `.exe` của ứng dụng ở chế độ độc lập. |
| Web App / Script đã sẵn sàng | **▶ Run .bat** | Mở cửa sổ dòng lệnh thực thi script khởi chạy (`start.bat`, `run.bat`...). |
| Thiếu file `.env` | **⚠️ Cấu hình .env** | Tự động copy `.env.example` → `.env` (nếu có) và mở ngay trong Notepad để bạn điền khóa cấu hình. |
| Chỉ có trên Cloud (Desktop) | **☁️ Install R2** | Tải gói `.zip`/`.exe` mới nhất từ Cloudflare R2 và giải nén vào `AppsDir`. |
| Chỉ có trên Cloud (Web) | **⬇️ Git Clone** | Tự động chạy `git clone` từ `Git Remote URL` về thư mục Workspace của máy. |
| Đang chạy tiến trình | **■ Dừng (:PORT)** | Hiển thị cổng TCP đang lắng nghe (ví dụ `:5173`) và cho phép tắt sạch cây tiến trình chỉ với 1 cú click. |

### 5.2. Kiểm tra sức khỏe toàn bộ Web Targets (`Health`)
- Bấm nút **Health** trên thanh công cụ trên cùng để T-Deck gửi truy vấn kiểm tra đồng thời tới tất cả địa chỉ Web URL đã khai báo của các dự án.
- Kết quả hiển thị trực tiếp bằng chấm trạng thái (`LIVE` / `DOWN` / `DRIFT`), mã phản hồi HTTP, độ trễ (`ms`) và phiên bản đang chạy thực tế trên môi trường Web.

---

## 6. Quản lý `.env`, Tạo `start.bat` & Tiện ích Máy trạm

Bấm vào nút **Chi tiết & Cấu hình** trên thẻ dự án để mở cửa sổ **Project Detail Modal** → tab **Cấu hình & Hệ thống**:

1. **Tiện ích tại máy (1-Click Launchers)**:
   - **Explorer**: Mở ngay thư mục gốc của dự án trong Windows File Explorer.
   - **VS Code**: Mở thư mục dự án trong Visual Studio Code (`code .`).
   - **Terminal**: Mở Windows Terminal (`wt.exe`) hoặc Command Prompt ngay tại thư mục dự án.
   - Ba đèn chỉ báo trạng thái trực quan cho `.env`, `.env.example` và `start.bat` (bấm vào từng ô để mở file đó trong Notepad).
2. **Tạo `start.bat` / `run.bat` mẫu tự động**:
   - Chọn mẫu phù hợp với công nghệ của dự án từ danh sách thả xuống (`Node / Vite (npm run dev)`, `Python VirtualEnv`, `Go / Wails Dev`, `Docker Compose Up`, `Custom Script`) → bấm **Tạo file**.
3. **Đồng bộ `.env.example` an toàn (`Sync .env`)**:
   - Khi bạn thêm biến mới vào file `.env`, bấm nút **Sync .env** để T-Deck tự động sinh/cập nhật file `.env.example` với toàn bộ tên biến giữ nguyên nhưng **xóa sạch mọi giá trị bí mật sau dấu `=`**.
4. **Nhận diện lại từ thư mục (`✨ Nhận diện từ thư mục`)**:
   - Nếu dự án vừa thay đổi cấu trúc, thêm thư viện mới hoặc cập nhật `README.md`, bấm **Nhận diện từ thư mục** để cập nhật lại Tech Stack, Tags và đường dẫn file thực thi.

---

## 7. Phát hành & Quản lý Phiên bản qua Cloudflare R2

Chuyển sang tab **Phát hành & Targets (R2)** trong cửa sổ Chi tiết dự án để quản lý vòng đời phát hành cho cả ứng dụng Web lẫn Desktop:

### 7.1. Quản lý Đích phát hành (Targets)
Một dự án có thể có nhiều Targets (ví dụ: `Web Production`, `Windows Portable ZIP`, `Windows NSIS Installer`):
- Bấm **+ Thêm Target** để khai báo:
  - **Loại Target**: `Desktop App` hoặc `Web App`.
  - **Thư mục con (`Subpath`)**: Dành cho dự án Monorepo (ví dụ: `apps/desktop-station` hoặc `web`).
  - **Lệnh Build (`Build Command`)**: Ví dụ `.\build.bat full` hoặc `npm run build`.
  - **Mẫu đường dẫn Artifact (`Artifact Glob`)**: Ví dụ `build/bin/*.exe` hoặc `dist`.
  - **URL Web & Health/Version Endpoint** (đối với Web Target).

### 7.2. Build trực tiếp & Đẩy bản phát hành lên R2
1. Điền số phiên bản (ví dụ `v1.2.0`), chọn kênh phát hành (`stable`, `beta`, `canary`).
2. Bấm **Tự tạo từ Git** để T-Deck tự động trích xuất danh sách commit gần nhất làm nội dung **Changelog**.
3. Nếu cần biên dịch mới trước khi tải lên, bấm **Chạy Build** để mở ngăn kéo **Build & Release Drawer** xem trực tiếp luồng log `stdout`/`stderr` theo thời gian thực.
4. Bấm **Đóng gói & Publish R2**:
   - T-Deck tự động nén thư mục/file đích thành gói `.zip` (hoặc giữ nguyên bộ cài `.exe`), tính mã băm **SHA-256**, tải lên Cloudflare R2 qua cơ chế **Multipart Upload** kèm thanh tiến trình trực tiếp.
   - Tự động cập nhật `releases/<slug>/index.json`, `latest.json` và đồng bộ `catalog.json`.

### 7.3. Chia sẻ Link tải & Khôi phục phiên bản (Rollback)
Trong bảng **Lịch sử các bản phát hành**:
- **Copy Link / Presigned URL**: Tạo đường dẫn tải trực tiếp (có thời hạn hiệu lực chữ ký S3 hoặc qua Public Domain) để gửi cho đồng nghiệp/khách hàng.
- **Cài bản này (Update/Downgrade)**: Tải phiên bản bất kỳ trong lịch sử về thay thế bản hiện tại; bản cũ được sao lưu nguyên tử sang thư mục `.bak`.
- **Rollback `.bak`**: Khôi phục tức thì phiên bản trước đó nếu bản mới cập nhật gặp sự cố.

---

## 8. Đồng bộ Đa máy & Cài đặt Dự án sang Máy mới

Khi làm việc trên một máy tính mới:
1. Cài đặt `TDeck.exe`, mở **Cài đặt (⚙️)** và nhập thông tin kết nối tới cùng Bucket **Cloudflare R2**.
2. Bấm nút **Sync R2** trên thanh công cụ.
3. Toàn bộ danh sách dự án, nhóm phân loại, lịch sử phiên bản và ảnh poster sẽ được kéo về máy mới với trạng thái **Chưa tải (Cloud)**.
4. Bấm **Install R2** trên các ứng dụng Desktop để tự động tải và giải nén về máy, hoặc bấm **Git Clone** trên các dự án Web để kéo mã nguồn về thư mục Workspace.

---

## 9. Bảng phím tắt & Mẹo thao tác nhanh

| Phím tắt / Thao tác | Tác dụng |
|---|---|
| `Ctrl + K` | Đưa con trỏ vào ô tìm kiếm dự án ngay lập tức. |
| `Esc` | Đóng hộp thoại (Modal) hoặc bảng xác nhận đang mở. |
| `Ctrl + V` *(trên khung ảnh Card)* | Dán ảnh trực tiếp từ Clipboard làm ảnh đại diện cho dự án đó. |
| **Kéo & thả ảnh vào Card** | Cập nhật nhanh ảnh preview 16:9 từ file ảnh bất kỳ. |
| **Click vào `#tag` trên Card** | Bật/tắt bộ lọc nhanh theo tag đó. |
| **Click vào Nhãn Nhóm trên Card** | Lọc nhanh toàn bộ dự án thuộc nhóm phân loại đó. |

---

## 10. Xử lý sự cố thường gặp (Troubleshooting)

### Hỏi: Sau khi bấm `Scan Local`, một số dự án không xuất hiện?
- Kiểm tra xem thư mục cha của dự án đã được thêm vào **Thư mục Workspace (`WorkspaceDirs`)** trong **Cài đặt (⚙️)** chưa.
- Nếu dự án đã lâu không chỉnh sửa, hãy tăng giá trị **Giới hạn ngày hoạt động (`ScanRecentDays`)** hoặc đặt về `0` (quét không giới hạn thời gian).
- Kiểm tra xem tên thư mục dự án có nằm trong danh sách **Thư mục loại trừ (`ScanExcludeDirs`)** hay không.

### Hỏi: Làm sao để làm mới hoàn toàn danh sách dự án và quét lại từ đầu?
- Bấm nút **Chọn nhiều** → **Chọn tất cả** → **Xóa đã chọn** (thao tác này chỉ xóa bản ghi trên giao diện, không đụng tới thư mục gốc trên ổ cứng), sau đó bấm lại **Scan Local**.

### Hỏi: Làm sao để một dự án tự động hiển thị App Icon cạnh tiêu đề khi quét?
- Đặt một file icon vuông (`.png` hoặc `.ico`, kích thước từ `48×48` đến `2048×2048`) tại một trong các đường dẫn chuẩn của dự án như `build/appicon.png`, `build/windows/icon.ico`, `public/apple-touch-icon.png`, `assets/icon.png` hoặc `icons/icon128.png`, sau đó bấm **Scan Local** (hoặc mở Chi tiết dự án → bấm **✨ Tạo theo mô tả**).

