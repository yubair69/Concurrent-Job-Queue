package config

import (
	"os"
	"strconv"
	"time"
)

type Config struct {
	Port            string
	DatabasePath    string
	WorkerCount     int
	QueueCapacity   int
	DefaultRetries  int
	RetryBaseDelay  time.Duration
	RetryMaxDelay   time.Duration
	ShutdownTimeout time.Duration
	LogLevel        string
}

func Load() *Config {
	return &Config{
		Port:            getEnv("GOTASK_PORT", "8080"),
		DatabasePath:    getEnv("GOTASK_DB_PATH", "gotask.db"),
		WorkerCount:     getEnvAsInt("GOTASK_WORKER_COUNT", 4),
		QueueCapacity:   getEnvAsInt("GOTASK_QUEUE_CAPACITY", 1000),
		DefaultRetries:  getEnvAsInt("GOTASK_DEFAULT_RETRIES", 3),
		RetryBaseDelay:  getEnvAsDuration("GOTASK_RETRY_BASE_DELAY", 1*time.Second),
		RetryMaxDelay:   getEnvAsDuration("GOTASK_RETRY_MAX_DELAY", 30*time.Second),
		ShutdownTimeout: getEnvAsDuration("GOTASK_SHUTDOWN_TIMEOUT", 10*time.Second),
		LogLevel:        getEnv("GOTASK_LOG_LEVEL", "info"),
	}
}

func getEnv(key, fallback string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return fallback
}

func getEnvAsInt(key string, fallback int) int {
	if valueStr, exists := os.LookupEnv(key); exists {
		if value, err := strconv.Atoi(valueStr); err == nil {
			return value
		}
	}
	return fallback
}

func getEnvAsDuration(key string, fallback time.Duration) time.Duration {
	if valueStr, exists := os.LookupEnv(key); exists {
		if value, err := time.ParseDuration(valueStr); err == nil {
			return value
		}
	}
	return fallback
}
