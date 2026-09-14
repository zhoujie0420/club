#!/usr/bin/env bash
# Emergency local publish: test, build, pack, scp, apply.
# Prefer pushing main and letting GitHub Actions + club-watch do this.
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
ART="$ROOT/.artifacts"
mkdir -p "$ART"

(cd "$ROOT/server" && go test ./...)
(cd "$ROOT/front" && npm ci && npm run type-check && npm run build:h5)
(cd "$ROOT/server" && CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags='-s -w' -o "$ART/club-api" ./cmd/api)
bash "$ROOT/deploy/pack-release.sh"

scp -o ConnectTimeout=15 "$ART/club-test.tar.gz" my-cloud:/tmp/club-test.tar.gz
ssh -o ConnectTimeout=15 my-cloud 'bash /home/deploy/club-test/apply-release.sh /tmp/club-test.tar.gz && rm -f /tmp/club-test.tar.gz'
echo "manual release applied"
