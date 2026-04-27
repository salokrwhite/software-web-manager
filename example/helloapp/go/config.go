package main

import (
	"encoding/json"
	"fmt"
	"net"
	"os"
	"runtime"
	"strconv"
	"strings"
	"time"
)

const (
	defaultSWMBaseURL       = "http://localhost:8080"
	defaultSWMAppID         = "9bc50eaa-234d-43e2-a2b8-c28b8231b2d2"
	defaultSWMAppSecret     = "c0489d1c6ee291d2f3c2d0a55622c08eba875134b647b794bf3088260cd0711c"
	placeholderSWMAppID     = "00000000-0000-0000-0000-000000000000"
	placeholderSWMAppSecret = "replace-with-your-app-secret"
)

type swmRuntimeConfig struct {
	BaseURL   string
	AppID     string
	AppSecret string
}

type swmClientRuntimeConfig struct {
	Channel          string
	Platform         string
	Arch             string
	DeviceID         string
	UserID           string
	Attributes       map[string]interface{}
	VerifySignature  bool
	PublicKey        string
	HTTPTimeout      time.Duration
	Retries          int
	Backoff          time.Duration
	EnableHTTPDebug  bool
	ClientIPOverride string
}

var runtimeSWMConfig = loadSWMConfig()
var runtimeSWMClientConfig, runtimeSWMClientConfigErr = loadSWMClientConfig()

var (
	SWMBaseURL   = runtimeSWMConfig.BaseURL
	SWMAppID     = runtimeSWMConfig.AppID
	SWMAppSecret = runtimeSWMConfig.AppSecret
)

func loadSWMConfig() swmRuntimeConfig {
	baseURL := readFirstNonEmptyEnv("SWM_BASE_URL")
	if baseURL == "" {
		baseURL = defaultSWMBaseURL
	}
	baseURL = strings.TrimRight(baseURL, "/")

	appID := readFirstNonEmptyEnv("SWM_APP_ID")
	appSecret := readFirstNonEmptyEnv("SWM_APP_SECRET")

	if appID == "" {
		appID = defaultSWMAppID
	}
	if appSecret == "" {
		appSecret = defaultSWMAppSecret
	}

	return swmRuntimeConfig{
		BaseURL:   baseURL,
		AppID:     strings.TrimSpace(appID),
		AppSecret: strings.TrimSpace(appSecret),
	}
}

func loadSWMClientConfig() (swmClientRuntimeConfig, error) {
	cfg := swmClientRuntimeConfig{
		Channel:          strings.TrimSpace(readFirstNonEmptyEnv("HELLOAPP_CHANNEL", "SWM_CHANNEL_CODE")),
		Platform:         strings.TrimSpace(readFirstNonEmptyEnv("HELLOAPP_PLATFORM")),
		Arch:             strings.TrimSpace(readFirstNonEmptyEnv("HELLOAPP_ARCH")),
		DeviceID:         strings.TrimSpace(readFirstNonEmptyEnv("HELLOAPP_DEVICE_ID")),
		UserID:           strings.TrimSpace(readFirstNonEmptyEnv("HELLOAPP_USER_ID", "SWM_USER_ID")),
		VerifySignature:  parseBoolEnvOrDefault("SWM_VERIFY_SIGNATURE", false),
		PublicKey:        strings.TrimSpace(readFirstNonEmptyEnv("SWM_PUBLIC_KEY")),
		EnableHTTPDebug:  parseBoolEnvOrDefault("SWM_HTTP_DEBUG", true),
		ClientIPOverride: strings.TrimSpace(readFirstNonEmptyEnv("HELLOAPP_CLIENT_IP_OVERRIDE", "SWM_TEST_CLIENT_IP")),
	}
	if cfg.Channel == "" {
		cfg.Channel = "stable"
	}
	if cfg.Platform == "" {
		cfg.Platform = runtime.GOOS
	}
	if cfg.Arch == "" {
		cfg.Arch = runtime.GOARCH
	}

	timeoutSeconds := parseIntEnvOrDefault("SWM_HTTP_TIMEOUT_SECONDS", 30)
	if timeoutSeconds < 5 {
		timeoutSeconds = 30
	}
	cfg.HTTPTimeout = time.Duration(timeoutSeconds) * time.Second

	retries := parseIntEnvOrDefault("SWM_HTTP_RETRIES", 2)
	if retries < 0 {
		retries = 0
	}
	if retries > 10 {
		retries = 10
	}
	cfg.Retries = retries

	backoffMS := parseIntEnvOrDefault("SWM_HTTP_BACKOFF_MS", 500)
	if backoffMS < 50 {
		backoffMS = 500
	}
	cfg.Backoff = time.Duration(backoffMS) * time.Millisecond

	cfg.Attributes = map[string]interface{}{}
	rawAttrJSON := strings.TrimSpace(readFirstNonEmptyEnv("HELLOAPP_ATTRIBUTES_JSON", "SWM_ATTRIBUTES_JSON"))
	if rawAttrJSON != "" {
		if err := json.Unmarshal([]byte(rawAttrJSON), &cfg.Attributes); err != nil {
			return swmClientRuntimeConfig{}, fmt.Errorf("invalid attributes json in HELLOAPP_ATTRIBUTES_JSON/SWM_ATTRIBUTES_JSON: %w", err)
		}
	}

	if cfg.VerifySignature && cfg.PublicKey == "" {
		return swmClientRuntimeConfig{}, fmt.Errorf("SWM_VERIFY_SIGNATURE=true requires SWM_PUBLIC_KEY")
	}
	if cfg.ClientIPOverride != "" && net.ParseIP(cfg.ClientIPOverride) == nil {
		return swmClientRuntimeConfig{}, fmt.Errorf("invalid client ip override: %s", cfg.ClientIPOverride)
	}

	return cfg, nil
}

func readFirstNonEmptyEnv(keys ...string) string {
	for _, key := range keys {
		v := strings.TrimSpace(os.Getenv(key))
		if v != "" {
			return v
		}
	}
	return ""
}

func parseIntEnvOrDefault(key string, def int) int {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return def
	}
	v, err := strconv.Atoi(raw)
	if err != nil {
		return def
	}
	return v
}

func parseBoolEnvOrDefault(key string, def bool) bool {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return def
	}
	v, err := strconv.ParseBool(raw)
	if err != nil {
		return def
	}
	return v
}

func validateSWMConfig() error {
	if strings.TrimSpace(SWMBaseURL) == "" {
		return fmt.Errorf("missing SWM_BASE_URL")
	}
	if strings.TrimSpace(SWMAppID) == "" || SWMAppID == placeholderSWMAppID {
		return fmt.Errorf("missing SWM app id, set SWM_APP_ID")
	}
	if strings.TrimSpace(SWMAppSecret) == "" || SWMAppSecret == placeholderSWMAppSecret {
		return fmt.Errorf("missing SWM app secret, set SWM_APP_SECRET")
	}
	if runtimeSWMClientConfigErr != nil {
		return runtimeSWMClientConfigErr
	}
	return nil
}

func newConfiguredSWMClient() (*SWMClient, error) {
	if err := validateSWMConfig(); err != nil {
		return nil, err
	}
	client := NewSWMClient(SWMBaseURL, SWMAppID, SWMAppSecret)
	client.Channel = runtimeSWMClientConfig.Channel
	client.Platform = runtimeSWMClientConfig.Platform
	client.Arch = runtimeSWMClientConfig.Arch
	if runtimeSWMClientConfig.DeviceID != "" {
		client.DeviceID = runtimeSWMClientConfig.DeviceID
	}
	client.UserID = runtimeSWMClientConfig.UserID
	client.Attributes = cloneInterfaceMap(runtimeSWMClientConfig.Attributes)
	client.VerifySignature = runtimeSWMClientConfig.VerifySignature
	client.PublicKey = runtimeSWMClientConfig.PublicKey
	client.HTTPTimeout = runtimeSWMClientConfig.HTTPTimeout
	client.Retries = runtimeSWMClientConfig.Retries
	client.Backoff = runtimeSWMClientConfig.Backoff
	client.EnableHTTPDebug = runtimeSWMClientConfig.EnableHTTPDebug
	client.ClientIPOverride = runtimeSWMClientConfig.ClientIPOverride
	client.applyHTTPClientSettings()
	client.syncInner()
	return client, nil
}
