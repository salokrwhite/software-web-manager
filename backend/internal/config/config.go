package config

import (
	"fmt"
	"net"
	"os"
	"strings"

	"github.com/joho/godotenv"
)

// Dev-only default secrets. These are intentionally weak placeholders for local
// development. Config.Validate refuses to start in prod while any of these are
// still in effect, so they can never silently ship to production.
const (
	DevJWTSecret          = "dev-secret"
	DevAppSecretMasterKey = "dev-app-secret-master-key"
	// DevAuthzKeyID/DevAuthzSigningKey are a throwaway Ed25519 dev keypair. The
	// matching public key is embedded in the client for local testing only.
	// Prod MUST override AUTHZ_SIGNING_PRIVATE_KEY / AUTHZ_KEY_ID via env.
	DevAuthzKeyID      = "authz-dev"
	DevAuthzSigningKey = "3d6bd866a72a631ad51e4c495b6dd81062d8a36acb28dbf245bc37bfb3734b28"

	// minSecretLen is the minimum length we require for production secrets.
	minSecretLen = 32
)

type Config struct {
	Env                string
	HTTPAddr           string
	DatabaseURL        string
	RedisURL           string
	JWTSecret          string
	JWTIssuer          string
	AppSecretMasterKey string
	AuthzSigningKey    string
	AuthzKeyID         string
	AccessTokenMinutes int
	RefreshTokenHours  int
	CORSOrigins        []string
	ClientIPAllowlist  []string
	ClientRateLimitRPS int
	ClientRateLimitBurst int
	PersonalAppLimit   int
	OnlineWindowSeconds int
	OnlineStreamIntervalSeconds int

	StorageDriver      string
	LocalStoragePath   string
	LocalPublicBaseURL string
	WebBaseURL         string

	S3Endpoint      string
	S3Region        string
	S3Bucket        string
	S3AccessKey     string
	S3SecretKey     string
	S3UsePathStyle  bool
	S3PublicBaseURL string

	IP2RegionV4Path     string
	IP2RegionV6Path     string
	IP2RegionCachePolicy string
	IP2RegionPoolSize    int

	// ESA（阿里云边缘安全加速）托管转换注入的可信头。仅在确认流量必经 ESA 时
	// 才信任（TrustESAGeoHeaders=true）；否则普通客户端可伪造这些头绕过地域限制。
	TrustESAGeoHeaders bool
	ESARealIPHeader    string
	ESACountryHeader   string
	ESACityHeader      string
	// PreferServerSideRegion：地域来源"服务端优先"总开关（防伪造）。
	// true（默认）= ESA > ip2region > 客户端自报（自报降为最低兜底）；
	// false = 回退旧版"客户端自报最优先"。与是否接入 ESA 无关。
	PreferServerSideRegion bool

	RunMigrations bool
	EnableEmbeddedWorker bool
	WorkerIntervalSeconds int
}

func Load() Config {
	_ = godotenv.Load()

	cfg := Config{
		Env:                getEnv("APP_ENV", "dev"),
		HTTPAddr:           getEnv("HTTP_ADDR", ":8080"),
		DatabaseURL:        getEnv("DATABASE_URL", "swm:swm@tcp(localhost:3306)/swmanager?charset=utf8mb4&parseTime=true&loc=Local&multiStatements=true"),
		RedisURL:           getEnv("REDIS_URL", "redis://localhost:6379/0"),
		JWTSecret:          getEnv("JWT_SECRET", DevJWTSecret),
		JWTIssuer:          getEnv("JWT_ISSUER", "swm"),
		AppSecretMasterKey: getEnv("APP_SECRET_MASTER_KEY", DevAppSecretMasterKey),
		AuthzSigningKey:    getEnv("AUTHZ_SIGNING_PRIVATE_KEY", DevAuthzSigningKey),
		AuthzKeyID:         getEnv("AUTHZ_KEY_ID", DevAuthzKeyID),
		AccessTokenMinutes: getEnvInt("ACCESS_TOKEN_MINUTES", 30),
		RefreshTokenHours:  getEnvInt("REFRESH_TOKEN_HOURS", 720),
		CORSOrigins:        splitCSV(getEnv("CORS_ORIGINS", "*")),
		ClientIPAllowlist:  splitCSV(getEnv("CLIENT_IP_ALLOWLIST", "")),
		ClientRateLimitRPS: getEnvInt("CLIENT_RATE_LIMIT_RPS", 20),
		ClientRateLimitBurst: getEnvInt("CLIENT_RATE_LIMIT_BURST", 40),
		PersonalAppLimit:   getEnvInt("PERSONAL_APP_LIMIT", 10),
		OnlineWindowSeconds: getEnvInt("ONLINE_WINDOW_SECONDS", 120),
		OnlineStreamIntervalSeconds: getEnvInt("ONLINE_STREAM_INTERVAL_SECONDS", 3),
		StorageDriver:      getEnv("STORAGE_DRIVER", "local"),
		LocalStoragePath:   getEnv("LOCAL_STORAGE_PATH", "./data/files"),
		LocalPublicBaseURL: getEnv("LOCAL_PUBLIC_BASE_URL", ""),
		WebBaseURL:         getEnv("WEB_BASE_URL", ""),
		S3Endpoint:         getEnv("S3_ENDPOINT", "http://localhost:9000"),
		S3Region:           getEnv("S3_REGION", "us-east-1"),
		S3Bucket:           getEnv("S3_BUCKET", "swm"),
		S3AccessKey:        getEnv("S3_ACCESS_KEY", "minioadmin"),
		S3SecretKey:        getEnv("S3_SECRET_KEY", "minioadmin"),
		S3UsePathStyle:     getEnvBool("S3_USE_PATH_STYLE", true),
		S3PublicBaseURL:    getEnv("S3_PUBLIC_BASE_URL", ""),
		IP2RegionV4Path:    getEnv("IP2REGION_V4_XDB_PATH", "./iplocation/data/ip2region_v4.xdb"),
		IP2RegionV6Path:    getEnv("IP2REGION_V6_XDB_PATH", "./iplocation/data/ip2region_v6.xdb"),
		IP2RegionCachePolicy: getEnv("IP2REGION_CACHE_POLICY", "vindex"),
		IP2RegionPoolSize:  getEnvInt("IP2REGION_POOL_SIZE", 20),
		TrustESAGeoHeaders:     getEnvBool("ESA_GEO_HEADERS_TRUSTED", false),
		ESARealIPHeader:        getEnv("ESA_REAL_IP_HEADER", "ali-real-client-ip"),
		ESACountryHeader:       getEnv("ESA_IP_COUNTRY_HEADER", "ali-ip-country"),
		ESACityHeader:          getEnv("ESA_IP_CITY_HEADER", "ali-ip-city"),
		PreferServerSideRegion: getEnvBool("PREFER_SERVERSIDE_REGION", true),
		RunMigrations:      getEnvBool("RUN_MIGRATIONS", true),
		EnableEmbeddedWorker: getEnvBool("ENABLE_EMBEDDED_WORKER", true),
		WorkerIntervalSeconds: getEnvInt("WORKER_INTERVAL_SECONDS", 3600),
	}

	if cfg.WorkerIntervalSeconds < 60 {
		cfg.WorkerIntervalSeconds = 3600
	}

	if cfg.LocalPublicBaseURL == "" {
		cfg.LocalPublicBaseURL = deriveLocalPublicBaseURL(cfg.HTTPAddr)
	}

	return cfg
}

// IsProd reports whether the service is configured for a production environment.
func (c Config) IsProd() bool {
	env := strings.ToLower(strings.TrimSpace(c.Env))
	return env == "prod" || env == "production"
}

// Validate enforces that production deployments do not run with the weak
// development defaults. It is a no-op outside of prod so local development keeps
// working out of the box. Returns a non-nil error listing every problem found.
func (c Config) Validate() error {
	if !c.IsProd() {
		return nil
	}

	var problems []string

	checkSecret := func(name, value, devDefault string) {
		v := strings.TrimSpace(value)
		switch {
		case v == "" || v == devDefault:
			problems = append(problems, name+" must be set to a strong, non-default value in prod")
		case len(v) < minSecretLen:
			problems = append(problems, fmt.Sprintf("%s must be at least %d characters in prod", name, minSecretLen))
		}
	}

	checkSecret("JWT_SECRET", c.JWTSecret, DevJWTSecret)
	checkSecret("APP_SECRET_MASTER_KEY", c.AppSecretMasterKey, DevAppSecretMasterKey)

	authzKey := strings.TrimSpace(c.AuthzSigningKey)
	if authzKey == "" || authzKey == DevAuthzSigningKey {
		problems = append(problems, "AUTHZ_SIGNING_PRIVATE_KEY must be set to a production Ed25519 key (the dev key is rejected in prod)")
	}
	if strings.TrimSpace(c.AuthzKeyID) == "" || strings.TrimSpace(c.AuthzKeyID) == DevAuthzKeyID {
		problems = append(problems, "AUTHZ_KEY_ID must be set to a non-default value in prod")
	}

	if len(problems) > 0 {
		return fmt.Errorf("insecure production config:\n  - %s", strings.Join(problems, "\n  - "))
	}
	return nil
}

func getEnv(key, def string) string {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return def
	}
	return v
}

func getEnvInt(key string, def int) int {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return def
	}
	parsed := 0
	_, err := fmt.Sscanf(v, "%d", &parsed)
	if err != nil {
		return def
	}
	return parsed
}

func getEnvBool(key string, def bool) bool {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return def
	}
	switch strings.ToLower(v) {
	case "1", "true", "yes", "y", "on":
		return true
	case "0", "false", "no", "n", "off":
		return false
	default:
		return def
	}
}

func splitCSV(input string) []string {
	if input == "" {
		return nil
	}
	parts := strings.Split(input, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

func deriveLocalPublicBaseURL(httpAddr string) string {
	addr := strings.TrimSpace(httpAddr)
	if addr == "" {
		return "http://localhost:8080/files"
	}
	host := ""
	port := ""
	if strings.HasPrefix(addr, ":") {
		port = strings.TrimPrefix(addr, ":")
	} else {
		var err error
		host, port, err = net.SplitHostPort(addr)
		if err != nil {
			if isAllDigits(addr) {
				port = addr
			} else {
				host = addr
			}
		}
	}
	if host == "" || host == "0.0.0.0" || host == "::" {
		host = "localhost"
	}
	if port != "" {
		return fmt.Sprintf("http://%s:%s/files", host, port)
	}
	return fmt.Sprintf("http://%s/files", host)
}

func isAllDigits(input string) bool {
	if input == "" {
		return false
	}
	for _, ch := range input {
		if ch < '0' || ch > '9' {
			return false
		}
	}
	return true
}

