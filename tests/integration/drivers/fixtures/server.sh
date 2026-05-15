#!/usr/bin/env bash
# Start a python http.server on a free port. Echoes the port.
set -euo pipefail
DIR="$(cd "$(dirname "$0")" && pwd)"
PORT=$(python3 -c 'import socket; s=socket.socket(); s.bind(("",0)); print(s.getsockname()[1]); s.close()')
cd "$DIR" && python3 -m http.server "$PORT" >/dev/null 2>&1 &
echo $! > /tmp/gpowers-fixture-server.pid
echo "$PORT"
