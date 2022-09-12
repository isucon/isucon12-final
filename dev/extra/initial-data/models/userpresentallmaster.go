package models

import (
	"fmt"
)

type UserPresentAllMaster struct {
	ID                int    `db:"id"`
	RegisteredStartAt int64  `db:"registered_start_at"`
	RegisteredEndAt   int64  `db:"registered_end_at"`
	ItemType          int    `db:"item_type"`
	ItemID            int    `db:"item_id"`
	Amount            int    `db:"amount"`
	PresentMessage    string `db:"present_message"`
	CreatedAt         int64  `db:"created_at"`
}

var UserPresentAllMasters []UserPresentAllMaster

func InitUserPresentAllMaster() {
	err := Db.Select(&UserPresentAllMasters, "SELECT *  FROM `present_all_masters` ORDER BY id")
	if err != nil {
		fmt.Println("error", err)
		_ = fmt.Errorf("select present_all_masters: %w", err)
	}
}
