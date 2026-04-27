#!/bin/bash

# SWM 后端开发模式启动脚本
# 使用 go run 直接运行，无需编译

echo "==================================="
echo "   SWM 后端开发模式"
echo "==================================="
echo ""

# 设置环境变量（兼容旧 SWM_* 变量，同时输出当前实际生效变量）
export STORAGE_DRIVER=${STORAGE_DRIVER:-${SWM_STORAGE_TYPE:-local}}
export LOCAL_STORAGE_PATH=${LOCAL_STORAGE_PATH:-${SWM_STORAGE_LOCAL_PATH:-../release/data}}
export JWT_SECRET=${JWT_SECRET:-${SWM_JWT_SECRET:-dev-secret-key}}
export ENABLE_EMBEDDED_WORKER=${ENABLE_EMBEDDED_WORKER:-true}
export WORKER_INTERVAL_SECONDS=${WORKER_INTERVAL_SECONDS:-60}

echo "环境变量:"
echo "  STORAGE_DRIVER: $STORAGE_DRIVER"
echo "  LOCAL_STORAGE_PATH: $LOCAL_STORAGE_PATH"
echo "  ENABLE_EMBEDDED_WORKER: $ENABLE_EMBEDDED_WORKER"
echo "  WORKER_INTERVAL_SECONDS: $WORKER_INTERVAL_SECONDS"
echo ""

# 检查是否需要运行迁移
if [ "$1" == "--migrate" ] || [ "$1" == "-m" ]; then
    echo "[1/2] 运行数据库迁移..."
    go run ./cmd/migrate
    if [ $? -ne 0 ]; then
        echo "迁移失败!"
        exit 1
    fi
    echo "✓ 迁移完成"
    echo ""
fi

# 启动 API 服务
echo "[2/2] 启动 API 服务 (go run)..."
echo "服务将运行在 http://localhost:8080"
echo "按 Ctrl+C 停止服务"
echo ""

go run ./cmd/api/main.go
