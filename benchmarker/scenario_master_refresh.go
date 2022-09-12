package main

import (
	"context"

	"github.com/isucon/isucandar"
	"github.com/isucon/isucandar/failure"
	"github.com/isucon/isucandar/worker"
)

// 30秒経過時に発火させるマスター更新バージョンアップのシナリオ。
func (s *Scenario) FireRefreshingMasterVersion(step *isucandar.BenchmarkStep) (*worker.Worker, error) {
	worker, err := worker.NewWorker(func(ctx context.Context, _ int) {
		// `MasterRefreshStartTime` 分だけ待つ。
		timeout, cancel := context.WithTimeout(ctx, MasterRefreshStartTime)
		defer cancel()

		<-timeout.Done()

		// 終わったらマスター更新シナリオを開始。
		refreshCtx, cancel := context.WithCancel(ctx)
		defer cancel()

		err := s.RefreshMasterDataScenario(refreshCtx, step, cancel)
		if err != nil {
			ContestantLogger.Printf("マスター更新に失敗しました。マスター更新に失敗した場合、失格となります。エラー: %v", err)
			AdminLogger.Printf("競技者がマスター更新に失敗しました。ベンチマーカーを終了します。エラー: %+v", err)
			step.AddError(failure.NewError(ErrCannotRefreshMasterVersion, err))
		}

		s.AdminUser.ClearAgent()

	}, worker.WithLoopCount(1), worker.WithMaxParallelism(1))

	worker.SetParallelism(1)
	return worker, err
}

// 最新のマスターバージョンを取得する。マスターバージョンはシナリオの最中に書き換えが発生する。
func (s *Scenario) LatestMasterVersion() string {
	var masterVersion string
	s.mu.RLock()
	defer s.mu.RUnlock()
	masterVersion = s.MasterVersion
	return masterVersion
}

// ベンチマーカーのシナリオ全体で管理するマスターバージョンを指定したバージョンに更新する。
func (s *Scenario) UpdateMasterVersion(masterVersion string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.MasterVersion = masterVersion
}

// マスターデータを更新するシナリオ。
// マスターデータの更新が走ると、サーバー側でマスター情報のバージョンが上げられる。
// 更新以降は古かったマスターバージョンは無効判定になる。
// シナリオは下記の通り。
// 1. 管理者ユーザーのログインを実施
// 2. マスター更新を実施
// 3. 管理者ユーザーのログアウトを実施
func (s *Scenario) RefreshMasterDataScenario(ctx context.Context, step *isucandar.BenchmarkStep, cancel context.CancelFunc) error {
	PrintScenarioStarted(ScenarioMasterRefresh)
	defer PrintScenarioFinished(ScenarioMasterRefresh)

	report := TimeReporter("マスター更新シナリオ", s.Option)
	defer report()

	login, err := s.LoginAdminUser(ctx, step)
	if err != nil {
		return err
	}

	err = s.RefreshMasterData(ctx, step, login)
	if err != nil {
		return err
	}

	err = s.LogoutAdminUser(ctx, step, login)
	if err != nil {
		return err
	}

	return nil
}

// 管理者ユーザーでログインするシナリオ。
func (s *Scenario) LoginAdminUser(ctx context.Context, step *isucandar.BenchmarkStep) (*Login, error) {
	agent, err := s.AdminUser.GetAgent(s.Option)
	if err != nil {
		return nil, err
	}

	res, err := PostAdminLoginAction(ctx, agent, s.AdminUser, s.LatestMasterVersion())
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	adminLoginResponse := &AdminLoginResponse{}

	adminLoginValidation := ValidateResponse(
		res,
		WithStatusCode(200),
		WithJsonBody(adminLoginResponse),
	)
	adminLoginValidation.Add(step)

	if adminLoginValidation.IsEmpty() {
		login := &Login{
			SessionID: adminLoginResponse.Session.SessionID,
		}
		return login, nil
	} else {
		return nil, adminLoginValidation
	}
}

// マスターデータを更新するシナリオ。
// マスターデータ更新用のエンドポイントにアクセスし、更新できたら更新結果をレスポンスで取得する。
// レスポンスで取得したバージョンをScenario全体で管理するバージョンに設定する。
func (s *Scenario) RefreshMasterData(ctx context.Context, step *isucandar.BenchmarkStep, login *Login) error {
	agent, err := s.AdminUser.GetAgent(s.Option)
	if err != nil {
		return err
	}

	res, err := PutRefreshMasterData(ctx, agent, login.SessionID, s.LatestMasterVersion())
	if err != nil {
		return err
	}
	defer res.Body.Close()

	masterDataResponse := &MasterDataResponse{}

	masterDataValidation := ValidateResponse(
		res,
		WithStatusCode(200),
		WithJsonBody(masterDataResponse),
	)
	masterDataValidation.Add(step)

	if masterDataValidation.IsEmpty() {
		s.UpdateMasterVersion(masterDataResponse.VersionMaster.MasterVersion)
		ContestantLogger.Printf("マスターデータの更新が完了しました。次のマスターバージョンは「%s」です。\n", s.MasterVersion)
	} else {
		return masterDataValidation
	}

	return nil
}

// 管理者ユーザーをログアウトさせるシナリオ。
func (s *Scenario) LogoutAdminUser(ctx context.Context, step *isucandar.BenchmarkStep, login *Login) error {
	agent, err := s.AdminUser.GetAgent(s.Option)
	if err != nil {
		return err
	}

	res, err := DeleteAdminLogoutAction(ctx, agent, login.SessionID, s.LatestMasterVersion())
	if err != nil {
		return err
	}
	defer res.Body.Close()

	adminLogoutValidation := ValidateResponse(
		res,
		WithStatusCode(204),
	)

	adminLogoutValidation.Add(step)

	if !adminLogoutValidation.IsEmpty() {
		return adminLogoutValidation
	}

	return nil
}
