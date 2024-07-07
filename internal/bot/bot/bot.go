package bot

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/rs/zerolog/log"
	"github.com/smakimka/balb/internal/bot/dialog"
	"github.com/smakimka/balb/internal/bot/storage"
	"github.com/smakimka/balb/internal/model"
)

type Bot struct {
	a           *tgbotapi.BotAPI
	c           http.Client
	d           *dialog.Dialog
	s           storage.Storage
	adminChatID int
}

func New(a *tgbotapi.BotAPI, startToken string, c http.Client, s storage.Storage, adminChatID int) *Bot {
	d := dialog.New(startToken, c)
	return &Bot{a: a, c: c, d: d, s: s, adminChatID: adminChatID}
}

func (b *Bot) StartPolling(ctx context.Context) {
	u := tgbotapi.NewUpdate(0)
	updates := b.a.GetUpdatesChan(u)

	for update := range updates {
		if update.Message == nil {
			continue
		}

		if update.Message.IsCommand() {
			if update.Message.Text == "/start" {
				go b.handleCommand(ctx, update.Message)
				continue
			}

			if b.d.IsRegistered(update.Message.From.ID) {
				go b.handleCommand(ctx, update.Message)
				continue
			}
		}

		go func() {
			msg := b.d.HandleMessage(ctx, update.Message.From.ID, update.Message.Text)
			if msg != nil {
				b.a.Send(msg)
			}
		}()
	}
}

func (b *Bot) handleCommand(ctx context.Context, message *tgbotapi.Message) {
	switch message.Command() {
	case "start":
		msg := tgbotapi.NewMessage(message.From.ID, "Для использования этого бота необходимо авторизоваться, введите токен")
		b.d.Reset(message.From.ID)
		b.a.Send(msg)
	case "list":
		b.list(ctx, message)
	case "subscribe":
		b.subscribe(ctx, message)
	case "unsubscribe":
		b.unsubscribe(ctx, message)
	case "birthday":
		b.birthday(ctx, message)
	}
}

func (b *Bot) birthday(ctx context.Context, message *tgbotapi.Message) {
	if message.From.ID != int64(b.adminChatID) {
		msg := tgbotapi.NewMessage(message.From.ID, "Эту команду можно использовать только админу")
		b.a.Send(msg)
		return
	}
	if !message.Chat.IsGroup() {
		msg := tgbotapi.NewMessage(message.From.ID, "Эту команду можно использовать только в групповых чатах")
		b.a.Send(msg)
		return
	}

	link, err := b.a.GetInviteLink(tgbotapi.ChatInviteLinkConfig{ChatConfig: tgbotapi.ChatConfig{ChatID: message.Chat.ID}})
	if err != nil {
		msg := tgbotapi.NewMessage(message.From.ID, "Не получилось создать ссылку, я точно админ?, проверьте и попробуйте ещё раз")
		b.a.Send(msg)
		return
	}

	if err = b.s.UpdateLinkAndChatIDByCode(ctx, message.CommandArguments(), fmt.Sprint(message.Chat.ID), link); err != nil {
		log.Err(err).Msg("error updating chat link")
		msg := tgbotapi.NewMessage(message.From.ID, "ошибка, попробуйте позже")
		b.a.Send(msg)
		return
	}

	msg := tgbotapi.NewMessage(
		message.From.ID,
		fmt.Sprintf("Ссылка в чате по коду %s успешно создана, рассылка приглашений скоро начнется", message.CommandArguments()),
	)
	b.a.Send(msg)

	birthday, err := b.s.GetBirthdayByCode(ctx, message.CommandArguments())
	if err != nil {
		log.Err(err).Msg("error getting data on group message")
		return
	}

	msg = tgbotapi.NewMessage(
		message.Chat.ID,
		fmt.Sprintf("Это беседа дня рождения %s (%s) wishlist:\n%s", birthday.FIO, birthday.Date.Format("02.01"), birthday.Wishlist),
	)
	b.a.Send(msg)
}

func (b *Bot) subscribe(_ context.Context, message *tgbotapi.Message) {
	data := model.SubscriptionData{
		Front:         model.TelegramFront,
		SubscriberUID: fmt.Sprint(message.From.ID),
		UserUID:       message.CommandArguments(),
	}
	body, err := json.Marshal(data)
	if err != nil {
		log.Err(err).Msg("error marshailling data")
		return
	}

	resp, err := b.c.Post(fmt.Sprintf("http://server:8090/subscriptions/subscribe"), "application/json", bytes.NewReader(body))
	if err != nil {
		log.Err(err).Msg("error sending subscribe request")
		msg := tgbotapi.NewMessage(message.From.ID, "Ошибка, попробуйте позже")
		b.a.Send(msg)
		return
	}
	defer resp.Body.Close()

	body, err = io.ReadAll(resp.Body)
	if err != nil {
		log.Err(err).Msg("error reading body")
		msg := tgbotapi.NewMessage(message.From.ID, "Ошибка, попробуйте позже")
		b.a.Send(msg)
		return
	}

	if resp.StatusCode != 200 {
		var response model.Response
		if err = json.Unmarshal(body, &response); err != nil {
			log.Err(err).Msg("error unmarshalling response")
			msg := tgbotapi.NewMessage(message.From.ID, "Ошибка, попробуйте позже")
			b.a.Send(msg)
			return
		}

		if response.Msg == "subscription already exists" {
			msg := tgbotapi.NewMessage(message.From.ID, "Вы уже подписаны")
			b.a.Send(msg)
			return
		}

		msg := tgbotapi.NewMessage(message.From.ID, "Ошибка, попробуйте позже")
		b.a.Send(msg)
		return
	}

	msg := tgbotapi.NewMessage(message.From.ID, "Подписка оформлена")
	b.a.Send(msg)
}

func (b *Bot) unsubscribe(_ context.Context, message *tgbotapi.Message) {
	data := model.SubscriptionData{
		Front:         model.TelegramFront,
		SubscriberUID: fmt.Sprint(message.From.ID),
		UserUID:       message.CommandArguments(),
	}
	body, err := json.Marshal(data)
	if err != nil {
		log.Err(err).Msg("error marshailling data")
		return
	}

	resp, err := b.c.Post(fmt.Sprintf("http://server:8090/subscriptions/unsubscribe"), "application/json", bytes.NewReader(body))
	if err != nil {
		log.Err(err).Msg("error sending subscribe request")
		msg := tgbotapi.NewMessage(message.From.ID, "Ошибка, попробуйте позже")
		b.a.Send(msg)
		return
	}
	defer resp.Body.Close()

	body, err = io.ReadAll(resp.Body)
	if err != nil {
		log.Err(err).Msg("error reading body")
		msg := tgbotapi.NewMessage(message.From.ID, "Ошибка, попробуйте позже")
		b.a.Send(msg)
		return
	}

	if resp.StatusCode != 200 {
		var response model.Response
		if err = json.Unmarshal(body, &response); err != nil {
			log.Err(err).Msg("error unmarshalling response")
			msg := tgbotapi.NewMessage(message.From.ID, "Ошибка, попробуйте позже")
			b.a.Send(msg)
			return
		}

		if response.Msg == "subscription not found" {
			msg := tgbotapi.NewMessage(message.From.ID, "Вы не были подписаны")
			b.a.Send(msg)
			return
		}

		msg := tgbotapi.NewMessage(message.From.ID, "Ошибка, попробуйте позже")
		b.a.Send(msg)
		return
	}

	msg := tgbotapi.NewMessage(message.From.ID, "Подписка отменена")
	b.a.Send(msg)
}

func (b *Bot) list(_ context.Context, message *tgbotapi.Message) {
	resp, err := b.c.Get(fmt.Sprintf("http://server:8090/users/get/%d", model.TelegramFront))
	if err != nil {
		log.Err(err).Msg("error getting users")

		msg := tgbotapi.NewMessage(message.From.ID, "Ошибка, попробуйте позже")
		b.a.Send(msg)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		log.Error().Int("code", resp.StatusCode).Msg("error getting users")

		msg := tgbotapi.NewMessage(message.From.ID, "Ошибка, попробуйте позже")
		b.a.Send(msg)
		return
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Err(err).Msg("error unmarshalling users")

		msg := tgbotapi.NewMessage(message.From.ID, "Ошибка, попробуйте позже")
		b.a.Send(msg)
	}

	var users []model.User
	err = json.Unmarshal(body, &users)
	if err != nil {
		log.Err(err).Msg("error unmarshalling users")

		msg := tgbotapi.NewMessage(message.From.ID, "Ошибка, попробуйте позже")
		b.a.Send(msg)
	}

	msgText := []string{"Пользователи:"}
	for i, user := range users {
		msgText = append(msgText, fmt.Sprintf("%d. %s - %s", i+1, user.FIO, user.UID))
	}

	msg := tgbotapi.NewMessage(message.From.ID, strings.Join(msgText, "\n"))
	b.a.Send(msg)
}
