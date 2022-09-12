package models

import (
	"fmt"
)

type UserPresent struct {
	ID             int64  `db:"id"`
	UserID         int64  `db:"user_id"`
	SentAt         int64  `db:"sent_at"`
	ItemType       int    `db:"item_type"`
	ItemID         int    `db:"item_id"`
	Amount         int    `db:"amount"`
	PresentMessage string `db:"present_message"`
	CreatedAt      int64  `db:"created_at"`
	UpdatedAt      int64  `db:"updated_at"`
	DeletedAt      *int64 `db:"deleted_at"`
}

func NewUserPresent(i int64, user User, sentAt int64, itemType int, itemID int, amount int, presentMessage string) UserPresent {
	return UserPresent{i, user.ID, sentAt, itemType, itemID, amount, presentMessage, sentAt + 10*60, sentAt + 10*60, nil}
}

func (u UserPresent) Create() error {
	if _, err := Db.Exec("INSERT INTO user_presents VALUES (?,?,?,?,?,?,?,?,?,?)",
		u.ID,
		u.UserID,
		u.SentAt,
		u.ItemType,
		u.ItemID,
		u.Amount,
		u.PresentMessage,
		u.CreatedAt,
		u.UpdatedAt,
		nil); err != nil {
		fmt.Println(err)
		return fmt.Errorf("insert user_presents: %w", err)
	}
	return nil
}

func (u UserPresent) ReceivedCreate() error {
	if _, err := Db.Exec("INSERT INTO user_presents VALUES (?,?,?,?,?,?,?,?,?,?)",
		u.ID,
		u.UserID,
		u.SentAt,
		u.ItemType,
		u.ItemID,
		u.Amount,
		u.PresentMessage,
		u.CreatedAt,
		u.UpdatedAt,
		u.UpdatedAt); err != nil {
		fmt.Println(err)
		return fmt.Errorf("insert user_presents: %w", err)
	}
	return nil
}

// bulk insert
// https://ipeblog.com/go-sqlx/100/

func PresentReceivedBulkCreate(userPresents *[]UserPresent) error {
	_, err := Db.NamedExec("INSERT INTO user_presents "+
		" (`id`, `user_id`, `sent_at`, `item_type`, `item_id`, `amount`, `present_message`, `created_at`, `updated_at`, `deleted_at`) "+
		" VALUES "+
		" (:id, :user_id, :sent_at, :item_type, :item_id, :amount, :present_message, :created_at, :updated_at, :updated_at)", *userPresents)
	if err != nil {
		fmt.Println(err)
		return fmt.Errorf("insert user_presents: %w", err)
	}
	return nil
}

func PresentBulkCreate(userPresents *[]UserPresent) error {
	_, err := Db.NamedExec("INSERT INTO user_presents "+
		" (`id`, `user_id`, `sent_at`, `item_type`, `item_id`, `amount`, `present_message`, `created_at`, `updated_at`) "+
		" VALUES "+
		" (:id, :user_id, :sent_at, :item_type, :item_id, :amount, :present_message, :created_at, :updated_at)", *userPresents)
	if err != nil {
		fmt.Println(err)
		return fmt.Errorf("insert user_presents: %w", err)
	}
	return nil
}
