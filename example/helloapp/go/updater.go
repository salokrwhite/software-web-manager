package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	swmsdk "software-web-manager/sdk-go"
)

type swmUpdateCheckResponse = swmsdk.UpdateCheckResponse

var (
	realtimeUpdateMu sync.Mutex
	seenReleaseIDs   = map[string]struct{}{}
	updateInProgress bool
)

// checkAndUpdate 检查并执行更新
func checkAndUpdate() error {
	// 创建 SWM 客户端
	client, err := newConfiguredSWMClient()
	if err != nil {
		return fmt.Errorf("SWM 配置错误: %w", err)
	}

	// 解析当前版本号
	currentVersion := Version
	versionCode := 0
	fmt.Sscanf(VersionCode, "%d", &versionCode)

	fmt.Printf("当前版本: %s (Build %d)\n", currentVersion, versionCode)
	fmt.Printf("设备ID: %s\n", client.DeviceID)
	fmt.Printf("平台: %s/%s\n", client.Platform, client.Arch)

	// 检查更新
	resp, err := client.CheckUpdate(currentVersion, versionCode)
	if err != nil {
		if errors.Is(err, swmsdk.ErrDeviceBlocked) {
			exitForDeviceBlocked("update-check", "device blocked")
		}
		if isSignatureVerificationFailed(err) {
			exitForSignatureVerificationFailed("update-check", err)
			return nil
		}
		if errors.Is(err, swmsdk.ErrUpdateRegionBlocked) {
			fmt.Println("当前地区命中更新黑名单（update_region_blocked），客户端即将退出。")
			_ = client.ReportEvent("update_region_blocked", map[string]interface{}{
				"version":      currentVersion,
				"version_code": versionCode,
				"reason":       "region_blacklist",
			})
			time.Sleep(300 * time.Millisecond)
			os.Exit(24)
		}
		_ = client.ReportEvent("update_failed", map[string]interface{}{
			"reason":       "check_update_failed",
			"error":        err.Error(),
			"version":      currentVersion,
			"version_code": versionCode,
		})
		return fmt.Errorf("检查更新失败: %w", err)
	}
	if resp.HeartbeatIntervalSeconds >= 5 {
		RuntimeHeartbeatIntervalSeconds = resp.HeartbeatIntervalSeconds
		fmt.Printf("已同步后台心跳间隔: %d 秒\n", RuntimeHeartbeatIntervalSeconds)
	}
	_ = client.ReportEvent("check_update", map[string]interface{}{
		"version":          currentVersion,
		"version_code":     versionCode,
		"update_available": resp.UpdateAvailable,
	})

	// 没有可用更新
	if !resp.UpdateAvailable {
		if err := verifyCurrentInstalledBinaryIntegrity(client, resp); err != nil {
			exitForSignatureVerificationFailed("startup-self-check", err)
			return nil
		}
		fmt.Println("✓ 当前已是最新版本")
		return nil
	}
	_ = client.ReportEvent("update_available", map[string]interface{}{
		"version":          currentVersion,
		"version_code":     versionCode,
		"target_version":   resp.Version,
		"target_build":     resp.VersionCode,
		"release_id":       resp.ReleaseID,
		"mandatory":        resp.Mandatory,
		"delivery_method":  resp.DeliveryMethod,
		"rollback_allowed": resp.RollbackAllowed,
	})

	// 发现新版本
	fmt.Println()
	fmt.Println("═══════════════════════════════════════")
	fmt.Println("           发现新版本!")
	fmt.Println("═══════════════════════════════════════")
	if resp.VersionCode > 0 {
		fmt.Printf("新版本: %s (Build %d)\n", resp.Version, resp.VersionCode)
	} else {
		fmt.Printf("新版本: %s\n", resp.Version)
	}
	fmt.Printf("当前版本: %s\n", currentVersion)
	fmt.Printf("文件大小: %.2f MB\n", float64(resp.Size)/(1024*1024))
	if resp.ReleaseID != "" {
		fmt.Printf("发布ID: %s\n", resp.ReleaseID)
	}
	if strings.TrimSpace(resp.DeliveryMethod) != "" {
		fmt.Printf("分发方式: %s\n", resp.DeliveryMethod)
	}
	if resp.RollbackAllowed {
		fmt.Println("支持回滚: 是")
	}
	if resp.Mandatory {
		fmt.Println("⚠️  这是一个强制更新，必须升级才能继续使用")
	}
	if resp.Notes != "" {
		fmt.Printf("\n更新说明:\n%s\n", resp.Notes)
	}
	fmt.Println("═══════════════════════════════════════")
	fmt.Println()

	// 询问用户是否更新（强制更新则跳过询问）
	if !resp.Mandatory {
		fmt.Print("是否现在更新? (y/n): ")
		var answer string
		fmt.Scanln(&answer)
		if strings.ToLower(answer) != "y" && strings.ToLower(answer) != "yes" {
			fmt.Println("已取消更新")
			return nil
		}
	}

	if shouldOpenInBrowser(resp) {
		fmt.Println("\n该版本使用外部下载链接，正在打开浏览器...")
		if err := openExternalURL(resp.DownloadURL); err != nil {
			_ = client.ReportEvent("update_failed", map[string]interface{}{
				"reason":         "open_external_link_failed",
				"error":          err.Error(),
				"target_version": resp.Version,
			})
			return fmt.Errorf("打开下载链接失败: %w", err)
		}
		_ = client.ReportEvent("external_link_opened", map[string]interface{}{
			"target_version": resp.Version,
			"url":            resp.DownloadURL,
		})
		fmt.Println("已打开外部下载页面，请按页面提示完成更新。")
		return nil
	}

	// 下载更新
	fmt.Println("\n开始下载更新...")
	updateFile := filepath.Join(os.TempDir(), fmt.Sprintf("helloapp_update_%s.exe", resp.Version))
	_ = client.ReportEvent("download_started", map[string]interface{}{
		"target_version": resp.Version,
		"size":           resp.Size,
		"release_id":     resp.ReleaseID,
	})

	startTime := time.Now()
	err = client.Download(resp.DownloadURL, updateFile, resp.ChecksumSHA256, resp.Signature,
		func(written, total int64) {
			if total > 0 {
				percent := float64(written) / float64(total) * 100
				fmt.Printf("\r下载进度: %.1f%% (%s / %s)",
					percent,
					formatBytes(written),
					formatBytes(total))
			}
		})
	if err != nil {
		if isSignatureVerificationFailed(err) {
			exitForSignatureVerificationFailed("download", err)
			return nil
		}
		_ = client.ReportEvent("update_failed", map[string]interface{}{
			"reason":         "download_failed",
			"error":          err.Error(),
			"target_version": resp.Version,
		})
		return fmt.Errorf("下载失败: %w", err)
	}
	downloadTime := time.Since(startTime)
	fmt.Printf("\n✓ 下载完成! 用时: %s\n", downloadTime.Round(time.Second))

	// 上报下载成功事件
	_ = client.ReportEvent("download_completed", map[string]interface{}{
		"version":         resp.Version,
		"release_id":      resp.ReleaseID,
		"size":            resp.Size,
		"download_time_s": downloadTime.Seconds(),
	})

	// 执行更新
	fmt.Println("\n准备安装更新...")
	fmt.Println("程序将重启以完成更新")
	time.Sleep(2 * time.Second)

	// 启动更新程序并退出当前程序
	if err := applyUpdate(updateFile); err != nil {
		_ = client.ReportEvent("update_failed", map[string]interface{}{
			"reason":         "apply_update_failed",
			"error":          err.Error(),
			"target_version": resp.Version,
		})
		return fmt.Errorf("启动更新失败: %w", err)
	}

	// 上报安装事件
	_ = client.ReportEvent("install_completed", map[string]interface{}{
		"version":    resp.Version,
		"release_id": resp.ReleaseID,
	})

	// 退出当前程序
	fmt.Println("正在退出当前程序...")
	time.Sleep(1 * time.Second)
	os.Exit(0)

	return nil
}

func startRealtimeUpdateWatcher() context.CancelFunc {
	client, err := newConfiguredSWMClient()
	if err != nil {
		fmt.Printf("[实时更新监听] 未启用: %v\n", err)
		return func() {}
	}
	versionCode := 0
	if parsed, err := strconv.Atoi(strings.TrimSpace(VersionCode)); err == nil {
		versionCode = parsed
	}
	opts := UpdateStreamOptions{
		ChannelCode:    client.Channel,
		Platform:       client.Platform,
		Arch:           client.Arch,
		DeviceID:       client.DeviceID,
		CurrentVersion: Version,
		VersionCode:    &versionCode,
		Reconnect:      true,
		OnError: func(err error) {
			if errors.Is(err, swmsdk.ErrDeviceBlocked) {
				exitForDeviceBlocked("update-stream", "device blocked")
				return
			}
			if errors.Is(err, swmsdk.ErrUpdateRegionBlocked) {
				fmt.Println("[实时更新监听] 当前地区命中更新黑名单（update_region_blocked）")
				return
			}
			fmt.Printf("[实时更新监听] %v\n", err)
		},
		OnControlEvent: func(evt ControlEvent) {
			if strings.EqualFold(strings.TrimSpace(evt.Type), swmsdk.ControlEventShutdown) {
				exitForDeviceBlocked("update-stream", evt.Reason)
			}
		},
	}
	ctx, cancel := context.WithCancel(context.Background())
	handle, err := client.WatchUpdates(ctx, opts, func(resp swmUpdateCheckResponse) {
		handleUpdateAvailable(client, UpdateResponse{
			UpdateAvailable:          resp.UpdateAvailable,
			Mandatory:                resp.Mandatory,
			HeartbeatIntervalSeconds: resp.HeartbeatIntervalSeconds,
			OpenInBrowser:            resp.OpenInBrowser,
			DeliveryMethod:           resp.DeliveryMethod,
			ReleaseID:                resp.ReleaseID,
			Version:                  resp.Version,
			VersionCode:              0,
			Notes:                    resp.Notes,
			DownloadURL:              resp.DownloadURL,
			ChecksumSHA256:           resp.ChecksumSHA256,
			Signature:                resp.Signature,
			Size:                     resp.Size,
			RollbackAllowed:          resp.RollbackAllowed,
		}, versionCode)
	})
	if err != nil {
		fmt.Printf("[实时更新监听] 启动失败: %v\n", err)
		return cancel
	}
	return func() {
		handle.Stop()
		cancel()
	}
}

func handleUpdateAvailable(client *SWMClient, resp UpdateResponse, currentVersionCode int) {
	realtimeUpdateMu.Lock()
	if updateInProgress {
		realtimeUpdateMu.Unlock()
		return
	}
	if resp.ReleaseID != "" {
		if _, ok := seenReleaseIDs[resp.ReleaseID]; ok {
			realtimeUpdateMu.Unlock()
			return
		}
		seenReleaseIDs[resp.ReleaseID] = struct{}{}
	}
	updateInProgress = true
	realtimeUpdateMu.Unlock()
	defer func() {
		realtimeUpdateMu.Lock()
		updateInProgress = false
		realtimeUpdateMu.Unlock()
	}()

	fmt.Println()
	fmt.Println("[实时更新] 检测到新版本推送")
	fmt.Printf("目标版本: %s\n", resp.Version)
	if resp.Notes != "" {
		fmt.Printf("[实时更新] 更新说明: %s\n", resp.Notes)
	}
	if resp.Mandatory {
		fmt.Println("[实时更新] 当前为强制更新，开始执行升级流程")
	} else {
		fmt.Print("[实时更新] 发现可选更新，是否立即更新? (y/n): ")
		var answer string
		fmt.Scanln(&answer)
		if strings.ToLower(strings.TrimSpace(answer)) != "y" && strings.ToLower(strings.TrimSpace(answer)) != "yes" {
			_ = client.ReportEvent("update_available", map[string]interface{}{
				"version":          Version,
				"version_code":     currentVersionCode,
				"target_version":   resp.Version,
				"target_build":     resp.VersionCode,
				"release_id":       resp.ReleaseID,
				"mandatory":        resp.Mandatory,
				"delivery_method":  resp.DeliveryMethod,
				"rollback_allowed": resp.RollbackAllowed,
				"decision":         "defer",
				"triggered_by":     "sse",
			})
			fmt.Println("[实时更新] 已选择稍后更新")
			return
		}
	}

	if err := performUpdate(client, resp, currentVersionCode); err != nil {
		fmt.Printf("[实时更新] 更新失败: %v\n", err)
	}
}

func performUpdate(client *SWMClient, resp UpdateResponse, currentVersionCode int) error {
	if shouldOpenInBrowser(&resp) {
		fmt.Println("[实时更新] 当前版本使用外部链接分发，正在打开浏览器...")
		if err := openExternalURL(resp.DownloadURL); err != nil {
			_ = client.ReportEvent("update_failed", map[string]interface{}{
				"reason":         "open_external_link_failed",
				"error":          err.Error(),
				"target_version": resp.Version,
				"triggered_by":   "sse",
			})
			return err
		}
		_ = client.ReportEvent("external_link_opened", map[string]interface{}{
			"target_version": resp.Version,
			"url":            resp.DownloadURL,
			"triggered_by":   "sse",
		})
		fmt.Println("[实时更新] 已打开外部下载页面")
		return nil
	}

	_ = client.ReportEvent("update_available", map[string]interface{}{
		"version":          Version,
		"version_code":     currentVersionCode,
		"target_version":   resp.Version,
		"target_build":     resp.VersionCode,
		"release_id":       resp.ReleaseID,
		"mandatory":        resp.Mandatory,
		"delivery_method":  resp.DeliveryMethod,
		"rollback_allowed": resp.RollbackAllowed,
		"triggered_by":     "sse",
	})

	fmt.Println("[实时更新] 开始下载更新包...")
	updateFile := filepath.Join(os.TempDir(), fmt.Sprintf("helloapp_update_%s.exe", resp.Version))
	_ = client.ReportEvent("download_started", map[string]interface{}{
		"target_version": resp.Version,
		"size":           resp.Size,
		"release_id":     resp.ReleaseID,
		"triggered_by":   "sse",
	})
	startTime := time.Now()
	if err := client.Download(resp.DownloadURL, updateFile, resp.ChecksumSHA256, resp.Signature,
		func(written, total int64) {
			if total > 0 {
				percent := float64(written) / float64(total) * 100
				fmt.Printf("\r[实时更新] 下载进度: %.1f%% (%s / %s)",
					percent,
					formatBytes(written),
					formatBytes(total))
			}
		}); err != nil {
		if isSignatureVerificationFailed(err) {
			exitForSignatureVerificationFailed("sse-download", err)
			return nil
		}
		_ = client.ReportEvent("update_failed", map[string]interface{}{
			"reason":         "download_failed",
			"error":          err.Error(),
			"target_version": resp.Version,
			"triggered_by":   "sse",
		})
		return err
	}
	fmt.Printf("\n[实时更新] 下载完成，用时: %s\n", time.Since(startTime).Round(time.Second))

	_ = client.ReportEvent("download_completed", map[string]interface{}{
		"version":         resp.Version,
		"release_id":      resp.ReleaseID,
		"size":            resp.Size,
		"download_time_s": time.Since(startTime).Seconds(),
		"triggered_by":    "sse",
	})

	if err := applyUpdate(updateFile); err != nil {
		_ = client.ReportEvent("update_failed", map[string]interface{}{
			"reason":         "apply_update_failed",
			"error":          err.Error(),
			"target_version": resp.Version,
			"triggered_by":   "sse",
		})
		return err
	}
	_ = client.ReportEvent("install_completed", map[string]interface{}{
		"version":      resp.Version,
		"release_id":   resp.ReleaseID,
		"triggered_by": "sse",
	})
	fmt.Println("[实时更新] 已启动升级流程，应用即将退出")
	time.Sleep(1 * time.Second)
	os.Exit(0)
	return nil
}

func shouldOpenInBrowser(resp *UpdateResponse) bool {
	if resp == nil {
		return false
	}
	if resp.OpenInBrowser {
		return true
	}
	return strings.EqualFold(strings.TrimSpace(resp.DeliveryMethod), "external_link")
}

func openExternalURL(target string) error {
	target = strings.TrimSpace(target)
	if target == "" {
		return fmt.Errorf("empty url")
	}
	if runtime.GOOS == "windows" {
		return exec.Command("cmd", "/c", "start", "", target).Start()
	}
	if runtime.GOOS == "darwin" {
		return exec.Command("open", target).Start()
	}
	return exec.Command("xdg-open", target).Start()
}

// applyUpdate 应用更新（Windows 平台）
func applyUpdate(updateFile string) error {
	// 获取当前程序路径
	currentExe, err := os.Executable()
	if err != nil {
		return fmt.Errorf("获取当前程序路径失败: %w", err)
	}

	// 创建更新脚本
	scriptContent := fmt.Sprintf(`@echo off
chcp 65001 >nul
echo 正在安装更新...
timeout /t 2 /nobreak >nul

:: 等待原程序退出
:wait_loop
tasklist | find /i "helloapp.exe" >nul
if not errorlevel 1 (
    timeout /t 1 /nobreak >nul
    goto wait_loop
)

:: 替换程序文件
copy /y "%s" "%s" >nul
if errorlevel 1 (
    echo 更新失败，请手动替换文件
    pause
    exit /b 1
)

:: 删除更新文件
del "%s" >nul 2>&1

:: 启动新版本
echo 更新完成，正在启动新版本...
start "" "%s"

:: 删除自身脚本
del "%%~f0" >nul 2>&1
`, updateFile, currentExe, updateFile, currentExe)

	// 保存脚本到临时文件
	scriptFile := filepath.Join(os.TempDir(), "helloapp_update.bat")
	if err := os.WriteFile(scriptFile, []byte(scriptContent), 0755); err != nil {
		return fmt.Errorf("创建更新脚本失败: %w", err)
	}

	// 启动更新脚本
	cmd := exec.Command("cmd", "/c", "start", scriptFile)
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("启动更新脚本失败: %w", err)
	}

	return nil
}

// formatBytes 格式化字节大小
func formatBytes(bytes int64) string {
	const (
		KB = 1024
		MB = 1024 * KB
		GB = 1024 * MB
	)

	switch {
	case bytes >= GB:
		return fmt.Sprintf("%.2f GB", float64(bytes)/GB)
	case bytes >= MB:
		return fmt.Sprintf("%.2f MB", float64(bytes)/MB)
	case bytes >= KB:
		return fmt.Sprintf("%.2f KB", float64(bytes)/KB)
	default:
		return fmt.Sprintf("%d B", bytes)
	}
}

func isSignatureVerificationFailed(err error) bool {
	if err == nil {
		return false
	}
	return strings.Contains(strings.ToLower(err.Error()), "signature verification failed")
}

func verifyCurrentInstalledBinaryIntegrity(client *SWMClient, resp *UpdateResponse) error {
	if client == nil || !client.VerifySignature {
		return nil
	}
	if resp == nil {
		return fmt.Errorf("signature verification failed: missing update-check response")
	}
	expectedChecksum := strings.ToLower(strings.TrimSpace(resp.ChecksumSHA256))
	expectedSignature := strings.TrimSpace(resp.Signature)
	if expectedChecksum == "" || expectedSignature == "" {
		return fmt.Errorf("signature verification failed: missing current package checksum or signature")
	}

	exePath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("signature verification failed: resolve executable path failed: %w", err)
	}
	raw, err := os.ReadFile(exePath)
	if err != nil {
		return fmt.Errorf("signature verification failed: read executable failed: %w", err)
	}
	sum := sha256.Sum256(raw)
	gotChecksum := hex.EncodeToString(sum[:])
	if !strings.EqualFold(gotChecksum, expectedChecksum) {
		return fmt.Errorf("signature verification failed: local package checksum mismatch")
	}
	return nil
}
