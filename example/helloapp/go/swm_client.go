package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"hash/fnv"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"time"

	swmsdk "software-web-manager/sdk-go"
)

// SWMClient SWM 更新客户端（封装 sdk-go，保持示例代码调用习惯）
type SWMClient struct {
	inner            *swmsdk.Client
	Channel          string
	Platform         string
	Arch             string
	DeviceID         string
	UserID           string
	Attributes       map[string]interface{}
	PublicKey        string
	VerifySignature  bool
	Retries          int
	Backoff          time.Duration
	HTTPTimeout      time.Duration
	EnableHTTPDebug  bool
	ClientIPOverride string
}

type UpdateStreamOptions = swmsdk.UpdateStreamOptions
type UpdatePushEvent = swmsdk.UpdatePushEvent
type ControlEvent = swmsdk.ControlEvent
type UpdateWatchHandle = swmsdk.UpdateWatchHandle

// UpdateResponse 更新检查响应
// sdk-go 当前不返回 version_code，这里保留字段以兼容现有示例输出。
type UpdateResponse struct {
	UpdateAvailable          bool                `json:"update_available"`
	Mandatory                bool                `json:"mandatory"`
	HeartbeatIntervalSeconds int                 `json:"heartbeat_interval_seconds"`
	OpenInBrowser            bool                `json:"open_in_browser"`
	DeliveryMethod           string              `json:"delivery_method"`
	ReleaseID                string              `json:"release_id"`
	Version                  string              `json:"version"`
	VersionCode              int                 `json:"version_code"`
	Notes                    string              `json:"notes"`
	DownloadURL              string              `json:"download_url"`
	ChecksumSHA256           string              `json:"checksum_sha256"`
	Signature                string              `json:"signature"`
	Size                     int64               `json:"size"`
	RollbackAllowed          bool                `json:"rollback_allowed"`
	Maintenance              *swmsdk.Maintenance `json:"maintenance,omitempty"`
}

// NewSWMClient 创建新的 SWM 客户端
func NewSWMClient(baseURL, appID, appSecret string) *SWMClient {
	cli := swmsdk.New(baseURL, appID, appSecret)
	channel := "stable"
	platform := runtime.GOOS
	arch := runtime.GOARCH
	deviceID := getDeviceID()

	client := &SWMClient{
		inner:            cli,
		Channel:          channel,
		Platform:         platform,
		Arch:             arch,
		DeviceID:         deviceID,
		UserID:           "",
		Attributes:       map[string]interface{}{},
		PublicKey:        "",
		VerifySignature:  false,
		Retries:          getOptionalIntField(cli, "Retries", 2),
		Backoff:          getOptionalDurationField(cli, "Backoff", 500*time.Millisecond),
		HTTPTimeout:      30 * time.Second,
		EnableHTTPDebug:  true,
		ClientIPOverride: "",
	}
	client.applyHTTPClientSettings()
	client.syncInner()
	return client
}

// SDK 返回底层 SDK 客户端，用于调用管理类接口。
func (c *SWMClient) SDK() *swmsdk.Client {
	c.syncInner()
	return c.inner
}

// CheckUpdate 检查是否有可用更新
func (c *SWMClient) CheckUpdate(currentVersion string, versionCode int) (*UpdateResponse, error) {
	c.syncInner()
	var vc *int
	if versionCode > 0 {
		vc = &versionCode
	}
	resp, err := c.inner.CheckUpdate(context.Background(), currentVersion, vc)
	if err != nil {
		return nil, fmt.Errorf("检查更新失败: %w", err)
	}
	return &UpdateResponse{
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
		Maintenance:              resp.Maintenance,
	}, nil
}

// Download 下载更新文件
func (c *SWMClient) Download(url, destPath string, expectedChecksum string, signature string, progress func(written, total int64)) error {
	c.syncInner()
	return c.inner.Download(context.Background(), url, destPath, expectedChecksum, signature, progress)
}

// ReportEvent 上报事件
func (c *SWMClient) ReportEvent(eventName string, properties map[string]interface{}) error {
	c.syncInner()
	return c.inner.ReportEvent(context.Background(), eventName, properties)
}

// ReportHeartbeat 上报心跳
func (c *SWMClient) ReportHeartbeat(appVersion string) error {
	c.syncInner()
	return c.inner.ReportHeartbeat(context.Background(), appVersion)
}

// ReportEvents 批量上报事件
func (c *SWMClient) ReportEvents(events []swmsdk.Event) error {
	c.syncInner()
	return c.inner.ReportEvents(context.Background(), events)
}

// ReportFeedback 上报反馈
func (c *SWMClient) ReportFeedback(content string, rating *int, contact string, attachments []string, metadata map[string]interface{}) error {
	c.syncInner()
	return c.inner.ReportFeedback(context.Background(), content, rating, contact, attachments, metadata)
}

func (c *SWMClient) syncInner() {
	c.inner.Channel = c.Channel
	c.inner.Platform = c.Platform
	c.inner.Arch = c.Arch
	c.inner.DeviceID = c.DeviceID
	setOptionalStringField(c.inner, "UserID", c.UserID)
	setOptionalMapField(c.inner, "Attributes", cloneInterfaceMap(c.Attributes))
	setOptionalStringField(c.inner, "PublicKey", c.PublicKey)
	setOptionalBoolField(c.inner, "VerifySignature", c.VerifySignature)
	setOptionalIntField(c.inner, "Retries", c.Retries)
	setOptionalDurationField(c.inner, "Backoff", c.Backoff)
}

func (c *SWMClient) applyHTTPClientSettings() {
	timeout := c.HTTPTimeout
	if timeout <= 0 {
		timeout = 30 * time.Second
	}
	var transport http.RoundTripper = http.DefaultTransport
	if c.EnableHTTPDebug {
		transport = &debugTransport{base: transport}
	}
	if strings.TrimSpace(c.ClientIPOverride) != "" {
		transport = &clientIPOverrideTransport{
			base:     transport,
			clientIP: strings.TrimSpace(c.ClientIPOverride),
		}
	}
	c.inner.HTTPClient = &http.Client{
		Timeout:   timeout,
		Transport: transport,
	}
}

func cloneInterfaceMap(in map[string]interface{}) map[string]interface{} {
	if len(in) == 0 {
		return map[string]interface{}{}
	}
	out := make(map[string]interface{}, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}

func getOptionalIntField(target interface{}, name string, fallback int) int {
	v := reflect.ValueOf(target)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}
	if !v.IsValid() || v.Kind() != reflect.Struct {
		return fallback
	}
	f := v.FieldByName(name)
	if !f.IsValid() || !f.CanInt() {
		return fallback
	}
	return int(f.Int())
}

func getOptionalDurationField(target interface{}, name string, fallback time.Duration) time.Duration {
	v := reflect.ValueOf(target)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}
	if !v.IsValid() || v.Kind() != reflect.Struct {
		return fallback
	}
	f := v.FieldByName(name)
	if !f.IsValid() || !f.CanInt() {
		return fallback
	}
	return time.Duration(f.Int())
}

func setOptionalStringField(target interface{}, name, value string) {
	v := reflect.ValueOf(target)
	if v.Kind() != reflect.Ptr {
		return
	}
	e := v.Elem()
	if !e.IsValid() || e.Kind() != reflect.Struct {
		return
	}
	f := e.FieldByName(name)
	if !f.IsValid() || !f.CanSet() || f.Kind() != reflect.String {
		return
	}
	f.SetString(value)
}

func setOptionalBoolField(target interface{}, name string, value bool) {
	v := reflect.ValueOf(target)
	if v.Kind() != reflect.Ptr {
		return
	}
	e := v.Elem()
	if !e.IsValid() || e.Kind() != reflect.Struct {
		return
	}
	f := e.FieldByName(name)
	if !f.IsValid() || !f.CanSet() || f.Kind() != reflect.Bool {
		return
	}
	f.SetBool(value)
}

func setOptionalIntField(target interface{}, name string, value int) {
	v := reflect.ValueOf(target)
	if v.Kind() != reflect.Ptr {
		return
	}
	e := v.Elem()
	if !e.IsValid() || e.Kind() != reflect.Struct {
		return
	}
	f := e.FieldByName(name)
	if !f.IsValid() || !f.CanSet() || !f.CanInt() {
		return
	}
	f.SetInt(int64(value))
}

func setOptionalDurationField(target interface{}, name string, value time.Duration) {
	v := reflect.ValueOf(target)
	if v.Kind() != reflect.Ptr {
		return
	}
	e := v.Elem()
	if !e.IsValid() || e.Kind() != reflect.Struct {
		return
	}
	f := e.FieldByName(name)
	if !f.IsValid() || !f.CanSet() || !f.CanInt() {
		return
	}
	f.SetInt(int64(value))
}

func setOptionalMapField(target interface{}, name string, value map[string]interface{}) {
	v := reflect.ValueOf(target)
	if v.Kind() != reflect.Ptr {
		return
	}
	e := v.Elem()
	if !e.IsValid() || e.Kind() != reflect.Struct {
		return
	}
	f := e.FieldByName(name)
	if !f.IsValid() || !f.CanSet() || f.Kind() != reflect.Map {
		return
	}
	if f.Type().Key().Kind() != reflect.String || f.Type().Elem().Kind() != reflect.Interface {
		return
	}
	f.Set(reflect.ValueOf(value))
}

func (c *SWMClient) StartUpdateStream(ctx context.Context, options UpdateStreamOptions, onEvent func(UpdatePushEvent)) (*UpdateWatchHandle, error) {
	c.syncInner()
	return c.inner.StartUpdateStream(ctx, options, onEvent)
}

func (c *SWMClient) WatchUpdates(ctx context.Context, options UpdateStreamOptions, onUpdateAvailable func(swmsdk.UpdateCheckResponse)) (*UpdateWatchHandle, error) {
	c.syncInner()
	return c.inner.WatchUpdates(ctx, options, onUpdateAvailable)
}

// getDeviceID 获取设备唯一标识
func getDeviceID() string {
	configDir, err := os.UserConfigDir()
	if err != nil {
		configDir = "."
	}

	deviceFile := filepath.Join(configDir, "helloapp", "device.id")

	if data, err := os.ReadFile(deviceFile); err == nil && len(data) > 0 {
		return string(data)
	}

	hostname, _ := os.Hostname()
	deviceID := generateDeviceID(hostname)

	_ = os.MkdirAll(filepath.Dir(deviceFile), 0o755)
	_ = os.WriteFile(deviceFile, []byte(deviceID), 0o644)

	return deviceID
}

// generateDeviceID 基于主机名生成设备ID
func generateDeviceID(hostname string) string {
	h := fnv.New64a()
	_, _ = h.Write([]byte(hostname + runtime.GOOS + runtime.GOARCH))
	return fmt.Sprintf("%s-%016x", runtime.GOOS, h.Sum64())
}

type debugTransport struct {
	base http.RoundTripper
}

type clientIPOverrideTransport struct {
	base     http.RoundTripper
	clientIP string
}

func (t *clientIPOverrideTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	base := t.base
	if base == nil {
		base = http.DefaultTransport
	}
	cloned := req.Clone(req.Context())
	cloned.Header = req.Header.Clone()
	if ip := strings.TrimSpace(t.clientIP); ip != "" {
		cloned.Header.Set("X-Forwarded-For", ip)
		cloned.Header.Set("X-Real-IP", ip)
	}
	return base.RoundTrip(cloned)
}

func (t *debugTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	base := t.base
	if base == nil {
		base = http.DefaultTransport
	}

	start := time.Now()
	bodyPreview := "<empty>"
	if req.Body != nil {
		raw, _ := io.ReadAll(req.Body)
		req.Body = io.NopCloser(bytes.NewReader(raw))
		bodyPreview = previewBody(raw, req.Header.Get("Content-Type"))
	}

	fmt.Printf("\n[HTTP Request] %s %s\n", req.Method, req.URL.String())
	fmt.Printf("[HTTP Request] Headers: %s\n", marshalHeaders(req.Header))
	fmt.Printf("[HTTP Request] Body: %s\n", bodyPreview)

	resp, err := base.RoundTrip(req)
	elapsed := time.Since(start)
	if err != nil {
		fmt.Printf("[HTTP Error] %s %s -> %v (elapsed=%s)\n", req.Method, req.URL.String(), err, elapsed)
		return nil, err
	}

	respBodyPreview := "<skipped>"
	if shouldPreviewResponse(resp) {
		raw, _ := io.ReadAll(resp.Body)
		resp.Body = io.NopCloser(bytes.NewReader(raw))
		respBodyPreview = previewBody(raw, resp.Header.Get("Content-Type"))
	}

	fmt.Printf("[HTTP Response] %s %s -> %d %s (elapsed=%s)\n", req.Method, req.URL.String(), resp.StatusCode, http.StatusText(resp.StatusCode), elapsed)
	fmt.Printf("[HTTP Response] Headers: %s\n", marshalHeaders(resp.Header))
	fmt.Printf("[HTTP Response] Body: %s\n", respBodyPreview)
	return resp, nil
}

func shouldPreviewResponse(resp *http.Response) bool {
	ct := strings.ToLower(resp.Header.Get("Content-Type"))
	if strings.Contains(ct, "application/json") || strings.Contains(ct, "text/") {
		return true
	}
	if resp.ContentLength >= 0 && resp.ContentLength <= 64*1024 {
		return true
	}
	return false
}

func previewBody(raw []byte, contentType string) string {
	if len(raw) == 0 {
		return "<empty>"
	}
	max := 4096
	if len(raw) > max {
		raw = raw[:max]
	}

	ct := strings.ToLower(contentType)
	if strings.Contains(ct, "application/json") {
		var out interface{}
		if err := json.Unmarshal(raw, &out); err == nil {
			if b, e := json.Marshal(out); e == nil {
				return string(b)
			}
		}
	}
	return string(raw)
}

func marshalHeaders(h http.Header) string {
	safe := http.Header{}
	for k, v := range h {
		if strings.EqualFold(k, "Authorization") && len(v) > 0 {
			safe[k] = []string{maskToken(v[0])}
			continue
		}
		if strings.EqualFold(k, "X-Signature") && len(v) > 0 {
			safe[k] = []string{maskToken(v[0])}
			continue
		}
		safe[k] = append([]string{}, v...)
	}
	raw, _ := json.Marshal(safe)
	return string(raw)
}

func maskToken(v string) string {
	if len(v) <= 10 {
		return "***"
	}
	return v[:10] + "..."
}
