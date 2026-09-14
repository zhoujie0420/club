#!/usr/bin/env bash
# Poll the rolling GitHub release via the API (github.com download URLs
# time out from this China host) and apply it when REVISION changes.
set -euo pipefail

CLUB_ROOT="${CLUB_ROOT:-/home/deploy/club-test}"
TAG="${CLUB_RELEASE_TAG:-club-test}"
REPO="${CLUB_GITHUB_REPO:-zhoujie0420/club}"
API="https://api.github.com/repos/${REPO}/releases/tags/${TAG}"
LOG="${CLUB_WATCH_LOG:-$CLUB_ROOT/watch.log}"
LOCK="${CLUB_WATCH_LOCK:-$CLUB_ROOT/.watch.lock}"
UA="club-watch/1.0"

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

asset_url() {
  python3 -c 'import json,sys
name, path = sys.argv[1], sys.argv[2]
rel = json.load(open(path, encoding="utf-8"))
assets = {a.get("name"): a.get("url") for a in rel.get("assets") or []}
url = assets.get(name) or ""
if not url:
    raise SystemExit(2)
print(url)
' "$1" "$TMP/release.json"
}

TMP="$(mktemp -d "${TMPDIR:-/tmp}/club-watch.XXXXXX")"
cleanup() { rm -rf "$TMP"; }
trap cleanup EXIT

if ! curl -fsSL --max-time 30 -A "$UA" -H 'Accept: application/vnd.github+json' \
  -o "$TMP/release.json" "$API"; then
  log "no release yet or API fetch failed ($API)"
  exit 0
fi

download() {
  local name="$1" dest="$2" url
  url="$(asset_url "$name" < "$TMP/release.json")"
  curl -fsSL --max-time 180 -A "$UA" -L \
    -H 'Accept: application/octet-stream' \
    -o "$dest" "$url"
}

if ! download REVISION "$TMP/REVISION"; then
  log "REVISION asset missing"
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
download club-test.tar.gz "$TMP/club-test.tar.gz"
download club-test.tar.gz.sha256 "$TMP/club-test.tar.gz.sha256"

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
