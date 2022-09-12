package main

import (
	"fmt"
	"math/rand"
	"net/http"
	"strings"

	"github.com/isucon/isucandar/failure"
)

func validateAdminFailLogin(adminLoginFailResponse *validateFailResponse) ResponseValidator {
	return func(r *http.Response) error {
		// HTTP status codeのチェックで失敗したら続行しない
		if err := IsuAssertStatus(LoginFailStatusCode, r.StatusCode, Hint(PostAdminLogin, "")); err != nil {
			return err
		}

		if err := parseJsonBody(r, adminLoginFailResponse); err != nil {
			AdminLogger.Printf("adminLoginFailResponse:%#v\n", adminLoginFailResponse)
			return failure.NewError(ValidationErrInvalidResponseBody, fmt.Errorf("%s: %v ", PostAdminLogin+" のBodyの Json decodeに失敗しました", err))
		}

		expected := &validateFailResponse{
			StatusCode: LoginFailStatusCode,
			Message:    LoginFailMessage,
		}

		if err := Diff(expected, adminLoginFailResponse, Hint(PostAdminLogin, "")); err != nil {
			return err
		}

		return nil
	}
}

func validateAdminPasswordFailLogin(adminLoginPasswordFailResponse *validateFailResponse) ResponseValidator {
	return func(r *http.Response) error {
		// HTTP status codeのチェックで失敗したら続行しない
		if err := IsuAssertStatus(LoginUnauthorizedStatusCode, r.StatusCode, Hint(PostAdminLogin, "")); err != nil {
			return err
		}

		if err := parseJsonBody(r, adminLoginPasswordFailResponse); err != nil {
			AdminLogger.Printf("adminLoginPasswordFailResponse:%#v\n", adminLoginPasswordFailResponse)
			return failure.NewError(ValidationErrInvalidResponseBody, fmt.Errorf("%s: %v ", PostAdminLogin+" のBodyの Json decodeに失敗しました", err))
		}

		expected := &validateFailResponse{
			StatusCode: LoginUnauthorizedStatusCode,
			Message:    LoginUnauthorizedMessage,
		}

		if err := Diff(expected, adminLoginPasswordFailResponse, Hint(PostAdminLogin, "")); err != nil {
			return err
		}
		return nil
	}
}

func validateAdminLogin(adminLoginResponse *adminLoginResponse, adminUser *AdminUser) ResponseValidator {
	return func(r *http.Response) error {
		// HTTP status codeのチェックで失敗したら続行しない
		if err := IsuAssertStatus(http.StatusOK, r.StatusCode, Hint(PostAdminLogin, "")); err != nil {
			return err
		}

		if err := parseJsonBody(r, adminLoginResponse); err != nil {
			AdminLogger.Printf("adminLoginResponse:%#v\n", adminLoginResponse)
			return failure.NewError(ValidationErrInvalidResponseBody, fmt.Errorf("%s: %v ", PostAdminLogin+" のBodyの Json decodeに失敗しました", err))
		}

		expected := &Session{
			UserID:    adminUser.GetID(),
			DeletedAt: nil,
		}

		if err := Diff(expected, &adminLoginResponse.Session, Hint(PostAdminLogin, "session."), IgnoreWhat("id", "sessionId", "createdAt", "updatedAt", "expiredAt")...); err != nil {
			AdminLogger.Printf("expectedSession:%#v\n actualSession:%#v\n", expected, adminLoginResponse.Session)
			return err
		}

		return nil
	}
}

func validateAdminUserBan(postAdminUserBanRes *postAdminUserBanResponse, user *ValidationUser) ResponseValidator {
	return func(r *http.Response) error {

		endpoint := strings.Replace(PostAdminUserBan, ":userId", fmt.Sprintf("%d", user.ID), 1)

		// HTTP status codeのチェックで失敗したら続行しない
		if err := IsuAssertStatus(http.StatusOK, r.StatusCode, Hint(endpoint, "")); err != nil {
			return err
		}

		if err := parseJsonBody(r, postAdminUserBanRes); err != nil {
			AdminLogger.Printf("postAdminUserBanRes:%#v\n", postAdminUserBanRes)
			return failure.NewError(ValidationErrInvalidResponseBody, fmt.Errorf("%s: %v ", endpoint+" のBodyの Json decodeに失敗しました", err))
		}

		expected := &JsonUser{
			ID:           user.ID,
			RegisteredAt: user.JsonUser.RegisteredAt,
			CreatedAt:    user.JsonUser.CreatedAt,
		}

		if err := Diff(expected, &postAdminUserBanRes.User, Hint(endpoint, "user."), IgnoreWhat("isuCoin", "lastGetRewardAt", "lastActivatedAt", "updatedAt", "deletedAt")...); err != nil {
			AdminLogger.Printf("expectedUser:%#v\n actualUser:%#v\n", expected, postAdminUserBanRes.User)
			return err
		}

		return nil
	}
}

func validateAdminMaster(getAdminMasterResponse *getAdminMasterResponse, gachaData []GachaData, loginBonusRewardMasters LoginBonusRewardMasters) ResponseValidator {
	return func(r *http.Response) error {
		// HTTP status codeのチェックで失敗したら続行しない
		if err := IsuAssertStatus(http.StatusOK, r.StatusCode, Hint(GetAdminMaster, "")); err != nil {
			return err
		}

		if err := parseJsonBody(r, getAdminMasterResponse); err != nil {
			AdminLogger.Printf("getAdminMasterResponse:%#v\n", getAdminMasterResponse)
			return failure.NewError(ValidationErrInvalidResponseBody, fmt.Errorf("%s: %v ", GetAdminMaster+" のBodyの Json decodeに失敗しました", err))
		}

		//gachaDataMaster
		resGachaMaster := getAdminMasterResponse.GachaMaster
		//個数
		if err := IsuAssert(len(gachaData), len(resGachaMaster), Hint(GetAdminMaster, "gachas の個数")); err != nil {
			return err
		}

		// ランダムにピックアップ
		num := rand.Intn(len(gachaData) - 1)
		for _, v := range resGachaMaster {

			if gachaData[num].Gacha.ID != v.ID {
				continue
			}

			if err := Diff(&gachaData[num].Gacha, &v, Hint(GetAdminMaster, "gachas.")); err != nil {
				AdminLogger.Printf("expectedGacha:%#v\n actualGacha:%#v\n", gachaData[num].Gacha, v)
				return err
			}

			break
		}

		//gachaItemsMaster
		gachaID := gachaData[num].Gacha.ID
		expectedGachaItemMaster := gachaData[num].GachaItem
		var resGachaItemMasters []GachaItemMaster
		for _, v := range getAdminMasterResponse.GachaItemMaster {
			if gachaID == v.GachaID {
				resGachaItemMasters = append(resGachaItemMasters, v)
			}
		}
		//個数のチェック
		if err := IsuAssert(len(expectedGachaItemMaster), len(resGachaItemMasters), Hint(GetAdminMaster, "gachaItems の個数")); err != nil {
			return err
		}

		// ランダムにピック
		nAt := rand.Intn(len(expectedGachaItemMaster) - 1)
		for _, v := range resGachaItemMasters {
			if expectedGachaItemMaster[nAt].ID != v.ID {
				continue
			}

			if err := Diff(&expectedGachaItemMaster[nAt], &v, Hint(GetAdminMaster, "gachaItemList.")); err != nil {
				AdminLogger.Printf("expectedGachaItem:%#v\n actualGachaItem:%#v\n", expectedGachaItemMaster[nAt], v)
				return err
			}

			break
		}

		//loginBonusRewardMasters
		resLoginBonusRewardMasters := getAdminMasterResponse.LoginBonusRewardMaster
		//個数のチェク
		if err := IsuAssert(len(loginBonusRewardMasters), len(resLoginBonusRewardMasters), Hint(GetAdminMaster, "loginBonusRewards の個数")); err != nil {
			return err
		}

		// ランダムにピック
		num = rand.Intn(len(loginBonusRewardMasters) - 1)
		for _, v := range resLoginBonusRewardMasters {
			if loginBonusRewardMasters[num].ID != v.ID {
				continue
			}

			if err := Diff(&loginBonusRewardMasters[num], &v, Hint(GetAdminMaster, "loginBonusRewards.")); err != nil {
				AdminLogger.Printf("expectedLoginBonuseReward:%#v\n actualLoginBonusReward:%#v\n", loginBonusRewardMasters[num], v)
				return err
			}
			break
		}
		return nil
	}
}

func validateAdminUser(getAdminUserRes *getAdminUserResponse, user *ValidationUser) ResponseValidator {
	return func(r *http.Response) error {
		endpoint := strings.Replace(GetAdminUser, ":userId", fmt.Sprintf("%d", user.ID), 1)

		// HTTP status codeのチェックで失敗したら続行しない
		if err := IsuAssertStatus(http.StatusOK, r.StatusCode, Hint(endpoint, "")); err != nil {
			return err
		}

		if err := parseJsonBody(r, getAdminUserRes); err != nil {
			AdminLogger.Printf("getAdminUserRes:%#v\n", getAdminUserRes)
			return failure.NewError(ValidationErrInvalidResponseBody, fmt.Errorf("%s: %v ", endpoint+" のBodyの Json decodeに失敗しました", err))
		}

		//user
		{
			responseUser := getAdminUserRes.User
			if responseUser.ID == 0 {
				if err := IsuAssert("user", "", Hint(endpoint, " のBodyに user がないため想定 ")); err != nil {
					return err
				}
			}

			if err := Diff(&user.JsonUser, &responseUser, Hint(endpoint, "user.")); err != nil {
				AdminLogger.Printf("expectedUser:%#v\n actualUser:%#v\n", user.JsonUser, responseUser)
				return err
			}

		}
		//userCards
		{
			responseUserCards := getAdminUserRes.UserCards

			//個数のチェク
			if err := IsuAssert(len(user.UserCards), len(responseUserCards), Hint(endpoint, "userCards の個数")); err != nil {
				return err
			}

			//ランダムにピックアップ
			num := rand.Intn(len(user.UserCards) - 1)
			expectedUserCard := user.UserCards[num]

			for i, v := range responseUserCards {
				if expectedUserCard.ID != v.ID {
					continue
				}

				if err := Diff(&expectedUserCard, &v, Hint(endpoint+fmt.Sprintf(" userCards[%d].id:%d", i, v.ID), "userCards.")); err != nil {
					AdminLogger.Printf("expectedUserCard:%#v\n actualUserCard:%#v\n", expectedUserCard, v)
					return err
				}
				break
			}
		}
		// userDecks
		{
			responseUserDecks := getAdminUserRes.UserDecks
			expectedUserDeck := user.UserDeck

			for i, v := range responseUserDecks {
				// 既存のものだけをチェック
				if expectedUserDeck.ID != v.ID {
					continue
				}
				if err := Diff(&expectedUserDeck, &v, Hint(endpoint+fmt.Sprintf(" userDecks[%d].id:%d", i, v.ID), "userDecks.")); err != nil {
					AdminLogger.Printf("expectedUserDeck:%#v\n actualUserDeck:%#v\n", expectedUserDeck, v)
					return err
				}
				break
			}
		}
		// userDevices
		{
			responseUserDevices := getAdminUserRes.UserDevices
			expectedUserDevices := user.UserDevices

			//個数のチェク
			if err := IsuAssert(len(expectedUserDevices), len(responseUserDevices), Hint(endpoint, "userDevices の個数")); err != nil {
				return err
			}

			for _, expect := range expectedUserDevices {
				for i, res := range responseUserDevices {
					if expect.ID != res.ID {
						continue
					}
					if err := Diff(&expect, &res, Hint(endpoint+fmt.Sprintf(" userDevices[%d].id:%d", i, res.ID), "userDevices.")); err != nil {
						AdminLogger.Printf("expectedUserDevices:%#v\n actualUserDevices:%#v\n", expect, res)
						return err
					}
					break
				}
			}
		}
		//userItems
		{
			responseUserItems := getAdminUserRes.UserItems
			expextedUserItems := user.GetItemList

			//個数のチェク
			if err := IsuAssert(len(expextedUserItems), len(responseUserItems), Hint(endpoint, "userItems の個数")); err != nil {
				return err
			}

			for _, expect := range expextedUserItems {
				for i, res := range responseUserItems {
					if expect.ID != res.ID {
						continue
					}
					if err := Diff(&expect, &res, Hint(endpoint+fmt.Sprintf(" userItems[%d].id:%d", i, res.ID), "userItems.")); err != nil {
						AdminLogger.Printf("expectedUserItems:%#v\n actualUserItems:%#v\n", expect, res)
						return err
					}
					break
				}
			}

		}

		// userPresents
		{
			responseUserPresents := getAdminUserRes.UserPresents
			expectedUserPresents := user.UserAllPresents

			//個数のチェク
			if err := IsuAssert(len(expectedUserPresents), len(responseUserPresents), Hint(endpoint, "userPresents の個数")); err != nil {
				return err
			}

			//ランダムピック 200回
			count := 200
			if len(expectedUserPresents) < count {
				count = len(expectedUserPresents)
			}

			for i := 0; i < count; i++ {
				num := rand.Intn(len(expectedUserPresents) - 1)
				for _, v := range expectedUserPresents {
					if responseUserPresents[num].ID != v.ID {
						continue
					}
					if err := Diff(&v, &responseUserPresents[num], Hint(endpoint+fmt.Sprintf(" userPresents[%d].id:%d", i, responseUserPresents[num].ID), "userPresents."), IgnoreWhat("deletedAt")...); err != nil {
						AdminLogger.Printf("expectedUserPresents:%#v\n actualUserPresents:%#v\n", v, responseUserPresents[num])
						return err
					}

					//deletedAt
					if v.DeletedAt == nil {
						if responseUserPresents[num].DeletedAt != nil {
							return failure.NewError(
								ValidationErrInvalidResponseBody,
								fmt.Errorf("%s : expected(%v) != actual(%d)", endpoint+fmt.Sprintf(" userPresents[%d].id:%d", i, responseUserPresents[num].ID)+" deletedAt が違います",
									"null",
									*responseUserPresents[num].DeletedAt,
								),
							)
						}
					} else {
						if responseUserPresents[num].DeletedAt == nil {
							return failure.NewError(
								ValidationErrInvalidResponseBody,
								fmt.Errorf("%s : expected(%d) != actual(%s)", endpoint+fmt.Sprintf(" userPresents[%d].id:%d", i, responseUserPresents[num].ID)+" deletedAt が違います",
									*v.DeletedAt,
									"null",
								),
							)
						} else if *v.DeletedAt != *responseUserPresents[num].DeletedAt {
							return failure.NewError(
								ValidationErrInvalidResponseBody,
								fmt.Errorf("%s : expected(%d) != actual(%d)", endpoint+fmt.Sprintf(" userPresents[%d].id:%d", i, responseUserPresents[num].ID)+" deletedAt が違います",
									*v.DeletedAt,
									*responseUserPresents[num].DeletedAt,
								),
							)
						}
					}
					break
				}
			}
		}

		//userPresentAllReceivedHistory
		{
			responseUserPresentAllReceivedHistory := getAdminUserRes.UserPresentAllReceivedHistory
			expectedUserPresentAllReceivedHistory := user.UserPresentAllReceiveHistories

			//個数のチェク
			if err := IsuAssert(len(expectedUserPresentAllReceivedHistory), len(responseUserPresentAllReceivedHistory), Hint(endpoint, "userPresentAllReceivedHistory の個数")); err != nil {
				return err
			}

			for _, expect := range expectedUserPresentAllReceivedHistory {
				for i, res := range responseUserPresentAllReceivedHistory {
					if expect.ID != res.ID {
						continue
					}
					if err := Diff(&expect, &res, Hint(endpoint+fmt.Sprintf(" userPresentAllReceivedHistory[%d].id:%d", i, res.ID), "userPresentAllReceivedHistory."), IgnoreWhat("deletedAt")...); err != nil {
						AdminLogger.Printf("expectedUserPresentAllReceivedHistory:%#v\n actualUserPresentsAllReceivedHistory:%#v\n", expect, res)
						return err
					}

					if expect.DeletedAt == nil {
						if res.DeletedAt != nil {
							return failure.NewError(
								ValidationErrInvalidResponseBody,
								fmt.Errorf("%s : expected(%v) != actual(%d)", endpoint+fmt.Sprintf(" userPresentAllReceivedHistory[%d].id:%d", i, res.ID)+" deletedAt が違います",
									"null",
									*res.DeletedAt,
								),
							)
						}
					} else {
						if res.DeletedAt == nil {
							return failure.NewError(
								ValidationErrInvalidResponseBody,
								fmt.Errorf("%s : expected(%d) != actual(%s)", endpoint+fmt.Sprintf(" userPresentAllReceivedHistory[%d].id:%d", i, res.ID)+" deletedAt が違います",
									*expect.DeletedAt,
									"null",
								),
							)
						} else if *expect.DeletedAt != *res.DeletedAt {
							return failure.NewError(
								ValidationErrInvalidResponseBody,
								fmt.Errorf("%s : expected(%d) != actual(%d)", endpoint+fmt.Sprintf(" userPresentAllReceivedHistory[%d].id:%d", i, res.ID)+" deletedAt が違います",
									*expect.DeletedAt,
									*res.DeletedAt,
								),
							)
						}
					}
					break
				}
			}
		}
		return nil
	}
}
