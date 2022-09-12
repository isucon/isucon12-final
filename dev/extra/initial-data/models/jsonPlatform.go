package models

import (
	"encoding/json"
	"fmt"
	"os"
)

type Platform struct {
	ID   int64 `json:"platform_id"`
	Type int   `json:"platform_type"`
}

type JsonPlatform []*Platform

func (j *JsonPlatform) Commit(outdir string) error {
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

func (j *JsonPlatform) Rename(fileName string) error {
	err := os.Rename(localOutdir+"/"+"initialize.json", localOutdir+"/"+fileName)
	if err != nil {
		fmt.Println(err)
		return err
	}
	return nil
}
