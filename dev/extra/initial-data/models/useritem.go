package models

import (
	"fmt"
)

type UserItem struct {
	ID        int64 `db:"id"`
	UserID    int64 `db:"user_id"`
	ItemType  int   `db:"item_type"`
	ItemID    int   `db:"item_id"`
	Amount    int   `db:"amount"`
	CreatedAt int64 `db:"created_at"`
	UpdatedAt int64 `db:"updated_at"`
}

func NewUserItem(i int64, user User, itemType int, itemID int, amount int, createdAt int64) UserItem {
	return UserItem{i, user.ID, itemType, itemID, amount, createdAt, createdAt}
}

func (u UserItem) Create() error {
	if _, err := Db.Exec("INSERT INTO user_items VALUES (?,?,?,?,?,?,?,?)",
		u.ID,
		u.UserID,
		u.ItemType,
		u.ItemID,
		u.Amount,
		u.CreatedAt,
		u.UpdatedAt,
		nil); err != nil {
		fmt.Println(err)
		return fmt.Errorf("insert user_items: %w", err)
	}
	return nil
}

func UserItemBulkCreate(userItems *[]UserItem) error {
	_, err := Db.NamedExec("INSERT INTO user_items "+
		" (`id`, `user_id`, `item_type`, `item_id`, `amount`, `created_at`, `updated_at`) "+
		" VALUES "+
		" (:id, :user_id, :item_type, :item_id, :amount, :created_at, :updated_at)", *userItems)
	if err != nil {
		fmt.Println(err)
		return fmt.Errorf("insert user_items: %w", err)
	}
	return nil
}
