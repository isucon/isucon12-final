package models

import (
	"encoding/json"
	"fmt"
	"os"
)

type ExpItemMasters []ExpItemMaster

type ExpItemMaster struct {
	ID        int64 `json:"id" db:"id"`
	GainedExp int   `json:"gainedExp" db:"gained_exp"`
}

func GetExpItemMaster() ExpItemMasters {
	var err error
	var expItemMasters []ExpItemMaster
	err = Db.Select(&expItemMasters, "SELECT `id`, `gained_exp`  FROM `item_masters` WHERE item_type = 3 ORDER BY id")
	if err != nil {
		fmt.Println("err:", err)
		_ = fmt.Errorf("select item_masters: %w", err)
	}
	return expItemMasters
}

func (eim ExpItemMasters) Commit(outdir string) error {
	localOutdir = outdir
	data, err := json.Marshal(eim)
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

func (eim ExpItemMasters) Rename(fileName string) error {
	err := os.Rename(localOutdir+"/"+"master.json", localOutdir+"/"+fileName)
	if err != nil {
		fmt.Println(err)
		return err
	}
	return nil
}
