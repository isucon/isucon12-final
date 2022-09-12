package models

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
)

type CardMaster struct {
	CardID           int `json:"id" db:"id"`
	BaseExpPerLevel  int `json:"baseExpPerLevel" db:"base_exp_per_level"`
	MaxAmountPerSec  int `json:"maxAmountPerSec" db:"max_amount_per_sec"`
	BaseAmountPerSec int `json:"amountPerSec" db:"amount_per_sec"`
	MaxLevel         int `json:"maxLevel" db:"max_level"`
}

type CardMasters []CardMaster

var cardMasters CardMasters

func InitCardMaster() {
	err := Db.Select(&cardMasters, "SELECT `id`, `base_exp_per_level`, `max_amount_per_sec`, `amount_per_sec`, `max_level`  FROM `item_masters` WHERE item_type = 2 ORDER BY id")
	if err != nil {
		fmt.Println("err:", err)
		_ = fmt.Errorf("select item_masters: %w", err)
	}
}

func GetCardMaster(cardID int) CardMaster {
	cardMaster := CardMaster{}
	for _, v := range cardMasters {
		if v.CardID == cardID {
			cardMaster = v
			break
		}
	}
	return cardMaster
}

func GetCardMasters() CardMasters {
	return cardMasters
}

func GetCardLevelAndAmountPerSec(card CardMaster, totalExp int64) (int, int) {
	level := 1
	amountPerSec := card.BaseAmountPerSec
	for {
		nextLvThreshold := int64(float64(card.BaseExpPerLevel) * math.Pow(1.2, float64(level-1)))
		if nextLvThreshold > totalExp {
			break
		}

		// lv up処理
		level += 1
		amountPerSec += (card.MaxAmountPerSec - card.BaseAmountPerSec) / (card.MaxLevel - 1)
	}
	return level, amountPerSec
}

func (cm CardMasters) Commit(outdir string) error {
	localOutdir = outdir
	data, err := json.Marshal(cm)
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

func (cm CardMasters) Rename(fileName string) error {
	err := os.Rename(localOutdir+"/"+"master.json", localOutdir+"/"+fileName)
	if err != nil {
		fmt.Println(err)
		return err
	}
	return nil
}
