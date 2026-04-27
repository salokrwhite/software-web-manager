#!/bin/bash

# SWM 项目一键打包脚本
# 用法: ./build.sh

set -e  # 遇到错误立即退出

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# 项目根目录
PROJECT_ROOT="$(cd "$(dirname "$0")" && pwd)"
RELEASE_DIR="$PROJECT_ROOT/release"
BUILD_WORKER_COMPAT="${BUILD_WORKER_COMPAT:-true}"
# 生产环境 API 地址（可通过环境变量配置）
VITE_API_BASE="${VITE_API_BASE:-http://localhost:8080}"

echo -e "${BLUE}========================================${NC}"
echo -e "${BLUE}   SWM 项目一键打包脚本${NC}"
echo -e "${BLUE}========================================${NC}"
echo ""

# 步骤 1: 清理旧的打包文件
echo -e "${YELLOW}[1/5] 清理旧的打包文件...${NC}"
if [ -d "$RELEASE_DIR" ]; then
    rm -rf "$RELEASE_DIR"/*
    echo -e "${GREEN}✓ 已清理 release/ 目录${NC}"
else
    mkdir -p "$RELEASE_DIR"
    echo -e "${GREEN}✓ 创建 release/ 目录${NC}"
fi

# 创建子目录
mkdir -p "$RELEASE_DIR/web" "$RELEASE_DIR/migrations"
echo ""

# 步骤 2: 打包前端
echo -e "${YELLOW}[2/5] 打包前端项目...${NC}"
cd "$PROJECT_ROOT/web"

# 检查 node_modules 是否存在
if [ ! -d "node_modules" ]; then
    echo -e "${BLUE}  → 安装依赖...${NC}"
    npm install
fi

# 更新生产环境配置文件
echo -e "${BLUE}  → 配置生产环境变量 VITE_API_BASE=$VITE_API_BASE...${NC}"
cat > .env.production << EOF
# 生产环境配置
# 部署时请修改为实际的后端 API 地址
VITE_API_BASE=$VITE_API_BASE
EOF

echo -e "${BLUE}  → 构建生产版本...${NC}"
npm run build

# 复制前端文件到 release
cp -r dist/* "$RELEASE_DIR/web/"
echo -e "${GREEN}✓ 前端打包完成${NC}"
echo ""

# 步骤 3: 打包后端 API 服务
echo -e "${YELLOW}[3/5] 打包后端 API 服务...${NC}"
cd "$PROJECT_ROOT/backend"

echo -e "${BLUE}  → 编译 swm-server...${NC}"
go build -o "$RELEASE_DIR/swm-server" ./cmd/api

echo -e "${GREEN}✓ API 服务编译完成${NC}"
echo ""

# 步骤 4: 打包后端 Worker（兼容，可选）
echo -e "${YELLOW}[4/5] 打包后端 Worker（兼容可选）...${NC}"
if [ "$BUILD_WORKER_COMPAT" = "true" ]; then
    echo -e "${BLUE}  → 编译 swm-worker...${NC}"
    go build -o "$RELEASE_DIR/swm-worker" ./cmd/worker
    echo -e "${GREEN}✓ Worker 编译完成（兼容模式）${NC}"
else
    echo -e "${BLUE}  → 跳过 swm-worker 编译（单二进制优先）${NC}"
fi
echo ""

# 步骤 5: 复制数据库迁移文件
echo -e "${YELLOW}[5/5] 复制数据库迁移文件...${NC}"
cp "$PROJECT_ROOT/backend/migrations/"*.sql "$RELEASE_DIR/migrations/"
echo -e "${GREEN}✓ 迁移文件复制完成${NC}"
echo ""

# 显示打包结果
echo -e "${BLUE}========================================${NC}"
echo -e "${GREEN}   打包完成！${NC}"
echo -e "${BLUE}========================================${NC}"
echo ""
echo -e "${BLUE}打包文件列表:${NC}"
echo ""

# 显示文件大小
cd "$RELEASE_DIR"

if [ -f "swm-server" ]; then
    SERVER_SIZE=$(du -h swm-server | cut -f1)
    echo -e "  ${GREEN}●${NC} swm-server          ${YELLOW}$SERVER_SIZE${NC}"
fi

if [ -f "swm-worker" ]; then
    WORKER_SIZE=$(du -h swm-worker | cut -f1)
    echo -e "  ${GREEN}●${NC} swm-worker          ${YELLOW}$WORKER_SIZE${NC}"
fi

if [ -d "web" ]; then
    WEB_SIZE=$(du -sh web | cut -f1)
    echo -e "  ${GREEN}●${NC} web/                ${YELLOW}$WEB_SIZE${NC}"
fi

MIGRATION_COUNT=$(ls -1 migrations/*.sql 2>/dev/null | wc -l)
echo -e "  ${GREEN}●${NC} migrations/         ${YELLOW}$MIGRATION_COUNT 个 SQL 文件${NC}"

echo ""
echo -e "${BLUE}目录结构:${NC}"
find "$RELEASE_DIR" -type f | sort | sed "s|$RELEASE_DIR|  release|"
echo ""

# 计算总大小
TOTAL_SIZE=$(du -sh "$RELEASE_DIR" | cut -f1)
echo -e "${GREEN}总大小: $TOTAL_SIZE${NC}"
echo ""

# 部署提示
echo -e "${BLUE}========================================${NC}"
echo -e "${BLUE}   部署指南${NC}"
echo -e "${BLUE}========================================${NC}"
echo ""
echo -e "1. 上传到服务器:"
echo -e "   ${YELLOW}scp -r release/ root@your-server:/opt/swm/${NC}"
echo ""
echo -e "2. 在服务器上运行:"
echo -e "   ${YELLOW}cd /opt/swm${NC}"
echo -e "   ${YELLOW}export DATABASE_URL=\"user:pass@tcp(localhost:3306)/dbname?charset=utf8mb4&parseTime=true\"${NC}"
echo -e "   ${YELLOW}export JWT_SECRET=\"your-secret-key\"${NC}"
echo -e "   ${YELLOW}export RUN_MIGRATIONS=true${NC}"
echo -e "   ${YELLOW}export ENABLE_EMBEDDED_WORKER=true${NC}"
echo -e "   ${YELLOW}export WORKER_INTERVAL_SECONDS=3600${NC}"
echo -e "   ${YELLOW}./swm-server${NC}"
echo -e "   ${GREEN}(宝塔单进程部署仅需选择 swm-server)${NC}"
echo ""
echo -e "3. 使用 Nginx 配置前端:"
echo -e "   ${YELLOW}location / {${NC}"
echo -e "   ${YELLOW}    root /opt/swm/web;${NC}"
echo -e "   ${YELLOW}    try_files \$uri \$uri/ /index.html;${NC}"
echo -e "   ${YELLOW}}${NC}"
echo ""
echo -e "${GREEN}打包脚本执行完毕！${NC}"
