package main

import (
	"context"
	"errors"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog/log"

	"github.com/smakimka/balb/internal/server/notifier"
	"github.com/smakimka/balb/internal/server/router"
	"github.com/smakimka/balb/internal/server/storage"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	pool, err := pgxpool.New(ctx, "postgres://server:server_password@postgres:5432/server_db")
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

	days := os.Getenv("DAYS_BEFORE_NOTIFICATION")
	daysInt, err := strconv.Atoi(days)
	if err != nil {
		log.Err(err).Msg("error convertin days to int")
	}

	notifier := notifier.New(http.Client{}, s, daysInt)
	go notifier.Run(ctx)

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
