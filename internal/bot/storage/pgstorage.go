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
        wishlist text
    )`)
	if err != nil {
		return err
	}

	_, err = tx.Exec(ctx, `create table if not exists invites (
        id serial primary key,
        birthday_id int references birthdays(id),
        chat_id text,
        status int,
        constraint c_birthdayinvite_uq unique (birthday_id, chat_id)
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

	var birthdayID int
	row := tx.QueryRow(ctx, `insert into birthdays as b (wishlist, fio, birthday) 
    values ($1, $2, $3) returning b.id`, r.Wishlist, r.FIO, r.Birthday)
	if err = row.Scan(&birthdayID); err != nil {
		return err
	}

	for _, user := range r.Users {
		_, err := tx.Exec(ctx, `insert into invites (birthday_id, chat_id, status) 
        values ($1, $2, $3)`, birthdayID, user, InviteNotSent)
		if err != nil {
			return err
		}
	}

	if err = tx.Commit(ctx); err != nil {
		return err
	}

	return nil
}

func (s *PGStorage) GetNewBirthdays(ctx context.Context) ([]BirthdayData, error) {
	res := []BirthdayData{}

	rows, err := s.p.Query(ctx, `select id, fio, birthday, wishlist from birthdays 
    where code like ''`)
	if err != nil {
		return res, err
	}
	defer rows.Close()

	for rows.Next() {
		data := BirthdayData{}

		if err = rows.Scan(&data.ID, &data.FIO, &data.Date, &data.Wishlist); err != nil {
			return res, err
		}

		res = append(res, data)
	}

	if err = rows.Err(); err != nil {
		return res, err
	}

	return res, nil
}

func (s *PGStorage) GetBirthdayByCode(ctx context.Context, code string) (BirthdayData, error) {
	res := BirthdayData{}

	row := s.p.QueryRow(ctx, `select id, fio, birthday, wishlist, chat_id from birthdays 
    where code like $1`, code)

	if err := row.Scan(&res.ID, &res.FIO, &res.Date, &res.Wishlist, &res.ChatID); err != nil {
		return res, err
	}

	return res, nil
}

func (s *PGStorage) GetNotSentInvites(ctx context.Context) ([]InviteData, error) {
	res := []InviteData{}

	rows, err := s.p.Query(ctx, `select i.id, b.fio, b.birthday, i.chat_id, b.invite_link 
    from invites as i
    join birthdays as b on b.id = i.birthday_id
    where i.status = $1 and b.invite_link is not null`, InviteNotSent)
	if err != nil {
		return res, err
	}

	for rows.Next() {
		invite := InviteData{}
		if err = rows.Scan(&invite.ID, &invite.FIO, &invite.Date, &invite.ChatID, &invite.Link); err != nil {
			return res, err
		}

		res = append(res, invite)
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

func (s *PGStorage) UpdateInviteStatus(ctx context.Context, inviteID int, status int) error {
	tx, err := s.p.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	cmd, err := tx.Exec(ctx, `update invites set status = $1 where id = $2`, status, inviteID)
	if err != nil {
		return err
	}
	if cmd.RowsAffected() == 0 {
		return ErrInviteNotFound
	}

	if err = tx.Commit(ctx); err != nil {
		return err
	}

	return nil
}
