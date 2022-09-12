package main

import (
	"fmt"
	"strings"
	"time"

	"github.com/isucon/isucandar/agent"
)

// アプリケーションの設定情報をまとめるファイル

// アプリケーション全体のデフォルト設定
const (
	DefaultTargetHost     = "localhost:8080"
	DefaultRequestTimeout = 3 * time.Second
	// `/initialize` へのリクエストは相対時間で15秒経過後タイムアウトする
	// ISUCON 関係者向けドキュメントの下記を参照のこと:
	// See: https://scrapbox.io/ISUCON12/%E3%83%99%E3%83%B3%E3%83%81%E3%83%9E%E3%83%BC%E3%82%AB%E3%83%BC%E3%83%86%E3%82%B9%E3%83%88%E3%82%B7%E3%83%8A%E3%83%AA%E3%82%AA
	DefaultInitializeRequestTimeout = 1 * time.Minute
	DefaultExitErrorOnFail          = true
	DefaultStage                    = "test"
	DefaultParallelism              = 100
	DefaultPrepareOnly              = false
)

// シナリオ関係の設定

const (
	// 負荷をかける時間
	LoadingDuration = 1 * time.Minute
	// マスター更新を発火させる時間
	MasterRefreshStartTime = 20 * time.Second
)

const (
	MaxErrors = 50
)

const (
	ScenarioLogin            string = "ログイン"
	ScenarioUserRegistration string = "新規ユーザー登録"
	ScenarioMasterRefresh    string = "マスター更新"
	ScenarioBanUserLogin     string = "Banユーザログイン"
)

// 検証シナリオ関係の設定

const (
	LastLoginBonusSequence      = 28
	LoginFailStatusCode         = 404
	LoginFailMessage            = "not found user"
	LoginBanStatusCode          = 403
	LoginBanMessage             = "forbidden"
	LoginAdminID                = 123456
	LoginAdminPassword          = "password"
	LoginUnauthorizedStatusCode = 401
	LoginUnauthorizedMessage    = "unauthorized user"
	BadRequest                  = 400
	InvalidToken                = "invalid token"
)

const (
	PostUser           string = "POST /user"
	PostAdminLogin     string = "POST /admin/login"
	PostAdminUserBan   string = "POST /admin/user/:userId/ban"
	GetAdminMaster     string = "GET  /admin/master"
	GetAdminUser       string = "GET  /admin/user/:userId"
	PostLogin          string = "POST /login"
	PostLoginBan       string = "POST /login(Ban)"
	GetUserHome        string = "GET  /user/:userId/home"
	PostRewardGet      string = "POST /user/:userId/reward"
	PostCardSet        string = "POST /user/:userId/card"
	GetItem            string = "GET  /user/:userId/item"
	PostCardAddexp     string = "POST /user/:userId/card/addexp/:cardId"
	GetPresent         string = "GET  /user/:userId/present/index/1"
	PostPresentReceive string = "POST /user/:userId/present/receive"
	GetGachaIndex      string = "GET  /user/:userId/gacha/index"
	PostGachaDraw      string = "POST /user/:userId/gacha/draw/:gachaId/10"
)

// 起動時の設定

type Option struct {
	TargetHost               string
	RequestTimeout           time.Duration
	InitializeRequestTimeout time.Duration
	ExitErrorOnFail          bool
	Stage                    string
	Parallelism              int
	PrepareOnly              bool
}

func (o Option) String() string {
	args := []string{
		"benchmarker",
		fmt.Sprintf("--target-host=%s", o.TargetHost),
		fmt.Sprintf("--request-timeout=%s", o.RequestTimeout.String()),
		fmt.Sprintf("--initialize-request-timeout=%s", o.InitializeRequestTimeout.String()),
		fmt.Sprintf("--exit-error-on-fail=%v", o.ExitErrorOnFail),
		fmt.Sprintf("--stage=%s", o.Stage),
		fmt.Sprintf("--max-parallelism=%d", o.Parallelism),
		fmt.Sprintf("--prepare-only=%v", o.PrepareOnly),
	}

	return strings.Join(args, " ")
}

func (o Option) NewAgent(forInitialize bool) (*agent.Agent, error) {
	agentOptions := []agent.AgentOption{
		agent.WithBaseURL(fmt.Sprintf("http://%s/", o.TargetHost)),
		agent.WithCloneTransport(agent.DefaultTransport),
	}

	if forInitialize {
		agentOptions = append(agentOptions, agent.WithTimeout(o.InitializeRequestTimeout))
	} else {
		agentOptions = append(agentOptions, agent.WithTimeout(o.RequestTimeout))
	}

	return agent.NewAgent(agentOptions...)
}
