package notifier

import (
	"context"
	"fmt"
	"strconv"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
	"github.com/smakimka/balb/internal/bot/storage"
)

type Notifier struct {
	s           storage.Storage
	a           *tgbotapi.BotAPI
	adminChatID int
}

func New(s storage.Storage, a *tgbotapi.BotAPI, adminChatID int) *Notifier {
	return &Notifier{s: s, a: a, adminChatID: adminChatID}
}

// Значения тикеров лучше брать из конфига, но норм
func (n *Notifier) Run(ctx context.Context) {
	log.Info().Msg("started notifier goroutine")
	askTiker := time.NewTicker(10 * time.Second)
	inviteTicker := time.NewTicker(10 * time.Second)

	for {
		select {
		case <-ctx.Done():
			return
		case <-askTiker.C:
			go n.askForChats(ctx)
		case <-inviteTicker.C:
			go n.inviteGuests(ctx)
		}
	}
}

func (n *Notifier) inviteGuests(ctx context.Context) {
	invites, err := n.s.GetNotSentInvites(ctx)
	if err != nil {
		log.Err(err).Msg("error getting unfinished invites")
		return
	}

	for _, invite := range invites {
		start := time.Now()

		chatID, err := strconv.Atoi(invite.ChatID)
		if err != nil {
			log.Err(err).Msg("error convering chat id, should be impossible")
			continue
		}

		msg := tgbotapi.NewMessage(
			int64(chatID),
			fmt.Sprintf("Скоро (%s) у %s день рождения, вы подписаны, поэтому заходите %s", invite.Date.Format("02.01"), invite.FIO, invite.Link),
		)
		_, err = n.a.Send(msg)
		if err != nil {
			log.Err(err).Msg("error sending invite")
			continue
		}

		elapsed := time.Since(start)
		if elapsed < time.Second {
			time.Sleep(time.Second - elapsed)
		}

		if err = n.s.UpdateInviteStatus(ctx, invite.ID, storage.InviteDone); err != nil {
			log.Err(err).Msg("error remembering sent invites, bad")
		}
	}
}

func (n *Notifier) askForChats(ctx context.Context) {
	birthdays, err := n.s.GetNewBirthdays(ctx)
	if err != nil {
		log.Err(err).Msg("error getting birthdays")
		return
	}

	for _, birthday := range birthdays {
		uuid, err := uuid.NewRandom()
		if err != nil {
			log.Err(err).Msg("error generating uuid")
			continue
		}
		code := uuid.String()

		if err = n.s.SetCode(ctx, birthday.ID, code); err != nil {
			log.Err(err).Msg("error setting code")
			continue
		}

		msg := tgbotapi.NewMessage(
			int64(n.adminChatID),
			fmt.Sprintf("Скоро (%s) день рождения у %s, пожадуйста создайте чат, дайте мне там админа и введите в нём команду '/birthday %s'",
				birthday.Date.Format("02.01"), birthday.FIO, code),
		)

		_, err = n.a.Send(msg)
		if err == nil {
			return
		} else {
			log.Err(err).Msg("error sending create chat request, this is bad")
		}

		if err = n.s.SetCode(ctx, birthday.ID, ""); err != nil {
			log.Err(err).Msg("critical, cerror deleting code after msg to admin couldn't be sent")
		}
	}
}
