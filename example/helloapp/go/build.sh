#!/bin/bash

echo "==================================="
echo "   HelloApp 构建脚本"
echo "==================================="
echo ""

# 设置版本信息
VERSION="1.0.0"
VERSION_CODE="101"
BUILD_TIME=$(date '+%Y-%m-%d %H:%M:%S')
# 独立示例模块构建，避免被仓库根 go.work 限制
GOWORK_MODE=off

echo "版本: $VERSION"
echo "构建时间: $BUILD_TIME"
echo ""

# 同步依赖
echo "[0/2] 正在同步依赖..."
GOWORK=$GOWORK_MODE go mod tidy
if [ $? -ne 0 ]; then
    echo "依赖同步失败!"
    exit 1
fi

# 创建输出目录
mkdir -p dist

# 构建 Windows x64 版本
echo "[1/2] 正在构建 Windows x64 版本..."
GOWORK=$GOWORK_MODE GOOS=windows GOARCH=amd64 go build \
    -ldflags "-s -w -X main.Version=$VERSION -X main.VersionCode=$VERSION_CODE -X 'main.BuildTime=$BUILD_TIME'" \
    -o dist/helloapp.exe

if [ $? -ne 0 ]; then
    echo "构建失败!"
    exit 1
fi

echo "✓ 构建成功: dist/helloapp.exe"
echo ""

# 显示文件信息
echo "[2/2] 文件信息:"
ls -lh dist/helloapp.exe

echo ""
echo "==================================="
echo "   构建完成!"
echo "==================================="
echo ""
echo "使用方法:"
echo "  1. 先启动 SWM 后端服务"
echo "  2. 在 SWM 后台创建应用并获取 AppID / AppSecret"
echo "  3. 设置环境变量 SWM_APP_ID + SWM_APP_SECRET"
echo "  4. 运行 dist/helloapp.exe"
echo ""
