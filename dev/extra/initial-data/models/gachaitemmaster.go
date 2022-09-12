package models

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
)

type GachaMaster struct {
	ID           int64  `json:"id" db:"id"`
	Name         string `json:"name" db:"name"`
	StartAt      int64  `json:"startAt" db:"start_at"`
	EndAt        int64  `json:"endAt" db:"end_at"`
	DisplayOrder int    `json:"displayOrder" db:"display_order"`
	CreatedAt    int64  `json:"createdAt" db:"created_at"`
}

type GachaMasters []GachaMaster

type GachaItemMaster struct {
	ID        int64 `json:"id" db:"id"`
	GachaID   int64 `json:"gachaId" db:"gacha_id"`
	ItemType  int   `json:"itemType" db:"item_type"`
	ItemID    int   `json:"itemId" db:"item_id"`
	Amount    int   `json:"amount" db:"amount"`
	Weight    int   `json:"weight" db:"weight"`
	CreatedAt int64 `json:"createdAt" db:"created_at"`
}

type GachaData struct {
	Gacha     GachaMaster       `json:"gacha"`
	GachaItem []GachaItemMaster `json:"gachaItemList"`
}

type GachaAllItemMasters []GachaData

var gachaAllMasters []GachaData
var gachaMasters []GachaMaster

func InitAllGachaItemMaster() {
	var gachaIMs []GachaData
	var err error

	err = Db.Select(&gachaMasters, "SELECT * FROM gacha_masters ORDER BY id")
	if err != nil {
		fmt.Println(err)
		_ = fmt.Errorf("select gacha_masters: %w", err)
		return
	}

	for _, v := range gachaMasters {
		var gachaItemMasters []GachaItemMaster

		err = Db.Select(&gachaItemMasters, "SELECT * FROM gacha_item_masters WHERE gacha_id=? ORDER BY id", v.ID)
		if err != nil {
			fmt.Println(err)
			_ = fmt.Errorf("select gacha_item_masters: %w", err)
			return
		}

		gachaIMs = append(gachaIMs, GachaData{
			Gacha:     v,
			GachaItem: gachaItemMasters,
		})
	}
	gachaAllMasters = gachaIMs
}

func DrawGacha(gachaID int) GachaItemMaster {
	gachaItemMasters := []GachaItemMaster{}

	for _, v := range gachaAllMasters {
		if int64(v.Gacha.ID) == int64(gachaID) {
			gachaItemMasters = v.GachaItem
		}
	}

	weightTotal := 0
	for _, v := range gachaItemMasters {
		weightTotal += v.Weight
	}

	random := rand.Intn(weightTotal)
	boundary := 0
	result := &GachaItemMaster{}

	for _, v := range gachaItemMasters {
		boundary += v.Weight
		if random < boundary {
			result = &v
			break
		}
	}
	return *result
}

func DrawManyGacha(gachaID int, drawCount int) []GachaItemMaster {
	var gachaResult []GachaItemMaster
	for i := 0; i < drawCount; i++ {
		gachaItemMaster := DrawGacha(gachaID)
		gachaResult = append(gachaResult, gachaItemMaster)
	}
	return gachaResult
}

func GetGachaAllItemMasters() GachaAllItemMasters {
	return gachaAllMasters
}

func (gaim GachaAllItemMasters) Commit(outdir string) error {
	localOutdir = outdir
	data, err := json.Marshal(gaim)
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

func (gaim GachaAllItemMasters) Rename(fileName string) error {
	err := os.Rename(localOutdir+"/"+"master.json", localOutdir+"/"+fileName)
	if err != nil {
		fmt.Println(err)
		return err
	}
	return nil
}
