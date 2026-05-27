#!/bin/sh
# gc dolt status — Check if the Dolt server is running.
#
# Exits 0 if the server is reachable, 1 otherwise.
# Lightweight status probe for manual checks and scripts; the dolt-health order
# uses structured `gc dolt health --json | gc dolt health-check` diagnostics.
#
# Environment: GC_CITY_PATH
set -e

: "${GC_CITY_PATH:?GC_CITY_PATH must be set}"
PACK_DIR="${GC_PACK_DIR:-$(CDPATH= cd -- "$(dirname "$0")/.." && pwd)}"
. "$PACK_DIR/assets/scripts/runtime.sh"

if [ ! -x "$GC_BEADS_BD_SCRIPT" ]; then
  echo "gc dolt status: gc-beads-bd not found" >&2
  exit 1
fi

probe_timeout="${GC_DOLT_STATUS_TIMEOUT:-5}"
port="${GC_DOLT_PORT:-}"
pid="$(managed_runtime_listener_pid "$port" || true)"
reachable=false
if managed_runtime_tcp_reachable "$port"; then
  reachable=true
fi

if [ -n "$pid" ] || [ "$reachable" = true ]; then
  [ -n "$pid" ] || pid="unknown"
  printf 'Server: running (PID %s, port %s, tcp_reachable=%s)\n' "$pid" "$port" "$reachable"
else
  printf 'Server: not running (port %s, tcp_reachable=false)\n' "$port"
fi

# probe exits 0 if running, 2 if not running. Keep it bounded so this
# command still writes useful diagnostics when Dolt itself is wedged. Capture
# via a temp file rather than command-substitution assignment so plain /bin/sh
# preserves the child exit status.
probe_output_file=$(mktemp "${TMPDIR:-/tmp}/gc-dolt-status.XXXXXX")
set +e
GC_CITY_PATH="$GC_CITY_PATH" run_bounded "$probe_timeout" "$GC_BEADS_BD_SCRIPT" probe >"$probe_output_file" 2>&1
probe_status=$?
set -e
probe_output=$(cat "$probe_output_file" 2>/dev/null || true)
rm -f "$probe_output_file"

if [ "$probe_status" -eq 0 ]; then
  printf 'Probe: ok\n'
  [ -n "$probe_output" ] && printf '%s\n' "$probe_output"
  exit 0
fi

case "$probe_status" in
  124)
    printf 'Probe: timed out after %ss\n' "$probe_timeout"
    ;;
  *)
    printf 'Probe: failed (exit %s)\n' "$probe_status"
    ;;
esac
[ -n "$probe_output" ] && printf '%s\n' "$probe_output"
exit 1
