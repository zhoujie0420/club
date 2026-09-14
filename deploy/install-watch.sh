#!/usr/bin/env bash
# Install the user systemd timer that pulls the rolling GitHub release.
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
DEST="${CLUB_ROOT:-$HOME/club-test}"
mkdir -p "$DEST" "$HOME/.config/systemd/user"
install -m 755 "$ROOT/deploy/apply-release.sh" "$DEST/apply-release.sh"
install -m 755 "$ROOT/deploy/watch-release.sh" "$DEST/watch-release.sh"
install -m 644 "$ROOT/deploy/club-watch.service" "$HOME/.config/systemd/user/club-watch.service"
install -m 644 "$ROOT/deploy/club-watch.timer" "$HOME/.config/systemd/user/club-watch.timer"
systemctl --user daemon-reload
systemctl --user enable --now club-watch.timer
systemctl --user --no-pager list-timers club-watch.timer
