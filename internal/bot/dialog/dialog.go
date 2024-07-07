package dialog

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/smakimka/balb/internal/model"
	"golang.org/x/net/context"
)

const (
	token    = iota
	fio      = iota
	birthday = iota
	wishlist = iota
	finished = iota
)

type UserData struct {
	status   int
	FIO      string
	Birthday time.Time
	Wishlist string
}

type Dialog struct {
	m         sync.RWMutex
	c         http.Client
	users     map[int64]UserData
	authToken string
}

func New(authToken string, c http.Client) *Dialog {
	return &Dialog{m: sync.RWMutex{}, c: c, users: map[int64]UserData{}, authToken: authToken}
}

func (d *Dialog) Reset(chatID int64) {
	d.m.Lock()
	defer d.m.Unlock()

	user, ok := d.users[chatID]
	if !ok {
		return
	}

	user.status = token
	d.users[chatID] = user
}

func (d *Dialog) HandleMessage(ctx context.Context, chatID int64, text string) *tgbotapi.MessageConfig {
	d.m.RLock()
	userData, ok := d.users[chatID]
	d.m.RUnlock()

	if !ok {
		user, err := d.getUser(chatID)
		if err != nil {
			msg := tgbotapi.NewMessage(chatID, "произошла ошибка, попробуйте позже")
			return &msg
		}

		d.m.Lock()
		if user != nil {
			userData = UserData{
				status:   finished,
				FIO:      user.FIO,
				Birthday: user.Birthday,
				Wishlist: user.Wishlist,
			}
			d.users[chatID] = userData
		} else {
			userData = UserData{
				status: token,
			}
			d.users[chatID] = userData
		}
		d.m.Unlock()
	}

	var msg tgbotapi.MessageConfig
	switch userData.status {
	case token:
		if text == d.authToken {
			userData.status = fio
			d.updateUserData(chatID, userData)

			msg = tgbotapi.NewMessage(chatID, "Введите ваше ФИО")
		} else {
			msg = tgbotapi.NewMessage(chatID, "Неправильно, ещё раз")
		}
	case fio:
		userData.status = birthday
		userData.FIO = text
		d.updateUserData(chatID, userData)
		msg = tgbotapi.NewMessage(chatID, "Введите вашу дату рождения в формате dd.mm.yyyy")
	case birthday:
		date, err := time.Parse("02.01.2006", text)
		if err != nil {
			msg = tgbotapi.NewMessage(chatID, "неверный формат даты")
		} else {
			userData.status = wishlist
			userData.Birthday = date
			d.updateUserData(chatID, userData)

			msg = tgbotapi.NewMessage(chatID, "Введите вишлист, пожайлуйста")
		}
	case wishlist:
		userData.status = finished
		userData.Wishlist = text
		d.updateUserData(chatID, userData)

		err := d.addUser(chatID, userData)
		if err != nil {
			msg = tgbotapi.NewMessage(chatID, "Что-то пошло не так, попробуйте начать сначала /start")
		} else {
			msg = tgbotapi.NewMessage(chatID, "Спасибо за регистрацию, ждите подарков ;)")
		}
	case finished:
		return nil
	default:
		msg = tgbotapi.NewMessage(chatID, "ошибка, не знаю что делать")
	}

	return &msg
}

func (d *Dialog) IsRegistered(chatID int64) bool {
	d.m.RLock()
	defer d.m.RUnlock()

	user, ok := d.users[chatID]
	if !ok {
		return false
	}

	if user.status != finished {
		return false
	}

	return true
}

func (d *Dialog) updateUserData(chatID int64, newData UserData) {
	d.m.Lock()
	defer d.m.Unlock()

	d.users[chatID] = newData
}

func (d *Dialog) getUser(chatID int64) (*model.User, error) {
	resp, err := d.c.Get(fmt.Sprintf("http://server:8090/users/get/%d/%d", model.TelegramFront, chatID))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, err
		}
		var user model.User
		if err = json.Unmarshal(body, &user); err != nil {
			return nil, err
		}

		return &user, nil
	}

	return nil, nil
}

func (d *Dialog) addUser(chatID int64, user UserData) error {
	data, err := json.Marshal(model.User{
		Front:    model.TelegramFront,
		UID:      fmt.Sprint(chatID),
		FIO:      user.FIO,
		Birthday: user.Birthday,
		Wishlist: user.Wishlist,
	})
	if err != nil {
		return err
	}

	resp, err := d.c.Post(fmt.Sprintf("http://server:8090/users/add"), "application/json", bytes.NewReader(data))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return err
		}

		var resp model.Response
		err = json.Unmarshal(body, &resp)
		if err != nil {
			return err
		}

		if resp.Msg == "user already exists" {
			return nil
		}

		return errors.New("error")
	}

	return nil
}
