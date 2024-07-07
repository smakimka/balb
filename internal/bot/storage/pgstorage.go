package storage

import (
	"context"

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

	_, err = tx.Exec(ctx, `create table if not exists birthdays (
        id serial primary key,
        chat_id text,
        code text default '',
        invite_link text,
        fio text,
        birthday timestamp,
        wishlist text,
        chat_ids text[],
        done_ids text[]
    )`)
	if err != nil {
		return err
	}

	if err = tx.Commit(ctx); err != nil {
		return err
	}

	return nil
}

func (s *PGStorage) CreateBirthday(ctx context.Context, r *model.NotifyRequest) error {
	tx, err := s.p.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	_, err = tx.Exec(ctx, `insert into birthdays (wishlist, fio, birthday, chat_ids, done_ids) 
    values ($1, $2, $3, $4, $5)`, r.Wishlist, r.FIO, r.Birthday, r.Users, []string{})

	if err = tx.Commit(ctx); err != nil {
		return err
	}

	return nil
}

func (s *PGStorage) GetNewBirthdays(ctx context.Context) ([]BirthdayData, error) {
	res := []BirthdayData{}

	rows, err := s.p.Query(ctx, `select id, fio, birthday, wishlist, chat_ids from birthdays 
    where code like ''`)
	if err != nil {
		return res, err
	}
	defer rows.Close()

	for rows.Next() {
		data := BirthdayData{}

		if err = rows.Scan(&data.ID, &data.FIO, &data.Date, &data.Wishlist, &data.GuestIDs); err != nil {
			return res, err
		}

		res = append(res, data)
	}

	if err = rows.Err(); err != nil {
		return res, err
	}

	return res, nil
}

func (s *PGStorage) GetUnFinishedBirthdays(ctx context.Context) ([]BirthdayData, error) {
	res := []BirthdayData{}

	rows, err := s.p.Query(ctx, `select id, fio, birthday, chat_id, invite_link, wishlist, chat_ids, done_ids from birthdays
    where chat_id is not null and (array_length(done_ids, 1) is null or array_length(done_ids, 1) < array_length(chat_ids, 1))`)
	if err != nil {
		return res, err
	}

	for rows.Next() {
		birthday := BirthdayData{}
		if err = rows.Scan(&birthday.ID, &birthday.FIO, &birthday.Date, &birthday.ChatID, &birthday.InviteLink, &birthday.Wishlist, &birthday.GuestIDs, &birthday.DoneIDs); err != nil {
			return res, err
		}

		res = append(res, birthday)
	}
	if err = rows.Err(); err != nil {
		return res, err
	}

	return res, nil
}

func (s *PGStorage) SetCode(ctx context.Context, birthdayID int, code string) error {
	tx, err := s.p.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	cmd, err := tx.Exec(ctx, `update birthdays set code = $1 where id = $2`, code, birthdayID)
	if err != nil {
		return err
	}
	if cmd.RowsAffected() == 0 {
		return ErrBirthdayNotFound
	}

	if err = tx.Commit(ctx); err != nil {
		return err
	}

	return nil
}

func (s *PGStorage) UpdateLinkAndChatIDByCode(ctx context.Context, code string, chatID string, link string) error {
	tx, err := s.p.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	cmd, err := tx.Exec(ctx, `update birthdays set chat_id = $1, invite_link = $2 where code = $3`, chatID, link, code)
	if err != nil {
		return err
	}
	if cmd.RowsAffected() == 0 {
		return ErrBirthdayNotFound
	}

	if err = tx.Commit(ctx); err != nil {
		return err
	}

	return nil
}

func (s *PGStorage) UpdateDoneIDS(ctx context.Context, birthdayID int, doneIDS []string) error {
	tx, err := s.p.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	cmd, err := tx.Exec(ctx, `update birthdays set done_ids = $1 where id = $2`, doneIDS, birthdayID)
	if err != nil {
		return err
	}
	if cmd.RowsAffected() == 0 {
		return ErrBirthdayNotFound
	}

	if err = tx.Commit(ctx); err != nil {
		return err
	}

	return nil
}
