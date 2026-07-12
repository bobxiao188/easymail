#!/usr/bin/env bash
# =============================================================================
# EasyMail control script — macOS / Darwin
# Pre-flight checks before launching, supports start / stop / status.
#
# Usage:
#   ./easymail.sh start                 # start the service
#   ./easymail.sh stop                  # stop the service
#   ./easymail.sh status                # check running state
#   EASYMAIL_HOME=/app ./easymail.sh start   # custom install path
# =============================================================================

set -euo pipefail

EASYMAIL_HOME="${EASYMAIL_HOME:-/opt/easymail}"
CONFIG="$EASYMAIL_HOME/config/easymail.yaml"
BINARY="$EASYMAIL_HOME/bin/easymail"
PIDFILE="/tmp/easymail.pid"

# ---------------------------------------------------------------------------
# Color helpers
# ---------------------------------------------------------------------------
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

info()  { echo -e "${GREEN}[INFO]${NC}  $1"; }
warn()  { echo -e "${YELLOW}[WARN]${NC}  $1"; }
fatal() { echo -e "${RED}[FATAL]${NC} $1" >&2; exit 1; }

# ---------------------------------------------------------------------------
# status — check if running and return the PID
# ---------------------------------------------------------------------------
status() {
    if [ ! -f "$PIDFILE" ]; then
        return 1
    fi
    local pid
    pid=$(cat "$PIDFILE" 2>/dev/null || echo "")
    if [ -z "$pid" ]; then
        return 1
    fi
    if kill -0 "$pid" 2>/dev/null; then
        echo "$pid"
        return 0
    fi
    # Stale PID
    rm -f "$PIDFILE"
    return 1
}

# ---------------------------------------------------------------------------
# start — pre-flight checks then launch
# ---------------------------------------------------------------------------
start() {
    if status >/dev/null; then
        local pid
        pid=$(status)
        fatal "EasyMail is already running (PID $pid). Use '$0 stop' first."
    fi

    info "EasyMail pre-flight checks ..."

    # 1. Config file
    if [ ! -f "$CONFIG" ]; then
        fatal "Config file not found: $CONFIG"
    fi
    info "  Config: $CONFIG"

    # 2. Binary
    if [ ! -x "$BINARY" ]; then
        fatal "Binary not found or not executable: $BINARY"
    fi
    local ver
    ver=$("$BINARY" --version 2>/dev/null || echo "version info unavailable")
    info "  Binary: $BINARY ($ver)"

    # 3. Required directories
    for dir in logs storage; do
        local path="$EASYMAIL_HOME/$dir"
        if [ ! -d "$path" ]; then
            warn "  Directory missing, creating: $path"
            mkdir -p "$path"
        else
            info "  Directory: $path"
        fi
    done

    # 4. Launch — exec replaces the script process so systemd (Type=simple) sees the binary directly
    echo $$ > "$PIDFILE"
    info "Starting EasyMail (PID $$) ..."
    cd "$EASYMAIL_HOME"
    exec "$BINARY" -config "$CONFIG"
}

# ---------------------------------------------------------------------------
# stop — graceful shutdown
# ---------------------------------------------------------------------------
stop() {
    if ! status >/dev/null; then
        warn "EasyMail is not running"
        rm -f "$PIDFILE"
        return 0
    fi

    local pid
    pid=$(status)
    info "Stopping EasyMail (PID $pid) ..."
    kill "$pid" 2>/dev/null || true

    # Wait up to 30s for graceful shutdown
    local waited=0
    while kill -0 "$pid" 2>/dev/null; do
        sleep 1
        waited=$((waited + 1))
        if [ $waited -ge 30 ]; then
            warn "Graceful shutdown timed out, forcing kill ..."
            kill -9 "$pid" 2>/dev/null || true
            break
        fi
    done

    rm -f "$PIDFILE"
    info "EasyMail stopped"
}

# ---------------------------------------------------------------------------
# Main
# ---------------------------------------------------------------------------
case "${1:-}" in
    start)
        start
        ;;
    stop)
        stop
        ;;
    status)
        if status >/dev/null; then
            pid=$(status)
            echo "EasyMail is running (PID $pid)"
            exit 0
        else
            echo "EasyMail is not running"
            exit 3
        fi
        ;;
    restart)
        stop
        start
        ;;
    *)
        echo "Usage: $0 {start|stop|status|restart}"
        exit 1
        ;;
esac