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

# Runtime .env generation inputs (written to release/.env after the build so the
# artifact is directly runnable: swm-server loads ./.env via godotenv).
GENERATE_ENV="${GENERATE_ENV:-true}"
APP_ENV="${APP_ENV:-dev}"
HTTP_ADDR="${HTTP_ADDR:-:8080}"
WEB_BASE_URL="${WEB_BASE_URL:-http://localhost:5173}"
DATABASE_URL="${DATABASE_URL:-user:pass@tcp(localhost:3306)/swmanager?charset=utf8mb4&parseTime=true&loc=Local&multiStatements=true}"
JWT_SECRET="${JWT_SECRET:-}"
JWT_ISSUER="${JWT_ISSUER:-swm}"
APP_SECRET_MASTER_KEY="${APP_SECRET_MASTER_KEY:-}"
CORS_ORIGINS="${CORS_ORIGINS:-*}"
LOCAL_PUBLIC_BASE_URL="${LOCAL_PUBLIC_BASE_URL:-}"
ESA_GEO_HEADERS_TRUSTED="${ESA_GEO_HEADERS_TRUSTED:-false}"
PREFER_SERVERSIDE_REGION="${PREFER_SERVERSIDE_REGION:-true}"

SERVER_BINARY="swm-server"
WORKER_BINARY="swm-worker"

print_usage() {
  cat <<'EOF'
Usage: ./build.sh [options]

Options:
  --target-os <os>              Go target OS, default linux. Common: linux/windows
  --target-arch <arch>          Go target arch, default amd64. Common: amd64/arm64
  --cgo-enabled <0|1>           CGO_ENABLED, default 0
  --api-base <url>              后端 API 域名（前端打包用 + 派生 LOCAL_PUBLIC_BASE_URL），e.g. https://api.example.com
  --vite-api-base <url>         Same as --api-base
  --build-worker-compat <bool>  Build swm-worker, default true
  -h, --help                    Show help

Runtime .env generation (写入 release/.env)：
  --web-base <url>              前端站点域名 → WEB_BASE_URL（SSO 回跳用），e.g. https://app.example.com
  --database-url <dsn>          数据库 DSN → DATABASE_URL
  --jwt-secret <s>              JWT_SECRET（prod 必须为 >=32 位强随机值）
  --app-secret-key <s>          APP_SECRET_MASTER_KEY（加密 app secret，勿丢失）
  --app-env <dev|prod>          APP_ENV，default dev
  --http-addr <addr>            HTTP_ADDR，default :8080
  --cors-origins <csv>          CORS_ORIGINS，default *
  --local-public-base <url>     LOCAL_PUBLIC_BASE_URL，default <api-base>/files
  --esa-trusted <bool>          ESA_GEO_HEADERS_TRUSTED，default false（仅流量必经 ESA 时设 true）
  --prefer-serverside-region <bool>  PREFER_SERVERSIDE_REGION，default true（服务端优先/防伪造）
  --no-env                      不生成 release/.env

Env vars are also supported (same names as the flags' targets):
  TARGET_OS=linux TARGET_ARCH=amd64 VITE_API_BASE=https://api.example.com WEB_BASE_URL=https://app.example.com ./build.sh
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
    --web-base=*) WEB_BASE_URL="${1#*=}" ;;
    --web-base) WEB_BASE_URL="$(read_option_value "$1" "${2:-}")"; shift ;;
    --database-url=*) DATABASE_URL="${1#*=}" ;;
    --database-url) DATABASE_URL="$(read_option_value "$1" "${2:-}")"; shift ;;
    --jwt-secret=*) JWT_SECRET="${1#*=}" ;;
    --jwt-secret) JWT_SECRET="$(read_option_value "$1" "${2:-}")"; shift ;;
    --app-secret-key=*) APP_SECRET_MASTER_KEY="${1#*=}" ;;
    --app-secret-key) APP_SECRET_MASTER_KEY="$(read_option_value "$1" "${2:-}")"; shift ;;
    --app-env=*) APP_ENV="${1#*=}" ;;
    --app-env) APP_ENV="$(read_option_value "$1" "${2:-}")"; shift ;;
    --http-addr=*) HTTP_ADDR="${1#*=}" ;;
    --http-addr) HTTP_ADDR="$(read_option_value "$1" "${2:-}")"; shift ;;
    --cors-origins=*) CORS_ORIGINS="${1#*=}" ;;
    --cors-origins) CORS_ORIGINS="$(read_option_value "$1" "${2:-}")"; shift ;;
    --local-public-base=*) LOCAL_PUBLIC_BASE_URL="${1#*=}" ;;
    --local-public-base) LOCAL_PUBLIC_BASE_URL="$(read_option_value "$1" "${2:-}")"; shift ;;
    --esa-trusted=*) ESA_GEO_HEADERS_TRUSTED="${1#*=}" ;;
    --esa-trusted) ESA_GEO_HEADERS_TRUSTED="$(read_option_value "$1" "${2:-}")"; shift ;;
    --prefer-serverside-region=*) PREFER_SERVERSIDE_REGION="${1#*=}" ;;
    --prefer-serverside-region) PREFER_SERVERSIDE_REGION="$(read_option_value "$1" "${2:-}")"; shift ;;
    --no-env) GENERATE_ENV="false" ;;
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
say "  GENERATE_ENV:  ${GENERATE_ENV}"
if [ "$GENERATE_ENV" = "true" ]; then
  say "  APP_ENV:       ${APP_ENV}"
  say "  HTTP_ADDR:     ${HTTP_ADDR}"
  say "  WEB_BASE_URL:  ${WEB_BASE_URL}"
  say "  ESA_GEO_HEADERS_TRUSTED: ${ESA_GEO_HEADERS_TRUSTED}"
  say "  PREFER_SERVERSIDE_REGION: ${PREFER_SERVERSIDE_REGION}"
fi
say

say "[0/7] Checking build dependencies..."
require_cmd go
require_cmd npm
say "Done: dependencies ok"
say

say "[1/7] Cleaning old release files..."
if [ -d "$RELEASE_DIR" ]; then
  rm -rf "$RELEASE_DIR"/*
  say "Done: cleaned release/"
else
  mkdir -p "$RELEASE_DIR"
  say "Done: created release/"
fi

mkdir -p "$RELEASE_DIR/web" "$RELEASE_DIR/migrations"
say

say "[2/7] Building frontend..."
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

say "[3/7] Building API server..."
cd "$PROJECT_ROOT/backend"

say "  -> Compiling $SERVER_BINARY (${TARGET_OS}/${TARGET_ARCH})..."
GOOS="$TARGET_OS" GOARCH="$TARGET_ARCH" CGO_ENABLED="$CGO_ENABLED" \
  go build -trimpath -ldflags="-s -w" -o "$RELEASE_DIR/$SERVER_BINARY" ./cmd/api

say "Done: API server built"
say

say "[4/7] Building worker (optional)..."
if [ "$BUILD_WORKER_COMPAT" = "true" ]; then
  say "  -> Compiling $WORKER_BINARY (${TARGET_OS}/${TARGET_ARCH})..."
  GOOS="$TARGET_OS" GOARCH="$TARGET_ARCH" CGO_ENABLED="$CGO_ENABLED" \
    go build -trimpath -ldflags="-s -w" -o "$RELEASE_DIR/$WORKER_BINARY" ./cmd/worker
  say "Done: worker built"
else
  say "  -> Skipping swm-worker (single-binary mode)"
fi
say

say "[5/7] Copying database migrations..."
cp "$PROJECT_ROOT/backend/migrations/"*.sql "$RELEASE_DIR/migrations/"
say "Done: migrations copied"
say

say "[6/7] Copying ip2region data files..."
GEO_SRC="$PROJECT_ROOT/backend/third_party/ip2region/data"
GEO_DST="$RELEASE_DIR/third_party/ip2region/data"
mkdir -p "$GEO_DST"
cp "$GEO_SRC/"*.csv "$GEO_DST/"
cp "$GEO_SRC/"*.xdb "$GEO_DST/"
say "Done: ip2region data copied"
say

say "[7/7] Generating runtime .env..."
if [ "$GENERATE_ENV" = "true" ]; then
  if [ -z "$LOCAL_PUBLIC_BASE_URL" ]; then
    LOCAL_PUBLIC_BASE_URL="${VITE_API_BASE%/}/files"
  fi
  # Heredoc is unquoted so $VARS expand; values may contain & ( ) etc. which are
  # literal inside a heredoc. Secrets are written as-is (release/ is a deploy
  # artifact and must stay out of version control).
  cat > "$RELEASE_DIR/.env" << EOF
# Generated by build.sh — runtime config for swm-server (loaded via ./.env).
# Edit as needed. For APP_ENV=prod, JWT_SECRET / APP_SECRET_MASTER_KEY MUST be
# strong (>=32 chars) or the server refuses to start.
APP_ENV=$APP_ENV
HTTP_ADDR=$HTTP_ADDR
DATABASE_URL=$DATABASE_URL
JWT_SECRET=$JWT_SECRET
JWT_ISSUER=$JWT_ISSUER
APP_SECRET_MASTER_KEY=$APP_SECRET_MASTER_KEY
STORAGE_DRIVER=local
LOCAL_STORAGE_PATH=./data/files
LOCAL_PUBLIC_BASE_URL=$LOCAL_PUBLIC_BASE_URL
WEB_BASE_URL=$WEB_BASE_URL
RUN_MIGRATIONS=true
ENABLE_EMBEDDED_WORKER=true
WORKER_INTERVAL_SECONDS=3600
IP2REGION_V4_XDB_PATH=./third_party/ip2region/data/ip2region_v4.xdb
IP2REGION_V6_XDB_PATH=./third_party/ip2region/data/ip2region_v6.xdb
IP2REGION_REGION_CSV_PATH=./third_party/ip2region/data/global_region.csv
CORS_ORIGINS=$CORS_ORIGINS

# --- 阿里云 ESA 托管转换头（真实 IP / 地域识别）---
# 仅当回源流量"必经 ESA"时设 true，否则可被客户端伪造绕过地域限制。
ESA_GEO_HEADERS_TRUSTED=$ESA_GEO_HEADERS_TRUSTED
ESA_REAL_IP_HEADER=ali-real-client-ip
ESA_IP_COUNTRY_HEADER=ali-ip-country
ESA_IP_CITY_HEADER=ali-ip-city
# 服务端优先（防伪造）：true=ESA>ip2region>客户端自报；false=旧版客户端自报最优先。
PREFER_SERVERSIDE_REGION=$PREFER_SERVERSIDE_REGION
EOF
  say "Done: wrote release/.env (APP_ENV=$APP_ENV, HTTP_ADDR=$HTTP_ADDR)"
  if [ "$APP_ENV" = "prod" ]; then
    if [ -z "$JWT_SECRET" ] || [ "${#JWT_SECRET}" -lt 32 ]; then
      say "  !! WARNING: APP_ENV=prod 但 JWT_SECRET 为空或<32位 — 服务端将拒绝启动，请用 --jwt-secret 提供强随机值。"
    fi
    if [ -z "$APP_SECRET_MASTER_KEY" ] || [ "${#APP_SECRET_MASTER_KEY}" -lt 32 ]; then
      say "  !! WARNING: APP_ENV=prod 但 APP_SECRET_MASTER_KEY 为空或<32位 — 请用 --app-secret-key 提供强随机值。"
    fi
  fi
else
  say "  -> Skipping .env generation (--no-env)"
fi
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
say "   # release/.env 已由本脚本生成并随包上传；按需核对/修改后直接启动："
say "   ./swm-server"
say "   # swm-server 会自动加载同目录的 .env（godotenv）。prod 务必填强密钥。"
say "   For single-process (e.g. aaPanel) deploy, run swm-server only"
say
say "   接入阿里云 ESA 后（流量必经 ESA）："
say "   - ESA 控制台开启「添加真实客户端IP标头」「添加访问者位置标头」"
say "   - 在 .env 设 ESA_GEO_HEADERS_TRUSTED=true（确保源站只接受 ESA 回源，防伪造）"
say
say "3. Nginx config for frontend:"
say "   location / {"
say "       root /opt/swm/web;"
say "       try_files \$uri \$uri/ /index.html;"
say "   }"
say
say "Build script finished."
