#!/usr/bin/env bash
set -euo pipefail

export LC_ALL="${LC_ALL:-C.UTF-8}"
export LANG="${LANG:-C.UTF-8}"

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
APP_NAME="T-Deck"
DEFAULT_INSTALL_DIR="${HOME}/.local/opt/T-Deck"

INSTALL_DIR="${DEFAULT_INSTALL_DIR}"
OPT_DESKTOP=1
OPT_MENU=1
OPT_BIN=1
OPT_AUTOSTART=0
NON_INTERACTIVE=0

usage() {
  cat <<'EOF'
Trình cài đặt T-Deck cho Linux (UTF-8)
Cách dùng:
  ./install.sh [tùy chọn]

Tùy chọn:
  --prefix=<thư_mục>   Thư mục cài đặt (mặc định: ~/.local/opt/T-Deck)
  --yes, -y            Cài đặt nhanh với tùy chọn mặc định (không hỏi tương tác)
  --no-desktop         Không tạo biểu tượng ngoài Desktop
  --no-menu            Không tạo lối tắt trong Menu Ứng dụng
  --no-bin             Không tạo symlink trong ~/.local/bin
  --autostart          Tự động khởi chạy T-Deck khi đăng nhập
  --help, -h           Hiển thị hướng dẫn này
EOF
}

for arg in "$@"; do
  case "$arg" in
    --prefix=*)
      INSTALL_DIR="${arg#*=}"
      ;;
    --yes|-y)
      NON_INTERACTIVE=1
      ;;
    --no-desktop)
      OPT_DESKTOP=0
      ;;
    --no-menu)
      OPT_MENU=0
      ;;
    --no-bin)
      OPT_BIN=0
      ;;
    --autostart)
      OPT_AUTOSTART=1
      ;;
    --help|-h)
      usage
      exit 0
      ;;
    *)
      echo "[LỖI] Tùy chọn không hợp lệ: $arg" >&2
      usage
      exit 1
      ;;
  esac
done

expand_tilde() {
  local p="$1"
  case "$p" in
    "~"*)
      printf '%s\n' "${HOME}${p#\~}"
      ;;
    *)
      printf '%s\n' "$p"
      ;;
  esac
}

prompt_gui_zenity() {
  local picked
  picked=$(zenity --entry \
    --title="Cài đặt ${APP_NAME}" \
    --text="Chọn thư mục cài đặt ${APP_NAME}.\nLưu ý: Dữ liệu (tdeck.db, thumbnails/, settings.json, apps/) sẽ được lưu trực tiếp trong thư mục này:" \
    --entry-text="${INSTALL_DIR}" 2>/dev/null) || exit 1
  if [ -n "${picked}" ]; then
    INSTALL_DIR="${picked}"
  fi

  local choices
  choices=$(zenity --list --checklist \
    --title="Tùy chọn cài đặt ${APP_NAME}" \
    --text="Đánh dấu các tùy chọn tạo lối tắt và khởi động:" \
    --width=560 --height=300 \
    --column="Chọn" --column="Mã" --column="Tùy chọn" \
    --hide-column=2 --separator="|" \
    TRUE  "desktop"   "Tạo biểu tượng ngoài Màn hình chính (Desktop Shortcut)" \
    TRUE  "menu"      "Tạo lối tắt trong Menu Ứng dụng (Applications Menu)" \
    TRUE  "bin"       "Tạo lệnh 'tdeck' và 'tdeck-sync' trong ~/.local/bin" \
    FALSE "autostart" "Tự động khởi chạy T-Deck khi đăng nhập hệ thống" \
    2>/dev/null) || exit 1

  OPT_DESKTOP=0
  OPT_MENU=0
  OPT_BIN=0
  OPT_AUTOSTART=0
  IFS='|' read -ra SELECTED <<< "$choices"
  for item in "${SELECTED[@]}"; do
    case "$item" in
      desktop) OPT_DESKTOP=1 ;;
      menu) OPT_MENU=1 ;;
      bin) OPT_BIN=1 ;;
      autostart) OPT_AUTOSTART=1 ;;
    esac
  done
}

prompt_tui_whiptail() {
  local picked
  picked=$(whiptail --title "Cài đặt ${APP_NAME}" \
    --inputbox "Nhập thư mục cài đặt ${APP_NAME} (tdeck.db, thumbnails/, settings.json, apps/ sẽ lưu tại đây):" \
    11 72 "${INSTALL_DIR}" 3>&1 1>&2 2>&3) || exit 1
  if [ -n "${picked}" ]; then
    INSTALL_DIR="${picked}"
  fi

  local choices
  choices=$(whiptail --title "Tùy chọn cài đặt ${APP_NAME}" \
    --checklist "Dùng phím Space để bật/tắt checkbox, Enter để tiếp tục:" 15 74 4 \
    "desktop"   "Tạo biểu tượng ngoài Màn hình chính (Desktop)" ON \
    "menu"      "Tạo lối tắt trong Menu Ứng dụng" ON \
    "bin"       "Thêm lệnh 'tdeck' & 'tdeck-sync' vào ~/.local/bin" ON \
    "autostart" "Khởi động T-Deck cùng hệ thống" OFF \
    3>&1 1>&2 2>&3) || exit 1

  OPT_DESKTOP=0
  OPT_MENU=0
  OPT_BIN=0
  OPT_AUTOSTART=0
  [[ "$choices" == *"\"desktop\""* ]] && OPT_DESKTOP=1
  [[ "$choices" == *"\"menu\""* ]] && OPT_MENU=1
  [[ "$choices" == *"\"bin\""* ]] && OPT_BIN=1
  [[ "$choices" == *"\"autostart\""* ]] && OPT_AUTOSTART=1
}

mark() {
  if [ "$1" -eq 1 ]; then printf "x"; else printf " "; fi
}

prompt_cli_interactive() {
  echo "============================================================"
  echo "  TRÌNH CÀI ĐẶT T-DECK CHO LINUX (UTF-8)"
  echo "============================================================"
  echo "Dữ liệu (tdeck.db, thumbnails/, settings.json, apps/) sẽ được"
  echo "lưu trực tiếp bên trong thư mục cài đặt bạn chọn."
  echo
  read -r -p "Thư mục cài đặt [${INSTALL_DIR}]: " custom_dir
  if [ -n "${custom_dir}" ]; then
    INSTALL_DIR="${custom_dir}"
  fi

  while true; do
    echo
    echo "Chọn các tùy chọn (nhập số 1-4 để bật/tắt, nhấn Enter để cài đặt):"
    echo "  [1] [$(mark $OPT_DESKTOP)] Tạo biểu tượng ngoài Màn hình chính (Desktop Shortcut)"
    echo "  [2] [$(mark $OPT_MENU)] Tạo lối tắt trong Menu Ứng dụng (~/.local/share/applications)"
    echo "  [3] [$(mark $OPT_BIN)] Tạo lệnh 'tdeck' & 'tdeck-sync' trong ~/.local/bin"
    echo "  [4] [$(mark $OPT_AUTOSTART)] Tự động khởi chạy T-Deck khi đăng nhập"
    read -r -p "Lựa chọn [1-4 hoặc Enter để bắt đầu cài]: " toggle
    case "${toggle}" in
      "") break ;;
      1) OPT_DESKTOP=$((1 - OPT_DESKTOP)) ;;
      2) OPT_MENU=$((1 - OPT_MENU)) ;;
      3) OPT_BIN=$((1 - OPT_BIN)) ;;
      4) OPT_AUTOSTART=$((1 - OPT_AUTOSTART)) ;;
      *) echo "Vui lòng nhập 1, 2, 3, 4 hoặc Enter." ;;
    esac
  done
}

if [ "${NON_INTERACTIVE}" -eq 0 ] && [ -t 0 ]; then
  if { [ -n "${DISPLAY:-}" ] || [ -n "${WAYLAND_DISPLAY:-}" ]; } && command -v zenity >/dev/null 2>&1; then
    prompt_gui_zenity
  elif command -v whiptail >/dev/null 2>&1; then
    prompt_tui_whiptail
  else
    prompt_cli_interactive
  fi
fi

INSTALL_DIR="$(expand_tilde "${INSTALL_DIR}")"
echo "[INFO] Đang cài đặt ${APP_NAME} vào: ${INSTALL_DIR}"

mkdir -p "${INSTALL_DIR}" "${INSTALL_DIR}/thumbnails" "${INSTALL_DIR}/apps"

if [ -f "${SCRIPT_DIR}/TDeck" ]; then
  install -m 0755 "${SCRIPT_DIR}/TDeck" "${INSTALL_DIR}/TDeck"
fi
if [ -f "${SCRIPT_DIR}/tdeck-sync" ]; then
  install -m 0755 "${SCRIPT_DIR}/tdeck-sync" "${INSTALL_DIR}/tdeck-sync"
fi
if [ -f "${SCRIPT_DIR}/t-deck.png" ]; then
  install -m 0644 "${SCRIPT_DIR}/t-deck.png" "${INSTALL_DIR}/t-deck.png"
elif [ -f "${SCRIPT_DIR}/appicon.png" ]; then
  install -m 0644 "${SCRIPT_DIR}/appicon.png" "${INSTALL_DIR}/t-deck.png"
fi
for doc in README.md CHANGELOG.md .env.example; do
  if [ -f "${SCRIPT_DIR}/${doc}" ]; then
    install -m 0644 "${SCRIPT_DIR}/${doc}" "${INSTALL_DIR}/${doc}"
  fi
done
if [ -f "${SCRIPT_DIR}/uninstall.sh" ]; then
  install -m 0755 "${SCRIPT_DIR}/uninstall.sh" "${INSTALL_DIR}/uninstall.sh"
fi

if [ ! -f "${INSTALL_DIR}/TDeck" ]; then
  cat > "${INSTALL_DIR}/TDeck" <<'WRAPPER'
#!/usr/bin/env bash
set -euo pipefail
SELF_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
echo "[T-Deck] Binary GUI 'TDeck' cho Linux chưa được đóng gói sẵn trong bản cross-build này."
echo "[T-Deck] Để build bản GUI Linux native tại máy, hãy cài Go >= 1.26, Node >= 20, libgtk-3-dev, libwebkit2gtk-4.1-dev rồi chạy:"
echo "  git clone https://github.com/kacher2097/t-deck.git && cd t-deck && wails build -s"
echo "  cp build/bin/TDeck '${SELF_DIR}/TDeck' && chmod +x '${SELF_DIR}/TDeck'"
exit 1
WRAPPER
  chmod 0755 "${INSTALL_DIR}/TDeck"
fi

render_desktop_file() {
  local dest="$1"
  mkdir -p "$(dirname "$dest")"
  if [ -f "${SCRIPT_DIR}/t-deck.desktop" ]; then
    sed "s|@INSTALL_DIR@|${INSTALL_DIR}|g" "${SCRIPT_DIR}/t-deck.desktop" > "$dest"
  else
    cat > "$dest" <<EOF
[Desktop Entry]
Version=1.0
Type=Application
Name=T-Deck
GenericName=Application Hub & Release Manager
Comment=Quản lý danh mục dự án, khởi chạy môi trường phát triển và phát hành qua Cloudflare R2
Exec=${INSTALL_DIR}/TDeck %U
Icon=${INSTALL_DIR}/t-deck.png
Terminal=false
Categories=Development;Utility;
StartupNotify=true
StartupWMClass=TDeck
EOF
  fi
  chmod 0755 "$dest"
}

if [ "${OPT_MENU}" -eq 1 ]; then
  MENU_FILE="${HOME}/.local/share/applications/t-deck.desktop"
  render_desktop_file "${MENU_FILE}"
  if command -v update-desktop-database >/dev/null 2>&1; then
    update-desktop-database "${HOME}/.local/share/applications" >/dev/null 2>&1 || true
  fi
  echo "[OK] Đã tạo lối tắt Menu Ứng dụng: ${MENU_FILE}"
fi

if [ "${OPT_DESKTOP}" -eq 1 ]; then
  DESKTOP_DIR="${XDG_DESKTOP_DIR:-${HOME}/Desktop}"
  if [ -d "${DESKTOP_DIR}" ] || mkdir -p "${DESKTOP_DIR}" 2>/dev/null; then
    DESKTOP_FILE="${DESKTOP_DIR}/t-deck.desktop"
    render_desktop_file "${DESKTOP_FILE}"
    if command -v gio >/dev/null 2>&1; then
      gio set "${DESKTOP_FILE}" metadata::trusted true >/dev/null 2>&1 || true
    fi
    echo "[OK] Đã tạo biểu tượng Desktop: ${DESKTOP_FILE}"
  fi
fi

if [ "${OPT_BIN}" -eq 1 ]; then
  mkdir -p "${HOME}/.local/bin"
  ln -sf "${INSTALL_DIR}/TDeck" "${HOME}/.local/bin/tdeck"
  if [ -f "${INSTALL_DIR}/tdeck-sync" ]; then
    ln -sf "${INSTALL_DIR}/tdeck-sync" "${HOME}/.local/bin/tdeck-sync"
  fi
  echo "[OK] Đã tạo lệnh tại: ${HOME}/.local/bin/tdeck"
fi

if [ "${OPT_AUTOSTART}" -eq 1 ]; then
  AUTOSTART_FILE="${HOME}/.config/autostart/t-deck.desktop"
  render_desktop_file "${AUTOSTART_FILE}"
  echo "[OK] Đã bật tự động khởi chạy: ${AUTOSTART_FILE}"
fi

echo
echo "============================================================"
echo "  CÀI ĐẶT T-DECK HOÀN TẤT!"
echo "  Thư mục cài đặt & lưu dữ liệu: ${INSTALL_DIR}"
echo "  Gỡ cài đặt: ${INSTALL_DIR}/uninstall.sh"
echo "============================================================"

