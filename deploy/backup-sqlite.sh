#!/usr/bin/env bash
set -euo pipefail
ROOT="${CLUB_DB_PATH:-/home/deploy/club-test/club.db}"
DIR="$(dirname "$ROOT")/backups"
mkdir -p "$DIR"
STAMP="$(date +%Y%m%d-%H%M%S)"
DEST="$DIR/club-$STAMP.db"
python3 - "$ROOT" "$DEST" <<'PY'
import sqlite3, sys
src, dest = sys.argv[1], sys.argv[2]
source = sqlite3.connect(f"file:{src}?mode=ro", uri=True)
backup = sqlite3.connect(dest)
source.backup(backup)
backup.close()
source.close()
print(dest)
PY
find "$DIR" -name 'club-*.db' -mtime +14 -delete
