@echo off
setlocal
chcp 65001 >nul
echo ===================================
echo    HelloApp 构建脚本
echo ===================================
echo.

:: 设置版本信息
set VERSION=1.0.0
set VERSION_CODE=100
set BUILD_TIME=%date:~0,4%-%date:~5,2%-%date:~8,2% %time:~0,8%
set GOWORK=off

echo 版本: %VERSION%
echo 构建时间: %BUILD_TIME%
echo.

:: 同步依赖
echo [0/2] 正在同步依赖...
go mod tidy
if %errorlevel% neq 0 (
    echo 依赖同步失败!
    exit /b 1
)

:: 创建输出目录
if not exist dist mkdir dist

:: 构建 Windows x64 版本
echo [1/2] 正在构建 Windows x64 版本...
set GOOS=windows
set GOARCH=amd64
go build -ldflags "-s -w -X main.Version=%VERSION% -X main.VersionCode=%VERSION_CODE% -X 'main.BuildTime=%BUILD_TIME%'" -o dist\helloapp.exe

if %errorlevel% neq 0 (
    echo 构建失败!
    exit /b 1
)

echo ✓ 构建成功: dist\helloapp.exe
echo.

:: 显示文件信息
echo [2/2] 文件信息:
dir dist\helloapp.exe /q

echo.
echo ===================================
echo    构建完成!
echo ===================================
echo.
echo 使用方法:
echo   1. 先启动 SWM 后端服务
echo   2. 在 SWM 后台创建应用并获取 AppID / AppSecret
echo   3. 设置环境变量 SWM_APP_ID + SWM_APP_SECRET
echo   4. 运行 dist\helloapp.exe
echo.
pause
endlocal
