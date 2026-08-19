#!/usr/bin/env bash
# Start the HomeGauge Go API. Reads runtime config from ./.env
# pm2 manages restarts; this script just execs the binary.
set -euo pipefail
cd "$(dirname "$0")"

ENV_FILE="${DEPLOY_REMOTE_ENV:-.env}"
if [[ -f "$ENV_FILE" ]]; then
  set -a
  # shellcheck disable=SC1090
  source <(grep -v '^\s*#' "$ENV_FILE" | grep -v '^\s*$' || true)
  set +a
fi

export APP_ENV="${APP_ENV:-production}"
# The binary expects to find the migrations directory beside it.
# We rsync migrations/ into DEPLOY_PATH/migrations so the auto-migrator finds them.
exec .build/homegauge-api
