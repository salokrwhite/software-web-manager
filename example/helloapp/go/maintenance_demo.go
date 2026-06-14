package main

import (
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	swmsdk "software-web-manager/sdk-go"
)

var (
	maintenanceMu    sync.Mutex
	maintenanceTimer *time.Timer
)

// applyMaintenanceFromCheck 处理轮询(update-check)返回的维护信息:
// 维护已开始 → 提示并退出;已排期 → 提示倒计时并安排到点退出。
func applyMaintenanceFromCheck(info *swmsdk.Maintenance) {
	if info == nil || !info.Enabled {
		cancelMaintenanceExit()
		return
	}
	if info.Active {
		showMaintenanceActive(info.Message)
		os.Exit(0)
	}
	scheduleMaintenanceExit(info.StartAt, info.Message)
}

// handleMaintenanceControl 处理 SSE 推送的维护控制事件。
func handleMaintenanceControl(evt ControlEvent) {
	switch strings.TrimSpace(evt.Type) {
	case swmsdk.ControlEventMaintenanceScheduled:
		startAt := ""
		if !evt.StartAt.IsZero() {
			startAt = evt.StartAt.UTC().Format(time.RFC3339)
		}
		scheduleMaintenanceExit(startAt, evt.Message)
	case swmsdk.ControlEventMaintenanceCancelled:
		fmt.Println("\n[维护模式] 维护计划已取消")
		cancelMaintenanceExit()
	}
}

func scheduleMaintenanceExit(startAtRFC3339, message string) {
	startAtRFC3339 = strings.TrimSpace(startAtRFC3339)
	if startAtRFC3339 == "" {
		showMaintenanceActive(message)
		os.Exit(0)
	}
	startAt, err := time.Parse(time.RFC3339, startAtRFC3339)
	if err != nil {
		fmt.Printf("[维护模式] 无法解析维护开始时间: %v\n", err)
		return
	}
	remaining := time.Until(startAt)
	if remaining <= 0 {
		showMaintenanceActive(message)
		os.Exit(0)
	}

	fmt.Println()
	fmt.Println("═══════════════════════════════════════")
	fmt.Println("           ⚠️  系统维护通知")
	fmt.Println("═══════════════════════════════════════")
	if strings.TrimSpace(message) != "" {
		fmt.Printf("公告: %s\n", strings.TrimSpace(message))
	}
	fmt.Printf("系统将在 %s 后进入维护(%s)\n", formatCountdown(remaining), startAt.Local().Format("2006-01-02 15:04:05"))
	fmt.Println("维护开始时本程序将自动退出,请及时保存。")
	fmt.Println("═══════════════════════════════════════")

	maintenanceMu.Lock()
	if maintenanceTimer != nil {
		maintenanceTimer.Stop()
	}
	maintenanceTimer = time.AfterFunc(remaining, func() {
		showMaintenanceActive(message)
		os.Exit(0)
	})
	maintenanceMu.Unlock()
}

func cancelMaintenanceExit() {
	maintenanceMu.Lock()
	if maintenanceTimer != nil {
		maintenanceTimer.Stop()
		maintenanceTimer = nil
	}
	maintenanceMu.Unlock()
}

func showMaintenanceActive(message string) {
	fmt.Println()
	fmt.Println("═══════════════════════════════════════")
	fmt.Println("           系统维护中")
	fmt.Println("═══════════════════════════════════════")
	if strings.TrimSpace(message) != "" {
		fmt.Printf("公告: %s\n", strings.TrimSpace(message))
	}
	fmt.Println("系统正在维护,请稍后再试。程序即将退出。")
	fmt.Println("═══════════════════════════════════════")
}

func formatCountdown(d time.Duration) string {
	if d < 0 {
		d = 0
	}
	d = d.Round(time.Second)
	h := int(d.Hours())
	m := int(d.Minutes()) % 60
	s := int(d.Seconds()) % 60
	if h > 0 {
		return fmt.Sprintf("%d小时%d分%d秒", h, m, s)
	}
	if m > 0 {
		return fmt.Sprintf("%d分%d秒", m, s)
	}
	return fmt.Sprintf("%d秒", s)
}
