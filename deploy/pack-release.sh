#!/usr/bin/env bash
# Pack linux club-api + H5 web/ into .artifacts/club-test.tar.gz.
# The archive must never contain club.db or service.env.
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
ART="${CLUB_ARTIFACT_DIR:-$ROOT/.artifacts}"
REV="${CLUB_REVISION:-}"
if [[ -z "$REV" ]]; then
  REV="$(git -C "$ROOT" rev-parse HEAD)"
fi
API="$ART/club-api"
WEB="${CLUB_WEB_DIR:-$ROOT/front/dist/build/h5}"

if [[ ! -f "$API" ]]; then
  echo "missing $API (build the linux binary first)" >&2
  exit 1
fi
if [[ ! -f "$WEB/index.html" ]]; then
  echo "missing H5 build at $WEB/index.html" >&2
  exit 1
fi

STAGE="$ART/stage"
rm -rf "$STAGE"
mkdir -p "$STAGE/web"
cp "$API" "$STAGE/club-api"
chmod 755 "$STAGE/club-api"
cp -R "$WEB/." "$STAGE/web/"
cp "$ROOT/deploy/apply-release.sh" "$STAGE/apply-release.sh"
cp "$ROOT/deploy/watch-release.sh" "$STAGE/watch-release.sh"
cp "$ROOT/deploy/backup-sqlite.sh" "$STAGE/backup-sqlite.sh"
cp "$ROOT/deploy/club-test.service" "$STAGE/club-test.service"
printf '%s\n' "$REV" > "$STAGE/REVISION"
printf '%s\n' "$REV" > "$ART/REVISION"

# Fail closed if a future copy step drops secrets into the stage.
if [[ -e "$STAGE/club.db" || -e "$STAGE/service.env" ]]; then
  echo "refusing to pack club.db or service.env" >&2
  exit 1
fi

tar -C "$STAGE" -czf "$ART/club-test.tar.gz" .
digest() {
  if command -v sha256sum >/dev/null 2>&1; then
    sha256sum "$1" | awk '{print $1}'
  else
    shasum -a 256 "$1" | awk '{print $1}'
  fi
}
digest "$ART/club-test.tar.gz" > "$ART/club-test.tar.gz.sha256"
echo "packed $ART/club-test.tar.gz revision $REV"
