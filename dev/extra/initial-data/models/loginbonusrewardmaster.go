package models

import (
	"encoding/json"
	"fmt"
	"os"
)

type LoginBonusRewardMasters []LoginBonusRewardMaster

type LoginBonusRewardMaster struct {
	ID             int64 `json:"id" db:"id"`
	LoginBonusID   int64 `json:"loginBonusId" db:"login_bonus_id"`
	RewardSequence int   `json:"rewardSequence" db:"reward_sequence"`
	ItemType       int   `json:"itemType" db:"item_type"`
	ItemID         int64 `json:"itemId" db:"item_id"`
	Amount         int64 `json:"amount" db:"amount"`
}

func GetLoginBonusRewardMaster() LoginBonusRewardMasters {
	var err error
	var loginBonusRewardMasters []LoginBonusRewardMaster
	err = Db.Select(&loginBonusRewardMasters, "SELECT `id`, `login_bonus_id` , `reward_sequence`, `item_type` , `item_id`, `amount` FROM `login_bonus_reward_masters` ORDER BY id")
	if err != nil {
		fmt.Println("err:", err)
		_ = fmt.Errorf("select login_bonus_reward_masters: %w", err)
	}
	return loginBonusRewardMasters
}

func (lbrm LoginBonusRewardMasters) Commit(outdir string) error {
	localOutdir = outdir
	data, err := json.Marshal(lbrm)
	if err != nil {
		fmt.Println(err)
		return err
	}
	err = os.WriteFile(localOutdir+"/"+"master.json", data, 0666)
	if err != nil {
		fmt.Println(err)
		return err
	}
	return nil
}

func (lbrm LoginBonusRewardMasters) Rename(fileName string) error {
	err := os.Rename(localOutdir+"/"+"master.json", localOutdir+"/"+fileName)
	if err != nil {
		return err
	}
	return nil
}
