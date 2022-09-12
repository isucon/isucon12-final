package main

import (
	"context"

	"github.com/isucon/isucandar"
	"github.com/isucon/isucandar/agent"
	"github.com/isucon/isucandar/failure"
)

// 初期データを読み込む。
func (s *Scenario) LoadInitialData() error {
	if err := s.Users.LoadJSON("./dump/royalUserInitialize.json"); err != nil {
		ContestantLogger.Println("初期データ（ロイヤルユーザー）のロードに失敗しました")
		return failure.NewError(ErrFailedToLoadJson, err)
	}
	if err := s.Users.LoadJSON("./dump/combackUserInitialize.json"); err != nil {
		ContestantLogger.Println("初期データ（カムバックユーザー）のロードに失敗しました")
		return failure.NewError(ErrFailedToLoadJson, err)
	}
	if err := s.Users.LoadJSON("./dump/oneYearUserInitialize.json"); err != nil {
		ContestantLogger.Println("初期データ（1周年ユーザー）のロードに失敗しました")
		return failure.NewError(ErrFailedToLoadJson, err)
	}
	if err := s.BanUsers.LoadJSON("./dump/banUserInitialize.json"); err != nil {
		ContestantLogger.Println("初期データ（Banユーザー）のロードに失敗しました")
		return failure.NewError(ErrFailedToLoadJson, err)
	}
	if err := s.Platforms.LoadJSON("./dump/platforms.json"); err != nil {
		ContestantLogger.Println("初期データ（Registユーザー）のロードに失敗しました")
		return failure.NewError(ErrFailedToLoadJson, err)
	}
	return nil
}

// webapp の POST /initialize を叩く。
func (s *Scenario) DoInitialize(ctx context.Context, step *isucandar.BenchmarkStep, agent *agent.Agent) error {
	res, err := PostInitializeAction(ctx, agent)
	if err != nil {
		return failure.NewError(ErrPrepareInvalidRequest, err)
	}
	defer res.Body.Close()

	initializeResponse := &InitializeResponse{}

	validationError := ValidateResponse(
		res,
		WithInitializationSuccess(initializeResponse),
	)
	validationError.Add(step)

	// 後の統計用に使用言語を取得し、ロギングしておく。
	s.Language = initializeResponse.Language
	AdminLogger.Printf("[LANGUAGE] %s", initializeResponse.Language)

	if !validationError.IsEmpty() {
		ContestantLogger.Printf("初期化リクエストに失敗しました")
		return validationError
	} else {
		return nil
	}
}
