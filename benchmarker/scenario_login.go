package main

// ログイン処理〜ガチャ成功までのシナリオを一通り管理するファイル。

import (
	"context"
	"math/rand"
	"net/http"
	"time"

	"github.com/isucon/isucandar"
	"github.com/isucon/isucandar/agent"
	"github.com/isucon/isucandar/failure"
	"github.com/isucon/isucandar/worker"
)

func (s *Scenario) NewLoginSuccessScenarioWorker(step *isucandar.BenchmarkStep, p int32) (*worker.Worker, error) {
	loginSuccess, err := worker.NewWorker(func(ctx context.Context, _ int) {
		PrintScenarioStarted(ScenarioLogin)
		defer PrintScenarioFinished(ScenarioLogin)

		// ユーザーデータ自体が、ユーザーの属性が均等になるように等分されているため、
		// 乱数インデックスで値を取り出してもランダムな属性のユーザーを取り出せるようになっている。
		var user *User
		for {
			trial := rand.Intn(s.Users.Len())
			if !s.ConsumedUserIDs.Exists(int64(trial)) {
				s.ConsumedUserIDs.Add(int64(trial))
				user = s.Users.At(trial)
				break
			}
		}
		defer s.ConsumedUserIDs.Remove(user.ID)

	Rewind:

		// ワーカー開始時点での最新のマスターバージョンを取得する。
		masterVersion := s.LatestMasterVersion()

		// 1. ログイン処理
		result, login := s.LoginSuccessScenario(ctx, step, user, masterVersion)
		if login == nil {
			return
		}
		if result.Rewind {
			goto Rewind
		}

		s.RecordLoginSuccessCount(1)

		// 2. ホーム画面表示
		result = s.ShowHomeViewSuccessScenario(ctx, step, user, masterVersion, login)
		if result.Rewind {
			goto Rewind
		}

		// 3. インゲーム報酬受け取り
		result = s.RedeemRewardSuccessScenario(ctx, step, user, masterVersion, login)
		if result.Rewind {
			goto Rewind
		}

		// 4. プレゼントの受け取り
		result = s.AcceptGiftSuccessScenario(ctx, step, user, masterVersion, login)
		if result.Rewind {
			goto Rewind
		}

		// 5. ガチャを引く
		result = s.RedeemGachaSuccessScenario(ctx, step, user, masterVersion, login)
		if result.Rewind {
			goto Rewind
		}

		user.ClearAgent()
	}, loopConfig(s), parallelismConfig(s))

	loginSuccess.SetParallelism(p)

	return loginSuccess, err
}

func (s *Scenario) GetAgentFromUser(step *isucandar.BenchmarkStep, user *User) (*agent.Agent, error) {
	agent, err := user.GetAgent(s.Option)
	if err != nil {
		step.AddError(failure.NewError(ErrCannotCreateNewAgent, err))
		return nil, err
	}
	return agent, nil
}

// ログインが成功するシナリオを実行できる。ログインの成功はスコアに加算される。
// リクエストを送ってステータスコードが成功状態であることと、レスポンスボディの形式が正しいかを確認する。
func (s *Scenario) LoginSuccessScenario(ctx context.Context, step *isucandar.BenchmarkStep, user *User, masterVersion string) (ScenarioResult, *Login) {
	report := TimeReporter("ログイン成功シナリオ", s.Option)
	defer report()

	agent, err := s.GetAgentFromUser(step, user)
	if err != nil {
		return NoRewind(), nil
	}

	now := time.Now()
	xIsuDate := time.Date(2022, 8, 27, now.Hour(), now.Minute(), now.Second(), now.Nanosecond(), myGMT)
	loginRes, err := PostLoginAction(ctx, agent, user.ID, user.ViewerID, masterVersion, xIsuDate)
	if err != nil {
		AddErrorIfNotCanceled(step, failure.NewError(ErrInvalidRequest, err))
		return NoRewind(), nil
	}
	defer loginRes.Body.Close()

	if loginRes.StatusCode == http.StatusUnprocessableEntity {
		return Rewind(), nil
	}

	loginResponse := &LoginResponse{}

	loginValidation := ValidateResponse(
		loginRes,
		WithStatusCode(200),
		WithJsonBody(loginResponse),
	)
	loginValidation.Add(step)

	if loginValidation.IsEmpty() {
		step.AddScore(ScoreLoginSuccess)

		login := &Login{
			SessionID: loginResponse.SessionID,
			ViewerID:  loginResponse.ViewerID,
		}
		return NoRewind(), login
	} else {
		return NoRewind(), nil
	}
}

// ホーム画面を表示できるシナリオを実行できる。ホーム画面の表示の成功はスコアに加算される。
func (s *Scenario) ShowHomeViewSuccessScenario(ctx context.Context, step *isucandar.BenchmarkStep, user *User, masterVersion string, login *Login) ScenarioResult {
	report := TimeReporter("ホーム画面表示シナリオ", s.Option)
	defer report()

	agent, err := s.GetAgentFromUser(step, user)
	if err != nil {
		return NoRewind()
	}

	now := time.Now()
	xIsuDate := time.Date(2022, 8, 27, now.Hour(), now.Minute(), now.Second(), now.Nanosecond(), myGMT)
	res, err := GetHome(ctx, agent, user.ID, masterVersion, xIsuDate, login)
	if err != nil {
		AddErrorIfNotCanceled(step, failure.NewError(ErrInvalidRequest, err))
		return NoRewind()
	}
	defer res.Body.Close()

	if res.StatusCode == http.StatusUnprocessableEntity {
		return Rewind()
	}

	validationResult := ValidateResponse(
		res,
		WithStatusCode(200),
	)
	validationResult.Add(step)

	if validationResult.IsEmpty() {
		step.AddScore(ScoreShowHomeViewSuccess)
	}

	return NoRewind()
}

// インゲーム報酬受け取りのシナリオを実行する。
// このシナリオの動作は加算対象になる。
func (s *Scenario) RedeemRewardSuccessScenario(ctx context.Context, step *isucandar.BenchmarkStep, user *User, masterVersion string, login *Login) ScenarioResult {
	report := TimeReporter("インゲーム報酬受け取りシナリオ", s.Option)
	defer report()

	agent, err := s.GetAgentFromUser(step, user)
	if err != nil {
		return NoRewind()
	}

	now := time.Now()
	xIsuDate := time.Date(2022, 8, 27, now.Hour(), now.Minute(), now.Second(), now.Nanosecond(), myGMT)
	res, err := PostReward(ctx, agent, user.ID, masterVersion, xIsuDate, login)
	if err != nil {
		AddErrorIfNotCanceled(step, failure.NewError(ErrInvalidRequest, err))
		return NoRewind()
	}
	defer res.Body.Close()

	if res.StatusCode == http.StatusUnprocessableEntity {
		return Rewind()
	}

	var rewardResponse = &validateRewardResponse{}

	validationResult := ValidateResponse(
		res,
		WithStatusCode(200),
		WithJsonBody(rewardResponse),
	)

	validationResult.Add(step)

	if validationResult.IsEmpty() {
		step.AddScore(ScoreRedeemRewardSuccess)
	}

	return NoRewind()
}

// プレゼント一覧画面表示〜プレゼント受け取り〜もう一度一覧画面表示をするシナリオを実行できる。
// これらの動作の成功はスコアの加算対象になる。
func (s *Scenario) AcceptGiftSuccessScenario(ctx context.Context, step *isucandar.BenchmarkStep, user *User, masterVersion string, login *Login) ScenarioResult {
	report := TimeReporter("プレゼント受け取りシナリオ", s.Option)
	defer report()

	agent, err := s.GetAgentFromUser(step, user)
	if err != nil {
		return NoRewind()
	}

	// ## プレゼント一覧画面の表示

	var listPresentResponse *ListPresentResponse

	{
		now := time.Now()
		xIsuDate := time.Date(2022, 8, 27, now.Hour(), now.Minute(), now.Second(), now.Nanosecond(), myGMT)
		res, err := GetPresentList(ctx, agent, user.ID, masterVersion, xIsuDate, login)
		if err != nil {
			AddErrorIfNotCanceled(step, failure.NewError(ErrInvalidRequest, err))
			return NoRewind()
		}
		defer res.Body.Close()

		if res.StatusCode == http.StatusUnprocessableEntity {
			return Rewind()
		}

		listPresentResponse = &ListPresentResponse{}

		validationResult := ValidateResponse(
			res,
			WithStatusCode(200),
			WithJsonBody(listPresentResponse),
		)
		validationResult.Add(step)

		if validationResult.IsEmpty() {
			step.AddScore(ScoreShowPresentListSuccess)
		}

		if checkIfContextOver(ctx) {
			return NoRewind()
		}
	}

	// ## プレゼント受け取り

	{
		presentIDs := []int64{}
		for _, present := range listPresentResponse.Presents {
			presentIDs = append(presentIDs, present.ID)
		}

		if len(presentIDs) == 0 {
			return NoRewind()
		}

		now := time.Now()
		xIsuDate := time.Date(2022, 8, 27, now.Hour(), now.Minute(), now.Second(), now.Nanosecond(), myGMT)
		res, err := PostReceivePresent(ctx, agent, user.ID, presentIDs, masterVersion, xIsuDate, login)
		if err != nil {
			AddErrorIfNotCanceled(step, failure.NewError(ErrInvalidRequest, err))
			return NoRewind()
		}
		defer res.Body.Close()

		if res.StatusCode == http.StatusUnprocessableEntity {
			return Rewind()
		}

		receivePresentResponse := &ReceivePresentResponse{}

		validationResult := ValidateResponse(
			res,
			WithStatusCode(200),
			WithJsonBody(receivePresentResponse),
		)
		validationResult.Add(step)

		if validationResult.IsEmpty() {
			step.AddScore(ScoreReceivePresentSuccess)
		}

	}

	return NoRewind()
}

// ガチャ一覧を閲覧し、ガチャを引く処理が成功するシナリオを実行する。これらの動作の成功はスコア加算になる。
func (s *Scenario) RedeemGachaSuccessScenario(ctx context.Context, step *isucandar.BenchmarkStep, user *User, masterVersion string, login *Login) ScenarioResult {
	report := TimeReporter("ガチャシナリオ", s.Option)
	defer report()

	agent, err := s.GetAgentFromUser(step, user)
	if err != nil {
		return NoRewind()
	}

	// ## ガチャ一覧画面表示

	var listGachaResponse *ListGachaResponse

	{
		now := time.Now()
		xIsuDate := time.Date(2022, 8, 27, now.Hour(), now.Minute(), now.Second(), now.Nanosecond(), myGMT)
		res, err := GetGachaList(ctx, agent, user.ID, masterVersion, xIsuDate, login)
		if err != nil {
			AddErrorIfNotCanceled(step, failure.NewError(ErrInvalidRequest, err))
			return NoRewind()
		}
		defer res.Body.Close()

		if res.StatusCode == http.StatusUnprocessableEntity {
			return Rewind()
		}

		listGachaResponse = &ListGachaResponse{}

		validationResult := ValidateResponse(
			res,
			WithStatusCode(200),
			WithJsonBody(listGachaResponse),
		)
		validationResult.Add(step)

		if validationResult.IsEmpty() {
			step.AddScore(ScoreShowGachaListSuccess)
		}

		if checkIfContextOver(ctx) {
			return NoRewind()
		}
	}

	// ## ガチャを引く

	gachaTypeID := 37
	oneTimeToken := listGachaResponse.OneTimeToken

	{
		now := time.Now()
		xIsuDate := time.Date(2022, 8, 27, now.Hour(), now.Minute(), now.Second(), now.Nanosecond(), myGMT)
		res, err := PostRedeemGacha(ctx, agent, user.ID, masterVersion, xIsuDate, gachaTypeID, oneTimeToken, login)
		if err != nil {
			AddErrorIfNotCanceled(step, failure.NewError(ErrInvalidRequest, err))
			return NoRewind()
		}
		defer res.Body.Close()

		if res.StatusCode == http.StatusUnprocessableEntity {
			return Rewind()
		}

		drawGachaResponse := &DrawGachaResponse{}

		validationResult := ValidateResponse(
			res,
			WithStatusCode(200),
			WithJsonBody(drawGachaResponse),
		)
		validationResult.Add(step)

		if validationResult.IsEmpty() {
			step.AddScore(ScoreRedeemGachaSuccess)
		}

	}

	return NoRewind()
}
