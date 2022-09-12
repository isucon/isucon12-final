package models

import (
	"fmt"
)

type UserPresentAllReceivedHistory struct {
	ID           int64  `json:"id" db:"id"`
	UserID       int64  `json:"userId" db:"user_id"`
	PresentAllID int64  `json:"presentAllId" db:"present_all_id"`
	ReceivedAt   int64  `json:"receivedAt" db:"received_at"`
	CreatedAt    int64  `json:"createdAt" db:"created_at"`
	UpdatedAt    int64  `json:"updatedAt" db:"updated_at"`
	DeletedAt    *int64 `json:"deletedAt,omitempty" db:"deleted_at"`
}

func NewUserPresentAllReceivedHistory(i int64, user User, presentAllID int64, receivedAt int64, createdAt int64) UserPresentAllReceivedHistory {
	return UserPresentAllReceivedHistory{i, user.ID, presentAllID, receivedAt, createdAt, receivedAt, nil}
}

func (u UserPresentAllReceivedHistory) Create() error {
	if _, err := Db.Exec("INSERT INTO user_present_all_received_history VALUES (?,?,?,?,?,?,?)",
		u.ID,
		u.UserID,
		u.PresentAllID,
		u.ReceivedAt,
		u.CreatedAt,
		u.UpdatedAt,
		nil); err != nil {
		fmt.Println(err)
		return fmt.Errorf("insert user_present_all_received_history: %w", err)
	}
	return nil
}

func (u UserPresentAllReceivedHistory) CreateDeleted() error {
	if _, err := Db.Exec("INSERT INTO user_present_all_received_history VALUES (?,?,?,?,?,?,?)",
		u.ID,
		u.UserID,
		u.PresentAllID,
		u.ReceivedAt,
		u.CreatedAt,
		u.UpdatedAt,
		u.UpdatedAt); err != nil {
		fmt.Println(err)
		return fmt.Errorf("insert user_present_all_received_history: %w", err)
	}
	return nil
}

func UserPresentAllReceivedHistoryBulkCreate(UserPresentAllReceivedHistories *[]UserPresentAllReceivedHistory) error {
	_, err := Db.NamedExec("INSERT INTO user_present_all_received_history "+
		" (`id`, `user_id`, `present_all_id`, `received_at`, `created_at`, `updated_at`) "+
		" VALUES "+
		" (:id, :user_id, :present_all_id, :received_at, :created_at, :updated_at)", *UserPresentAllReceivedHistories)
	if err != nil {
		fmt.Println(err)
		return fmt.Errorf("insert user_present_all_received_history: %w", err)
	}
	return nil
}
