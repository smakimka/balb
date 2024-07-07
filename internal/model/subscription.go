package model

import "net/http"

type SubscriptionData struct {
	Front         int    `json:"front"`
	SubscriberUID string `json:"subscriber_uid"`
	UserUID       string `json:"user_uid"`
}

func (d *SubscriptionData) Bind(r *http.Request) error {
	if d.SubscriberUID == "" || d.UserUID == "" {
		return ErrMissingFields
	}

	if d.Front < TelegramFront || d.Front > TelegramFront {
		return ErrWrongFront
	}

	return nil
}
