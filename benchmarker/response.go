package main

// リクエストの結果返ってくる JSON レスポンスを集約するファイル

type InitializeResponse struct {
	Language string `json:"language"`
}

type LoginResponse struct {
	SessionID string `json:"sessionId"`
	ViewerID  string `json:"viewerId"`
}

type AdminLoginResponse struct {
	Session  AdminSession `json:"session"`
	ViewerID string       `json:"viewerId"`
}

type AdminSession struct {
	SessionID string `json:"sessionId"`
}

type ListPresentResponse struct {
	Presents []*UserPresent `json:"presents"`
	IsNext   bool           `json:"isNext"`
}

type UserPresent struct {
	ID             int64  `json:"id" db:"id"`
	UserID         int64  `json:"userId" db:"user_id"`
	SentAt         int64  `json:"sentAt" db:"sent_at"`
	ItemType       int    `json:"itemType" db:"item_type"`
	ItemID         int64  `json:"itemId" db:"item_id"`
	Amount         int    `json:"amount" db:"amount"`
	PresentMessage string `json:"presentMessage" db:"present_message"`
	CreatedAt      int64  `json:"createdAt" db:"created_at"`
	UpdatedAt      int64  `json:"updatedAt" db:"updated_at"`
	DeletedAt      *int64 `json:"deletedAt" db:"deleted_at"`
}

type ReceivePresentResponse struct {
	UpdatedResources *UpdatedResource `json:"updatedResources"`
}

type ListGachaResponse struct {
	OneTimeToken string      `json:"oneTimeToken"`
	Gachas       []GachaData `json:"gachas"`
}

type GachaData struct {
	Gacha     GachaMaster       `json:"gacha"`
	GachaItem []GachaItemMaster `json:"gachaItemList"`
}

type GachaMaster struct {
	ID           int64  `json:"id"`
	Name         string `json:"name"`
	StartAt      int64  `json:"startAt"`
	EndAt        int64  `json:"endAt"`
	DisplayOrder int    `json:"displayOrder"`
	CreatedAt    int64  `json:"createdAt"`
}

type GachaItemMaster struct {
	ID        int64 `json:"id"`
	GachaID   int64 `json:"gachaId"`
	ItemType  int   `json:"itemType"`
	ItemID    int64 `json:"itemId"`
	Amount    int   `json:"amount"`
	Weight    int   `json:"weight"`
	CreatedAt int64 `json:"createdAt"`
}

type ItemMaster struct {
	ID              int64  `json:"id"`
	ItemType        int    `json:"itemType"`
	Name            string `json:"name"`
	Description     string `json:"description"`
	AmountPerSec    *int   `json:"amountPerSec"`
	MaxLevel        *int   `json:"maxLevel"`
	MaxAmountPerSec *int   `json:"maxAmountPerSec"`
	BaseExpPerLevel *int   `json:"baseExpPerLevel"`
	GainedExp       *int   `json:"gainedExp"`
	ShorteningMin   *int64 `json:"shorteningMin"`
}

type PresentAllMaster struct {
	ID                int64  `json:"id"`
	RegisteredStartAt int64  `json:"registeredStartAt"`
	RegisteredEndAt   int64  `json:"registeredEndAt"`
	ItemType          int    `json:"itemType"`
	ItemID            int64  `json:"itemID"`
	Amount            int64  `json:"amount"`
	PresentMessage    string `json:"presentMessage"`
	CreatedAt         int64  `json:"createdAt"`
}

type LoginBonusMaster struct {
	ID          int64 `json:"id"`
	StartAt     int64 `json:"startAt"`
	EndAt       int64 `json:"endAt"`
	ColumnCount int   `json:"columnCount"`
	Looped      bool  `json:"looped"`
	CreatedAt   int64 `json:"createdAt"`
}

type DrawGachaResponse struct {
	Present []UserPresent `json:"presents"`
}

type UpdatedResource struct {
	Now          int64         `json:"now"`
	UserPresents []UserPresent `json:"userPresents,omitempty"`
}

type CreateUserResponse struct {
	SessionID string `json:"sessionId"`
	UserID    int64  `json:"userId"`
	ViewerId  string `json:"viewerId"`
}

type MasterDataResponse struct {
	VersionMaster versionMaster `json:"versionMaster"`
}

type versionMaster struct {
	Id            int64  `json:"id"`
	MasterVersion string `json:"masterVersion"`
	Status        int64  `json:"status"`
}

type validatePostUserResponse struct {
	UserID           int64           `json:"userId"`
	ViewerID         string          `json:"viewerId"`
	SessionID        string          `json:"sessionId"`
	CreatedAt        int64           `json:"createdAt"`
	UpdatedResources updatedResource `json:"updatedResources"`
}

type validateLoginResponse struct {
	SessionID        string          `json:"sessionId"`
	ViewerID         string          `json:"viewerId"`
	UpdatedResources updatedResource `json:"updatedResources"`
}

type validateFailResponse struct {
	StatusCode int    `json:"status_code"`
	Message    string `json:"message"`
}

type validateHomeResponse struct {
	Now               int64    `json:"now"`
	User              JsonUser `json:"user"`
	Deck              UserDeck `json:"deck"`
	TotalAmountPerSec int      `json:"totalAmountPerSec"`
	PastTime          int64    `json:"pastTime"`
}

type validateRewardResponse struct {
	UpdatedResources updatedResource `json:"updatedResources"`
}

type validateItemListResponse struct {
	OneTimeToken string     `json:"oneTimeToken"`
	User         JsonUser   `json:"user"`
	UserItems    []UserItem `json:"items"`
	UserCards    []UserCard `json:"cards"`
}

type validatePostCardResponse struct {
	UpdatedResources updatedResource `json:"updatedResources"`
}

type validatePostAddExpCardResponse struct {
	UpdatedResources updatedResource `json:"updatedResources"`
}

type adminLoginResponse struct {
	Session Session `json:"session"`
}

type postAdminUserBanResponse struct {
	User JsonUser `json:"user"`
}

type getAdminMasterResponse struct {
	VersionMaster          []versionMaster          `json:"versionMaster"`
	ItemMaster             []ItemMaster             `json:"items"`
	GachaMaster            []GachaMaster            `json:"gachas"`
	GachaItemMaster        []GachaItemMaster        `json:"gachaItems"`
	PresentAllMaster       []PresentAllMaster       `json:"presentAlls"`
	LoginBonusMaster       []LoginBonusMaster       `json:"loginBonuses"`
	LoginBonusRewardMaster []LoginBonusRewardMaster `json:"loginBonusRewards"`
}

type getAdminUserResponse struct {
	User                          JsonUser                        `json:"user"`
	UserDevices                   []UserDevice                    `json:"userDevices"`
	UserCards                     []UserCard                      `json:"userCards"`
	UserDecks                     []UserDeck                      `json:"userDecks"`
	UserItems                     []UserItem                      `json:"userItems"`
	UserLoginBonus                []UserLoginBonus                `json:"userLoginBonuses"`
	UserPresents                  []UserPresent                   `json:"userPresents"`
	UserPresentAllReceivedHistory []UserPresentAllReceivedHistory `json:"userPresentAllReceivedHistory"`
}

type UserPresentAllReceivedHistory struct {
	ID           int64  `json:"id"`
	UserID       int64  `json:"userId"`
	PresentAllID int64  `json:"presentAllId"`
	ReceivedAt   int64  `json:"receivedAt"`
	CreatedAt    int64  `json:"createdAt"`
	UpdatedAt    int64  `json:"updatedAt"`
	DeletedAt    *int64 `json:"deletedAt,omitempty"`
}

type localExpItemMaster struct {
	ID        int64 `json:"id"`
	GainedExp int   `json:"gainedExp"`
}

type CardMaster struct {
	CardID           int `json:"id" db:"id"`
	BaseExpPerLevel  int `json:"baseExpPerLevel" db:"base_exp_per_level"`
	MaxAmountPerSec  int `json:"maxAmountPerSec" db:"max_amount_per_sec"`
	BaseAmountPerSec int `json:"amountPerSec" db:"amount_per_sec"`
	MaxLevel         int `json:"maxLevel" db:"max_level"`
}

type CardMasters []CardMaster

type LoginBonusRewardMaster struct {
	ID             int64 `json:"id" db:"id"`
	LoginBonusID   int64 `json:"loginBonusId" db:"login_bonus_id"`
	RewardSequence int   `json:"rewardSequence" db:"reward_sequence"`
	ItemType       int   `json:"itemType" db:"item_type"`
	ItemID         int64 `json:"itemId" db:"item_id"`
	Amount         int64 `json:"amount" db:"amount"`
}

type LoginBonusRewardMasters []LoginBonusRewardMaster

type updatedResource struct {
	Now  int64    `json:"now"`
	User JsonUser `json:"user,omitempty"`

	UserDevice       UserDevice       `json:"userDevice,omitempty"`
	UserCards        []UserCard       `json:"userCards,omitempty"`
	UserDecks        []UserDeck       `json:"userDecks,omitempty"`
	UserItems        []UserItem       `json:"userItems,omitempty"`
	UserLoginBonuses []UserLoginBonus `json:"userLoginBonuses,omitempty"`
	UserPresents     []UserPresent    `json:"userPresents,omitempty"`
}

type JsonUser struct {
	ID              int64  `json:"id" db:"id"`
	IsuCoin         int64  `json:"isuCoin" db:"isu_coin"`
	LastGetRewardAt int64  `json:"lastGetRewardAt" db:"last_getreward_at"`
	LastActivatedAt int64  `json:"lastActivatedAt" db:"last_activated_at"`
	RegisteredAt    int64  `json:"registeredAt" db:"registered_at"`
	CreatedAt       int64  `json:"createdAt" db:"created_at"`
	UpdatedAt       int64  `json:"updatedAt" db:"updated_at"`
	DeletedAt       *int64 `json:"deletedAt" db:"deleted_at"`
}

type UserDevice struct {
	ID           int64  `json:"id" db:"id"`
	UserID       int64  `json:"userId" db:"user_id"`
	PlatformID   string `json:"platformId" db:"platform_id"`
	PlatformType int    `json:"platformType" db:"platform_type"`
	CreatedAt    int64  `json:"createdAt" db:"created_at"`
	UpdatedAt    int64  `json:"updatedAt" db:"updated_at"`
	DeletedAt    *int64 `json:"deletedAt" db:"deleted_at"`
}

type UserCard struct {
	ID           int64  `json:"id" db:"id"`
	UserID       int64  `json:"userId" db:"user_id"`
	CardID       int64  `json:"cardId" db:"card_id"`
	AmountPerSec int    `json:"amountPerSec" db:"amount_per_sec"`
	Level        int    `json:"level" db:"level"`
	TotalExp     int64  `json:"totalExp" db:"total_exp"`
	CreatedAt    int64  `json:"createdAt" db:"created_at"`
	UpdatedAt    int64  `json:"updatedAt" db:"updated_at"`
	DeletedAt    *int64 `json:"deletedAt" db:"deleted_at"`
}

type UserDeck struct {
	ID        int64  `json:"id" db:"id"`
	UserID    int64  `json:"userId" db:"user_id"`
	CardID1   int64  `json:"cardId1" db:"user_card_id_1"`
	CardID2   int64  `json:"cardId2" db:"user_card_id_2"`
	CardID3   int64  `json:"cardId3" db:"user_card_id_3"`
	CreatedAt int64  `json:"createdAt" db:"created_at"`
	UpdatedAt int64  `json:"updatedAt" db:"updated_at"`
	DeletedAt *int64 `json:"deletedAt" db:"deleted_at"`
}

type UserItem struct {
	ID        int64  `json:"id" db:"id"`
	UserID    int64  `json:"userId" db:"user_id"`
	ItemType  int    `json:"itemType" db:"item_type"`
	ItemID    int64  `json:"itemId" db:"item_id"`
	Amount    int    `json:"amount" db:"amount"`
	CreatedAt int64  `json:"createdAt" db:"created_at"`
	UpdatedAt int64  `json:"updatedAt" db:"updated_at"`
	DeletedAt *int64 `json:"deletedAt" db:"deleted_at"`
}

type UserLoginBonus struct {
	ID                 int64  `json:"id" db:"id"`
	UserID             int64  `json:"userId" db:"user_id"`
	LoginBonusID       int64  `json:"loginBonusId" db:"login_bonus_id"`
	LastRewardSequence int    `json:"lastRewardSequence" db:"last_reward_sequence"`
	LoopCount          int    `json:"loopCount" db:"loop_count"`
	CreatedAt          int64  `json:"createdAt" db:"created_at"`
	UpdatedAt          int64  `json:"updatedAt" db:"updated_at"`
	DeletedAt          *int64 `json:"deletedAt" db:"deleted_at"`
}

type Session struct {
	ID        int64  `json:"id"`
	UserID    int64  `json:"userId"`
	SessionID string `json:"sessionId"`
	CreatedAt int64  `json:"createdAt"`
	UpdatedAt int64  `json:"updatedAt"`
	ExpiredAt int64  `json:"expiredAt"`
	DeletedAt *int64 `json:"deletedAt,omitempty"`
}
