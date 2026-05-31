package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
	_ "time/tzdata"

	"player-stats-backend-go/internal/auth"
	"player-stats-backend-go/internal/config"
	"player-stats-backend-go/internal/httpapi"
	"player-stats-backend-go/internal/importer"
	"player-stats-backend-go/internal/settings"
	"player-stats-backend-go/internal/share"
	"player-stats-backend-go/internal/stats"
	"player-stats-backend-go/internal/storage"
)

func main() {
	cfg := config.Load()
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))

	startupCtx, startupCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer startupCancel()
	db, err := storage.Open(startupCtx, cfg)
	if err != nil {
		logger.Error("mysql connection failed", "error", err)
		os.Exit(1)
	}
	defer db.Close()
	if err := storage.Migrate(startupCtx, db); err != nil {
		logger.Error("schema migration failed", "error", err)
		os.Exit(1)
	}
	if err := storage.EnsureDefaults(startupCtx, db, cfg); err != nil {
		logger.Error("default config initialization failed", "error", err)
		os.Exit(1)
	}
	authService := auth.NewService(db, cfg)
	if err := authService.EnsureAdmin(startupCtx); err != nil {
		logger.Error("admin initialization failed", "error", err)
		os.Exit(1)
	}
	settingsService := settings.NewService(db, cfg)
	statsService := stats.NewService(db, cfg)
	importService := importer.NewService(db, cfg, settingsService)
	importJobs := importer.NewJobService(importService)
	logQueryService := importer.NewLogQueryService(importService)
	xrayService := importer.NewXrayAnalysisService(importService)
	shareService := share.NewService(db, cfg, settingsService, statsService, xrayService)
	autoScheduler := importer.NewAutoTaskScheduler(settingsService, importService, logger, cfg.Location)
	autoScheduler.Start()
	defer autoScheduler.Stop()

	app := httpapi.NewServer(cfg, logger, authService, settingsService, shareService, importService, importJobs, logQueryService, xrayService, statsService)
	server := &http.Server{
		Addr:              ":" + cfg.ServerPort,
		Handler:           app.Handler(),
		ReadHeaderTimeout: 10 * time.Second,
	}

	go func() {
		logger.Info("go backend listening", "addr", server.Addr)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error("server stopped unexpectedly", "error", err)
			os.Exit(1)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := server.Shutdown(ctx); err != nil {
		logger.Error("graceful shutdown failed", "error", err)
	}
}
