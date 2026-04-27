package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	swmsdk "software-web-manager/sdk-go"
)

// 版本信息 - 编译时注入
var (
	Version                         = "1.0.0"
	VersionCode                     = "100"
	BuildTime                       = "unknown"
	RuntimeHeartbeatIntervalSeconds = 0
	exitOnce                        sync.Once
)

func main() {
	// 显示欢迎信息
	showBanner()

	// 检查更新
	fmt.Println("\n[1/4] 正在检查更新...")
	if err := checkAndUpdate(); err != nil {
		if isSignatureVerificationFailed(err) {
			exitForSignatureVerificationFailed("startup", err)
			return
		}
		fmt.Printf("检查更新失败: %v\n", err)
		fmt.Println("继续使用当前版本...")
	}

	// 仅上报一次真实应用启动事件（不与 SDK demo 混用）
	reportAppStarted()

	// 用户反馈 smoke demo（默认关闭，便于按需测试反馈页面与截图预览）
	runFeedbackSmokeDemoIfEnabled()

	// SDK 全量演示（默认关闭，避免干扰主流程）
	if parseBoolEnv("HELLOAPP_RUN_SDK_DEMO", false) {
		fmt.Println("\n[2/4] 正在执行 SDK 全量调用演示...")
		runFullSDKDemo()
	} else {
		fmt.Println("\n[2/4] 已跳过 SDK 全量调用演示（设置 HELLOAPP_RUN_SDK_DEMO=true 可启用）")
	}

	// 运行期实时更新监听（SSE）
	updateWatchCancel := startRealtimeUpdateWatcher()
	defer updateWatchCancel()

	// 运行期心跳（保持在线设备状态）
	hbCancel := startRuntimeHeartbeat()
	defer hbCancel()

	// 主程序逻辑
	fmt.Println("\n[3/4] 启动主程序...")
	runMainApp()

	// 等待用户退出（避免与实时更新线程并发读取 stdin）
	fmt.Println("\n[4/4] 运行中（按 Ctrl+C 退出）...")
	sigCtx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	<-sigCtx.Done()
	fmt.Println("收到退出信号，正在关闭...")
}

func reportAppStarted() {
	client, err := newConfiguredSWMClient()
	if err != nil {
		fmt.Printf("应用启动事件上报已跳过: %v\n", err)
		return
	}

	if err := client.ReportEvent("app_started", map[string]interface{}{
		"version":      Version,
		"version_code": VersionCode,
		"at":           time.Now().Format(time.RFC3339),
		"source":       "main",
	}); err != nil {
		if errors.Is(err, swmsdk.ErrDeviceBlocked) {
			exitForDeviceBlocked("app-started", "device blocked")
			return
		}
		fmt.Printf("应用启动事件上报失败: %v\n", err)
	}
}

func showBanner() {
	fmt.Println("=================================")
	fmt.Println("       HelloApp Demo")
	fmt.Println("       版本: ", Version)
	fmt.Println("       构建: ", BuildTime)
	fmt.Println("=================================")
}

func runMainApp() {
	fmt.Println("\n🎉 欢迎使用 HelloApp!")
	fmt.Println("这是一个演示 SWM 自动更新功能的示例程序。")
	fmt.Println()
	fmt.Println("功能列表:")
	fmt.Println("  1. 自动检查更新")
	fmt.Println("  2. 下载并安装新版本")
	fmt.Println("  3. 上报更新统计")
	fmt.Println()
	fmt.Printf("当前版本: %s (Build %s)\n", Version, VersionCode)
	fmt.Println()

	// 模拟一些工作
	fmt.Println("正在运行主程序逻辑...")
	fmt.Println("✓ 初始化完成")
	fmt.Println("✓ 服务已启动")
	fmt.Println("✓ 一切正常")
}

// 获取程序所在目录
func getAppDir() string {
	exe, err := os.Executable()
	if err != nil {
		return "."
	}
	return filepath.Dir(exe)
}

func startRuntimeHeartbeat() context.CancelFunc {
	client, err := newConfiguredSWMClient()
	if err != nil {
		fmt.Printf("未启用运行期心跳: %v\n", err)
		return func() {}
	}
	interval := resolveHeartbeatInterval()

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		// 先打一次，避免进入主逻辑前出现空窗期
		if err := client.ReportHeartbeat(Version); err != nil {
			if errors.Is(err, swmsdk.ErrDeviceBlocked) {
				exitForDeviceBlocked("heartbeat", "initial heartbeat rejected")
				return
			}
			fmt.Printf("心跳上报失败: %v\n", err)
		}

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if err := client.ReportHeartbeat(Version); err != nil {
					if errors.Is(err, swmsdk.ErrDeviceBlocked) {
						exitForDeviceBlocked("heartbeat", "device blocked")
						return
					}
					fmt.Printf("心跳上报失败: %v\n", err)
				}
			}
		}
	}()

	fmt.Printf("已启用运行期心跳，间隔: %s\n", interval)
	return cancel
}

func exitForDeviceBlocked(source, reason string) {
	exitOnce.Do(func() {
		fmt.Printf("\n[设备下线] 来源=%s 原因=%s，客户端即将退出。\n", source, reason)
		time.Sleep(300 * time.Millisecond)
		os.Exit(23)
	})
}

func exitForSignatureVerificationFailed(source string, err error) {
	exitOnce.Do(func() {
		msg := "signature verification failed"
		if err != nil {
			msg = err.Error()
		}
		fmt.Printf("\n[安全校验失败] 来源=%s 错误=%s，客户端即将退出。\n", source, msg)
		time.Sleep(300 * time.Millisecond)
		os.Exit(25)
	})
}

func resolveHeartbeatInterval() time.Duration {
	if RuntimeHeartbeatIntervalSeconds >= 5 {
		return time.Duration(RuntimeHeartbeatIntervalSeconds) * time.Second
	}

	raw := strings.TrimSpace(os.Getenv("HELLOAPP_HEARTBEAT_SECONDS"))
	if raw == "" {
		return 30 * time.Second
	}
	seconds, err := strconv.Atoi(raw)
	if err != nil || seconds < 5 {
		return 30 * time.Second
	}
	return time.Duration(seconds) * time.Second
}
