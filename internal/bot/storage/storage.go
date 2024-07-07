package storage

import (
	"context"
	"errors"
	"time"

	"github.com/smakimka/balb/internal/model"
)

var ErrBirthdayNotFound = errors.New("birthday not found")

type BirthdayData struct {
	ID         int
	Date       time.Time
	FIO        string
	Wishlist   string
	ChatID     string
	Code       string
	InviteLink string
	GuestIDs   []string
	DoneIDs    []string
}

type Storage interface {
	UpdateDoneIDS(ctx context.Context, birthdayID int, doneIDS []string) error
	UpdateLinkAndChatIDByCode(ctx context.Context, code string, chatID string, link string) error
	GetNewBirthdays(ctx context.Context) ([]BirthdayData, error)
	GetUnFinishedBirthdays(ctx context.Context) ([]BirthdayData, error)
	CreateBirthday(ctx context.Context, r *model.NotifyRequest) error
	SetCode(ctx context.Context, birthdayID int, code string) error
}
