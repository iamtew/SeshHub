package main

import (
	"context"
	"database/sql"
	"flag"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"seshhub/internal/auth"
	"seshhub/internal/bot"
	"seshhub/internal/config"
	"seshhub/internal/db"
	"seshhub/internal/spot"
	"seshhub/internal/twitch"
	"seshhub/internal/web"
	"seshhub/internal/yt"
)

func main() {
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stdout, nil)))
	cfg := config.Load()
	if err := cfg.ApplyFlags(flag.CommandLine, os.Args[1:]); err != nil {
		os.Exit(2)
	}
	spot.EpisodeURL = "https://" + cfg.SubottoInstance + "/api/get/episode/sesh-sofa"

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
	if err := auth.PurgeExpiredSessions(sqldb); err != nil {
		slog.Error("purge sessions", "err", err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	if cfg.YouTubeAPIKey != "" {
		go pollYouTube(ctx, sqldb, cfg.YouTubeAPIKey)
	}

	b := bot.Open(sqldb, cfg.DiscordBotToken, cfg.DiscordGuildID)
	defer b.Close()
	if cfg.TwitchEnabled() {
		go pollTwitch(ctx, sqldb, twitch.New(cfg.TwitchClientID, cfg.TwitchClientSecret), b)
	}

	srv := &http.Server{
		Addr:              cfg.ListenAddr(),
		Handler:           web.New(cfg, sqldb, b),
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

func pollYouTube(ctx context.Context, db *sql.DB, key string) {
	run := func() {
		n, err := yt.PollPublic(ctx, db, key)
		if err != nil {
			slog.Error("youtube poll", "err", err)
			return
		}
		slog.Info("youtube poll", "updated", n)
	}
	run()
	t := time.NewTicker(time.Hour)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			run()
		}
	}
}

// ponytail: 60s Helix poll; EventSub stream.online if delay is too slow.
func pollTwitch(ctx context.Context, db *sql.DB, c *twitch.Client, b *bot.Bot) {
	run := func() {
		went, err := twitch.Poll(ctx, db, c)
		if err != nil {
			slog.Error("twitch poll", "err", err)
			return
		}
		if len(went) == 0 {
			return
		}
		msg := twitch.DefaultMsg
		if b != nil {
			if s, err := b.Settings(); err == nil && strings.TrimSpace(s.TwitchMsg) != "" {
				msg = s.TwitchMsg
			}
		}
		for _, ev := range went {
			if b != nil {
				b.Announce("twitch", twitch.Render(msg, ev))
			}
		}
		slog.Info("twitch poll", "live", len(went))
	}
	run()
	t := time.NewTicker(time.Minute)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			run()
		}
	}
}
