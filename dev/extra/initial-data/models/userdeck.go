package models

import (
	"fmt"
)

type UserDeck struct {
	ID        int64 `json:"id" db:"id"`
	UserID    int64 `json:"userId" db:"user_id"`
	CardID1   int64 `json:"cardId1" db:"user_card_id_1"`
	CardID2   int64 `json:"cardId2" db:"user_card_id_2"`
	CardID3   int64 `json:"cardId3" db:"user_card_id_3"`
	CreatedAt int64 `json:"createdAt" db:"created_at"`
	UpdatedAt int64 `json:"updatedAt" db:"updated_at"`
}

func NewUserDeck(i int64, user User, userCardID1 int64, userCardID2 int64, userCardID3 int64, createdAt int64) UserDeck {
	return UserDeck{i, user.ID, userCardID1, userCardID2, userCardID3, createdAt, createdAt}
}

func (u UserDeck) Create() error {
	if _, err := Db.Exec("INSERT INTO user_decks VALUES (?,?,?,?,?,?,?,?)",
		u.ID,
		u.UserID,
		u.CardID1,
		u.CardID2,
		u.CardID3,
		u.CreatedAt,
		u.UpdatedAt,
		nil); err != nil {
		fmt.Println(err)
		return fmt.Errorf("insert user_decks: %w", err)
	}
	return nil
}
