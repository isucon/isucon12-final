package models

import (
	"encoding/json"
	"fmt"
	"os"
)

type PresentAllMaster struct {
	ID                int64  `json:"id" db:"id"`
	RegisteredStartAt int64  `json:"registeredStartAt" db:"registered_start_at"`
	RegisteredEndAt   int64  `json:"registeredEndAt" db:"registered_end_at"`
	ItemType          int    `json:"itemType" db:"item_type"`
	ItemID            int64  `json:"itemId" db:"item_id"`
	Amount            int64  `json:"amount" db:"amount"`
	PresentMessage    string `json:"presentMessage" db:"present_message"`
	CreatedAt         int64  `json:"createdAt" db:"created_at"`
}

type PresentAllMasters []PresentAllMaster

func GetPresentAllMasters() PresentAllMasters {
	presentAllMasters := []PresentAllMaster{}
	err := Db.Select(&presentAllMasters, "SELECT * FROM `present_all_masters` ORDER BY id")
	if err != nil {
		_ = fmt.Errorf("select present_all_masters: %w", err)
		return nil
	}
	return presentAllMasters
}

func (pam PresentAllMasters) Commit(outdir string) error {
	localOutdir = outdir
	data, err := json.Marshal(pam)
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

func (pam PresentAllMasters) Rename(fileName string) error {
	err := os.Rename(localOutdir+"/"+"master.json", localOutdir+"/"+fileName)
	if err != nil {
		return err
	}
	return nil
}
