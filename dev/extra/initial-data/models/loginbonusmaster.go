package models

import (
	"fmt"
)

type LoginBonusMaster struct {
	ID             int64 `db:"id"`
	LoginBonusType int   `db:"login_bonus_type"`
	StartAt        int64 `db:"start_at"`
	EndAt          int64 `db:"end_at"`
	ColumnCount    int   `db:"column_count"`
	Looped         int   `db:"looped"`
	CreatedAt      int64 `db:"created_at"`
}

func GetLoginBonusMasters() []LoginBonusMaster {
	loginBonusMasters := []LoginBonusMaster{}
	err := Db.Select(&loginBonusMasters, "SELECT * FROM `login_bonus_masters` ORDER BY id")
	if err != nil {
		_ = fmt.Errorf("select login_bonus_masters: %w", err)
		return nil
	}

	return loginBonusMasters
}
