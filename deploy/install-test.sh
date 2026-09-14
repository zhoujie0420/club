#!/usr/bin/env bash
set -euo pipefail
cd /home/deploy/club-test
umask 077
if [ ! -f service.env ]; then
  printf 'PORT=18080\nCLUB_DB_PATH=/home/deploy/club-test/club.db\nCLUB_WEB_DIR=/home/deploy/club-test/web\nCLUB_ACCESS_CODE=holeclub\nAPP_VERSION=0.1-test\n' > service.env
fi
chmod 600 service.env
chmod 755 club-api
mkdir -p /home/deploy/.config/systemd/user
install -m 644 club-test.service /home/deploy/.config/systemd/user/club-test.service
systemctl --user daemon-reload
systemctl --user enable --now club-test.service
systemctl --user restart club-test.service
systemctl --user --no-pager status club-test.service
