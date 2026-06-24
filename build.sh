#!/usr/bin/env bash

# SWM project build script
# Usage: ./build.sh [options]

set -euo pipefail

say() {
  printf '%s\n' "$*"
}

PROJECT_ROOT="$(cd "$(dirname "$0")" && pwd)"
RELEASE_DIR="$PROJECT_ROOT/release"
BUILD_WORKER_COMPAT="${BUILD_WORKER_COMPAT:-true}"
TARGET_OS="${TARGET_OS:-linux}"
TARGET_ARCH="${TARGET_ARCH:-amd64}"
CGO_ENABLED="${CGO_ENABLED:-0}"
VITE_API_BASE="${VITE_API_BASE:-http://localhost:8080}"

SERVER_BINARY="swm-server"
WORKER_BINARY="swm-worker"

print_usage() {
  cat <<'EOF'
Usage: ./build.sh [options]

Options:
  --target-os <os>              Go target OS, default linux. Common: linux/windows
  --target-arch <arch>          Go target arch, default amd64. Common: amd64/arm64
  --cgo-enabled <0|1>           CGO_ENABLED, default 0
  --api-base <url>              Frontend production API base, e.g. https://api.example.com
  --vite-api-base <url>         Same as --api-base
  --build-worker-compat <bool>  Build swm-worker, default true
  -h, --help                    Show help

Env vars are also supported:
  TARGET_OS=linux TARGET_ARCH=amd64 CGO_ENABLED=0 VITE_API_BASE=https://api.example.com ./build.sh
EOF
}

read_option_value() {
  local option="$1"
  local value="${2:-}"
  if [ -z "$value" ]; then
    say "Missing value for option: $option" >&2
    exit 1
  fi
  printf '%s' "$value"
}

# `./build.sh genkeys [key-id]` generates a fresh Ed25519 device-authorization
# keypair and prints both halves. Run this ONCE per environment / rotation — never
# as part of the normal build, because the public half is baked into the client
# and regenerating it would lock out every already-distributed client.
if [ "${1:-}" = "genkeys" ]; then
  shift
  KEY_ID_ARG=""
  if [ "${1:-}" != "" ]; then
    KEY_ID_ARG="--key-id=$1"
  fi
  cd "$PROJECT_ROOT/backend"
  go run ./cmd/genauthzkey $KEY_ID_ARG
  exit 0
fi

while [ "$#" -gt 0 ]; do
  case "$1" in
    --target-os=*) TARGET_OS="${1#*=}" ;;
    --target-os) TARGET_OS="$(read_option_value "$1" "${2:-}")"; shift ;;
    --target-arch=*) TARGET_ARCH="${1#*=}" ;;
    --target-arch) TARGET_ARCH="$(read_option_value "$1" "${2:-}")"; shift ;;
    --cgo-enabled=*) CGO_ENABLED="${1#*=}" ;;
    --cgo-enabled) CGO_ENABLED="$(read_option_value "$1" "${2:-}")"; shift ;;
    --api-base=*) VITE_API_BASE="${1#*=}" ;;
    --api-base) VITE_API_BASE="$(read_option_value "$1" "${2:-}")"; shift ;;
    --vite-api-base=*) VITE_API_BASE="${1#*=}" ;;
    --vite-api-base) VITE_API_BASE="$(read_option_value "$1" "${2:-}")"; shift ;;
    --build-worker-compat=*) BUILD_WORKER_COMPAT="${1#*=}" ;;
    --build-worker-compat) BUILD_WORKER_COMPAT="$(read_option_value "$1" "${2:-}")"; shift ;;
    -h|--help)
      print_usage
      exit 0
      ;;
    *)
      say "Unknown option: $1" >&2
      say "Run ./build.sh --help for usage." >&2
      exit 1
      ;;
  esac
  shift
done

if [ "$TARGET_OS" = "windows" ]; then
  SERVER_BINARY="swm-server.exe"
  WORKER_BINARY="swm-worker.exe"
fi

require_cmd() {
  if ! command -v "$1" >/dev/null 2>&1; then
    say "Missing required command: $1" >&2
    exit 1
  fi
}

say "========================================"
say "   SWM build script"
say "========================================"
say

say "Build config:"
say "  Target:       ${TARGET_OS}/${TARGET_ARCH}"
say "  CGO_ENABLED:  ${CGO_ENABLED}"
say "  VITE_API_BASE: ${VITE_API_BASE}"
say "  BUILD_WORKER_COMPAT: ${BUILD_WORKER_COMPAT}"
say

say "[0/6] Checking build dependencies..."
require_cmd go
require_cmd npm
say "Done: dependencies ok"
say

say "[1/6] Cleaning old release files..."
if [ -d "$RELEASE_DIR" ]; then
  rm -rf "$RELEASE_DIR"/*
  say "Done: cleaned release/"
else
  mkdir -p "$RELEASE_DIR"
  say "Done: created release/"
fi

mkdir -p "$RELEASE_DIR/web" "$RELEASE_DIR/migrations"
say

say "[2/6] Building frontend..."
cd "$PROJECT_ROOT/web"

if [ ! -d "node_modules" ]; then
  say "  -> Installing dependencies..."
  npm install
fi

say "  -> Writing production env VITE_API_BASE=$VITE_API_BASE..."
cat > .env.production << EOF
# Production config
# Set this to the real backend API base when deploying
VITE_API_BASE=$VITE_API_BASE
EOF

say "  -> Building production bundle..."
npm run build

cp -r dist/* "$RELEASE_DIR/web/"
say "Done: frontend built"
say

say "[3/6] Building API server..."
cd "$PROJECT_ROOT/backend"

say "  -> Compiling $SERVER_BINARY (${TARGET_OS}/${TARGET_ARCH})..."
GOOS="$TARGET_OS" GOARCH="$TARGET_ARCH" CGO_ENABLED="$CGO_ENABLED" \
  go build -trimpath -ldflags="-s -w" -o "$RELEASE_DIR/$SERVER_BINARY" ./cmd/api

say "Done: API server built"
say

say "[4/6] Building worker (optional)..."
if [ "$BUILD_WORKER_COMPAT" = "true" ]; then
  say "  -> Compiling $WORKER_BINARY (${TARGET_OS}/${TARGET_ARCH})..."
  GOOS="$TARGET_OS" GOARCH="$TARGET_ARCH" CGO_ENABLED="$CGO_ENABLED" \
    go build -trimpath -ldflags="-s -w" -o "$RELEASE_DIR/$WORKER_BINARY" ./cmd/worker
  say "Done: worker built"
else
  say "  -> Skipping swm-worker (single-binary mode)"
fi
say

say "[5/6] Copying database migrations..."
cp "$PROJECT_ROOT/backend/migrations/"*.sql "$RELEASE_DIR/migrations/"
say "Done: migrations copied"
say

say "[6/6] Copying ip2region data files..."
GEO_SRC="$PROJECT_ROOT/backend/third_party/ip2region/data"
GEO_DST="$RELEASE_DIR/third_party/ip2region/data"
mkdir -p "$GEO_DST"
cp "$GEO_SRC/"*.csv "$GEO_DST/"
cp "$GEO_SRC/"*.xdb "$GEO_DST/"
say "Done: ip2region data copied"
say

say "========================================"
say "   Build complete"
say "========================================"
say

say "Artifacts:"
say

cd "$RELEASE_DIR"

if [ -f "$SERVER_BINARY" ]; then
  SERVER_SIZE=$(du -h "$SERVER_BINARY" | cut -f1)
  say "  * $SERVER_BINARY          $SERVER_SIZE"
fi

if [ -f "$WORKER_BINARY" ]; then
  WORKER_SIZE=$(du -h "$WORKER_BINARY" | cut -f1)
  say "  * $WORKER_BINARY          $WORKER_SIZE"
fi

if [ -d "web" ]; then
  WEB_SIZE=$(du -sh web | cut -f1)
  say "  * web/                $WEB_SIZE"
fi

MIGRATION_COUNT=$(ls -1 migrations/*.sql 2>/dev/null | wc -l)
say "  * migrations/         $MIGRATION_COUNT SQL files"

if [ -d "third_party/ip2region/data" ]; then
  GEO_SIZE=$(du -sh third_party/ip2region/data | cut -f1)
  say "  * third_party/        $GEO_SIZE (ip2region data)"
fi

say
say "Directory tree:"
find "$RELEASE_DIR" -type f | sort | sed "s|$RELEASE_DIR|  release|"
say

TOTAL_SIZE=$(du -sh "$RELEASE_DIR" | cut -f1)
say "Total size: $TOTAL_SIZE"
say

say "========================================"
say "   Deploy guide"
say "========================================"
say
say "1. Upload to server:"
say "   scp -r release/ root@your-server:/opt/swm/"
say
say "2. Run on server:"
say "   cd /opt/swm"
say "   export DATABASE_URL=\"user:pass@tcp(localhost:3306)/dbname?charset=utf8mb4&parseTime=true\""
say "   export JWT_SECRET=\"your-secret-key\""
say "   export RUN_MIGRATIONS=true"
say "   export ENABLE_EMBEDDED_WORKER=true"
say "   export WORKER_INTERVAL_SECONDS=3600"
say "   ./swm-server"
say "   For single-process (e.g. aaPanel) deploy, run swm-server only"
say
say "3. Nginx config for frontend:"
say "   location / {"
say "       root /opt/swm/web;"
say "       try_files \$uri \$uri/ /index.html;"
say "   }"
say
say "Build script finished."
