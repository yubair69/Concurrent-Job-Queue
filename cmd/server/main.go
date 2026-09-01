package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/gotask/gotask/internal/api"
	"github.com/gotask/gotask/internal/config"
	"github.com/gotask/gotask/internal/logging"
)

func main() {
	cfg := config.Load()
	logger := logging.Setup(cfg.LogLevel)

	logger.Info("initializing gotask server", "port", cfg.Port, "db", cfg.DatabasePath)

	srv := api.NewServer(cfg, logger)

	// Channel to listen for interrupt or terminate signals from the OS
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		if err := srv.Start(); err != nil {
			logger.Error("server error", "error", err)
		}
	}()

	sig := <-sigChan
	logger.Info("received shutdown signal", "signal", sig.String())

	// Context with timeout for graceful shutdown
	ctx, cancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		logger.Error("server forced to shutdown", "error", err)
	}

	logger.Info("gotask server stopped gracefully")
}
