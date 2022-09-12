package main

import (
	"sync"

	"github.com/isucon/isucandar/agent"
)

// benchmarker 内部で使用するモデルを集約するファイル

// ログイン成功シナリオで使用する「ユーザー」を表現するデータ。
// 初期化処理（initialize）の時点でデータは生成済み。
type User struct {
	mu sync.RWMutex

	ID       int64  `json:"user_id"`
	UserType string `json:"user_type"`
	ViewerID string `json:"viewer_id"`

	Agent *agent.Agent
}

func (u *User) GetID() int64 {
	return u.ID
}

func (u *User) GetAgent(o Option) (*agent.Agent, error) {
	u.mu.RLock()
	agent := u.Agent
	u.mu.RUnlock()

	if agent != nil {
		return agent, nil
	}

	u.mu.Lock()
	defer u.mu.Unlock()

	agent, err := o.NewAgent(false)
	if err != nil {
		return nil, err
	}

	u.Agent = agent

	return agent, nil
}

func (u *User) ClearAgent() {
	u.mu.Lock()
	defer u.mu.Unlock()

	u.Agent = nil
}

// 新規ユーザー登録のシナリオで使用する新規ユーザー作成時に利用するデータ。
// 初期化処理の時点で生成済み。
type Platform struct {
	mu sync.RWMutex

	ID   int64 `json:"platform_id"`
	Type int   `json:"platform_type"`

	Agent *agent.Agent
}

func (p *Platform) GetID() int64 {
	return p.ID
}

func (p *Platform) GetAgent(o Option) (*agent.Agent, error) {
	p.mu.RLock()
	agent := p.Agent
	p.mu.RUnlock()

	if agent != nil {
		return agent, nil
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	agent, err := o.NewAgent(false)
	if err != nil {
		return nil, err
	}

	p.Agent = agent

	return agent, nil
}

func (p *Platform) ClearAgent() {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.Agent = nil
}

type AdminUser struct {
	mu sync.RWMutex

	ID       int64  `json:"user_id"`
	Password string `json:"password"`

	Agent *agent.Agent
}

func (u *AdminUser) GetID() int64 {
	return u.ID
}

func (u *AdminUser) GetAgent(o Option) (*agent.Agent, error) {
	u.mu.RLock()
	agent := u.Agent
	u.mu.RUnlock()

	if agent != nil {
		return agent, nil
	}

	u.mu.Lock()
	defer u.mu.Unlock()

	agent, err := o.NewAgent(false)
	if err != nil {
		return nil, err
	}

	u.Agent = agent

	return agent, nil
}

func (u *AdminUser) ClearAgent() {
	u.mu.Lock()
	defer u.mu.Unlock()
	u.Agent = nil
}

//ユーザBanシナリオで使用するユーザー
type UserBan struct {
	mu sync.RWMutex

	ID       int64  `json:"user_id"`
	UserType string `json:"user_type"`
	ViewerID string `json:"viewer_id"`

	Agent *agent.Agent
}

func (u *UserBan) GetID() int64 {
	return u.ID
}

func (u *UserBan) GetAgent(o Option) (*agent.Agent, error) {
	u.mu.RLock()
	agent := u.Agent
	u.mu.RUnlock()

	if agent != nil {
		return agent, nil
	}

	u.mu.Lock()
	defer u.mu.Unlock()

	agent, err := o.NewAgent(false)
	if err != nil {
		return nil, err
	}

	u.Agent = agent

	return agent, nil
}

func (u *UserBan) ClearAgent() {
	u.mu.Lock()
	defer u.mu.Unlock()

	u.Agent = nil
}

type ValidationUser struct {
	mu sync.RWMutex

	ID                             int64                           `json:"user_id"`
	UserType                       string                          `json:"user_type"`
	ViewerID                       string                          `json:"viewer_id"`
	UserLoginBonuses               []UserLoginBonus                `json:"userLoginBonuses,omitempty"`
	UserLoginAppendPresents        []UserPresent                   `json:"userLoginAppendPresents,omitempty"`
	JsonUser                       JsonUser                        `json:"user"`
	UserDeck                       UserDeck                        `json:"userDeck,omitempty"`
	UserDevices                    []UserDevice                    `json:"userDevices,omitempty"`
	TotalAmountPerSec              int64                           `json:"totalAmountPerSec,omitempty"`
	GetItemList                    []UserItem                      `json:"userItem,omitempty"`
	UserCards                      []UserCard                      `json:"userCard,omitempty"`
	UserPresents                   []UserPresent                   `json:"userPresent,omitempty"`
	UserAllPresents                []UserPresent                   `json:"userAllPresents,omitempty"`
	UserPresentAllReceiveHistories []UserPresentAllReceivedHistory `json:"userPresentAllReceivedHistory"`

	Agent *agent.Agent
}

func (u *ValidationUser) GetID() int64 {
	return u.ID
}

func (u *ValidationUser) GetAgent(o Option) (*agent.Agent, error) {
	u.mu.RLock()
	agent := u.Agent
	u.mu.RUnlock()

	if agent != nil {
		return agent, nil
	}

	u.mu.Lock()
	defer u.mu.Unlock()

	agent, err := o.NewAgent(false)
	if err != nil {
		return nil, err
	}

	u.Agent = agent

	return agent, nil
}

func (u *ValidationUser) ClearAgent() {
	u.mu.Lock()
	defer u.mu.Unlock()

	u.Agent = nil
}

type Login struct {
	SessionID string
	ViewerID  string
}

type UserCreated struct {
	SessionID string
	ViewerID  string
	UserID    int64
}
