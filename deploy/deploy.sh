#!/usr/bin/env bash
# Deploy HomeGauge Mortgage to themobileprof.com/mortgage
#
# Builds everything locally (Go API + Next standalone), rsyncs to the server,
# runs DB migrations, and restarts pm2 processes — no `npm run build` on the server.
#
# Usage:
#   bash deploy/deploy.sh
#   # or add to Makefile: make deploy
#
# Prerequisites (local):
#   - Go toolchain
#   - Node.js ≥ 20
#   - rsync, ssh
#   - .env.deploy  (copy from .env.deploy.example)
#   - .env.prod    (copy from .env.prod.example)
#
# Prerequisites (remote):
#   - pm2 installed globally (npm i -g pm2)
#   - PostgreSQL running; DB and user created
#   - Redis running
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

# ── Load deploy config ────────────────────────────────────────────────────────
ENV_FILE="${DEPLOY_ENV_FILE:-$ROOT/.env.deploy}"
if [[ ! -f "$ENV_FILE" ]]; then
  echo "ERROR: $ENV_FILE not found."
  echo "  cp .env.deploy.example .env.deploy  # then fill it in"
  exit 1
fi
set -a
# Strip full-line comments, blank lines, and inline comments; require KEY=VALUE form.
# shellcheck disable=SC1090
source <(grep -v '^\s*#' "$ENV_FILE" | grep -v '^\s*$' | grep '=' | sed 's/[[:space:]]*#.*//' || true)
set +a

: "${DEPLOY_USER:?}"
: "${DEPLOY_HOST:?}"
: "${DEPLOY_PATH:?}"
DEPLOY_PORT="${DEPLOY_PORT:-22}"
DEPLOY_PM2_API_NAME="${DEPLOY_PM2_API_NAME:-mortgage-api}"
DEPLOY_PM2_WEB_NAME="${DEPLOY_PM2_WEB_NAME:-mortgage-web}"
DEPLOY_MIGRATE="${DEPLOY_MIGRATE:-true}"
DEPLOY_SEED="${DEPLOY_SEED:-false}"

PROD_ENV="${DEPLOY_PROD_ENV_FILE:-$ROOT/.env.prod}"
if [[ ! -f "$PROD_ENV" ]]; then
  echo "ERROR: $PROD_ENV not found."
  echo "  cp .env.prod.example .env.prod  # then fill in secrets"
  exit 1
fi
if grep -Eq '^SESSION_SECRET=\s*$' "$PROD_ENV" || ! grep -Eq '^SESSION_SECRET=.' "$PROD_ENV"; then
  echo "ERROR: SESSION_SECRET is empty in $PROD_ENV — set a long random value first." >&2
  exit 1
fi

# ── SSH / rsync helpers ───────────────────────────────────────────────────────
SSH_OPTS=(-p "$DEPLOY_PORT" -o StrictHostKeyChecking=accept-new)
RSYNC_SSH="ssh -p $DEPLOY_PORT -o StrictHostKeyChecking=accept-new"
SCP_OPTS=(-P "$DEPLOY_PORT" -o StrictHostKeyChecking=accept-new)
if [[ -n "${DEPLOY_SSH_KEY:-}" ]]; then
  KEY="${DEPLOY_SSH_KEY/#\~/$HOME}"
  SSH_OPTS+=(-i "$KEY")
  SCP_OPTS+=(-i "$KEY")
  RSYNC_SSH="ssh -p $DEPLOY_PORT -i $KEY -o StrictHostKeyChecking=accept-new"
fi
REMOTE="${DEPLOY_USER}@${DEPLOY_HOST}"
TARGET="${REMOTE}:${DEPLOY_PATH}"

# ── Optional SSH tunnel for remote DB seed from laptop ───────────────────────
DB_TUNNEL_PID=""
cleanup_db_tunnel() {
  if [[ -n "${DB_TUNNEL_PID:-}" ]] && kill -0 "$DB_TUNNEL_PID" 2>/dev/null; then
    kill "$DB_TUNNEL_PID" 2>/dev/null || true
    wait "$DB_TUNNEL_PID" 2>/dev/null || true
  fi
}
trap cleanup_db_tunnel EXIT

ensure_db_tunnel() {
  local db_url="$1"
  local host port
  host="$(printf '%s' "$db_url" | sed -E 's|^[^:]+://([^@/]*@)?([^:/?]+).*|\2|')"
  port="$(printf '%s' "$db_url" | sed -E 's|^[^:]+://([^@/]*@)?[^:/?]+:([0-9]+).*|\2|')"
  [[ "$port" == "$db_url" ]] && port="5432"
  local tunnel_mode="${DEPLOY_DB_TUNNEL:-auto}"
  [[ "$tunnel_mode" == "false" || "$tunnel_mode" == "no" ]] && return 0
  [[ "$tunnel_mode" == "auto" && "$host" != "127.0.0.1" && "$host" != "localhost" ]] && return 0
  if (echo >/dev/tcp/127.0.0.1/"$port") >/dev/null 2>&1; then
    echo "==> DB tunnel port $port already open — reusing"
    return 0
  fi
  local remote_target="${DEPLOY_DB_TUNNEL_REMOTE:-127.0.0.1:5432}"
  echo "==> Opening SSH DB tunnel localhost:$port → $REMOTE:$remote_target"
  ssh -N -p "$DEPLOY_PORT" -o StrictHostKeyChecking=accept-new \
    -o ExitOnForwardFailure=yes -o ServerAliveInterval=30 \
    ${DEPLOY_SSH_KEY:+-i "${DEPLOY_SSH_KEY/#\~/$HOME}"} \
    -L "${port}:${remote_target}" "$REMOTE" &
  DB_TUNNEL_PID=$!
  for _ in $(seq 1 50); do
    kill -0 "$DB_TUNNEL_PID" 2>/dev/null || { echo "SSH tunnel exited early" >&2; exit 1; }
    (echo >/dev/tcp/127.0.0.1/"$port") >/dev/null 2>&1 && sleep 0.5 && return 0
    sleep 0.1
  done
  echo "SSH tunnel did not become ready on port $port" >&2; exit 1
}

# ── 1. Build Go API ───────────────────────────────────────────────────────────
echo "==> Building Go API (linux/amd64)"
GOARCH=amd64 GOOS=linux go -C backend build -o "$ROOT/.build/homegauge-api" ./cmd/api
GOARCH=amd64 GOOS=linux go -C backend build -o "$ROOT/.build/homegauge-seed" ./cmd/seed
echo "    binaries → .build/"

# ── 2. Build Next.js standalone ───────────────────────────────────────────────
echo "==> Building Next.js frontend (standalone, basePath=/mortgage)"
cd "$ROOT/frontend"
[[ -d node_modules ]] || npm ci
NEXT_PUBLIC_BASE_PATH=/mortgage npm run build
cd "$ROOT"

if [[ ! -f frontend/.next/standalone/server.js ]]; then
  echo "ERROR: frontend/.next/standalone/server.js not found after build." >&2
  exit 1
fi

# Copy static assets next to standalone server.js (Next doesn't bundle them)
cp -a frontend/.next/static  frontend/.next/standalone/.next/static
rm -rf                        frontend/.next/standalone/public
[[ -d frontend/public ]] && cp -a frontend/public frontend/.next/standalone/public

# ── 3. Optionally seed from laptop ───────────────────────────────────────────
if [[ "${DEPLOY_SEED}" == "true" ]]; then
  DB_URL="${REMOTE_DATABASE_URL:-}"
  [[ -z "$DB_URL" ]] && { echo "DEPLOY_SEED=true requires REMOTE_DATABASE_URL in .env.deploy" >&2; exit 1; }
  ensure_db_tunnel "$DB_URL"
  echo "==> Running seed against remote DB"
  DATABASE_URL="$DB_URL" .build/homegauge-seed
  cleanup_db_tunnel
fi

# ── 4. Rsync to server ────────────────────────────────────────────────────────
echo "==> Rsyncing to ${TARGET}"
ssh "${SSH_OPTS[@]}" "$REMOTE" "mkdir -p $(printf '%q' "$DEPLOY_PATH")/data/docs"

# Go binaries + migration files
rsync -az --delete -e "$RSYNC_SSH" \
  --exclude '.env' --exclude '.env.*' \
  "$ROOT/.build/"                   "$TARGET/.build/"
rsync -az --delete -e "$RSYNC_SSH" \
  "$ROOT/backend/migrations/"       "$TARGET/migrations/"

# Next standalone
rsync -az --delete -e "$RSYNC_SSH" \
  --exclude '.env' --exclude '.env.*' \
  "$ROOT/frontend/.next/standalone/"  "$TARGET/web/"

# Start scripts
rsync -az -e "$RSYNC_SSH" \
  "$ROOT/deploy/start-api.sh" \
  "$ROOT/deploy/start-web.sh" \
  "$TARGET/"

echo "==> Uploading .env.prod → $DEPLOY_PATH/.env"
scp "${SCP_OPTS[@]}" "$PROD_ENV" "${REMOTE}:$(printf '%q' "$DEPLOY_PATH/.env")"

# ── 5. Ensure pgcrypto extension (safe to run every time) ────────────────────
# The Go migration needs pgcrypto; a regular DB user can't CREATE EXTENSION.
# On a fresh server run once: sudo -u postgres psql -c 'CREATE EXTENSION IF NOT EXISTS pgcrypto;' homegauge
# We try non-interactively here; it is a no-op if already installed.
ssh "${SSH_OPTS[@]}" "$REMOTE" \
  "echo '${SUDO_PASS:-}' | sudo -S -u postgres psql -c 'CREATE EXTENSION IF NOT EXISTS pgcrypto;' homegauge 2>/dev/null || true"

# ── 6. Migrate + restart on remote ───────────────────────────────────────────
echo "==> Remote: migrate + restart"

REMOTE_SCRIPT=$(cat <<'EOS'
set -euo pipefail
cd "$DEPLOY_PATH"

# Bootstrap nvm / pm2 path
export NVM_DIR="${NVM_DIR:-$HOME/.nvm}"
[[ -s "$NVM_DIR/nvm.sh" ]] && . "$NVM_DIR/nvm.sh" && (nvm use default >/dev/null 2>&1 || nvm use 22 >/dev/null 2>&1 || true)
if ! command -v pm2 >/dev/null 2>&1; then
  for d in "$HOME"/.nvm/versions/node/*/bin; do [[ -x "$d/pm2" ]] && export PATH="$d:$PATH" && break; done
fi

chmod 600 .env 2>/dev/null || true
chmod +x .build/homegauge-api .build/homegauge-seed start-api.sh start-web.sh 2>/dev/null || true

# Load env so the binary can read DATABASE_URL
set -a
source <(grep -v '^\s*#' .env | grep -v '^\s*$' || true)
set +a

if [[ "${DEPLOY_MIGRATE}" == "true" ]]; then
  echo "==> Running DB migrations"
  # The Go API auto-migrates on startup; we can also run the seed binary in migrate-only mode.
  # Simplest: just let the API migrate on first start. Nothing to do here unless you add a
  # separate migrate binary. The API will apply pending migrations before serving.
  echo "    (migrations will apply when API starts)"
fi

if [[ "${DEPLOY_SEED_REMOTE}" == "true" ]]; then
  echo "==> Running seed on remote"
  .build/homegauge-seed
fi

pm2_restart() {
  local name="$1"; local script="$2"
  if pm2 describe "$name" >/dev/null 2>&1; then
    pm2 restart "$name" --update-env
  else
    pm2 start "$script" --name "$name"
    pm2 save
  fi
}

if command -v pm2 >/dev/null 2>&1; then
  pm2_restart "$DEPLOY_PM2_API_NAME" "./start-api.sh"
  pm2_restart "$DEPLOY_PM2_WEB_NAME" "./start-web.sh"
  echo "==> pm2 status"
  pm2 ls
else
  echo "pm2 not found — start manually:"
  echo "  cd $DEPLOY_PATH && ./start-api.sh &"
  echo "  cd $DEPLOY_PATH && ./start-web.sh &"
fi

echo "==> Remote deploy finished"
EOS
)

ssh "${SSH_OPTS[@]}" "$REMOTE" \
  "DEPLOY_PATH=$(printf '%q' "$DEPLOY_PATH") \
   DEPLOY_PM2_API_NAME=$(printf '%q' "$DEPLOY_PM2_API_NAME") \
   DEPLOY_PM2_WEB_NAME=$(printf '%q' "$DEPLOY_PM2_WEB_NAME") \
   DEPLOY_MIGRATE=$(printf '%q' "$DEPLOY_MIGRATE") \
   DEPLOY_SEED_REMOTE=false \
   bash -s" <<<"$REMOTE_SCRIPT"

echo ""
echo "✓  Deploy complete"
echo "   https://themobileprof.com/mortgage"
