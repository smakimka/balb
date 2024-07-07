package main

import (
	"context"
	"errors"
	"net/http"
	"os"
	"strconv"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog/log"
	"github.com/smakimka/balb/internal/bot/bot"
	"github.com/smakimka/balb/internal/bot/notifier"
	"github.com/smakimka/balb/internal/bot/router"
	"github.com/smakimka/balb/internal/bot/storage"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	pool, err := pgxpool.New(ctx, "postgres://bot:bot_password@postgres:5432/bot_db")
	if err != nil {
		log.Err(err).Msg("eror creating pool")
		return
	}

	s := storage.NewPGStorage(pool)
	if err = waitForPostgres(ctx, s); err != nil {
		log.Err(err).Msg("error connecting to postgres")
		return
	}

	if err = s.Init(ctx); err != nil {
		log.Err(err).Msg("eror initializing storage")
		return
	}

	authToken := os.Getenv("AUTH_TOKEN")
	botToken := os.Getenv("BOT_TOKEN")

	if authToken == "" || botToken == "" {
		log.Error().Msg("auth token or bot token or both are empty")
		return
	}

	adminChatIDStr := os.Getenv("ADMIN_CHAT_ID")
	adminChatID, err := strconv.Atoi(adminChatIDStr)
	if err != nil {
		log.Err(err).Msg("error converting admin chat id")
		return
	}

	api, err := tgbotapi.NewBotAPI(botToken)
	if err != nil {
		log.Err(err).Msg("error creating api")
		return
	}

	notifier := notifier.New(s, api, adminChatID)
	go notifier.Run(ctx)

	bot := bot.New(
		api,
		"test",
		http.Client{},
		s,
		adminChatID,
	)

	go bot.StartPolling(ctx)

	log.Info().Msg("listening on :8090")
	if err := http.ListenAndServe(":8090", router.New(s)); err != nil {
		log.Err(err).Msg("error")
		return
	}
}

func waitForPostgres(ctx context.Context, s *storage.PGStorage) error {
	timeoutTimer := time.NewTimer(10 * time.Second)

	okChan := make(chan struct{})
	go func() {
		for {
			err := s.Ping(ctx)
			if err != nil {
				time.Sleep(1 * time.Second)
				continue
			}

			okChan <- struct{}{}
			break
		}
	}()

	select {
	case <-timeoutTimer.C:
		return errors.New("Couldn't reach postgres'")
	case <-okChan:
		return nil
	}
}
