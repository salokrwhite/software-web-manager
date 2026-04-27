package config

import "testing"

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

