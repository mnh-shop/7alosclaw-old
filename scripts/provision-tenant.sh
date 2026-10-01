#!/usr/bin/env bash
# provision-tenant.sh — onboard one tenant: workspace dir + XFS disk quota + API tenant + operator key.
#
# Usage:   provision-tenant.sh <slug> <name> [--quota-soft 10G] [--quota-hard 15G]
# Env:     ZALOSCLAW_API      gateway base URL   (default http://127.0.0.1:18790)
#          ZALOSCLAW_TOKEN    gateway token with owner rights (required)
#          ZALOSCLAW_USER     owner user id      (default system)
#          WORKSPACE_ROOT     workspace root     (default /data/7alosclaw/workspace)
# Requires: jq, root (for XFS quota); run on the host or via ssh.
set -euo pipefail

SLUG="${1:?usage: provision-tenant.sh <slug> <name> [--quota-soft 10G] [--quota-hard 15G]}"
NAME="${2:?usage: provision-tenant.sh <slug> <name> [--quota-soft 10G] [--quota-hard 15G]}"
shift 2
SOFT="10G"; HARD="15G"
while [[ $# -gt 0 ]]; do
    case "$1" in
        --quota-soft) SOFT="$2"; shift 2;;
        --quota-hard) HARD="$2"; shift 2;;
        *) echo "unknown flag: $1"; exit 1;;
    esac
done

: "${ZALOSCLAW_API:=http://127.0.0.1:18790}"
: "${ZALOSCLAW_USER:=system}"
: "${WORKSPACE_ROOT:=/data/7alosclaw/workspace}"
: "${ZALOSCLAW_TOKEN:?ZALOSCLAW_TOKEN (gateway token) required}"

# Slug: lowercase alnum + dashes, 2-63 chars — same charset the harness sanitizes to
[[ "$SLUG" =~ ^[a-z0-9][a-z0-9-]{1,62}$ ]] || { echo "invalid slug: $SLUG"; exit 1; }

TENANT_DIR="$WORKSPACE_ROOT/tenants/$SLUG"

# 1. Workspace dir (created before any agent can write)
mkdir -p "$TENANT_DIR"
chmod 0750 "$TENANT_DIR"

# 2. XFS project quota (kernel-enforced disk cap)
MAX_PROJID=$(awk -F: '{print $2}' /etc/projid 2>/dev/null | sort -n | tail -1 || true)
PROJID=$((10000 + ${MAX_PROJID:-0}))
grep -q "^${SLUG}:" /etc/projid 2>/dev/null || echo "${SLUG}:${PROJID}" >> /etc/projid
grep -q ":${PROJID}:" /etc/projects 2>/dev/null || echo "${PROJID}:${TENANT_DIR}" >> /etc/projects
xfs_quota -x -c "project -s ${SLUG}" "$WORKSPACE_ROOT"
xfs_quota -x -c "limit -p bsoft=${SOFT} bhard=${HARD} ${SLUG}" "$WORKSPACE_ROOT"

# 3. Tenant record
RESP=$(curl -sfS -X POST "$ZALOSCLAW_API/v1/tenants" \
    -H "Authorization: Bearer $ZALOSCLAW_TOKEN" \
    -H "X-ZalosClaw-User-Id: $ZALOSCLAW_USER" \
    -H "Content-Type: application/json" \
    -d "{\"name\":\"$NAME\",\"slug\":\"$SLUG\"}")
TENANT_ID=$(echo "$RESP" | jq -r '.id // .tenant.id')
[[ -n "$TENANT_ID" && "$TENANT_ID" != "null" ]] || { echo "tenant create failed: $RESP"; exit 1; }

# 4. Tenant-bound operator API key (raw key shown exactly once)
KEYRESP=$(curl -sfS -X POST "$ZALOSCLAW_API/v1/api-keys" \
    -H "Authorization: Bearer $ZALOSCLAW_TOKEN" \
    -H "X-ZalosClaw-User-Id: $ZALOSCLAW_USER" \
    -H "Content-Type: application/json" \
    -d "{\"name\":\"$SLUG-operator\",\"scopes\":[\"operator.read\",\"operator.write\"],\"tenant_id\":\"$TENANT_ID\"}")
API_KEY=$(echo "$KEYRESP" | jq -r '.key // empty')
KEY_ID=$(echo "$KEYRESP" | jq -r '.id // empty')
[[ -n "$API_KEY" ]] || { echo "api key create failed: $KEYRESP"; exit 1; }

echo "──────────────────────────────────────────────────"
echo "Tenant provisioned: $NAME ($SLUG)"
echo "  tenant_id : $TENANT_ID"
echo "  disk      : soft $SOFT / hard $HARD  ($TENANT_DIR)"
echo "  key_id    : $KEY_ID"
echo "  api_key   : $API_KEY"
echo "  ⚠ store the key now — it is never shown again"
echo "──────────────────────────────────────────────────"
