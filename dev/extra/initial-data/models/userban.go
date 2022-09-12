package models

import (
	"fmt"
)

type UserBan struct {
	ID        int64 `json:"id" db:"id"`
	UserID    int64 `json:"userId" db:"user_id"`
	CreatedAt int64 `json:"createdAt" db:"created_at"`
	UpdatedAt int64 `json:"updatedAt" db:"updated_at"`
}

func NewUserBan(i int64, userID int64, createdAt int64) UserBan {
	return UserBan{i, userID, createdAt, createdAt}
}

func (u UserBan) Create() error {
	if _, err := Db.Exec("INSERT INTO user_bans VALUES (?,?,?,?,?)",
		u.ID,
		u.UserID,
		u.CreatedAt,
		u.UpdatedAt,
		nil); err != nil {
		fmt.Println(err)
		return fmt.Errorf("insert user_bans: %w", err)
	}
	return nil
}
