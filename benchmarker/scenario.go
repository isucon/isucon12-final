package main

import (
	"context"
	"sync"

	"github.com/isucon/isucandar"
	"github.com/isucon/isucandar/failure"
	"github.com/isucon/isucandar/worker"
	"github.com/isucon/isucon12-final/benchmarker/data"
)

type Scenario struct {
	mu sync.RWMutex

	Option Option
	// 競技者が使用した言語。ポータルへのレポーティングで使用される。
	Language string

	Users         data.Set[*User]
	ValidateUsers data.Set[*ValidationUser]
	Platforms     data.Set[*Platform]
	BanUsers      data.Set[*User]

	AdminUser     *AdminUser
	MasterVersion string

	ConsumedUserIDs *data.LightSet

	ScenarioControlWg sync.WaitGroup
	LoginCountMu      sync.Mutex
	LoginSuccessCount int

	UserRegistrationMu    sync.Mutex
	UserRegistrationCount int
}

// 初期化処理を行うが、初期化処理を正しく実行しているかをチェックする。
// 初期化処理自体は `main.DefaultInitializeRequestTimeout` 秒以内に終了する必要がある。
func (s *Scenario) Prepare(ctx context.Context, step *isucandar.BenchmarkStep) error {
	err := s.LoadInitialData()
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(ctx, s.Option.InitializeRequestTimeout)
	defer cancel()

	report := TimeReporter("初期化処理", s.Option)
	defer report()

	agent, err := s.Option.NewAgent(true)
	if err != nil {
		return failure.NewError(ErrCannotCreateNewAgent, err)
	}

	// 管理者ユーザーをシナリオに登録しておき、マスターデータ更新で利用する。
	// 管理者 ID は決め打ちで1が入る。
	s.AdminUser = &AdminUser{
		ID:       LoginAdminID,
		Password: LoginAdminPassword,
	}
	s.MasterVersion = "1"

	err = s.DoInitialize(ctx, step, agent)
	if err != nil {
		return err
	}

	// 検証シナリオを1回まわす
	if err := s.ValidationScenario(ctx, step); err != nil {
		ContestantLogger.Println("整合性チェックに失敗しました")
		return err
	}

	ContestantLogger.Println("整合性チェックに成功しました！")
	ContestantLogger.Println("初期化処理が成功しました！")

	return nil
}

// ISUCON 12関係者向け: 負荷フェーズの細かい処理段階や成功条件については、下記ドキュメントに意図などをまとめているので参照のこと。
// See: https://scrapbox.io/ISUCON12/%E3%83%99%E3%83%B3%E3%83%81%E3%83%9E%E3%83%BC%E3%82%AB%E3%83%BC%E3%83%86%E3%82%B9%E3%83%88%E3%82%B7%E3%83%8A%E3%83%AA%E3%82%AA
//
// 主なシナリオとしては次の通り。
// 1. ログインシナリオ
// 2. ユーザー登録シナリオ
// 3. マスター更新シナリオ
func (s *Scenario) Load(ctx context.Context, step *isucandar.BenchmarkStep) error {
	if s.Option.PrepareOnly {
		return nil
	}

	ContestantLogger.Println("アプリケーションへの負荷走行を開始します")

	wg := &sync.WaitGroup{}

	// 各シナリオを走らせる。
	loginSuccess, err := s.NewLoginSuccessScenarioWorker(step, 1)
	if err != nil {
		return err
	}

	userRegistration, err := s.NewUserRegistrationScenarioWorker(step, 1)
	if err != nil {
		return err
	}

	banUserLogin, err := s.NewBanUserLoginScenarioWorker(step, 1)
	if err != nil {
		return err
	}

	masterRefresh, err := s.FireRefreshingMasterVersion(step)
	if err != nil {
		return err
	}

	workers := []*worker.Worker{
		loginSuccess,
		userRegistration,
		banUserLogin,
		masterRefresh,
	}

	for _, w := range workers {
		wg.Add(1)
		worker := w
		go func() {
			defer wg.Done()
			worker.Process(ctx)
		}()
	}

	// ベンチマーカー走行中の負荷調整を10秒ごとにかける。
	wg.Add(1)
	go func() {
		defer wg.Done()
		s.loadAdjustor(ctx, step, loginSuccess, userRegistration, banUserLogin)
	}()

	wg.Wait()
	s.ScenarioControlWg.Wait()

	ContestantLogger.Println("負荷走行がすべて終了しました")
	AdminLogger.Println("負荷走行がすべて終了しました")

	return nil
}

func (s *Scenario) Validation(ctx context.Context, step *isucandar.BenchmarkStep) error {
	if s.Option.PrepareOnly {
		return nil
	}

	return nil
}
