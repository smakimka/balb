package model

import (
	"net/http"
	"time"
)

type NotifyRequest struct {
	ID       int
	Front    int
	Users    []string  `json:"users"`
	FIO      string    `json:"fio"`
	Birthday time.Time `json:"birthday"`
	Wishlist string    `json:"wishlist"`
}

func (n *NotifyRequest) Bind(r *http.Request) error {
	if len(n.Users) == 0 {
		return ErrMissingFields
	}

	return nil
}
