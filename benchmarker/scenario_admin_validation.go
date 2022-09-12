package main

import (
	"context"
	"math/rand"
	"strconv"

	"github.com/isucon/isucandar"
)

func (s *Scenario) postAdminLoginFailValidateScenario(ctx context.Context, step *isucandar.BenchmarkStep) error {
	report := TimeReporter("Adminログイン 失敗時　整合性チェック", s.Option)
	defer report()

	agent, err := s.AdminUser.GetAgent(s.Option)
	if err != nil {
		return err
	}

	min := int64(10000)

	fakeID := rand.Int63n(min) + 1
	fakePassword := strconv.FormatInt(rand.Int63n(min)+1, 10)
	adminUser := &AdminUser{
		ID:       fakeID,
		Password: fakePassword,
	}

	res, err := PostAdminLoginAction(ctx, agent, adminUser, s.LatestMasterVersion())
	if err != nil {
		return err
	}
	defer res.Body.Close()

	adminLoginFailResponse := &validateFailResponse{}

	adminLoginFailValidation := ValidateResponse(
		res,
		validateAdminFailLogin(adminLoginFailResponse),
	)
	adminLoginFailValidation.Add(step)

	if !adminLoginFailValidation.IsEmpty() {
		return adminLoginFailValidation
	}

	adminUser = &AdminUser{
		ID:       LoginAdminID,
		Password: fakePassword,
	}

	res, err = PostAdminLoginAction(ctx, agent, adminUser, s.LatestMasterVersion())
	if err != nil {
		return err
	}
	defer res.Body.Close()

	adminLoginPasswordFailResponse := &validateFailResponse{}

	adminLoginPasswordFailValidation := ValidateResponse(
		res,
		validateAdminPasswordFailLogin(adminLoginPasswordFailResponse),
	)
	adminLoginPasswordFailValidation.Add(step)

	if adminLoginPasswordFailValidation.IsEmpty() {
		return nil
	} else {
		return adminLoginPasswordFailValidation
	}

}

func (s *Scenario) postAdminLoginValidateSuccessScenario(ctx context.Context, step *isucandar.BenchmarkStep) (*Login, error) {
	report := TimeReporter("Adminログイン 整合性チェック", s.Option)
	defer report()

	agent, err := s.AdminUser.GetAgent(s.Option)
	if err != nil {
		return nil, err
	}

	res, err := PostAdminLoginAction(ctx, agent, s.AdminUser, s.LatestMasterVersion())
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	adminLoginResponse := &adminLoginResponse{}

	adminLoginValidation := ValidateResponse(
		res,
		validateAdminLogin(adminLoginResponse, s.AdminUser),
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

func (s *Scenario) postAdminUserBanValidateSuccessScenario(ctx context.Context, step *isucandar.BenchmarkStep, adminLogin *Login, user *ValidationUser) error {
	report := TimeReporter("AdminユーザBan 整合性チェック", s.Option)
	defer report()

	agent, err := s.AdminUser.GetAgent(s.Option)
	if err != nil {
		return err
	}

	res, err := PostAdminUserBanAction(ctx, agent, adminLogin.SessionID, s.LatestMasterVersion(), user.ID)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	postAdminUserBanResponse := &postAdminUserBanResponse{}

	adminValidation := ValidateResponse(
		res,
		validateAdminUserBan(postAdminUserBanResponse, user),
	)
	adminValidation.Add(step)

	if adminValidation.IsEmpty() {
		return nil
	} else {
		return adminValidation
	}
}

func (s *Scenario) getAdminMasterValidateSuccessScenario(ctx context.Context, step *isucandar.BenchmarkStep, adminLogin *Login, gachaData []GachaData, loginBonusRewardMasters LoginBonusRewardMasters) error {
	report := TimeReporter("Adminマスター照会 整合性チェック", s.Option)
	defer report()

	agent, err := s.AdminUser.GetAgent(s.Option)
	if err != nil {
		return err
	}

	res, err := GetAdminMasterAction(ctx, agent, adminLogin.SessionID, s.LatestMasterVersion())
	if err != nil {
		return err
	}
	defer res.Body.Close()

	getAdminMasterResponse := &getAdminMasterResponse{}

	adminValidation := ValidateResponse(
		res,
		validateAdminMaster(getAdminMasterResponse, gachaData, loginBonusRewardMasters),
	)
	adminValidation.Add(step)

	if adminValidation.IsEmpty() {
		return nil
	} else {
		return adminValidation
	}
}

func (s *Scenario) getAdminUserValidateSuccessScenario(ctx context.Context, step *isucandar.BenchmarkStep, adminLogin *Login, user *ValidationUser) error {
	report := TimeReporter("Adminユーザー照会 整合性チェック", s.Option)
	defer report()

	agent, err := s.AdminUser.GetAgent(s.Option)
	if err != nil {
		return err
	}

	res, err := GetAdminUserAction(ctx, agent, adminLogin.SessionID, s.LatestMasterVersion(), user.ID)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	getAdminUserResponse := &getAdminUserResponse{}

	adminValidation := ValidateResponse(
		res,
		validateAdminUser(getAdminUserResponse, user),
	)
	adminValidation.Add(step)

	if adminValidation.IsEmpty() {
		return nil
	} else {
		return adminValidation
	}
}
