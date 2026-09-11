package main

import (
	"context"
	"flag"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"seshhub/internal/config"
	"seshhub/internal/db"
	"seshhub/internal/web"
	"seshhub/internal/yt"
)

func main() {
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stdout, nil)))
	cfg := config.Load()
	if err := cfg.ApplyFlags(flag.CommandLine, os.Args[1:]); err != nil {
		os.Exit(2)
	}

	sqldb, err := db.Open(cfg.DatabaseURL)
	if err != nil {
		slog.Error("database", "err", err)
		os.Exit(1)
	}
	defer sqldb.Close()
	if err := db.Migrate(sqldb, cfg.MigrationsDir); err != nil {
		slog.Error("migrate", "err", err)
		os.Exit(1)
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	go yt.Loop(ctx, sqldb, yt.Client{Key: cfg.YouTubeAPIKey, Channel: cfg.YouTubeChannelID}, cfg.YouTubeSyncEvery)

	srv := &http.Server{
		Addr:              cfg.ListenAddr(),
		Handler:           web.New(cfg, sqldb),
		ReadHeaderTimeout: 10 * time.Second,
	}
	go func() {
		slog.Info("listening", "addr", srv.Addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("listen", "err", err)
			os.Exit(1)
		}
	}()
	<-ctx.Done()
	shut, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = srv.Shutdown(shut)
}
