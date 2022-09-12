package models

import (
	"encoding/json"
	"fmt"
	"os"
)

type JsonValidates []*JsonValidate

func (j *JsonValidates) Commit(outdir string) error {
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

func (j *JsonValidates) Rename(fileName string) error {
	err := os.Rename(localOutdir+"/"+"initialize.json", localOutdir+"/"+fileName)
	if err != nil {
		fmt.Println(err)
		return err
	}
	return nil
}

type JsonValidate struct {
	UserType                       string                          `json:"user_type"`
	UserID                         int64                           `json:"user_id"`
	ViewerID                       string                          `json:"viewer_id"`
	UserLoginBonuses               []UserLoginBonus                `json:"userLoginBonuses,omitempty"`
	UserLoginAppendPresents        []UserPresent                   `json:"userLoginAppendPresents,omitempty"`
	JsonUser                       User                            `json:"user,omitempty"`
	UserDeck                       UserDeck                        `json:"userDeck,omitempty"`
	UserDevices                    []UserDevice                    `json:"userDevices,omitempty"`
	TotalAmountPerSec              int64                           `json:"totalAmountPerSec,omitempty"`
	GetItemList                    []UserItem                      `json:"userItem,omitempty"`
	UserCards                      []UserCard                      `json:"userCard,omitempty"`
	UserPresents                   []UserPresent                   `json:"userPresent,omitempty"`
	UserAllPresents                []UserPresent                   `json:"userAllPresents,omitempty"`
	UserPresentAllReceiveHistories []UserPresentAllReceivedHistory `json:"userPresentAllReceivedHistory,omitempty"`
}
