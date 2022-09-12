package models

import (
	"fmt"
)

type UserCard struct {
	ID           int64 `db:"id"`
	UserID       int64 `db:"user_id"`
	CardID       int   `db:"card_id"`
	AmountPerSec int   `db:"amount_per_sec"`
	Level        int   `db:"level"`
	TotalExp     int64 `db:"total_exp"`
	CreatedAt    int64 `db:"created_at"`
	UpdatedAt    int64 `db:"updated_at"`
}

func NewUserCard(i int64, user User, cardID int, amountPerSec int, level int, totalExp int64) UserCard {
	return UserCard{i, user.ID, cardID, amountPerSec, level, totalExp, user.RegisteredAt, user.RegisteredAt}
}

func (u UserCard) Create() error {
	if _, err := Db.Exec("INSERT INTO user_cards VALUES (?,?,?,?,?,?,?,?,?)",
		u.ID,
		u.UserID,
		u.CardID,
		u.AmountPerSec,
		u.Level,
		u.TotalExp,
		u.CreatedAt,
		u.UpdatedAt,
		nil); err != nil {
		fmt.Println(err)
		return fmt.Errorf("insert user_cards: %w", err)
	}
	return nil
}

func UserCardBulkCreate(userCards *[]UserCard) error {
	_, err := Db.NamedExec("INSERT INTO user_cards "+
		" (`id`, `user_id`, `card_id`, `amount_per_sec`, `level`, `total_exp`, `created_at`, `updated_at`) "+
		" VALUES "+
		" (:id, :user_id, :card_id, :amount_per_sec, :level, :total_exp, :created_at, :updated_at)", *userCards)
	if err != nil {
		fmt.Println(err)
		return fmt.Errorf("insert user_cards: %w", err)
	}
	return nil
}
