package storage

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/smakimka/balb/internal/model"
)

type PGStorage struct {
	p *pgxpool.Pool
}

func NewPGStorage(p *pgxpool.Pool) *PGStorage {
	return &PGStorage{p: p}
}

func (s *PGStorage) Ping(ctx context.Context) error {
	return s.p.Ping(ctx)
}

func (s *PGStorage) Init(ctx context.Context) error {
	tx, err := s.p.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	_, err = tx.Exec(ctx, `create table if not exists users (
        id serial primary key,
        front int,
        uid text,
        fio text,
        birthday timestamp,
        wishlist text,
        notified bool default false,
        constraint c_username_uq unique (front, uid)
    )`)
	if err != nil {
		return err
	}

	_, err = tx.Exec(ctx, `create table if not exists subscriptions (
        id serial primary key,
        subscriber_id int references users(id),
        user_id int references users(id),
        constraint c_sub_uq unique (subscriber_id, user_id)
    )`)
	if err != nil {
		return err
	}

	if err = tx.Commit(ctx); err != nil {
		return err
	}

	return nil
}

func (s *PGStorage) GetUser(ctx context.Context, front int, uid string) (*model.User, error) {
	user := &model.User{Front: front, UID: uid}

	row := s.p.QueryRow(ctx, `select id, fio, birthday, wishlist from users 
    where front = $1 and uid like $2`, front, uid)

	if err := row.Scan(&user.ID, &user.FIO, &user.Birthday, &user.Wishlist); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return user, ErrUserNotFound
		}
		return user, err
	}

	return user, nil
}

func (s *PGStorage) GetUsers(ctx context.Context, front int) ([]model.User, error) {
	users := []model.User{}

	rows, err := s.p.Query(ctx, `select id, fio, birthday, wishlist, uid from users where front = $1`, front)
	if err != nil {
		return users, err
	}
	defer rows.Close()

	for rows.Next() {
		user := model.User{}

		if err = rows.Scan(&user.ID, &user.FIO, &user.Birthday, &user.Wishlist, &user.UID); err != nil {
			return users, err
		}

		users = append(users, user)
	}

	if err = rows.Err(); err != nil {
		return users, err
	}

	return users, err
}

func (s *PGStorage) txGetUser(ctx context.Context, tx pgx.Tx, front int, uid string) (*model.User, error) {
	user := &model.User{Front: front, UID: uid}

	row := tx.QueryRow(ctx, `select id, fio, birthday, wishlist from users 
    where front = $1 and uid like $2`, front, uid)

	if err := row.Scan(&user.ID, &user.FIO, &user.Birthday, &user.Wishlist); err != nil {
		return user, err
	}

	return user, nil
}

func (s *PGStorage) UpdateUser(ctx context.Context, u *model.User) error {
	tx, err := s.p.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	cmd, err := tx.Exec(ctx, `update users set 
    fio = $1, birthday = $2, wishlist = $3 where
    front = $4 and uid like $5)`, u.FIO, u.Birthday, u.Wishlist, u.Front, u.UID)
	if err != nil {
		return err
	}
	if cmd.RowsAffected() == 0 {
		return ErrUserNotFound
	}

	if err = tx.Commit(ctx); err != nil {
		return err
	}

	return nil
}

func (s *PGStorage) CreateUser(ctx context.Context, u *model.User) (int, error) {
	var newUserID int

	tx, err := s.p.Begin(ctx)
	if err != nil {
		return newUserID, err
	}
	defer tx.Rollback(ctx)

	row := tx.QueryRow(ctx, `insert into users as u (front, uid, fio, birthday, wishlist) 
    values ($1, $2, $3, $4, $5) returning u.id`, u.Front, u.UID, u.FIO, u.Birthday, u.Wishlist)

	if err = row.Scan(&newUserID); err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) {
			// 23505 - нарушение constraint-a
			if pgErr.Code == "23505" {
				return newUserID, ErrUserAlreadyExists
			}
		}
		return newUserID, err
	}

	if err = tx.Commit(ctx); err != nil {
		return newUserID, err
	}

	return newUserID, nil
}

func (s *PGStorage) Subscribe(ctx context.Context, data *model.SubscriptionData) error {
	tx, err := s.p.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	subcriber, err := s.txGetUser(ctx, tx, data.Front, data.SubscriberUID)
	if err != nil {
		return err
	}

	user, err := s.txGetUser(ctx, tx, data.Front, data.UserUID)
	if err != nil {
		return err
	}

	_, err = tx.Exec(ctx, `insert into subscriptions (subscriber_id, user_id)
    values ($1, $2)`, subcriber.ID, user.ID)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) {
			// 23505 - нарушение constraint-a
			if pgErr.Code == "23505" {
				return ErrSubscriptionAlreadyExists
			}
		}
		return err
	}

	if err = tx.Commit(ctx); err != nil {
		return err
	}

	return nil
}

func (s *PGStorage) Unsubscribe(ctx context.Context, data *model.SubscriptionData) error {
	tx, err := s.p.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	cmd, err := tx.Exec(ctx, `delete from subscriptions as s 
    using users as sub, users as u
    where s.user_id = u.id
    and s.subscriber_id = sub.id
    and sub.front = $1 
    and u.front = $1
    and sub.uid like $2 
    and u.uid like $3`, data.Front, data.SubscriberUID, data.UserUID)
	if err != nil {
		return err
	}
	if cmd.RowsAffected() == 0 {
		return ErrSubscriptionNotFound
	}

	if err = tx.Commit(ctx); err != nil {
		return err
	}

	return nil
}

func (s *PGStorage) GetBirthdays(ctx context.Context, daysLimit int) ([]model.NotifyRequest, error) {
	res := []model.NotifyRequest{}

	rows, err := s.p.Query(ctx, fmt.Sprintf(`SELECT u.id, u.front, u.fio, u.birthday, u.wishlist, array_agg(sub.uid) as subscriber_uids
    FROM users as u
    join subscriptions as s on u.id = s.user_id
    join users as sub on s.subscriber_id = sub.id
    where u.notified = false 
    and ((CURRENT_DATE + INTERVAL '%d days') - (DATE_TRUNC('year', CURRENT_DATE) + (u.birthday - DATE_TRUNC('year', u.birthday)))) < interval '%d days'
    and ((CURRENT_DATE + INTERVAL '%d days') - (DATE_TRUNC('year', CURRENT_DATE) + (u.birthday - DATE_TRUNC('year', u.birthday)))) > interval '0 days'
    group by u.id`, daysLimit, daysLimit, daysLimit))
	defer rows.Close()

	for rows.Next() {
		req := model.NotifyRequest{}

		err = rows.Scan(&req.ID, &req.Front, &req.FIO, &req.Birthday, &req.Wishlist, &req.Users)
		if err != nil {
			return nil, err
		}

		res = append(res, req)
	}

	if err = rows.Err(); err != nil {
		return res, err
	}

	return res, nil
}

func (s *PGStorage) SetOldBirthdays(ctx context.Context, daysLimit int) error {
	tx, err := s.p.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	_, err = tx.Exec(ctx, fmt.Sprintf(`update users set notified = false 
    where (DATE_TRUNC('year', CURRENT_DATE) + (birthday - DATE_TRUNC('year', birthday))) - CURRENT_DATE > interval '%d days'`,
		daysLimit))
	if err != nil {
		return err
	}

	if err = tx.Commit(ctx); err != nil {
		return err
	}

	return nil
}

func (s *PGStorage) SetNotified(ctx context.Context, userID int, notified bool) error {
	tx, err := s.p.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	cmd, err := tx.Exec(ctx, `update users set notified = $1 where id = $2`, notified, userID)
	if err != nil {
		return err
	}
	if cmd.RowsAffected() == 0 {
		return ErrUserNotFound
	}

	if err = tx.Commit(ctx); err != nil {
		return err
	}

	return nil
}
