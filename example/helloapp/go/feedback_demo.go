package main

import (
	"errors"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"os"
	"strconv"
	"strings"
	"time"

	swmsdk "software-web-manager/sdk-go"
)

func runFeedbackSmokeDemoIfEnabled() {
	if !parseBoolEnv("HELLOAPP_REPORT_FEEDBACK", false) {
		return
	}
	client, err := newConfiguredSWMClient()
	if err != nil {
		fmt.Printf("[Feedback Demo] 跳过：%v\n", err)
		return
	}

	content := strings.TrimSpace(os.Getenv("HELLOAPP_FEEDBACK_CONTENT"))
	if content == "" {
		content = "HelloApp feedback smoke test"
	}
	contact := strings.TrimSpace(os.Getenv("HELLOAPP_FEEDBACK_CONTACT"))
	if contact == "" {
		contact = "helloapp@example.local"
	}
	rating := parseFeedbackRating()

	attachmentPath := strings.TrimSpace(os.Getenv("HELLOAPP_FEEDBACK_ATTACHMENT"))
	tempAttachment := ""
	if attachmentPath == "" {
		tempAttachment, err = createFeedbackDemoPNG()
		if err != nil {
			fmt.Printf("[Feedback Demo] 生成测试截图失败: %v\n", err)
			return
		}
		defer os.Remove(tempAttachment)
		attachmentPath = tempAttachment
	}

	err = client.ReportFeedback(content, rating, contact, []string{attachmentPath}, map[string]interface{}{
		"app_version":  Version,
		"version_code": VersionCode,
		"platform":     client.Platform,
		"arch":         client.Arch,
		"demo_source":  "helloapp",
		"submitted_at": time.Now().Format(time.RFC3339),
	})
	if errors.Is(err, swmsdk.ErrFeedbackDisabled) {
		fmt.Println("[Feedback Demo] 用户反馈已被服务端关闭，SDK 上报被拒绝。")
		return
	}
	if err != nil {
		fmt.Printf("[Feedback Demo] 上报失败: %v\n", err)
		return
	}
	fmt.Printf("[Feedback Demo] 已上报用户反馈，附件: %s\n", attachmentPath)
}

func parseFeedbackRating() *int {
	raw := strings.TrimSpace(os.Getenv("HELLOAPP_FEEDBACK_RATING"))
	if raw == "" {
		v := 5
		return &v
	}
	v, err := strconv.Atoi(raw)
	if err != nil || v < 1 || v > 5 {
		return nil
	}
	return &v
}

func createFeedbackDemoPNG() (string, error) {
	file, err := os.CreateTemp("", "helloapp-feedback-*.png")
	if err != nil {
		return "", err
	}
	defer file.Close()

	img := image.NewRGBA(image.Rect(0, 0, 640, 360))
	draw.Draw(img, img.Bounds(), &image.Uniform{C: color.RGBA{R: 246, G: 248, B: 250, A: 255}}, image.Point{}, draw.Src)
	draw.Draw(img, image.Rect(32, 32, 608, 328), &image.Uniform{C: color.RGBA{R: 25, G: 118, B: 210, A: 255}}, image.Point{}, draw.Src)
	draw.Draw(img, image.Rect(48, 52, 592, 120), &image.Uniform{C: color.RGBA{R: 255, G: 255, B: 255, A: 255}}, image.Point{}, draw.Src)
	draw.Draw(img, image.Rect(48, 150, 420, 182), &image.Uniform{C: color.RGBA{R: 187, G: 222, B: 251, A: 255}}, image.Point{}, draw.Src)
	draw.Draw(img, image.Rect(48, 204, 520, 236), &image.Uniform{C: color.RGBA{R: 187, G: 222, B: 251, A: 255}}, image.Point{}, draw.Src)
	draw.Draw(img, image.Rect(48, 258, 300, 290), &image.Uniform{C: color.RGBA{R: 187, G: 222, B: 251, A: 255}}, image.Point{}, draw.Src)

	if err := png.Encode(file, img); err != nil {
		_ = os.Remove(file.Name())
		return "", err
	}
	return file.Name(), nil
}
