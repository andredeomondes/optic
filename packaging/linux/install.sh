#!/bin/sh
# Installs OPTIC for the current user (no root needed).
set -e
cd "$(dirname "$0")"
BIN="$HOME/.local/bin"; APPS="$HOME/.local/share/applications"; ICONS="$HOME/.local/share/icons/hicolor/512x512/apps"
mkdir -p "$BIN" "$APPS" "$ICONS"
install -m 755 optic-desktop "$BIN/optic-desktop"
install -m 644 optic.png "$ICONS/optic.png"
sed "s|^Exec=.*|Exec=$BIN/optic-desktop|" optic.desktop > "$APPS/optic.desktop"
command -v update-desktop-database >/dev/null && update-desktop-database "$APPS" || true
echo "OPTIC instalado. Procure por \"OPTIC\" no menu de aplicativos."
