package config

import (
	"strings"
	"testing"
)

func TestLoadEmbeddedWorkerDefaultsWhenInvalidInterval(t *testing.T) {
	t.Setenv("ENABLE_EMBEDDED_WORKER", "true")
	t.Setenv("WORKER_INTERVAL_SECONDS", "10")

	cfg := Load()
	if !cfg.EnableEmbeddedWorker {
		t.Fatalf("expected EnableEmbeddedWorker=true")
	}
	if cfg.WorkerIntervalSeconds != 3600 {
		t.Fatalf("expected WorkerIntervalSeconds fallback to 3600, got %d", cfg.WorkerIntervalSeconds)
	}
}

func TestLoadEmbeddedWorkerDisabled(t *testing.T) {
	t.Setenv("ENABLE_EMBEDDED_WORKER", "false")
	t.Setenv("WORKER_INTERVAL_SECONDS", "120")

	cfg := Load()
	if cfg.EnableEmbeddedWorker {
		t.Fatalf("expected EnableEmbeddedWorker=false")
	}
	if cfg.WorkerIntervalSeconds != 120 {
		t.Fatalf("expected WorkerIntervalSeconds=120, got %d", cfg.WorkerIntervalSeconds)
	}
}

func strongProdConfig() Config {
	return Config{
		Env:                   "prod",
		JWTSecret:             strings.Repeat("a", 40),
		AppSecretMasterKey:    strings.Repeat("b", 40),
		AuthzSigningKey:       strings.Repeat("c", 64),
		AuthzKeyID:            "authz-prod-1",
		AuthzPlatformFallback: true, // production default (see Load)
	}
}

func TestValidateDevIsNoop(t *testing.T) {
	c := Config{Env: "dev"} // all weak defaults, but dev => allowed
	if err := c.Validate(); err != nil {
		t.Fatalf("dev should pass: %v", err)
	}
}

func TestValidateProdHappy(t *testing.T) {
	if err := strongProdConfig().Validate(); err != nil {
		t.Fatalf("strong prod config should pass: %v", err)
	}
}

func TestValidateProdRejectsWeakConfig(t *testing.T) {
	cases := map[string]func(c *Config){
		"jwt dev default":    func(c *Config) { c.JWTSecret = DevJWTSecret },
		"master dev default": func(c *Config) { c.AppSecretMasterKey = DevAppSecretMasterKey },
		"authz dev key":      func(c *Config) { c.AuthzSigningKey = DevAuthzSigningKey },
		"authz dev key id":   func(c *Config) { c.AuthzKeyID = DevAuthzKeyID },
		"jwt too short":      func(c *Config) { c.JWTSecret = "short" },
		"master too short":   func(c *Config) { c.AppSecretMasterKey = "short" },
		"authz key empty":    func(c *Config) { c.AuthzSigningKey = "" },
		"authz key id empty": func(c *Config) { c.AuthzKeyID = "" },
	}
	for name, mutate := range cases {
		c := strongProdConfig()
		mutate(&c)
		if err := c.Validate(); err == nil {
			t.Errorf("%s: expected validation error, got nil", name)
		}
	}
}

// When the platform fallback is disabled, every app is expected to carry its own
// authz key, so the platform key is no longer required and may be retired.
func TestValidateProdAllowsMissingPlatformKeyWhenFallbackOff(t *testing.T) {
	c := strongProdConfig()
	c.AuthzPlatformFallback = false
	c.AuthzSigningKey = ""
	c.AuthzKeyID = ""
	if err := c.Validate(); err != nil {
		t.Fatalf("expected no error when fallback off and platform key absent, got: %v", err)
	}
}
