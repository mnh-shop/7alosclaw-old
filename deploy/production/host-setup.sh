#!/usr/bin/env bash
# host-setup.sh — one-time Fedora host bootstrap for native 7alosclaw deployment.
# Run as root:  sudo ./host-setup.sh
set -euo pipefail

SERVICE_USER="${SERVICE_USER:-alos}"
DATA_ROOT="${DATA_ROOT:-/data}"

[[ $EUID -eq 0 ]] || { echo "run as root"; exit 1; }

# 1. Service user (non-login, rootless podman runtime owner)
if ! id "$SERVICE_USER" &>/dev/null; then
    useradd -m -s /bin/bash "$SERVICE_USER"
    echo "created user $SERVICE_USER"
fi

# 2. Subordinate UID/GID ranges — 5M ids ≈ 610 sandbox containers at 8196 each
if ! grep -q "^${SERVICE_USER}:" /etc/subuid; then
    usermod --add-subuids 100000-5100000 --add-subgids 100000-5100000 "$SERVICE_USER"
    echo "allocated subuid/subgid 100000-5100000 to $SERVICE_USER"
fi

# 3. Linger — user services (harness + quadlets) survive logout and run at boot
loginctl enable-linger "$SERVICE_USER"

# 4. Data dirs
install -d -o "$SERVICE_USER" -g "$SERVICE_USER" -m 0750 \
    "$DATA_ROOT/7alosclaw" "$DATA_ROOT/tenants"
echo "created $DATA_ROOT/7alosclaw and $DATA_ROOT/tenants"

# 5. XFS project quota check — required for per-tenant disk caps
if xfs_quota -x -c 'report -h' "$DATA_ROOT" &>/dev/null; then
    echo "OK: XFS project quotas active on $DATA_ROOT"
else
    echo "WARNING: $DATA_ROOT is not XFS with pquota — per-tenant disk caps WILL NOT ENFORCE."
    echo "  Fix: remount with -o pquota (default on modern xfs) or migrate to an XFS volume."
fi

# 6. Cloud metadata exposure — sandbox boxes with network can reach it
if ip route show | grep -q 169.254.169.254; then
    echo "NOTE: cloud metadata route present (169.254.169.254)."
    echo "  Harden: AWS → enable IMDSv2 with hop limit 1, or delete the route:"
    echo "  ip route del 169.254.169.254/32 dev $(ip route show | awk '/169.254.169.254/{print $3; exit}')"
fi

echo
echo "Next (as $SERVICE_USER): build sandbox image, install quadlet + service units."
echo "See deploy/production/README.md"
