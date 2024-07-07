package model

import (
	"errors"
	"net/http"
	"time"
)

var ErrMissingFields = errors.New("missing fields")
var ErrWrongFront = errors.New("wong front")

type User struct {
	ID       int
	Front    int       `json:"front"`
	UID      string    `json:"uid"`
	FIO      string    `json:"fio"`
	Birthday time.Time `json:"birthday"`
	Wishlist string    `json:"wishlist"`
}

func (u *User) Bind(r *http.Request) error {
	if u.UID == "" {
		return ErrMissingFields
	}

	if u.Front < TelegramFront || u.Front > TelegramFront {
		return ErrWrongFront
	}

	return nil
}
