# HelloApp - SWM 自动更新示例程序

这是一个演示 SWM (Software Web Manager) 自动更新功能的 Windows 示例程序。

## 项目结构

```
helloapp/
├── main.go           # 主程序入口
├── swm_client.go     # 基于 software-web-manager/sdk-go 的客户端封装
├── updater.go        # 自动更新逻辑
├── go.mod            # Go 模块定义
├── build.bat         # Windows 构建脚本
├── build.sh          # Linux/Mac 构建脚本
└── README.md         # 本文件
```

## 功能特性

- ✅ 自动检查更新
- ✅ 显示更新信息（版本、大小、更新说明）
- ✅ 下载进度显示
- ✅ SHA256 文件校验
- ✅ Ed25519 签名校验（可选）
- ✅ 自动安装更新并重启
- ✅ 更新事件上报
- ✅ 上报 UserID / Attributes（可选）
- ✅ 运行期 SSE 实时更新监听
- ✅ 支持强制更新

## 快速开始

### 1. 准备工作

确保 SWM 系统已启动：

```bash
# 在 SWM 项目根目录
cd /www/wwwroot/goprj/release
./swm-api
```

### 2. 在 SWM 后台配置应用

1. 打开浏览器访问 `http://localhost:5173`
2. 登录后创建新应用：
   - 应用名称: HelloApp
   - 应用标识: helloapp
3. 创建渠道：
   - 名称: Stable
   - Code: stable
   - 设为默认
4. 获取 AppID / AppSecret 并复制

### 3. 设置凭据

示例程序已改为读取环境变量，不需要再修改源码常量。

Windows（PowerShell）：
```powershell
$env:SWM_BASE_URL="http://localhost:8080"
$env:SWM_APP_ID="你的-app-id"
$env:SWM_APP_SECRET="你的-app-secret"
```

Linux/Mac：
```bash
export SWM_BASE_URL="http://localhost:8080"
export SWM_APP_ID="你的-app-id"
export SWM_APP_SECRET="你的-app-secret"
```

### 4. 构建程序

**Windows:**
```bash
cd /www/wwwroot/goprj/example/helloapp
build.bat
```

**Linux/Mac:**
```bash
cd /www/wwwroot/goprj/example/helloapp
chmod +x build.sh
./build.sh
```

### 5. 发布第一个版本

1. 在 SWM 后台创建版本：
   - 版本号: 1.0.0
   - Version Code: 100
   - 更新说明: 初始版本

2. 上传构建产物：
   - 选择 `dist/helloapp.exe`
   - 平台: windows
   - 架构: amd64

3. 发布版本：
   - 渠道: stable
   - 灰度: 100%
   - 非强制更新

### 6. 测试运行

```bash
dist\helloapp.exe
```

你应该看到：
```
=================================
       HelloApp Demo
       版本:  1.0.0
       构建:  2024-01-15 10:30:00
=================================

[1/4] 正在检查更新...
当前版本: 1.0.0 (Build 100)
设备ID: windows-a1b2c3d4e5f6...
平台: windows/amd64
✓ 当前已是最新版本

[2/4] 已跳过 SDK 全量调用演示（设置 HELLOAPP_RUN_SDK_DEMO=true 可启用）

[3/4] 启动主程序...
🎉 欢迎使用 HelloApp!
...
```

### 7. 测试更新功能

1. 修改版本号为 1.0.1：
   ```bash
   # 编辑 build.bat
   set VERSION=1.0.1
   set VERSION_CODE=101
   ```

2. 重新构建：
   ```bash
   build.bat
   ```

3. 在 SWM 后台发布 1.0.1 版本

4. 运行旧版本（1.0.0）：
   ```bash
   dist\helloapp.exe
   ```

5. 你应该看到更新提示：
   ```
   ═══════════════════════════════════════
              发现新版本!
   ═══════════════════════════════════════
   新版本: 1.0.1 (Build 101)
   当前版本: 1.0.0
   文件大小: 2.35 MB
   
   更新说明:
   - 修复已知问题
   - 优化性能
   
   ═══════════════════════════════════════
   
   是否现在更新? (y/n): 
   ```

## 代码说明

### SWM 客户端 (`swm_client.go`)

核心功能：
- `CheckUpdate()` - 检查是否有新版本
- `Download()` - 下载更新文件（带进度回调）
- `ReportEvent()` - 上报更新事件

说明：
- 当前示例已接入仓库内最新 Go SDK：`software-web-manager/sdk-go`
- `go.mod` 使用本地 `replace` 指向 `../../../sdk/go`

### 更新逻辑 (`updater.go`)

更新流程：
1. 检查更新 → 2. 用户确认 → 3. 下载文件 → 4. 校验文件 → 5. 执行更新 → 6. 重启程序

Windows 更新机制：
- 创建批处理脚本等待原程序退出
- 替换可执行文件
- 启动新版本
- 自动清理临时文件

### 主程序 (`main.go`)

简单的 HelloWorld 程序，演示：
- 版本信息注入（编译时）
- 更新检查调用
- 主程序逻辑

## 自定义配置

### 修改服务器地址

```bash
# Windows PowerShell
$env:SWM_BASE_URL="https://your-swm-server.com"

# Linux/Mac
export SWM_BASE_URL="https://your-swm-server.com"
```

### 修改渠道

```go
client, err := newConfiguredSWMClient()
if err != nil {
    panic(err)
}
client.Channel = "beta"  // 使用 Beta 渠道
```

### 最新 SDK 高级配置（可选）

```bash
# 客户端身份与定向
set HELLOAPP_CHANNEL=stable
set HELLOAPP_USER_ID=user-1001
set HELLOAPP_ATTRIBUTES_JSON={"tier":"vip","region":"CN"}

# 传输层参数
set SWM_HTTP_TIMEOUT_SECONDS=30
set SWM_HTTP_RETRIES=2
set SWM_HTTP_BACKOFF_MS=500
set SWM_HTTP_DEBUG=true
# 测试用客户端IP覆盖（用于验证地区策略）
set HELLOAPP_CLIENT_IP_OVERRIDE=223.5.5.5

# 下载签名校验（启用后必须配置公钥）
set SWM_VERIFY_SIGNATURE=true
set SWM_PUBLIC_KEY=你的-ed25519-public-key-base64-or-hex
```

说明：
- `HELLOAPP_ATTRIBUTES_JSON` 必须是合法 JSON 对象。
- `HELLOAPP_CLIENT_IP_OVERRIDE` 会通过 `X-Forwarded-For/X-Real-IP` 覆盖客户端IP（仅测试建议使用）。
- `SWM_VERIFY_SIGNATURE=true` 时，demo 会校验更新响应与下载文件签名。
- `SWM_HTTP_DEBUG=false` 可关闭详细 HTTP 请求日志。
- 命中地区黑名单时，后端 `update-check` 会返回 `update_region_blocked`，demo 启动检查阶段会立即退出。

### SDK 全量调用演示

默认不执行，按需启用：

```bash
set HELLOAPP_RUN_SDK_DEMO=true
```

说明：
- 该演示会调用客户端上报接口，以及（在配置 `SWM_AUTH_TOKEN` 时）管理类接口。

管理类接口调用需要额外环境变量：

```bash
set SWM_APP_SECRET=你的应用密钥
set SWM_AUTH_TOKEN=你的登录token
set SWM_APP_ID=你的应用ID
set SWM_APP_SECRET_ID=已有密钥ID(可选，用于撤销演示)
set SWM_DEMO_ALLOW_WRITE=true
```

说明：
- `SWM_AUTH_TOKEN` 未设置时，只会执行客户端上报类接口，管理接口自动跳过。
- `SWM_DEMO_ALLOW_WRITE=false`（默认）时，会用占位 ID 演示调用，不会真正改动你的业务数据。
- `SWM_DEMO_ALLOW_WRITE=true` 时，会尝试真实创建/更新资源（请在测试环境使用）。
- SDK Demo 创建的 app_secret 默认 scopes 已包含 `update:check` 与 `event:write`，可直接用于客户端签名调用。

### 测试用户反馈与截图上报

如需单独验证后台“用户反馈”页面，可以开启反馈 smoke demo。未指定附件时，示例程序会自动生成一张临时 PNG 诊断图作为截图附件。

Windows（PowerShell）：
```powershell
$env:SWM_BASE_URL="http://localhost:8080"
$env:SWM_APP_ID="你的-app-id"
$env:SWM_APP_SECRET="你的-app-secret"
$env:HELLOAPP_REPORT_FEEDBACK="true"
$env:HELLOAPP_FEEDBACK_CONTENT="HelloApp 手动反馈测试"
$env:HELLOAPP_FEEDBACK_RATING="5"
$env:HELLOAPP_FEEDBACK_CONTACT="tester@example.local"
go run .
```

可选指定本地截图：
```powershell
$env:HELLOAPP_FEEDBACK_ATTACHMENT="C:\path\to\screenshot.png"
go run .
```

上报成功后，在后台 `/feedback` 页面选择对应应用即可查看反馈内容、截图附件和 metadata。若应用详情中关闭了“用户反馈”，服务端会拒绝新的上报，demo 会输出“用户反馈已被服务端关闭”。

### 强制更新

在 SWM 后台发布时勾选"强制更新"，客户端将跳过确认直接更新。

## 故障排查

### 检查更新失败

- 确认 SWM 服务已启动
- 确认 BaseURL 正确
- 确认 AppID / AppSecret 正确
- 确认客户端系统时间误差不超过 5 分钟（签名时间窗）
- 检查防火墙设置

### 下载失败

- 检查网络连接
- 确认存储配置正确（本地/S3）
- 查看后端日志

### 更新后无法启动

- 检查是否有杀毒软件拦截
- 确认文件权限
- 查看更新脚本日志

## 进阶功能

### 静默更新

修改 `updater.go`，移除用户确认：

```go
// 删除或注释掉这部分
// if !resp.Mandatory {
//     fmt.Print("是否现在更新? (y/n): ")
//     ...
// }
```

### 定时检查更新

```go
func main() {
    // 启动时检查
    checkAndUpdate()
    
    // 每 1 小时检查一次
    ticker := time.NewTicker(1 * time.Hour)
    go func() {
        for range ticker.C {
            checkAndUpdate()
        }
    }()
    
    // 主程序...
}
```

### 更新进度窗口

可以使用 Walk 库创建 GUI 进度窗口：
```go
import "github.com/lxn/walk"
```

## 许可证

MIT License
