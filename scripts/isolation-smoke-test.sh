#!/usr/bin/env bash
# isolation-smoke-test.sh — P0 acceptance drill: verify two tenants cannot see each other.
#
# Usage:   isolation-smoke-test.sh <keyA> <keyB> [api_base]
# Env:     ZALOSCLAW_USER  owner user id (default system)
# Requires: jq
#
# Exit 0 = all isolation checks pass. Exit 1 = ISOLATION VIOLATION.
set -euo pipefail

KEY_A="${1:?usage: isolation-smoke-test.sh <keyA> <keyB> [api_base]}"
KEY_B="${2:?usage: isolation-smoke-test.sh <keyA> <keyB> [api_base]}"
API="${3:-${ZALOSCLAW_API:-http://127.0.0.1:18790}}"
USER_ID="${ZALOSCLAW_USER:-system}"

fetch_ids() { # $1 = key, $2 = path → sorted id list
    curl -sfS "$API$2" \
        -H "Authorization: Bearer $1" \
        -H "X-ZalosClaw-User-Id: $USER_ID" |
        jq -r '.[].id // empty' | sort
}

FAIL=0
check_disjoint() { # $1 = resource label, $2 = path
    local a b overlap
    a=$(fetch_ids "$KEY_A" "$2") || { echo "FAIL  $2 unreachable with key A"; FAIL=1; return; }
    b=$(fetch_ids "$KEY_B" "$2") || { echo "FAIL  $2 unreachable with key B"; FAIL=1; return; }
    overlap=$(comm -12 <(echo "$a") <(echo "$b"))
    if [[ -n "$overlap" ]]; then
        echo "VIOLATION  tenant B sees tenant A $2 ids: $overlap"
        FAIL=1
    else
        echo "PASS  $2 lists are disjoint"
    fi
}

echo "ZalosClaw tenant isolation drill → $API"
check_disjoint "agents"   "/v1/agents"
check_disjoint "sessions" "/v1/sessions"
check_disjoint "skills"   "/v1/skills"

# Cross-tenant write probe: B must not be able to touch an A-scoped resource by guessing ids
A_AGENT=$(fetch_ids "$KEY_A" "/v1/agents" | head -1)
if [[ -n "$A_AGENT" ]]; then
    CODE=$(curl -s -o /dev/null -w '%{http_code}' "$API/v1/agents/$A_AGENT" \
        -H "Authorization: Bearer $KEY_B" -H "X-ZalosClaw-User-Id: $USER_ID")
    if [[ "$CODE" =~ ^(200|201)$ ]]; then
        echo "VIOLATION  tenant B read tenant A's agent $A_AGENT (HTTP $CODE)"
        FAIL=1
    else
        echo "PASS  cross-tenant agent read denied (HTTP $CODE)"
    fi
fi

if [[ $FAIL -eq 0 ]]; then
    echo "─────────────────────────────────"
    echo "ALL ISOLATION CHECKS PASSED"
else
    echo "─────────────────────────────────"
    echo "ISOLATION VIOLATIONS FOUND — DO NOT SHIP"
fi
exit $FAIL
