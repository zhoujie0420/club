#!/usr/bin/env bash
# Poll the rolling GitHub release via the API and apply it when REVISION
# changes. github.com download URLs and the API octet-stream path are both
# slow from this China host (~15KB/s), so tarball fetches resume in chunks.
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

# Resume a slow GitHub asset download. Each attempt is time-capped so a
# stalled connection cannot block the timer forever.
download() {
  local name="$1" dest="$2" url
  url="$(asset_url "$name")"
  rm -f "$dest"
  local i
  for i in $(seq 1 15); do
    if curl -fsSL -C - --connect-timeout 20 --max-time 90 \
      -A "$UA" -H 'Accept: application/octet-stream' \
      -o "$dest" "$url"; then
      return 0
    fi
    sleep 2
  done
  return 1
}

TMP="$(mktemp -d "${TMPDIR:-/tmp}/club-watch.XXXXXX")"
cleanup() { rm -rf "$TMP"; }
trap cleanup EXIT

if ! curl -fsSL --connect-timeout 20 --max-time 30 -A "$UA" \
  -H 'Accept: application/vnd.github+json' \
  -o "$TMP/release.json" "$API"; then
  log "no release yet or API fetch failed ($API)"
  exit 0
fi

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
if ! download club-test.tar.gz "$TMP/club-test.tar.gz"; then
  log "tarball download failed"
  exit 1
fi
if ! download club-test.tar.gz.sha256 "$TMP/club-test.tar.gz.sha256"; then
  log "sha256 asset missing"
  exit 1
fi

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
