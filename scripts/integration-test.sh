#!/usr/bin/env bash
# WARPture integration test suite
set -euo pipefail

AGENT_URL="${TUNNEL_AGENT_URL:-http://127.0.0.1:8080}"
PASS=0; FAIL=0

green() { echo -e "\033[32m✓ $*\033[0m"; }
red()   { echo -e "\033[31m✗ $*\033[0m"; }

check() {
    local name="$1"; local cmd="$2"; local expected="${3:-200}"
    local result
    result=$(eval "$cmd" 2>&1) && code=$(echo "$result" | tail -1) || code="000"
    if [[ "$code" == "$expected"* ]] || echo "$result" | grep -q "$expected"; then
        green "$name"
        PASS=$((PASS+1))
    else
        red "$name (got: $result)"
        FAIL=$((FAIL+1))
    fi
}

echo "═══════════════════════════════════════"
echo "  WARPture Integration Tests"
echo "  Agent: $AGENT_URL"
echo "═══════════════════════════════════════"

# 1. Health check
check "health endpoint responds" \
    "curl -sf '$AGENT_URL/health' | python3 -c \"import sys,json; d=json.load(sys.stdin); print(d['status'])\"" \
    "ok"

# 2. Version endpoint
check "version endpoint" \
    "curl -sf '$AGENT_URL/version' | python3 -c \"import sys,json; d=json.load(sys.stdin); print('ok' if 'version' in d else 'fail')\"" \
    "ok"

# 3. Get WARP status
check "WARP status endpoint" \
    "curl -sf '$AGENT_URL/api/v1/warp/status' | python3 -c \"import sys,json; d=json.load(sys.stdin); print('ok' if 'status' in d else 'fail')\"" \
    "ok"

# 4. Get config
check "get config" \
    "curl -sf '$AGENT_URL/api/v1/config' | python3 -c \"import sys,json; d=json.load(sys.stdin); print('ok' if 'defaultPolicy' in d else 'fail')\"" \
    "ok"

# 5. List apps
check "list apps" \
    "curl -sf '$AGENT_URL/api/v1/apps' | python3 -c \"import sys,json; d=json.load(sys.stdin); print('ok' if isinstance(d,list) else 'fail')\"" \
    "ok"

# 6. Merge apps
check "merge apps" \
    "curl -sf -X POST '$AGENT_URL/api/v1/apps/merge' \
        -H 'Content-Type: application/json' \
        -d '[{\"id\":\"test-app\",\"name\":\"Test App\",\"path\":\"/Applications/Test.app\",\"policy\":\"default\",\"running\":false}]' \
     | python3 -c \"import sys,json; d=json.load(sys.stdin); print('ok' if d.get('ok') else 'fail')\"" \
    "ok"

# 7. Set app policy
check "set app policy" \
    "curl -sf -X POST '$AGENT_URL/api/v1/apps/policy' \
        -H 'Content-Type: application/json' \
        -d '{\"appId\":\"test-app\",\"policy\":\"include\"}' \
     | python3 -c \"import sys,json; d=json.load(sys.stdin); print('ok' if d.get('ok') else 'fail')\"" \
    "ok"

# 8. Apply preset
check "apply work preset" \
    "curl -sf -X POST '$AGENT_URL/api/v1/presets/apply' \
        -H 'Content-Type: application/json' \
        -d '{\"preset\":\"work\"}' \
     | python3 -c \"import sys,json; d=json.load(sys.stdin); print('ok' if d.get('ok') else 'fail')\"" \
    "ok"

# 9. Set default policy
check "set default policy to bypass" \
    "curl -sf -X POST '$AGENT_URL/api/v1/config/default-policy' \
        -H 'Content-Type: application/json' \
        -d '{\"policy\":\"bypass\"}' \
     | python3 -c \"import sys,json; d=json.load(sys.stdin); print('ok' if d.get('ok') else 'fail')\"" \
    "ok"

# 10. Get stats
check "get stats" \
    "curl -sf '$AGENT_URL/api/v1/stats' | python3 -c \"import sys,json; d=json.load(sys.stdin); print('ok')\"" \
    "ok"

echo ""
echo "═══════════════════════════════════════"
echo "  Results: $PASS passed, $FAIL failed"
echo "═══════════════════════════════════════"
[[ "$FAIL" -eq 0 ]] && exit 0 || exit 1
