package models

import (
	"encoding/json"
	"fmt"
	"os"
)

type JsonArray []*Json

var localOutdir string

func (j *JsonArray) Commit(outdir string) error {
	localOutdir = outdir
	data, err := json.Marshal(j)
	if err != nil {
		fmt.Println(err)
		return err
	}
	err = os.WriteFile(localOutdir+"/"+"initialize.json", data, 0666)
	if err != nil {
		fmt.Println(err)
		return err
	}
	return nil
}

func (j *JsonArray) Rename(fileName string) error {
	err := os.Rename(localOutdir+"/"+"initialize.json", localOutdir+"/"+fileName)
	if err != nil {
		fmt.Println(err)
		return err
	}
	return nil
}

type Json struct {
	UserType string `json:"user_type"`
	UserID   int64  `json:"user_id"`
	ViewerID string `json:"viewer_id"`
}
