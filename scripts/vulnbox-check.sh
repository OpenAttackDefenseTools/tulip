#!/bin/bash
# Sunflower vulnbox healthcheck — run from ~/sunflower inside the box.
# Verifies: docker, stack status, .env paths, capture, ingest, dashboard.
# Exit 0 = all good, 1 = something needs attention.
set -u
cd "$(dirname "$0")/.." || exit 1

PASS=0; FAIL=0
ok()   { PASS=$((PASS+1)); echo "  [OK] $1"; }
fail() { FAIL=$((FAIL+1)); echo "  [FAIL] $1"; }

echo "== box =="
echo "  ips: $(hostname -I)"
command -v docker >/dev/null && ok "docker present ($(docker --version | head -n1))" || fail "docker missing"
sudo systemctl is-active --quiet docker && ok "docker daemon running" || fail "docker daemon not running"

echo "== stack =="
if sudo docker compose ps --format '{{.Service}} {{.Status}}' 2>/dev/null | grep -q .; then
    sudo docker compose ps --format '  {{.Service}}: {{.Status}}'
    sudo docker compose ps --format '{{.Status}}' | grep -qv "^Up" && fail "some containers not Up" || ok "all containers Up"
else
    fail "stack not running (docker compose up -d)"
fi

echo "== config =="
if [ -f .env ]; then
    # shellcheck disable=SC1091
    set -a; source .env; set +a
    case "${TRAFFIC_DIR_HOST:-}" in
        /*) ok "TRAFFIC_DIR_HOST is absolute: $TRAFFIC_DIR_HOST" ;;
        *)  fail "TRAFFIC_DIR_HOST must be absolute, got: '${TRAFFIC_DIR_HOST:-unset}'" ;;
    esac
else
    fail ".env missing (cp .env.example .env)"
fi

echo "== capture =="
if pgrep -a tcpdump | grep -q pcap; then
    ok "tcpdump running: $(pgrep -a tcpdump | head -n1)"
else
    fail "tcpdump not running (run ./capture.sh or /opt/vulnbox/capture.sh)"
fi
if [ -n "${TRAFFIC_DIR_HOST:-}" ] && [ -d "$TRAFFIC_DIR_HOST" ]; then
    N=$(ls "$TRAFFIC_DIR_HOST"/*.pcap 2>/dev/null | wc -l)
    [ "$N" -gt 0 ] && ok "$N pcap(s) in $TRAFFIC_DIR_HOST" || fail "no pcaps in $TRAFFIC_DIR_HOST"
    NEWEST=$(ls -t "$TRAFFIC_DIR_HOST"/*.pcap 2>/dev/null | head -n1)
    if [ -n "$NEWEST" ]; then
        AGE=$(( $(date +%s) - $(stat -c %Y "$NEWEST") ))
        [ "$AGE" -lt 600 ] && ok "newest pcap ${AGE}s old" || fail "newest pcap ${AGE}s old (capture stalled?)"
    fi
else
    fail "traffic dir missing: ${TRAFFIC_DIR_HOST:-unset}"
fi

echo "== ingest =="
if sudo docker compose ps -q assembler 2>/dev/null | grep -q .; then
    sudo docker exec "$(sudo docker compose ps -q assembler)" getent hosts timescale >/dev/null 2>&1 \
        && ok "assembler resolves timescale" \
        || fail "assembler cannot resolve timescale (is timescale container Up?)"
    LOG=$(sudo docker compose logs --tail=30 assembler 2>/dev/null | grep -E "Copied|Unable to connect" | tail -n1)
    echo "  last ingest: $LOG"
    echo "$LOG" | grep -q "Unable" && fail "assembler cannot reach DB" || ok "assembler ingesting"
fi

echo "== frontends =="
curl -s -m 5 -o /dev/null http://127.0.0.1:3000/ \
    && ok "sunflower dashboard on :3000" \
    || fail "dashboard not on :3000"
curl -s -m 5 -o /dev/null http://127.0.0.1:8080/ \
    && ok "service on :8080" \
    || echo "  [INFO] nothing on :8080 (adjust port if your service differs)"

echo
echo "result: $PASS ok, $FAIL failed"
[ "$FAIL" -eq 0 ]
