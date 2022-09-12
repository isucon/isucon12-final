package main

// リクエストを送信する際にリクエストボディに詰める JSON を書くファイル

type LoginRequest struct {
	UserID   int64  `json:"userId"`
	ViewerID string `json:"viewerId"`
}

type AdminUserLoginRequest struct {
	UserID   int64  `json:"userId"`
	Password string `json:"password"`
}

type ReceivePresentRequest struct {
	ViewerID   string  `json:"viewerId"`
	PresentIDs []int64 `json:"presentIds"`
}

type PostAddCardExpRequest struct {
	ViewerID     string           `json:"viewerId"`
	OneTimeToken string           `json:"oneTimeToken"`
	Items        []AddCardExpItem `json:"items"`
}

type AddCardExpItem struct {
	Amount int   `json:"amount"`
	ID     int64 `json:"id"`
}

type DrawGachaRequest struct {
	ViewerID     string `json:"viewerId"`
	OneTimeToken string `json:"oneTimeToken"`
}

type CreateUserRequest struct {
	ViewerID     string `json:"viewerId"`
	PlatformType int    `json:"platformType"`
}

type RewardRequest struct {
	ViewerID string `json:"viewerId"`
}

type PostCardRequest struct {
	ViewerID string  `json:"viewerId"`
	CardIDs  []int64 `json:"cardIds"`
}
