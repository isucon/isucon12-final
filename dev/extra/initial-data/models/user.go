package models

import (
	"fmt"
)

type User struct {
	ID              int64
	IsuCoin         int64
	LastGetRewardAt int64
	LastActivatedAt int64
	RegisteredAt    int64
	CreatedAt       int64
	UpdatedAt       int64
}

func NewUser(i int64, isuCoin int64, registeredAt int64, activeAt int64) User {
	return User{i, isuCoin, activeAt, activeAt, registeredAt, registeredAt, activeAt}
}

func (u User) Create() error {
	if _, err := Db.Exec("INSERT INTO users VALUES (?,?,?,?,?,?,?,?)",
		u.ID,
		u.IsuCoin,
		u.LastGetRewardAt,
		u.LastActivatedAt,
		u.RegisteredAt,
		u.CreatedAt,
		u.UpdatedAt,
		nil); err != nil {
		fmt.Println(err)
		return fmt.Errorf("insert users: %w", err)
	}
	return nil
}
