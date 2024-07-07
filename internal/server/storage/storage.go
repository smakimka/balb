package storage

import (
	"context"
	"errors"

	"github.com/smakimka/balb/internal/model"
)

var ErrUserAlreadyExists = errors.New("user already exists")
var ErrUserNotFound = errors.New("user not found")
var ErrSubscriptionAlreadyExists = errors.New("subscription already exists")
var ErrSubscriptionNotFound = errors.New("subscription not found")

type Storage interface {
	Init(ctx context.Context) error
	Getter
	Updater
	Creater
	Subscriber
}

type Getter interface {
	GetBirthdays(ctx context.Context, daysLimit int) ([]model.NotifyRequest, error)
	GetUser(ctx context.Context, front int, uid string) (*model.User, error)
	GetUsers(ctx context.Context, front int) ([]model.User, error)
}

type Updater interface {
	SetOldBirthdays(ctx context.Context, daysLimit int) error
	SetNotified(ctx context.Context, userID int, notified bool) error
	UpdateUser(ctx context.Context, u *model.User) error
}

type Creater interface {
	CreateUser(ctx context.Context, u *model.User) (int, error)
}

type Subscriber interface {
	Subscribe(ctx context.Context, data *model.SubscriptionData) error
	Unsubscribe(ctx context.Context, data *model.SubscriptionData) error
}
