#!/usr/bin/env bash
# Offline checks for pack/apply: secrets stay put, bad tarballs are refused.
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
TMP="$(mktemp -d "${TMPDIR:-/tmp}/club-selftest.XXXXXX")"
cleanup() { rm -rf "$TMP"; }
trap cleanup EXIT

mkdir -p "$TMP/root/web" "$TMP/web" "$TMP/art"
printf 'sentinel-db\n' > "$TMP/root/club.db"
printf 'CLUB_ACCESS_CODE=do-not-touch\n' > "$TMP/root/service.env"
printf 'old-web\n' > "$TMP/root/web/index.html"
printf '#!/bin/sh\necho ok\n' > "$TMP/art/club-api"
chmod 755 "$TMP/art/club-api"
printf '<html>new</html>\n' > "$TMP/web/index.html"

CLUB_ARTIFACT_DIR="$TMP/art" \
  CLUB_WEB_DIR="$TMP/web" \
  CLUB_REVISION='deadbeef' \
  bash "$ROOT/deploy/pack-release.sh"

if tar -tzf "$TMP/art/club-test.tar.gz" | grep -E 'club\.db|service\.env'; then
  echo "packed archive leaked secrets" >&2
  exit 1
fi

CLUB_ROOT="$TMP/root" CLUB_SKIP_SERVICE=1 \
  bash "$ROOT/deploy/apply-release.sh" "$TMP/art/club-test.tar.gz"

grep -qx 'sentinel-db' "$TMP/root/club.db"
grep -qx 'CLUB_ACCESS_CODE=do-not-touch' "$TMP/root/service.env"
grep -q 'new' "$TMP/root/web/index.html"
test -x "$TMP/root/club-api"
grep -qx 'deadbeef' "$TMP/root/.release-sha"

mkdir -p "$TMP/bad"
printf 'hacked\n' > "$TMP/bad/club.db"
printf 'x\n' > "$TMP/bad/club-api"
chmod 755 "$TMP/bad/club-api"
tar -C "$TMP/bad" -czf "$TMP/bad.tar.gz" .
if CLUB_ROOT="$TMP/root" CLUB_SKIP_SERVICE=1 bash "$ROOT/deploy/apply-release.sh" "$TMP/bad.tar.gz"; then
  echo "apply-release accepted a tarball that contains club.db" >&2
  exit 1
fi
grep -qx 'sentinel-db' "$TMP/root/club.db"
grep -qx 'CLUB_ACCESS_CODE=do-not-touch' "$TMP/root/service.env"

echo "apply-release self-test passed"
