# Chính sách & Kiến trúc Bảo mật (Security Policy)

**T-Deck** hoạt động trực tiếp với các thư mục mã nguồn cục bộ, tệp biến môi trường `.env` và kho lưu trữ đám mây **Cloudflare R2**. Vì vậy, dự án áp dụng các cơ chế bảo vệ đa tầng ngay từ kiến trúc cốt lõi để ngăn ngừa rò rỉ khóa bí mật và chống tấn công qua gói cài đặt.

---

## 1. Các Cơ chế Bảo vệ Cốt lõi trong T-Deck

### 1.1. Cô lập Khóa Bí mật Đám mây (`settings.json` vs `.env`)
- Khi người dùng cấu hình `Access Key ID` và `Secret Access Key` của Cloudflare R2 trên giao diện **Cài đặt (⚙️)**:
  - `SettingsManager.Save()` (`internal/config/settings.go`) **luôn xóa trắng** hai trường `access_key_id` và `secret_access_key` trước khi ghi cấu hình chung xuống `<InstallDir>\settings.json` (thư mục cài đặt người dùng).
  - Hai khóa bí mật này chỉ được ghi riêng vào tệp `<InstallDir>\.env` với quyền tệp hạn chế (`0600`).
  - Khóa bí mật không bao giờ xuất hiện trong `catalog.json`, trong cơ sở dữ liệu `tdeck.db` hay trong bất kỳ bản ghi log nào.

### 1.2. Tự động Làm sạch Bí mật khi Đồng bộ `.env.example`
- Công cụ **Sync `.env.example`** (`internal/scanner/env_parser.go`) khi đọc file `.env` của dự án sẽ giữ lại tên biến và chú thích nhưng **xóa bỏ hoàn toàn giá trị sau dấu `=`** (ví dụ `OPENAI_API_KEY=sk-xxxx` → `OPENAI_API_KEY=`).
- Giúp lập trình viên cập nhật `.env.example` để commit lên Git mà không bao giờ vô tình đẩy lộ API Key hay mật khẩu cơ sở dữ liệu.

### 1.3. Chống Lỗ hổng Giải nén Đường dẫn (Zip-Slip Protection)
- Khi tải gói phát hành `.zip` từ Cloudflare R2 về máy (`internal/cloud/archive.go` — hàm `unzipSecure`):
  - Mọi đường dẫn tệp bên trong gói `.zip` đều được chuẩn hóa và kiểm tra tiền tố tuyệt đối (`strings.HasPrefix(targetPath, filepath.Clean(destDir) + string(os.PathSeparator))`).
  - Bất kỳ mục nén nào chứa chuỗi leo thang thư mục (`../` hoặc đường dẫn tuyệt đối) đều bị chặn đứng ngay lập tức trước khi chạm vào ổ đĩa.

### 1.4. Xác thực Toàn vẹn `SHA-256` & Cập nhật Nguyên tử (Atomic Swap & Rollback)
- Mỗi bản phát hành tải lên Cloudflare R2 đều được tính toán và lưu kèm mã băm `SHA-256` trong `releases/<slug>/index.json` và ` catalog.json`.
- Khi tải bản cập nhật về máy trạm:
  1. Tệp tải về được đặt trong thư mục cách ly tạm thời (`staging`) và đối chiếu mã băm `SHA-256` khớp 100% trước khi giải nén.
  2. Phiên bản đang dùng được chuyển nguyên tử sang thư mục sao lưu `.bak` trước khi kích hoạt bản mới.
  3. Nếu quá trình xác thực hoặc giải nén thất bại, hệ thống tự động hủy bỏ bản tải lỗi và khôi phục nguyên trạng từ `.bak`.

### 1.5. An toàn Dữ liệu Nguồn khi Xóa Dự án
- Các thao tác **Xóa dự án** hoặc **Chọn nhiều → Xóa đã chọn** trên giao diện T-Deck được thiết kế chỉ đánh dấu xóa mềm (`deleted_at` tombstone) trong SQLite và gỡ ảnh cache `.png`/`.icon.png` trong `<InstallDir>\thumbnails`, **tuyệt đối không xóa thư mục mã nguồn gốc** trên ổ đĩa của người dùng.

---

## 2. Báo cáo Lỗ hổng Bảo mật

Nếu bạn phát hiện bất kỳ rủi ro bảo mật nào liên quan đến:
- Rò rỉ thông tin nhạy cảm từ `.env` hoặc cấu hình R2,
- Lỗ hổng thực thi lệnh ngoài ý muốn trong `internal/launcher`,
- Lỗi vượt quyền đường dẫn khi giải nén hoặc phục vụ ảnh qua middleware `/thumbnails/*`,

vui lòng không công khai chi tiết khai thác trên Issue công cộng mà liên hệ trực tiếp với nhóm bảo trì dự án kèm theo các bước tái hiện chi tiết để được khắc phục và phát hành bản vá sớm nhất.

