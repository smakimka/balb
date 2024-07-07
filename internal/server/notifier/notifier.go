package notifier

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/smakimka/balb/internal/model"
	"github.com/smakimka/balb/internal/server/storage"
)

type Notifier struct {
	c                  http.Client
	s                  storage.Storage
	daysBeforeBirthday int
}

func New(c http.Client, s storage.Storage, daysBeforeBirthday int) *Notifier {
	return &Notifier{c: c, s: s, daysBeforeBirthday: daysBeforeBirthday}
}

// Run Тикеры или не тикеры, а что-то лучше должны срабатывать один раз в день, например в 9 часов, но для теста пусть будет так
func (n *Notifier) Run(ctx context.Context) {
	log.Info().Msg("started notifier goroutine")
	notifyTicker := time.NewTicker(10 * time.Second)
	oldTicker := time.NewTicker(10 * time.Second)

	for {
		select {
		case <-ctx.Done():
			return
		case <-notifyTicker.C:
			go n.sendNotifications(ctx)
		case <-oldTicker.C:
			go n.resetNotified(ctx)
		}
	}
}

func (n *Notifier) sendNotifications(ctx context.Context) {
	res, err := n.s.GetBirthdays(ctx, n.daysBeforeBirthday)
	if err != nil {
		log.Err(err).Msg("error getting birthdays")
		return
	}

	for _, birthday := range res {
		log.Info().Msgf("sending notification for %s", birthday.FIO)

		body, err := json.Marshal(birthday)
		if err != nil {
			log.Err(err).Msg("error marshaling notification body")
			continue
		}

		var resp *http.Response
		switch birthday.Front {
		case model.TelegramFront:
			resp, err = n.c.Post(fmt.Sprintf("http://bot:8090/notify"), "application/json", bytes.NewReader(body))
			if err != nil {
				log.Err(err).Msg("error sending notification")
				continue
			}
		default:
			continue
		}

		if resp.StatusCode != http.StatusOK {
			log.Error().Int("code", resp.StatusCode).Msg("error sending notification")
			continue
		}

		if err = n.s.SetNotified(ctx, birthday.ID, true); err != nil {
			log.Err(err).Msg("error setting notified, message will be repeated")
		}
	}
}

func (n *Notifier) resetNotified(ctx context.Context) {
	if err := n.s.SetOldBirthdays(ctx, n.daysBeforeBirthday); err != nil {
		log.Err(err).Msg("error resetting notified")
	}
}
