#!/usr/bin/env bash
# Poll the rolling GitHub release and apply it when REVISION changes.
set -euo pipefail

CLUB_ROOT="${CLUB_ROOT:-/home/deploy/club-test}"
TAG="${CLUB_RELEASE_TAG:-club-test}"
REPO="${CLUB_GITHUB_REPO:-zhoujie0420/club}"
BASE="https://github.com/${REPO}/releases/download/${TAG}"
LOG="${CLUB_WATCH_LOG:-$CLUB_ROOT/watch.log}"
LOCK="${CLUB_WATCH_LOCK:-$CLUB_ROOT/.watch.lock}"

mkdir -p "$CLUB_ROOT"
exec 9>"$LOCK"
if ! flock -n 9; then
  exit 0
fi

log() {
  printf '%s %s\n' "$(date -Iseconds)" "$*" >> "$LOG"
}

digest() {
  if command -v sha256sum >/dev/null 2>&1; then
    sha256sum "$1" | awk '{print $1}'
  else
    shasum -a 256 "$1" | awk '{print $1}'
  fi
}

TMP="$(mktemp -d "${TMPDIR:-/tmp}/club-watch.XXXXXX")"
cleanup() { rm -rf "$TMP"; }
trap cleanup EXIT

if ! curl -fsSL --max-time 45 -A 'club-watch/1.0' -H 'Cache-Control: no-cache' \
  -o "$TMP/REVISION" "$BASE/REVISION"; then
  log "no release yet or fetch failed ($BASE/REVISION)"
  exit 0
fi

new="$(tr -d '[:space:]' < "$TMP/REVISION")"
old=""
if [[ -f "$CLUB_ROOT/.release-sha" ]]; then
  old="$(tr -d '[:space:]' < "$CLUB_ROOT/.release-sha")"
fi
if [[ -z "$new" || "$new" == "$old" ]]; then
  exit 0
fi

log "downloading $new (was ${old:-none})"
curl -fsSL --max-time 180 -A 'club-watch/1.0' -H 'Cache-Control: no-cache' \
  -o "$TMP/club-test.tar.gz" "$BASE/club-test.tar.gz"
curl -fsSL --max-time 45 -A 'club-watch/1.0' -H 'Cache-Control: no-cache' \
  -o "$TMP/club-test.tar.gz.sha256" "$BASE/club-test.tar.gz.sha256"

got="$(digest "$TMP/club-test.tar.gz")"
want="$(awk '{print $1}' "$TMP/club-test.tar.gz.sha256" | tr -d '[:space:]')"
if [[ -z "$want" || "$got" != "$want" ]]; then
  log "sha256 mismatch got=$got want=$want"
  exit 1
fi

mkdir -p "$TMP/extract"
tar -xzf "$TMP/club-test.tar.gz" -C "$TMP/extract"
APPLY="$TMP/extract/apply-release.sh"
if [[ ! -f "$APPLY" ]]; then
  log "tarball missing apply-release.sh"
  exit 1
fi
chmod +x "$APPLY"
CLUB_ROOT="$CLUB_ROOT" "$APPLY" "$TMP/club-test.tar.gz"
log "applied $new"
