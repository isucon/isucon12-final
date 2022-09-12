package models

import (
	"fmt"
)

type UserDevice struct {
	ID           int64  `db:"id"`
	UserID       int64  `db:"user_id"`
	PlatformID   string `db:"platform_id"`
	PlatformType int    `db:"platform_type"`
	CreatedAt    int64  `db:"created_at"`
	UpdatedAt    int64  `db:"updated_at"`
}

func NewUserDevice(i int64, user User, platformID string) UserDevice {
	return UserDevice{i, user.ID, platformID, 1, user.CreatedAt, user.CreatedAt}
}

func NewUserDeviceOther(i int64, user User, platformID string, platformType int) UserDevice {
	return UserDevice{i, user.ID, platformID, platformType, user.CreatedAt, user.CreatedAt}
}

func (u UserDevice) Create() error {
	if _, err := Db.Exec("INSERT INTO user_devices VALUES (?,?,?,?,?,?,?)",
		u.ID,
		u.UserID,
		u.PlatformID,
		u.PlatformType,
		u.CreatedAt,
		u.UpdatedAt,
		nil); err != nil {
		fmt.Println(err)
		return fmt.Errorf("insert user_devices: %w", err)
	}
	return nil
}
