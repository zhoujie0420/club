#!/usr/bin/env bash
# Replace club-api and web/ from a release tarball.
# Never writes club.db, club.db-*, or service.env.
set -euo pipefail

CLUB_ROOT="${CLUB_ROOT:-/home/deploy/club-test}"
SKIP_SERVICE="${CLUB_SKIP_SERVICE:-0}"
HEALTH_URL="${CLUB_HEALTH_URL:-http://127.0.0.1:18080/healthz}"
TAR="${1:-}"

if [[ -z "$TAR" || ! -f "$TAR" ]]; then
  echo "usage: apply-release.sh <tarball>" >&2
  exit 2
fi
if [[ ! -d "$CLUB_ROOT" ]]; then
  echo "CLUB_ROOT does not exist: $CLUB_ROOT" >&2
  exit 1
fi

listing="$(tar -tzf "$TAR")"
if grep -E '(^|/)(\.\.|club\.db|club\.db-wal|club\.db-shm|service\.env)(/|$)' <<<"$listing" >/dev/null; then
  echo "refusing tarball that contains club.db, service.env, or .. paths" >&2
  exit 1
fi
if ! grep -qx 'club-api' <<<"$listing" && ! grep -qx './club-api' <<<"$listing"; then
  echo "tarball missing club-api" >&2
  exit 1
fi

STAGE="$(mktemp -d "${TMPDIR:-/tmp}/club-apply.XXXXXX")"
cleanup() { rm -rf "$STAGE"; }
trap cleanup EXIT
tar -xzf "$TAR" -C "$STAGE"
# Belt and suspenders: drop secrets if a packer regression reintroduces them.
rm -f "$STAGE/club.db" "$STAGE/club.db-wal" "$STAGE/club.db-shm" "$STAGE/service.env"
rm -rf "$STAGE/club.db"

if [[ ! -f "$STAGE/club-api" ]]; then
  echo "extracted tree missing club-api" >&2
  exit 1
fi
chmod 755 "$STAGE/club-api"

install -m 755 "$STAGE/club-api" "$CLUB_ROOT/club-api.next"
if [[ -d "$STAGE/web" ]]; then
  rm -rf "$CLUB_ROOT/web.next"
  cp -R "$STAGE/web" "$CLUB_ROOT/web.next"
fi
# Operator scripts (watch-release, systemd units) live in ~/club-watch
# so a release cannot overwrite the puller with a broken copy.
for f in apply-release.sh backup-sqlite.sh; do
  if [[ -f "$STAGE/$f" ]]; then
    install -m 755 "$STAGE/$f" "$CLUB_ROOT/$f"
  fi
done
if [[ -f "$STAGE/club-test.service" ]]; then
  install -m 644 "$STAGE/club-test.service" "$CLUB_ROOT/club-test.service"
fi
if [[ -f "$STAGE/REVISION" ]]; then
  install -m 644 "$STAGE/REVISION" "$CLUB_ROOT/.release-sha.next"
fi

swap_in() {
  if [[ -f "$CLUB_ROOT/club-api" ]]; then
    rm -f "$CLUB_ROOT/club-api.prev"
    mv "$CLUB_ROOT/club-api" "$CLUB_ROOT/club-api.prev"
  fi
  mv "$CLUB_ROOT/club-api.next" "$CLUB_ROOT/club-api"
  chmod 755 "$CLUB_ROOT/club-api"
  if [[ -d "$CLUB_ROOT/web.next" ]]; then
    rm -rf "$CLUB_ROOT/web.prev"
    if [[ -d "$CLUB_ROOT/web" ]]; then
      mv "$CLUB_ROOT/web" "$CLUB_ROOT/web.prev"
    fi
    mv "$CLUB_ROOT/web.next" "$CLUB_ROOT/web"
  fi
}

roll_back() {
  if [[ -f "$CLUB_ROOT/club-api.prev" ]]; then
    mv "$CLUB_ROOT/club-api.prev" "$CLUB_ROOT/club-api"
  fi
  if [[ -d "$CLUB_ROOT/web.prev" ]]; then
    rm -rf "$CLUB_ROOT/web"
    mv "$CLUB_ROOT/web.prev" "$CLUB_ROOT/web"
  fi
}

if [[ "$SKIP_SERVICE" != "1" ]] && command -v systemctl >/dev/null 2>&1; then
  systemctl --user stop club-test.service || true
  swap_in
  if [[ -f "$CLUB_ROOT/club-test.service" ]]; then
    mkdir -p "${HOME}/.config/systemd/user"
    install -m 644 "$CLUB_ROOT/club-test.service" "${HOME}/.config/systemd/user/club-test.service"
    systemctl --user daemon-reload
  fi
  systemctl --user start club-test.service
  ok=0
  for _ in $(seq 1 25); do
    if curl -fsS --max-time 2 "$HEALTH_URL" >/dev/null 2>&1; then
      ok=1
      break
    fi
    sleep 0.4
  done
  if [[ "$ok" != "1" ]]; then
    echo "health check failed after swap; rolling back" >&2
    systemctl --user stop club-test.service || true
    roll_back
    systemctl --user start club-test.service || true
    exit 1
  fi
else
  swap_in
fi

if [[ -f "$CLUB_ROOT/.release-sha.next" ]]; then
  mv "$CLUB_ROOT/.release-sha.next" "$CLUB_ROOT/.release-sha"
fi
rm -f "$CLUB_ROOT/club-api.next" "$CLUB_ROOT/club-api.new.gz" "$CLUB_ROOT/club-api.next.gz"
echo "applied $(tr -d '[:space:]' < "$CLUB_ROOT/.release-sha" 2>/dev/null || echo unknown) to $CLUB_ROOT"
