#!/usr/bin/env bash
set -euo pipefail

export LC_ALL="${LC_ALL:-C.UTF-8}"
export LANG="${LANG:-C.UTF-8}"

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
INSTALL_DIR="${1:-${SCRIPT_DIR}}"
PURGE_DATA=0

for arg in "$@"; do
  case "$arg" in
    --purge)
      PURGE_DATA=1
      ;;
    --prefix=*)
      INSTALL_DIR="${arg#*=}"
      ;;
  esac
done

echo "[T-Deck] Đang gỡ lối tắt và liên kết..."
rm -f "${HOME}/.local/share/applications/t-deck.desktop"
rm -f "${XDG_DESKTOP_DIR:-${HOME}/Desktop}/t-deck.desktop"
rm -f "${HOME}/.config/autostart/t-deck.desktop"
rm -f "${HOME}/.local/bin/tdeck" "${HOME}/.local/bin/tdeck-sync"

if command -v update-desktop-database >/dev/null 2>&1; then
  update-desktop-database "${HOME}/.local/share/applications" >/dev/null 2>&1 || true
fi

if [ "${PURGE_DATA}" -eq 0 ] && [ -t 0 ]; then
  if { [ -n "${DISPLAY:-}" ] || [ -n "${WAYLAND_DISPLAY:-}" ]; } && command -v zenity >/dev/null 2>&1; then
    if zenity --question \
      --title="Gỡ cài đặt T-Deck" \
      --text="Bạn có muốn xóa toàn bộ dữ liệu T-Deck (tdeck.db, settings.json, .env, thumbnails/, apps/) tại:\n${INSTALL_DIR}?" \
      --default-cancel 2>/dev/null; then
      PURGE_DATA=1
    fi
  else
    read -r -p "Bạn có muốn xóa luôn dữ liệu (tdeck.db, settings.json, thumbnails, apps) tại '${INSTALL_DIR}'? [y/N]: " ans
    case "${ans}" in
      [yY]|[yY][eE][sS]) PURGE_DATA=1 ;;
    esac
  fi
fi

if [ "${PURGE_DATA}" -eq 1 ]; then
  rm -rf "${INSTALL_DIR}"
  echo "[OK] Đã xóa toàn bộ thư mục cài đặt và dữ liệu: ${INSTALL_DIR}"
else
  rm -f "${INSTALL_DIR}/TDeck" "${INSTALL_DIR}/tdeck-sync" "${INSTALL_DIR}/t-deck.png" "${INSTALL_DIR}/uninstall.sh"
  echo "[OK] Đã gỡ chương trình T-Deck. Dữ liệu cá nhân tại '${INSTALL_DIR}' được giữ nguyên."
fi

