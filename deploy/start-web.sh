#!/usr/bin/env bash
# Start the HomeGauge Next.js standalone server. Reads runtime config from ./.env
# pm2 manages restarts; this script just execs node server.js
set -euo pipefail
cd "$(dirname "$0")"

# Prefer nvm Node (system node may be too old for Next 15+)
export NVM_DIR="${NVM_DIR:-$HOME/.nvm}"
if [[ -s "$NVM_DIR/nvm.sh" ]]; then
  # shellcheck disable=SC1091
  . "$NVM_DIR/nvm.sh"
  nvm use default >/dev/null 2>&1 || nvm use 22 >/dev/null 2>&1 || true
fi

ENV_FILE="${DEPLOY_REMOTE_ENV:-.env}"
if [[ -f "$ENV_FILE" ]]; then
  set -a
  # shellcheck disable=SC1090
  source <(grep -v '^\s*#' "$ENV_FILE" | grep -v '^\s*$' || true)
  set +a
fi

export NODE_ENV=production
export PORT="${PORT:-3035}"
export HOSTNAME=0.0.0.0
# Tell Next where the API lives (it proxies /api/* internally)
export API_PROXY_TARGET="${API_PROXY_TARGET:-http://127.0.0.1:8085}"

exec node web/server.js
