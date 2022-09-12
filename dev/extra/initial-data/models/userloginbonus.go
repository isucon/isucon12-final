package models

import (
	"fmt"
)

type UserLoginBonus struct {
	ID                 int64 `json:"id" db:"id"`
	UserID             int64 `json:"userId" db:"user_id"`
	LoginBonusID       int   `json:"loginBonusId" db:"login_bonus_id"`
	LastRewardSequence int   `json:"lastRewardSequence" db:"last_reward_sequence"`
	LoopCount          int   `json:"loopCount" db:"loop_count"`
	CreatedAt          int64 `json:"createdAt" db:"created_at"`
	UpdatedAt          int64 `json:"updatedAt" db:"updated_at"`
}

func NewUserLoginBonus(i int64, user User, bonusID int, loopCount int, rewardSequence int) UserLoginBonus {
	return UserLoginBonus{i, user.ID, bonusID, rewardSequence, loopCount, user.RegisteredAt, user.LastActivatedAt}
}

func (u UserLoginBonus) Create() error {
	if _, err := Db.Exec("INSERT INTO user_login_bonuses VALUES (?,?,?,?,?,?,?,?)",
		u.ID,
		u.UserID,
		u.LoginBonusID,
		u.LastRewardSequence,
		u.LoopCount,
		u.CreatedAt,
		u.UpdatedAt,
		nil); err != nil {
		fmt.Println(err)
		return fmt.Errorf("insert user_login_bonuses: %w", err)
	}
	return nil
}
