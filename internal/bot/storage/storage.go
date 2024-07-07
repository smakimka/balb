package storage

import (
	"context"
	"errors"
	"time"

	"github.com/smakimka/balb/internal/model"
)

var ErrBirthdayNotFound = errors.New("birthday not found")
var ErrInviteNotFound = errors.New("invite not found")

const (
	InviteNotSent   = iota
	InviteDone      = iota
	InviteCancelled = iota
)

type BirthdayData struct {
	ID         int
	Date       time.Time
	FIO        string
	Wishlist   string
	ChatID     string
	Code       string
	InviteLink string
}

type InviteData struct {
	ID     int
	Date   time.Time
	FIO    string
	ChatID string
	Link   string
}

type Storage interface {
	UpdateInviteStatus(ctx context.Context, inviteID int, status int) error
	UpdateLinkAndChatIDByCode(ctx context.Context, code string, chatID string, link string) error
	GetNewBirthdays(ctx context.Context) ([]BirthdayData, error)
	GetBirthdayByCode(ctx context.Context, code string) (BirthdayData, error)
	GetNotSentInvites(ctx context.Context) ([]InviteData, error)
	CreateBirthday(ctx context.Context, r *model.NotifyRequest) error
	SetCode(ctx context.Context, birthdayID int, code string) error
}
