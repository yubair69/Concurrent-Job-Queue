package config

import (
	"os"
	"testing"
	"time"
)

func TestLoadDefaults(t *testing.T) {
	// Clear relevant env vars to test defaults
	os.Unsetenv("GOTASK_PORT")
	os.Unsetenv("GOTASK_DB_PATH")
	os.Unsetenv("GOTASK_WORKER_COUNT")
	os.Unsetenv("GOTASK_QUEUE_CAPACITY")
	os.Unsetenv("GOTASK_DEFAULT_RETRIES")
	os.Unsetenv("GOTASK_RETRY_BASE_DELAY")
	os.Unsetenv("GOTASK_RETRY_MAX_DELAY")
	os.Unsetenv("GOTASK_SHUTDOWN_TIMEOUT")
	os.Unsetenv("GOTASK_LOG_LEVEL")

	cfg := Load()

	if cfg.Port != "8080" {
		t.Errorf("expected port '8080', got '%s'", cfg.Port)
	}
	if cfg.DatabasePath != "gotask.db" {
		t.Errorf("expected database path 'gotask.db', got '%s'", cfg.DatabasePath)
	}
	if cfg.WorkerCount != 4 {
		t.Errorf("expected worker count 4, got %d", cfg.WorkerCount)
	}
	if cfg.QueueCapacity != 1000 {
		t.Errorf("expected queue capacity 1000, got %d", cfg.QueueCapacity)
	}
	if cfg.DefaultRetries != 3 {
		t.Errorf("expected default retries 3, got %d", cfg.DefaultRetries)
	}
	if cfg.RetryBaseDelay != 1*time.Second {
		t.Errorf("expected retry base delay 1s, got %v", cfg.RetryBaseDelay)
	}
	if cfg.LogLevel != "info" {
		t.Errorf("expected log level 'info', got '%s'", cfg.LogLevel)
	}
}

func TestLoadEnvOverrides(t *testing.T) {
	os.Setenv("GOTASK_PORT", "9090")
	os.Setenv("GOTASK_WORKER_COUNT", "8")
	os.Setenv("GOTASK_LOG_LEVEL", "debug")
	defer func() {
		os.Unsetenv("GOTASK_PORT")
		os.Unsetenv("GOTASK_WORKER_COUNT")
		os.Unsetenv("GOTASK_LOG_LEVEL")
	}()

	cfg := Load()

	if cfg.Port != "9090" {
		t.Errorf("expected port '9090', got '%s'", cfg.Port)
	}
	if cfg.WorkerCount != 8 {
		t.Errorf("expected worker count 8, got %d", cfg.WorkerCount)
	}
	if cfg.LogLevel != "debug" {
		t.Errorf("expected log level 'debug', got '%s'", cfg.LogLevel)
	}
}
