package main

// ユーザー登録のシナリオを一通り管理するファイル。

import (
	"context"
	"math/rand"
	"net/http"
	"time"

	"github.com/isucon/isucandar"
	"github.com/isucon/isucandar/failure"
	"github.com/isucon/isucandar/worker"
)

// Banユーザーのログイン試行シナリオを実行する。ログインのみ実施
func (s *Scenario) NewBanUserLoginScenarioWorker(step *isucandar.BenchmarkStep, p int32) (*worker.Worker, error) {
	banUserLogin, err := worker.NewWorker(func(ctx context.Context, _ int) {
		PrintScenarioStarted(ScenarioBanUserLogin)
		defer PrintScenarioFinished(ScenarioBanUserLogin)

		user := s.BanUsers.At(rand.Intn(s.BanUsers.Len()))

	Rewind:

		// ワーカー開始時点での最新のマスターバージョンを取得する。
		masterVersion := s.LatestMasterVersion()

		// 1. ログイン処理
		result := s.loginBanScenario(ctx, step, user, masterVersion)
		if result.Rewind {
			goto Rewind
		}

		SleepWithCtx(ctx, time.Millisecond*500)

		user.ClearAgent()
	}, loopConfig(s), parallelismConfig(s))

	banUserLogin.SetParallelism(p)

	return banUserLogin, err
}

func (s *Scenario) loginBanScenario(ctx context.Context, step *isucandar.BenchmarkStep, user *User, masterVersion string) ScenarioResult {
	report := TimeReporter("ログインBan シナリオ", s.Option)
	defer report()

	agent, err := s.GetAgentFromUser(step, user)
	if err != nil {
		return NoRewind()
	}

	loginRes, err := PostLoginAction(ctx, agent, user.ID, user.ViewerID, masterVersion, time.Now())
	if err != nil {
		AddErrorIfNotCanceled(step, failure.NewError(ErrInvalidRequest, err))
		return NoRewind()
	}
	defer loginRes.Body.Close()

	if loginRes.StatusCode == http.StatusUnprocessableEntity {
		return Rewind()
	}

	validateLoginBanResponse := &validateFailResponse{}

	loginBanValidation := ValidateResponse(
		loginRes,
		WithStatusCode(LoginBanStatusCode),
		WithJsonBody(validateLoginBanResponse),
	)
	loginBanValidation.Add(step)

	if loginBanValidation.IsEmpty() {
		step.AddScore(ScoreBanLogin)
	}

	return NoRewind()
}
