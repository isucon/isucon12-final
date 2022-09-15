package main

// ユーザー登録のシナリオを一通り管理するファイル。

import (
	"context"
	"net/http"
	"time"

	"github.com/isucon/isucandar"
	"github.com/isucon/isucandar/failure"
	"github.com/isucon/isucandar/worker"
)

// ユーザー登録のシナリオを実行する。ユーザー登録後のホーム画面表示やプレゼント受け取り、ガチャ処理はログインシナリオ側のものをそのまま再利用している。
func (s *Scenario) NewUserRegistrationScenarioWorker(step *isucandar.BenchmarkStep, p int32) (*worker.Worker, error) {
	userRegistration, err := worker.NewWorker(func(ctx context.Context, _ int) {
		PrintScenarioStarted(ScenarioUserRegistration)
		defer PrintScenarioFinished(ScenarioUserRegistration)

		var sucess bool
		var platform *Platform
		if s.Option.Stage == "test" {
			platform = &Platform{
				ID:   12345678,
				Type: 1,
			}
		} else {
			platform, sucess = s.Platforms.Pop()
			if !sucess {
				return
			}
		}

	Rewind:

		// ワーカー開始時点での最新のマスターバージョンを取得する。
		masterVersion := s.LatestMasterVersion()

		// 1. 新規ユーザー登録
		result, userCreated, err := s.UserRegistrationScenario(ctx, step, platform, masterVersion)
		if err != nil {
			return
		}
		if result.Rewind {
			goto Rewind
		}

		s.RecordUserRegistrationCount(1)

		// 受け取ったユーザー ID を元にユーザーを一時的に作り、その Agent 等を利用する
		user := &User{
			ID: userCreated.UserID,
		}
		login := &Login{
			SessionID: userCreated.SessionID,
			ViewerID:  userCreated.ViewerID,
		}

		// 2. ログイン情報を使ってホーム画面を表示
		result = s.ShowHomeViewSuccessScenario(ctx, step, user, masterVersion, login)
		if result.Rewind {
			goto Rewind
		}
		// 3. プレゼント受け取り
		result = s.AcceptGiftSuccessScenario(ctx, step, user, masterVersion, login)
		if result.Rewind {
			goto Rewind
		}

		// 4. ガチャを引く
		result = s.RedeemGachaSuccessScenario(ctx, step, user, masterVersion, login)
		if result.Rewind {
			goto Rewind
		}

		// 4. プレゼント受け取り
		result = s.AcceptGiftSuccessScenario(ctx, step, user, masterVersion, login)
		if result.Rewind {
			goto Rewind
		}

		// 5. itemリスト
		var userItems []UserItem
		var userCards []UserCard
		result, oneTimeToken, err := s.ShowItemListSuccessScenario(ctx, step, user, masterVersion, login, &userItems, &userCards)
		if err != nil {
			return
		}
		if result.Rewind {
			goto Rewind
		}

		// 6. カード強化
		result, err = s.postAddExpCardIDSuccessScenario(ctx, step, user, masterVersion, login, oneTimeToken, userItems, userCards)
		if err != nil {
			return
		}
		if result.Rewind {
			goto Rewind
		}

		// 7. 装備
		result, err = s.postCardSuccessScenario(ctx, step, user, masterVersion, login, userCards)
		if err != nil {
			return
		}
		if result.Rewind {
			goto Rewind
		}
		platform.ClearAgent()
		user.ClearAgent()
	}, loopConfig(s), parallelismConfig(s))

	userRegistration.SetParallelism(p)

	return userRegistration, err
}

// ユーザー作成のシナリオ。このシナリオステップは成功しないと次のシナリオステップに進めないので、作成時のバリデーションに失敗したらそこで処理を落とす。
func (s *Scenario) UserRegistrationScenario(ctx context.Context, step *isucandar.BenchmarkStep, platform *Platform, masterVersion string) (ScenarioResult, *UserCreated, error) {
	report := TimeReporter("ユーザー登録シナリオ", s.Option)
	defer report()

	agent, err := platform.GetAgent(s.Option)
	if err != nil {
		step.AddError(failure.NewError(ErrCannotCreateNewAgent, err))
		return NoRewind(), nil, err
	}

	now := time.Now()
	xIsuDate := time.Date(2022, 8, 27, now.Hour(), now.Minute(), now.Second(), now.Nanosecond(), myGMT)
	res, err := PostUserAction(ctx, agent, platform, masterVersion, xIsuDate)
	if err != nil {
		AddErrorIfNotCanceled(step, failure.NewError(ErrInvalidRequest, err))
		return NoRewind(), nil, err
	}
	defer res.Body.Close()

	if res.StatusCode == http.StatusUnprocessableEntity {
		return Rewind(), nil, nil
	}

	createUserResponse := &CreateUserResponse{}

	createUserValidation := ValidateResponse(
		res,
		WithStatusCode(200),
		WithJsonBody(createUserResponse),
	)
	createUserValidation.Add(step)

	if createUserValidation.IsEmpty() {
		step.AddScore(ScoreCreateUser)
		return NoRewind(), &UserCreated{
			SessionID: createUserResponse.SessionID,
			ViewerID:  createUserResponse.ViewerId,
			UserID:    createUserResponse.UserID,
		}, nil
	} else {
		return NoRewind(), nil, createUserValidation
	}
}

func (s *Scenario) ShowItemListSuccessScenario(ctx context.Context, step *isucandar.BenchmarkStep, user *User, masterVersion string, login *Login, userItems *[]UserItem, userCards *[]UserCard) (ScenarioResult, string, error) {
	report := TimeReporter("item表示 シナリオ", s.Option)
	defer report()

	agent, err := s.GetAgentFromUser(step, user)
	if err != nil {
		return NoRewind(), "", err
	}

	now := time.Now()
	xIsuDate := time.Date(2022, 8, 27, now.Hour(), now.Minute(), now.Second(), now.Nanosecond(), myGMT)
	res, err := GetItemList(ctx, agent, user.ID, masterVersion, xIsuDate, login)
	if err != nil {
		AddErrorIfNotCanceled(step, failure.NewError(ErrInvalidRequest, err))
		return NoRewind(), "", err
	}
	defer res.Body.Close()

	if res.StatusCode == http.StatusUnprocessableEntity {
		return Rewind(), "", nil
	}

	showItemListResponse := &validateItemListResponse{}

	validationResult := ValidateResponse(
		res,
		WithStatusCode(200),
		WithJsonBody(showItemListResponse),
	)

	oneTimeToken := showItemListResponse.OneTimeToken

	*userCards = showItemListResponse.UserCards
	*userItems = showItemListResponse.UserItems

	validationResult.Add(step)

	if validationResult.IsEmpty() {
		step.AddScore(ScoreShowItems)
		return NoRewind(), oneTimeToken, nil
	} else {
		return NoRewind(), "", validationResult
	}
}

// card経験値アップ
func (s *Scenario) postAddExpCardIDSuccessScenario(ctx context.Context, step *isucandar.BenchmarkStep, user *User, masterVersion string, login *Login, oneTimeToken string, userItems []UserItem, userCards []UserCard) (ScenarioResult, error) {
	report := TimeReporter("Card経験値アップ シナリオ", s.Option)
	defer report()

	agent, err := s.GetAgentFromUser(step, user)
	if err != nil {
		return NoRewind(), err
	}

	//カードとアイテムがなければ中断
	if len(userItems) == 0 || len(userCards) == 0 {
		return NoRewind(), nil
	}

	//cardIDをきめる
	cardID := userCards[0].ID
	//itemsをきめるitemType=3のもの
	var item UserItem
	isExist := false
	for _, v := range userItems {
		if v.ItemType == 3 {
			isExist = true
			item = v
			break
		}
	}
	//強化アイテムがなければ中断
	if !isExist {
		return NoRewind(), nil
	}
	addCardExpItem := []AddCardExpItem{
		{
			ID:     item.ID,
			Amount: 1,
		},
	}

	now := time.Now()
	xIsuDate := time.Date(2022, 8, 27, now.Hour(), now.Minute(), now.Second(), now.Nanosecond(), myGMT)
	res, err := PostAddExpCard(ctx, agent, user.ID, cardID, addCardExpItem, oneTimeToken, masterVersion, xIsuDate, login)
	if err != nil {
		AddErrorIfNotCanceled(step, failure.NewError(ErrInvalidRequest, err))
		return NoRewind(), err
	}
	defer res.Body.Close()

	if res.StatusCode == http.StatusUnprocessableEntity {
		return Rewind(), err
	}

	postAddExpCardResponse := &validatePostAddExpCardResponse{}

	validationResult := ValidateResponse(
		res,
		WithStatusCode(200),
		WithJsonBody(postAddExpCardResponse),
	)

	validationResult.Add(step)

	if validationResult.IsEmpty() {
		step.AddScore(ScorePostAddCard)
		return NoRewind(), nil
	} else {
		return NoRewind(), validationResult
	}
}

// deck変更　整合性チェック
func (s *Scenario) postCardSuccessScenario(ctx context.Context, step *isucandar.BenchmarkStep, user *User, masterVersion string, login *Login, userCards []UserCard) (ScenarioResult, error) {
	report := TimeReporter("deck変更 シナリオ", s.Option)
	defer report()

	agent, err := s.GetAgentFromUser(step, user)
	if err != nil {
		return NoRewind(), err
	}

	if len(userCards) < 3 {
		return NoRewind(), nil
	}
	userCardIDs := make([]int64, 3)
	userCardIDs[0] = userCards[0].ID
	userCardIDs[1] = userCards[1].ID
	userCardIDs[2] = userCards[2].ID

	now := time.Now()
	xIsuDate := time.Date(2022, 8, 27, now.Hour(), now.Minute(), now.Second(), now.Nanosecond(), myGMT)
	res, err := PostCard(ctx, agent, user.ID, userCardIDs, masterVersion, xIsuDate, login)
	if err != nil {
		AddErrorIfNotCanceled(step, failure.NewError(ErrInvalidRequest, err))
		return NoRewind(), err
	}
	defer res.Body.Close()

	if res.StatusCode == http.StatusUnprocessableEntity {
		return Rewind(), err
	}

	postCardResponse := &validatePostCardResponse{}

	validationResult := ValidateResponse(
		res,
		WithStatusCode(200),
		WithJsonBody(postCardResponse),
	)

	validationResult.Add(step)

	if validationResult.IsEmpty() {
		step.AddScore(ScoreSetDeck)
		return NoRewind(), nil
	} else {
		return NoRewind(), validationResult
	}
}
