#!/bin/sh
# Optional local desktop install from an extracted Linux release archive.
set -eu
source_dir=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
install_dir="$HOME/.local/bin"
apps_dir="$HOME/.local/share/applications"
mkdir -p "$install_dir" "$apps_dir"
install -m 755 "$source_dir/player2-kanban" "$install_dir/player2-kanban"
# Desktop-entry quoting: refuse paths requiring escape interpretation.
case "$install_dir" in *'"'*|*'`'*|*'$'*|*'\'*) echo 'Open the portable binary directly: unusual home path cannot be added to the desktop menu.'; exit 1;; esac
cat > "$apps_dir/player2.desktop" <<ENTRY
[Desktop Entry]
Type=Application
Name=Player2
Comment=Your plans, tasks, and AI work
Exec="$install_dir/player2-kanban" open
Terminal=false
Categories=Office;ProjectManagement;
ENTRY
printf '%s\n' 'Player2 installed. Open Player2 from your applications menu.'
