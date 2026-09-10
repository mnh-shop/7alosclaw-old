#!/usr/bin/env bash
# Canonical source for /usr/local/bin/zalosclaw-deploy on the zuey VPS.
#
# This repo is the source of truth; the VPS copy is a downstream replica.
# Auto-synced on every beta release by the `Sync zuey ops scripts to VPS`
# step in `.github/workflows/dev-beta-release.yaml`. See
# `docs/deployment-guide.md` for the manual sync recipe and required
# GitHub secrets.
#
# Self-loop guard (2026-05-27 incident): when `/opt/zalosclaw/current` is a
# self-referential symlink (current -> current), `readlink -f` exits non-zero.
# With `set -euo pipefail`, that aborts the script before `ln -sfn` can fix
# the symlink — causing silent CI deploy failures. The guard below swallows
# the readlink failure and logs a warning when the symlink resolves to empty,
# so the deploy still completes (without a rollback target).
set -euo pipefail
release_dir="${1:?usage: zalosclaw-deploy /opt/zalosclaw/releases/<release-dir>}"
if [ ! -x "$release_dir/zalosclaw" ]; then echo "missing executable: $release_dir/zalosclaw" >&2; exit 2; fi
if [ ! -d "$release_dir/migrations" ]; then echo "missing migrations dir" >&2; exit 2; fi
previous=""
if [ -L /opt/zalosclaw/current ]; then previous="$(readlink -f /opt/zalosclaw/current 2>/dev/null || true)"; fi
if [ -z "$previous" ] && [ -L /opt/zalosclaw/current ]; then
  echo "warn: /opt/zalosclaw/current is a symlink but readlink -f failed (likely self-loop or broken target); overwriting without rollback target" >&2
fi
ln -sfn "$release_dir" /opt/zalosclaw/current
chown -h zalosclaw:zalosclaw /opt/zalosclaw/current || true
chown -R root:root "$release_dir"
chmod 755 "$release_dir/zalosclaw"
set -a
. /etc/zalosclaw/zalosclaw.env
set +a
systemctl daemon-reload
if systemctl is-active --quiet zalosclaw; then systemctl stop zalosclaw; fi
/opt/zalosclaw/current/zalosclaw upgrade
systemctl start zalosclaw
for i in $(seq 1 60); do
  if curl -fsS http://127.0.0.1:18790/health >/tmp/zalosclaw-health.json; then
    cat /tmp/zalosclaw-health.json
    echo
    exit 0
  fi
  sleep 2
done
echo "health check failed" >&2
journalctl -u zalosclaw -n 120 --no-pager >&2 || true
if [ -n "$previous" ] && [ -d "$previous" ]; then
  echo "rolling back to $previous" >&2
  ln -sfn "$previous" /opt/zalosclaw/current
  systemctl restart zalosclaw || true
fi
exit 1
