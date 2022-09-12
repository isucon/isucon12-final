package main

import (
	"context"
	"math/rand"
	"strconv"
	"time"

	"github.com/isucon/isucandar"
	"github.com/isucon/isucandar/failure"
)

var (
	myGMT *time.Location
)

func init() {
	myGMT = time.FixedZone("GMT", 0)
}

func generateFailUserID() int64 {
	min := int64(10000)
	return rand.Int63n(min) + 1
}

func generateFailViewerID() string {
	min := int(100000)
	return strconv.Itoa(rand.Intn(min) + 10)
}

func (s *Scenario) loginValidateFailScenario(ctx context.Context, step *isucandar.BenchmarkStep, user *ValidationUser, userID int64, viewerID string, masterVersion string) error {
	report := TimeReporter("ログイン失敗 整合性チェック", s.Option)
	defer report()
	agent, err := s.GetAgentFromUser(step, &User{
		ID:       user.ID,
		UserType: user.UserType,
		ViewerID: user.ViewerID,
		Agent:    user.Agent,
	})
	if err != nil {
		return err
	}

	now := time.Now()
	xIsuDate := time.Date(2022, 8, 27, now.Hour(), now.Minute(), now.Second(), now.Nanosecond(), myGMT)

	loginRes, err := PostLoginAction(ctx, agent, userID, viewerID, masterVersion, xIsuDate)

	if err != nil {
		return failure.NewError(ValidationErrInvalidRequest, err)
	}
	defer loginRes.Body.Close()

	validateLoginFailResponse := &validateFailResponse{}

	loginFailValidation := ValidateResponse(
		loginRes,
		validateLoginFailBody(validateLoginFailResponse),
	)

	loginFailValidation.Add(step)

	if loginFailValidation.IsEmpty() {
		return nil
	} else {
		return loginFailValidation
	}
}

func (s *Scenario) postUserValidateSuccessScenario(ctx context.Context, step *isucandar.BenchmarkStep, platformUser *Platform, masterVersion string, loginBonusRewardMasters []LoginBonusRewardMaster, presentAllMasters []PresentAllMaster) (*Login, *JsonUser, error) {
	report := TimeReporter("ユーザ登録成功 整合性チェック", s.Option)
	defer report()

	agent, err := platformUser.GetAgent(s.Option)
	if err != nil {
		return nil, nil, failure.NewError(ValidationErrInvalidRequest, err)
	}

	now := time.Now()
	xIsuDate := time.Date(2022, 8, 27, now.Hour(), now.Minute(), now.Second(), now.Nanosecond(), myGMT)
	res, err := PostUserAction(ctx, agent, platformUser, masterVersion, xIsuDate)
	if err != nil {
		return nil, nil, failure.NewError(ValidationErrInvalidRequest, err)
	}
	defer res.Body.Close()

	validatePostUserRes := &validatePostUserResponse{}
	user := &JsonUser{}

	postUserValidation := ValidateResponse(
		res,
		validatePostUser(validatePostUserRes, platformUser, user, loginBonusRewardMasters, presentAllMasters, xIsuDate),
	)
	postUserValidation.Add(step)

	if postUserValidation.IsEmpty() {

		login := Login{
			SessionID: validatePostUserRes.SessionID,
			ViewerID:  validatePostUserRes.ViewerID,
		}
		return &login, user, nil
	} else {
		return nil, user, postUserValidation
	}

}

func (s *Scenario) loginValidateAfterDaySuccessScenario(ctx context.Context, step *isucandar.BenchmarkStep, user *ValidationUser, masterVersion string, loginBonusRewardMasters []LoginBonusRewardMaster) error {
	report := TimeReporter("ログインボーナス日跨ぎ後確認 整合性チェック", s.Option)
	defer report()

	agent, err := s.GetAgentFromUser(step, &User{
		ID:       user.ID,
		UserType: user.UserType,
		ViewerID: user.ViewerID,
		Agent:    user.Agent,
	})
	if err != nil {
		return err
	}
	now := time.Date(2022, 8, 26, 15, 0, 01, 0, myGMT)
	loginRes, err := PostLoginAction(ctx, agent, user.ID, user.ViewerID, masterVersion, now)
	if err != nil {
		return failure.NewError(ValidationErrInvalidRequest, err)

	}
	defer loginRes.Body.Close()

	validateLoginResponse := &validateLoginResponse{}

	loginValidation := ValidateResponse(
		loginRes,
		validateLoginSecondUser(validateLoginResponse, user, now, loginBonusRewardMasters),
	)
	loginValidation.Add(step)

	if loginValidation.IsEmpty() {
		return nil
	} else {
		return loginValidation
	}
}

func (s *Scenario) loginValidateBeforDaySuccessScenario(ctx context.Context, step *isucandar.BenchmarkStep, user *ValidationUser, masterVersion string, loginBonusRewardMasters []LoginBonusRewardMaster) error {
	report := TimeReporter("ログインボーナス確認 整合性チェック", s.Option)
	defer report()

	agent, err := s.GetAgentFromUser(step, &User{
		ID:       user.ID,
		UserType: user.UserType,
		ViewerID: user.ViewerID,
		Agent:    user.Agent,
	})
	if err != nil {
		return err
	}
	now := time.Date(2022, 8, 26, 14, 59, 59, 0, myGMT)
	loginRes, err := PostLoginAction(ctx, agent, user.ID, user.ViewerID, masterVersion, now)
	if err != nil {
		return failure.NewError(ValidationErrInvalidRequest, err)

	}
	defer loginRes.Body.Close()

	validateLoginResponse := &validateLoginResponse{}

	loginValidation := ValidateResponse(
		loginRes,
		validateLoginUser(validateLoginResponse, user, now, loginBonusRewardMasters),
	)
	loginValidation.Add(step)

	if loginValidation.IsEmpty() {
		return nil
	} else {
		return loginValidation
	}
}

func (s *Scenario) loginValidateSuccessScenario(ctx context.Context, step *isucandar.BenchmarkStep, user *ValidationUser, masterVersion string, loginBonusRewardMasters []LoginBonusRewardMaster) (*Login, error) {
	report := TimeReporter("ログイン成功 整合性チェック", s.Option)
	defer report()

	agent, err := s.GetAgentFromUser(step, &User{
		ID:       user.ID,
		UserType: user.UserType,
		ViewerID: user.ViewerID,
		Agent:    user.Agent,
	})
	if err != nil {
		return nil, err
	}

	now := time.Now()
	xIsuDate := time.Date(2022, 8, 27, now.Hour(), now.Minute(), now.Second(), now.Nanosecond(), myGMT)
	loginRes, err := PostLoginAction(ctx, agent, user.ID, user.ViewerID, masterVersion, xIsuDate)
	if err != nil {
		step.AddError(failure.NewError(ValidationErrInvalidRequest, err))
		return nil, err
	}
	defer loginRes.Body.Close()

	validateLoginResponse := &validateLoginResponse{}

	loginValidation := ValidateResponse(
		loginRes,
		validateLoginUser(validateLoginResponse, user, xIsuDate, loginBonusRewardMasters),
	)
	loginValidation.Add(step)

	if loginValidation.IsEmpty() {

		login := Login{
			SessionID: validateLoginResponse.SessionID,
			ViewerID:  validateLoginResponse.ViewerID,
		}
		return &login, nil
	} else {
		return nil, loginValidation
	}
}

func (s *Scenario) loginBanValidateScenario(ctx context.Context, step *isucandar.BenchmarkStep, user *User, masterVersion string) error {
	report := TimeReporter("ログインBan 整合性チェック", s.Option)
	defer report()

	agent, err := s.GetAgentFromUser(step, &User{
		ID:       user.ID,
		UserType: user.UserType,
		ViewerID: user.ViewerID,
		Agent:    user.Agent,
	})
	if err != nil {
		return err
	}

	now := time.Now()
	xIsuDate := time.Date(2022, 8, 27, now.Hour(), now.Minute(), now.Second(), now.Nanosecond(), myGMT)
	loginRes, err := PostLoginAction(ctx, agent, user.ID, user.ViewerID, masterVersion, xIsuDate)
	if err != nil {
		return failure.NewError(ValidationErrInvalidRequest, err)
	}
	defer loginRes.Body.Close()

	validateLoginBanResponse := &validateFailResponse{}

	loginBanValidation := ValidateResponse(
		loginRes,
		validateLoginBan(validateLoginBanResponse, user),
	)
	loginBanValidation.Add(step)

	if loginBanValidation.IsEmpty() {
		return nil
	} else {
		return loginBanValidation
	}
}

func (s *Scenario) ShowHomeValidateFailScenario(ctx context.Context, step *isucandar.BenchmarkStep, user *ValidationUser, masterVersion string) error {
	report := TimeReporter("ホーム画面表示失敗 整合性チェック", s.Option)
	defer report()

	agent, err := s.GetAgentFromUser(step, &User{
		ID:       user.ID,
		UserType: user.UserType,
		ViewerID: user.ViewerID,
		Agent:    user.Agent,
	})
	if err != nil {
		return err
	}
	login := &Login{
		SessionID: "failsession",
		ViewerID:  user.ViewerID,
	}

	now := time.Now()
	xIsuDate := time.Date(2022, 8, 27, now.Hour(), now.Minute(), now.Second(), now.Nanosecond(), myGMT)
	res, err := GetHome(ctx, agent, user.ID, masterVersion, xIsuDate, login)
	if err != nil {
		return failure.NewError(ValidationErrInvalidRequest, err)
	}
	defer res.Body.Close()

	validateHomeFailRes := &validateFailResponse{}

	homeFailValidation := ValidateResponse(
		res,
		validateHomeFail(validateHomeFailRes, user),
	)
	homeFailValidation.Add(step)

	if homeFailValidation.IsEmpty() {
		return nil
	} else {
		return homeFailValidation
	}
}

// ホーム画面表示　整合性チェック
func (s *Scenario) ShowHomeValidateSuccessScenario(ctx context.Context, step *isucandar.BenchmarkStep, user *ValidationUser, masterVersion string, login *Login, loginBonusRewardMasters []LoginBonusRewardMaster) (*LoginBonusRewardMaster, error) {
	report := TimeReporter("ホーム画面表示 整合性チェック", s.Option)
	defer report()

	loginRewardItem := &LoginBonusRewardMaster{}

	agent, err := s.GetAgentFromUser(step, &User{
		ID:       user.ID,
		UserType: user.UserType,
		ViewerID: user.ViewerID,
		Agent:    user.Agent,
	})
	if err != nil {
		return loginRewardItem, err
	}

	now := time.Now()
	xIsuDate := time.Date(2022, 8, 27, now.Hour(), now.Minute(), now.Second(), now.Nanosecond(), myGMT)
	res, err := GetHome(ctx, agent, user.ID, masterVersion, xIsuDate, login)
	if err != nil {
		return loginRewardItem, failure.NewError(ValidationErrInvalidRequest, err)
	}
	defer res.Body.Close()

	validateHomeResponse := &validateHomeResponse{}

	validationResult := ValidateResponse(
		res,
		validateHomeResponseCheck(validateHomeResponse, user, loginBonusRewardMasters, loginRewardItem, xIsuDate),
	)

	validationResult.Add(step)

	if validationResult.IsEmpty() {
		return loginRewardItem, nil
	} else {
		return loginRewardItem, validationResult
	}
}

// 他人のセッションでホーム表示
func (s *Scenario) ShowOhterHomeValidateFailScenario(ctx context.Context, step *isucandar.BenchmarkStep, user *ValidationUser, masterVersion string, login *Login, postUserLogin *Login, postUser *JsonUser) error {
	report := TimeReporter("他人ホーム画面表示失敗 整合性チェック", s.Option)
	defer report()

	agentUser := &User{
		ID: postUser.ID,
	}

	agent, err := s.GetAgentFromUser(step, agentUser)
	if err != nil {
		return err
	}

	now := time.Now()
	xIsuDate := time.Date(2022, 8, 27, now.Hour(), now.Minute(), now.Second(), now.Nanosecond(), myGMT)
	res, err := GetHome(ctx, agent, user.ID, masterVersion, xIsuDate, postUserLogin)
	if err != nil {
		return failure.NewError(ValidationErrInvalidRequest, err)
	}
	defer res.Body.Close()

	validateHomeFailRes := &validateFailResponse{}

	homeFailValidation := ValidateResponse(
		res,
		validateOtherLogin(validateHomeFailRes, postUser),
	)
	homeFailValidation.Add(step)

	if homeFailValidation.IsEmpty() {
		return nil
	} else {
		return homeFailValidation
	}

}

// Reward受けとり　整合性チェック
func (s *Scenario) postRewardValidateSuccessScenario(ctx context.Context, step *isucandar.BenchmarkStep, user *ValidationUser, masterVersion string, login *Login) error {
	report := TimeReporter("Reward受け取り 整合性チェック", s.Option)
	defer report()

	agent, err := s.GetAgentFromUser(step, &User{
		ID:       user.ID,
		UserType: user.UserType,
		ViewerID: user.ViewerID,
		Agent:    user.Agent,
	})
	if err != nil {
		return err
	}

	now := time.Now()
	xIsuDate := time.Date(2022, 8, 27, now.Hour(), now.Minute(), now.Second(), now.Nanosecond(), myGMT)
	res, err := PostReward(ctx, agent, user.ID, masterVersion, xIsuDate, login)
	if err != nil {
		return failure.NewError(ValidationErrInvalidRequest, err)
	}
	defer res.Body.Close()

	validateRewardResponse := &validateRewardResponse{}

	validationResult := ValidateResponse(
		res,
		validateRewardResponseBody(validateRewardResponse, user, xIsuDate),
	)

	validationResult.Add(step)

	if validationResult.IsEmpty() {
		return nil
	} else {
		return validationResult
	}
}

// item表示　整合性チェック
func (s *Scenario) ShowItemValidateSuccessScenario(ctx context.Context, step *isucandar.BenchmarkStep, user *ValidationUser, masterVersion string, login *Login, loginRewardItem *LoginBonusRewardMaster) (string, error) {
	report := TimeReporter("item表示 整合性チェック", s.Option)
	defer report()

	agent, err := s.GetAgentFromUser(step, &User{
		ID:       user.ID,
		UserType: user.UserType,
		ViewerID: user.ViewerID,
		Agent:    user.Agent,
	})
	if err != nil {
		return "", err
	}

	now := time.Now()
	xIsuDate := time.Date(2022, 8, 27, now.Hour(), now.Minute(), now.Second(), now.Nanosecond(), myGMT)
	res, err := GetItemList(ctx, agent, user.ID, masterVersion, xIsuDate, login)
	if err != nil {
		step.AddError(failure.NewError(ValidationErrInvalidRequest, err))
		return "", err
	}
	defer res.Body.Close()

	validateItemListResponse := &validateItemListResponse{}

	validationResult := ValidateResponse(
		res,
		validateItemListResponseBody(validateItemListResponse, user, loginRewardItem, xIsuDate),
	)

	oneTimeToken := validateItemListResponse.OneTimeToken

	validationResult.Add(step)

	if validationResult.IsEmpty() {
		return oneTimeToken, nil
	} else {
		return "", validationResult
	}
}

// deck変更　整合性チェック
func (s *Scenario) postCardValidateSuccessScenario(ctx context.Context, step *isucandar.BenchmarkStep, user *ValidationUser, masterVersion string, login *Login) error {
	report := TimeReporter("deck変更 整合性チェック", s.Option)
	defer report()

	agent, err := s.GetAgentFromUser(step, &User{
		ID:       user.ID,
		UserType: user.UserType,
		ViewerID: user.ViewerID,
		Agent:    user.Agent,
	})
	if err != nil {
		return err
	}
	// Cardを３枚 うしろから３つ取得
	var validateUserCardIDs []int64
	for i := 1; i <= 3; i++ {
		validateUserCardIDs = append(validateUserCardIDs, user.UserCards[len(user.UserCards)-i].ID)
	}

	now := time.Now()
	xIsuDate := time.Date(2022, 8, 27, now.Hour(), now.Minute(), now.Second(), now.Nanosecond(), myGMT)
	res, err := PostCard(ctx, agent, user.ID, validateUserCardIDs, masterVersion, xIsuDate, login)
	if err != nil {
		return failure.NewError(ValidationErrInvalidRequest, err)
	}
	defer res.Body.Close()

	validatePostCardResponse := &validatePostCardResponse{}

	validationResult := ValidateResponse(
		res,
		validatePostCardResponseBody(validatePostCardResponse, user, validateUserCardIDs, xIsuDate),
	)

	validationResult.Add(step)

	if !validationResult.IsEmpty() {
		return validationResult
	}

	// homeのレスポンスが更新されているかチェック
	now = time.Now()
	xIsuDate = time.Date(2022, 8, 27, now.Hour(), now.Minute(), now.Second(), now.Nanosecond(), myGMT)
	res, err = GetHome(ctx, agent, user.ID, masterVersion, xIsuDate, login)
	if err != nil {
		return failure.NewError(ValidationErrInvalidRequest, err)
	}
	defer res.Body.Close()

	validateHomeResponse := &validateHomeResponse{}
	validationResult = ValidateResponse(
		res,
		WithStatusCode(200),
		validatePostCardAfterHomeResponseBody(validateHomeResponse, user, validateUserCardIDs, xIsuDate),
	)

	validationResult.Add(step)

	if validationResult.IsEmpty() {
		return nil
	} else {
		return validationResult
	}
}

// card経験値アップ
func (s *Scenario) postAddExpCardIDValidateSuccessScenario(ctx context.Context, step *isucandar.BenchmarkStep, user *ValidationUser, masterVersion string, login *Login, oneTimeToken string, expItemMasters []localExpItemMaster, cardMasters CardMasters, loginRewardItem *LoginBonusRewardMaster) error {
	report := TimeReporter("Card経験値アップ 整合性チェック", s.Option)
	defer report()

	agent, err := s.GetAgentFromUser(step, &User{
		ID:       user.ID,
		UserType: user.UserType,
		ViewerID: user.ViewerID,
		Agent:    user.Agent,
	})
	if err != nil {
		return err
	}

	//cardIDをきめる
	cardID := user.UserCards[len(user.UserCards)-1].ID
	//itemsをきめるitemType=3のもの
	var items []UserItem
	for _, v := range user.GetItemList {
		if v.ItemType == 3 {
			items = append(items, v)
			break
		}
	}
	addCardExpItem := []AddCardExpItem{
		{
			ID:     items[0].ID,
			Amount: 1,
		},
	}

	now := time.Now()
	xIsuDate := time.Date(2022, 8, 27, now.Hour(), now.Minute(), now.Second(), now.Nanosecond(), myGMT)
	res, err := PostAddExpCard(ctx, agent, user.ID, cardID, addCardExpItem, oneTimeToken, masterVersion, xIsuDate, login)
	if err != nil {
		return failure.NewError(ValidationErrInvalidRequest, err)
	}
	defer res.Body.Close()

	validatePostAddExpCardResponse := &validatePostAddExpCardResponse{}

	validationResult := ValidateResponse(
		res,
		validatePostAddExpCardResponseBody(validatePostAddExpCardResponse, user, cardID, addCardExpItem, expItemMasters, cardMasters, loginRewardItem, xIsuDate),
	)

	validationResult.Add(step)

	if !validationResult.IsEmpty() {
		return validationResult
	}

	//ダブルサブミットチェック(もう一度同じリクエストを投げる)
	now = time.Now()
	xIsuDate = time.Date(2022, 8, 27, now.Hour(), now.Minute(), now.Second(), now.Nanosecond(), myGMT)
	res, err = PostAddExpCard(ctx, agent, user.ID, cardID, addCardExpItem, oneTimeToken, masterVersion, xIsuDate, login)
	if err != nil {
		return failure.NewError(ValidationErrInvalidRequest, err)
	}
	defer res.Body.Close()

	validatePostAddExpCardFail := &validateFailResponse{}

	validationFailResult := ValidateResponse(
		res,
		validatePostAddExpCardFailBody(validatePostAddExpCardFail, user),
	)

	validationFailResult.Add(step)

	if validationFailResult.IsEmpty() {
		return nil
	} else {
		return validationFailResult
	}

}

func (s *Scenario) AcceptGiftValidateSeccessScenario(ctx context.Context, step *isucandar.BenchmarkStep, user *ValidationUser, masterVersion string, login *Login, receivePresents *[]UserPresent) error {
	report := TimeReporter("プレゼント一覧 整合性チェック", s.Option)
	defer report()

	agent, err := s.GetAgentFromUser(step, &User{
		ID:       user.ID,
		UserType: user.UserType,
		ViewerID: user.ViewerID,
		Agent:    user.Agent,
	})
	if err != nil {
		return err
	}

	// ## プレゼント一覧画面の表示
	var listPresentResponse *ListPresentResponse

	now := time.Now()
	xIsuDate := time.Date(2022, 8, 27, now.Hour(), now.Minute(), now.Second(), now.Nanosecond(), myGMT)
	res, err := GetPresentList(ctx, agent, user.ID, masterVersion, xIsuDate, login)
	if err != nil {
		return failure.NewError(ValidationErrInvalidRequest, err)
	}
	defer res.Body.Close()

	listPresentResponse = &ListPresentResponse{}

	validationResult := ValidateResponse(
		res,
		validatePresentListResponse(listPresentResponse, user),
	)
	validationResult.Add(step)

	for _, present := range listPresentResponse.Presents {
		*receivePresents = append(*receivePresents, *present)
	}

	if validationResult.IsEmpty() {
		return nil
	} else {
		return validationResult
	}
}

func (s *Scenario) postReceivePresentValidateSeccessScenario(ctx context.Context, step *isucandar.BenchmarkStep, user *ValidationUser, masterVersion string, login *Login, receiveUserPresents *[]UserPresent) error {
	report := TimeReporter("プレゼント受け取り 整合性チェック", s.Option)
	defer report()

	agent, err := s.GetAgentFromUser(step, &User{
		ID:       user.ID,
		UserType: user.UserType,
		ViewerID: user.ViewerID,
		Agent:    user.Agent,
	})
	if err != nil {
		return err
	}
	var presentIDs []int64
	for _, v := range *receiveUserPresents {
		presentIDs = append(presentIDs, v.ID)
	}

	now := time.Now()
	xIsuDate := time.Date(2022, 8, 27, now.Hour(), now.Minute(), now.Second(), now.Nanosecond(), myGMT)
	res, err := PostReceivePresent(ctx, agent, user.ID, presentIDs, masterVersion, xIsuDate, login)
	if err != nil {
		return failure.NewError(ValidationErrInvalidRequest, err)
	}
	defer res.Body.Close()

	receivePresentResponse := &ReceivePresentResponse{}

	validationResult := ValidateResponse(
		res,
		validatePostReceivePresent(receivePresentResponse, user, receiveUserPresents, xIsuDate),
	)
	validationResult.Add(step)

	if !validationResult.IsEmpty() {
		return validationResult
	}

	// 全部受け取り済みなのを確認する
	var listPresentResponse *ListPresentResponse

	now = time.Now()
	xIsuDate = time.Date(2022, 8, 27, now.Hour(), now.Minute(), now.Second(), now.Nanosecond(), myGMT)
	res, err = GetPresentList(ctx, agent, user.ID, masterVersion, xIsuDate, login)
	if err != nil {
		return failure.NewError(ValidationErrInvalidRequest, err)
	}
	defer res.Body.Close()

	listPresentResponse = &ListPresentResponse{}

	validationResult = ValidateResponse(
		res,
		validateAfterRecievePresentListResponse(listPresentResponse, user),
	)
	validationResult.Add(step)

	if !validationResult.IsEmpty() {
		return validationResult
	}

	res, err = GetItemList(ctx, agent, user.ID, masterVersion, xIsuDate, login)
	if err != nil {
		return failure.NewError(ValidationErrInvalidRequest, err)
	}
	defer res.Body.Close()

	validateItemListResponse := &validateItemListResponse{}

	validationResult = ValidateResponse(
		res,
		validateItemListAfterReceivePresentResponseBody(validateItemListResponse, user, receiveUserPresents, xIsuDate),
	)
	validationResult.Add(step)

	if validationResult.IsEmpty() {
		return nil
	} else {
		return validationResult
	}

}

// ガチャ一覧整合性チェック
func (s *Scenario) GetGachaListValidationSuccessScenario(ctx context.Context, step *isucandar.BenchmarkStep, user *ValidationUser, masterVersion string, login *Login, gachaAllMasters []GachaData) (string, error) {
	report := TimeReporter("ガチャ一覧整合性チェック", s.Option)
	defer report()

	agent, err := s.GetAgentFromUser(step, &User{
		ID:       user.ID,
		UserType: user.UserType,
		ViewerID: user.ViewerID,
		Agent:    user.Agent,
	})
	if err != nil {
		return "", err
	}

	var listGachaResponse *ListGachaResponse

	// ## ガチャ一覧画面表示
	now := time.Now()
	xIsuDate := time.Date(2022, 8, 27, now.Hour(), now.Minute(), now.Second(), now.Nanosecond(), myGMT)
	res, err := GetGachaList(ctx, agent, user.ID, masterVersion, xIsuDate, login)
	if err != nil {
		return "", failure.NewError(ValidationErrInvalidRequest, err)
	}
	defer res.Body.Close()

	listGachaResponse = &ListGachaResponse{}

	validationResult := ValidateResponse(
		res,
		validateGachaListResponse(listGachaResponse, user, gachaAllMasters, xIsuDate),
	)
	oneTimeToken := listGachaResponse.OneTimeToken

	validationResult.Add(step)

	if validationResult.IsEmpty() {
		return oneTimeToken, nil
	} else {
		return "", validationResult
	}

}

// ガチャ結果整合性チェック
func (s *Scenario) postGachaDrawValidateSuccessScenario(ctx context.Context, step *isucandar.BenchmarkStep, user *ValidationUser, masterVersion string, login *Login, oneTimeToken string, gachaAllMasters []GachaData) error {
	report := TimeReporter("ガチャ結果整合性チェック", s.Option)
	defer report()

	agent, err := s.GetAgentFromUser(step, &User{
		ID:       user.ID,
		UserType: user.UserType,
		ViewerID: user.ViewerID,
		Agent:    user.Agent,
	})
	if err != nil {
		return err
	}

	now := time.Now()
	xIsuDate := time.Date(2022, 8, 27, now.Hour(), now.Minute(), now.Second(), now.Nanosecond(), myGMT)
	nowUnix := xIsuDate.Unix()
	var gachaData []GachaData
	for _, v := range gachaAllMasters {
		if v.Gacha.StartAt < nowUnix && nowUnix < v.Gacha.EndAt {
			gachaData = append(gachaData, GachaData{
				Gacha:     v.Gacha,
				GachaItem: v.GachaItem,
			})
		}
	}
	num := rand.Intn(len(gachaData) - 1)

	gachaID := gachaData[num].Gacha.ID
	// ## ガチャを引く
	res, err := PostRedeemGacha(ctx, agent, user.ID, masterVersion, xIsuDate, int(gachaID), oneTimeToken, login)
	if err != nil {
		return failure.NewError(ValidationErrInvalidRequest, err)
	}
	defer res.Body.Close()

	drawGachaResponse := &DrawGachaResponse{}

	validationResult := ValidateResponse(
		res,
		validateGachaDrawResponse(drawGachaResponse, user, gachaData[num], xIsuDate),
	)

	validationResult.Add(step)

	if !validationResult.IsEmpty() {
		return validationResult
	}

	//ダブルサブミット（同じパラメータでもう一度引く）
	now = time.Now()
	xIsuDate = time.Date(2022, 8, 27, now.Hour(), now.Minute(), now.Second(), now.Nanosecond(), myGMT)
	doubleRes, err := PostRedeemGacha(ctx, agent, user.ID, masterVersion, xIsuDate, int(gachaID), oneTimeToken, login)
	if err != nil {
		return failure.NewError(ValidationErrInvalidRequest, err)
	}
	defer doubleRes.Body.Close()

	validatePostReadeemFail := &validateFailResponse{}

	validationFailResult := ValidateResponse(
		doubleRes,
		validatePostRedeemGachaFailBody(validatePostReadeemFail, user),
	)

	validationFailResult.Add(step)

	if validationFailResult.IsEmpty() {
		return nil
	} else {
		return validationFailResult
	}
}

// ベンチ実行後の整合性検証シナリオ
// isucandar.ValidateScenarioを満たすメソッド
// isucandar.Benchmark の validation ステップで実行される
func (sc *Scenario) ValidationScenario(ctx context.Context, step *isucandar.BenchmarkStep) error {
	report := TimeReporter("validation", sc.Option)
	defer report()

	ContestantLogger.Println("[ValidationScenario] 整合性チェックを開始します")
	defer ContestantLogger.Printf("[ValidationScenario] 整合性チェックを終了します")

	// validation用のvalidation.jsonを読む
	if err := sc.ValidateUsers.LoadJSON("./dump/validateUserInitialize.json"); err != nil {
		return failure.NewError(ValidationErrFailedToLoadJson, err)
	}

	// expItemMaster.jsonを読む
	var expItemMasters []localExpItemMaster
	if err := parseMasterJson("./dump/expItemMaster.json", &expItemMasters); err != nil {
		return failure.NewError(ValidationErrFailedToLoadJson, err)
	}
	// cardMaster.json
	var cardMasters CardMasters
	if err := parseMasterJson("./dump/cardMaster.json", &cardMasters); err != nil {
		return failure.NewError(ValidationErrFailedToLoadJson, err)
	}
	// loginBonusRewardMaster.json
	var loginBonusRewardMasters LoginBonusRewardMasters
	if err := parseMasterJson("./dump/loginBonusRewardMaster.json", &loginBonusRewardMasters); err != nil {
		return failure.NewError(ValidationErrFailedToLoadJson, err)
	}

	// gachaMaster.json
	var gachaAllMasters []GachaData
	if err := parseMasterJson("./dump/gachaAllItemMaster.json", &gachaAllMasters); err != nil {
		return failure.NewError(ValidationErrFailedToLoadJson, err)
	}

	// presentAllMaster.json
	var presentAllMasters []PresentAllMaster
	if err := parseMasterJson("./dump/presentAllMaster.json", &presentAllMasters); err != nil {
		return failure.NewError(ValidationErrFailedToLoadJson, err)
	}

	// ユーザーの払い出し
	validationBeforeUser, success := sc.ValidateUsers.Pop()
	if !success {
		return failure.NewError(ValidationErrFailedToLoadJson, nil)
	}

	// =============Loginボーナスチェック（日跨ぎ）
	// 0.0 ログイン処理
	ContestantLogger.Println("整合性チェック Request:POST /login (日跨ぎ前）")
	if err := sc.loginValidateBeforDaySuccessScenario(ctx, step, validationBeforeUser, sc.LatestMasterVersion(), loginBonusRewardMasters); err != nil {
		return err
	}

	// 0.1 ログイン処理
	ContestantLogger.Println("整合性チェック Request:POST /login (日跨ぎ後）")
	if err := sc.loginValidateAfterDaySuccessScenario(ctx, step, validationBeforeUser, sc.LatestMasterVersion(), loginBonusRewardMasters); err != nil {
		return err
	}
	// ==============ここまで

	// ユーザーの払い出し（次のユーザ）
	validationUser := sc.ValidateUsers.At(rand.Intn(sc.ValidateUsers.Len()))
	// admin =================== start
	ContestantLogger.Println("整合性チェック Request:POST /admin/login 失敗の場合")
	if err := sc.postAdminLoginFailValidateScenario(ctx, step); err != nil {
		return err
	}

	ContestantLogger.Println("整合性チェック Request:POST /admin/login")
	adminLogin, err := sc.postAdminLoginValidateSuccessScenario(ctx, step)
	if err != nil {
		return err
	}

	ContestantLogger.Println("整合性チェック Request:GET /admin/user/{userID}")
	if err := sc.getAdminUserValidateSuccessScenario(ctx, step, adminLogin, validationUser); err != nil {
		return err
	}

	// admin =================== end

	// ログイン失敗用のデータ作成
	userID := generateFailUserID()
	viewerID := generateFailViewerID()

	ContestantLogger.Println("整合性チェック Request:POST /login 失敗の場合")
	if err := sc.loginValidateFailScenario(ctx, step, validationUser, userID, viewerID, sc.LatestMasterVersion()); err != nil {
		return err
	}

	// 0. ユーザ作成
	platforms, success := sc.Platforms.Pop()
	if !success {
		return failure.NewError(ValidationErrFailedToLoadJson, nil)
	}
	ContestantLogger.Println("整合性チェック Request:POST /user")
	postUserlogin, postUser, err := sc.postUserValidateSuccessScenario(ctx, step, platforms, sc.LatestMasterVersion(), loginBonusRewardMasters, presentAllMasters)
	if err != nil {
		return err
	}

	// 1. ログイン処理
	ContestantLogger.Println("整合性チェック Request:POST /login")
	login, err := sc.loginValidateSuccessScenario(ctx, step, validationUser, sc.LatestMasterVersion(), loginBonusRewardMasters)
	if err != nil {
		return err
	}
	// 2.0 ホーム画面表示失敗
	ContestantLogger.Println("整合性チェック Request:GET /user/{userId}/home セッション無効の場合")
	if err := sc.ShowHomeValidateFailScenario(ctx, step, validationUser, sc.LatestMasterVersion()); err != nil {
		return err
	}

	// 2.1 ホーム画面表示
	ContestantLogger.Println("整合性チェック Request:GET /user/{userId}/home")
	loginRewardItem, err := sc.ShowHomeValidateSuccessScenario(ctx, step, validationUser, sc.LatestMasterVersion(), login, loginBonusRewardMasters)
	if err != nil {
		return err
	}

	// 2.2 他人のセッションでホーム画面表示チェック
	ContestantLogger.Println("整合性チェック Request:GET /user/{userId}/home 他人のホーム表示の場合")
	if err := sc.ShowOhterHomeValidateFailScenario(ctx, step, validationUser, sc.LatestMasterVersion(), login, postUserlogin, postUser); err != nil {
		return err
	}

	// 3. reward受け取り
	ContestantLogger.Println("整合性チェック Request:POST /user/{userId}/reward")
	if err := sc.postRewardValidateSuccessScenario(ctx, step, validationUser, sc.LatestMasterVersion(), login); err != nil {
		return err
	}

	// 4. deck変更
	ContestantLogger.Println("整合性チェック Request:POST /user/{userId}/card")
	if err := sc.postCardValidateSuccessScenario(ctx, step, validationUser, sc.LatestMasterVersion(), login); err != nil {
		return err
	}

	// 5. item画面表示
	ContestantLogger.Println("整合性チェック Request:GET /user/{userId}/item")
	oneTimeToken, err := sc.ShowItemValidateSuccessScenario(ctx, step, validationUser, sc.LatestMasterVersion(), login, loginRewardItem)
	if err != nil {
		return err
	}

	// 6. cardの経験値アップ
	ContestantLogger.Println("整合性チェック Request:POST /user/{userId}/addexp/{cardId}")
	if err := sc.postAddExpCardIDValidateSuccessScenario(ctx, step, validationUser, sc.LatestMasterVersion(), login, oneTimeToken, expItemMasters, cardMasters, loginRewardItem); err != nil {
		return err
	}

	// 7. プレゼント一覧
	var receiveUserPresents []UserPresent
	ContestantLogger.Println("整合性チェック Request:GET /user/{userId}/present/index/:n")
	if err := sc.AcceptGiftValidateSeccessScenario(ctx, step, validationUser, sc.LatestMasterVersion(), login, &receiveUserPresents); err != nil {
		return err
	}

	// 8. プレゼント受取
	ContestantLogger.Println("整合性チェック Request:POST /user/{userId}/present/receive")
	if err := sc.postReceivePresentValidateSeccessScenario(ctx, step, validationUser, sc.LatestMasterVersion(), login, &receiveUserPresents); err != nil {
		return err
	}

	// 9. ガチャ一覧
	ContestantLogger.Println("整合性チェック Request:GET /user/{userId}/gacha/index")
	gachaOneTimeToken, err := sc.GetGachaListValidationSuccessScenario(ctx, step, validationUser, sc.LatestMasterVersion(), login, gachaAllMasters)
	if err != nil {
		return err
	}
	// 10 ガチャ引く
	ContestantLogger.Println("整合性チェック Request:POST /user/{userId}/gacha/draw/{gachaID}/10")
	if err := sc.postGachaDrawValidateSuccessScenario(ctx, step, validationUser, sc.LatestMasterVersion(), login, gachaOneTimeToken, gachaAllMasters); err != nil {
		return err
	}

	// admin =================== start
	ContestantLogger.Println("整合性チェック Request:GET /admin/master")
	if err := sc.getAdminMasterValidateSuccessScenario(ctx, step, adminLogin, gachaAllMasters, loginBonusRewardMasters); err != nil {
		return err
	}

	ContestantLogger.Println("整合性チェック Request:POST /admin/user/{userID}/ban")
	if err := sc.postAdminUserBanValidateSuccessScenario(ctx, step, adminLogin, validationUser); err != nil {
		return err
	}
	// admin =================== end

	user := &User{
		ID:       validationUser.ID,
		UserType: validationUser.UserType,
		ViewerID: validationUser.ViewerID,
		Agent:    validationUser.Agent,
	}
	ContestantLogger.Println("整合性チェック Request:POST /login Banの場合")
	if err := sc.loginBanValidateScenario(ctx, step, user, sc.LatestMasterVersion()); err != nil {
		return err
	}

	return nil
}
