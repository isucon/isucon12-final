package main

import (
	"fmt"
	"math"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/isucon/isucandar/failure"
)

func validatePostUser(validatePostUserRes *validatePostUserResponse, platformUser *Platform, user *JsonUser, loginBonusRewardMasters []LoginBonusRewardMaster, presentAllMasters []PresentAllMaster, now time.Time) ResponseValidator {
	return func(r *http.Response) error {
		// HTTP status codeのチェックで失敗したら続行しない
		if err := IsuAssertStatus(200, r.StatusCode, Hint(PostUser, "")); err != nil {
			return err
		}

		if err := parseJsonBody(r, validatePostUserRes); err != nil {
			AdminLogger.Printf("ReponseBody:%#v\n", validatePostUserRes)
			return failure.NewError(ValidationErrInvalidResponseBody, fmt.Errorf("%s: %v ", PostUser+" のBodyの Json decodeに失敗しました", err))
		}

		//存在チェック
		if userID := validatePostUserRes.UserID; userID == 0 {
			return failure.NewError(ValidationErrInvalidResponseBody, fmt.Errorf("%s ", PostUser+" のBodyの userId がありません"))
		}
		if viewerID := validatePostUserRes.ViewerID; len(viewerID) == 0 {
			return failure.NewError(ValidationErrInvalidResponseBody, fmt.Errorf("%s ", PostUser+" のBodyの viewerId がありません"))
		}
		if sessionID := validatePostUserRes.SessionID; len(sessionID) == 0 {
			return failure.NewError(ValidationErrInvalidResponseBody, fmt.Errorf("%s ", PostUser+" のBodyの sessionId がありません"))
		}
		if createdAt := validatePostUserRes.CreatedAt; createdAt == 0 {
			return failure.NewError(ValidationErrInvalidResponseBody, fmt.Errorf("%s ", PostUser+" のBodyの createdAt がありません"))
		}
		//user
		if validatePostUserRes.UpdatedResources.User == (JsonUser{}) {
			return failure.NewError(ValidationErrInvalidResponseBody, fmt.Errorf("%s ", PostUser+" のBodyの updateResources.user がありません"))
		}
		user = &validatePostUserRes.UpdatedResources.User

		//userDevice
		if validatePostUserRes.UpdatedResources.UserDevice == (UserDevice{}) {
			return failure.NewError(ValidationErrInvalidResponseBody, fmt.Errorf("%s ", PostUser+" のBodyの updateResources.userDevice がありません"))
		}
		expectedUserDevice := UserDevice{
			UserID:       user.ID,
			PlatformID:   strconv.FormatInt(platformUser.ID, 10),
			PlatformType: platformUser.Type,
			CreatedAt:    now.Unix(),
			UpdatedAt:    now.Unix(),
		}
		if err := Diff(&expectedUserDevice, &validatePostUserRes.UpdatedResources.UserDevice, Hint(PostUser, "updatedResources.userDevice."), IgnoreWhat("id", "deletedAt")...); err != nil {
			AdminLogger.Printf("expectedUserDevice:%#v\n actualuserDevice:%#v\n", expectedUserDevice, validatePostUserRes.UpdatedResources.UserDevice)
			return err
		}

		//userCards
		if len(validatePostUserRes.UpdatedResources.UserCards) != 3 {
			return failure.NewError(ValidationErrInvalidResponseBody, fmt.Errorf("%s ", PostUser+" のBodyの updateResources.userCards の個数が違います"))
		}
		userCards := validatePostUserRes.UpdatedResources.UserCards

		expectedUserCard := UserCard{
			UserID:       user.ID,
			CardID:       2, // 初期値固定
			AmountPerSec: 1, // 初期値固定
			Level:        1, // 初期値固定
			TotalExp:     0, // 初期値固定
			CreatedAt:    now.Unix(),
			UpdatedAt:    now.Unix(),
		}
		for _, v := range userCards {
			if err := Diff(&expectedUserCard, &v, Hint(PostUser, "updatedResources.userCards."), IgnoreWhat("id", "deletedAt")...); err != nil {
				AdminLogger.Printf("expectedUserCard:%#v\n actualUserCard:%#v\n", expectedUserCard, v)
				return err
			}
		}

		//userDecks
		if len(validatePostUserRes.UpdatedResources.UserDecks) != 1 {
			return failure.NewError(ValidationErrInvalidResponseBody, fmt.Errorf("%s ", PostUser+" のBodyの updateResources.userDecks の個数が違います"))
		}

		expectedUserDeck := UserDeck{
			UserID:    user.ID,
			CardID1:   userCards[0].ID,
			CardID2:   userCards[1].ID,
			CardID3:   userCards[2].ID,
			CreatedAt: now.Unix(),
			UpdatedAt: now.Unix(),
		}
		if err := Diff(&expectedUserDeck, &validatePostUserRes.UpdatedResources.UserDecks[0], Hint(PostUser, "updatedResources.userDecks."), IgnoreWhat("id", "deletedAt")...); err != nil {
			AdminLogger.Printf("expectedUserDeck:%#v\n actualUserDeck:%#v\n", expectedUserDeck, validatePostUserRes.UpdatedResources.UserDecks[0])
			return err
		}
		var expectedLoginBonuses []UserLoginBonus
		//loginBonuses(3つ)
		for _, v := range loginBonusRewardMasters {
			if v.RewardSequence == 1 {
				expectedLoginBonuses = append(expectedLoginBonuses, UserLoginBonus{
					UserID:             user.ID,
					LoginBonusID:       v.LoginBonusID,
					LastRewardSequence: 1,
					LoopCount:          1,
					CreatedAt:          now.Unix(),
					UpdatedAt:          now.Unix(),
				})
			}
		}
		// 個数のチェック
		if err := IsuAssert(len(expectedLoginBonuses), len(validatePostUserRes.UpdatedResources.UserLoginBonuses), Hint(PostUser, "userLoginBonuses の個数")); err != nil {
			return err
		}

		for _, v := range expectedLoginBonuses {
			for _, w := range validatePostUserRes.UpdatedResources.UserLoginBonuses {
				if v.LoginBonusID != w.LoginBonusID {
					continue
				}
				if err := Diff(&v, &w, Hint(PostUser, "updatedResources.userLoginBonuses."), IgnoreWhat("id", "deletedAt")...); err != nil {
					AdminLogger.Printf("expectedLoginBonuses:%#v\n actualLoginBonuses:%#v\n", v, w)
					return err
				}
				break
			}
		}
		//userPresents
		nowUnix := now.Unix()
		var expectedUserPresents []UserPresent
		for _, v := range presentAllMasters {
			if v.RegisteredStartAt < nowUnix && nowUnix < v.RegisteredEndAt {
				expectedUserPresents = append(expectedUserPresents, UserPresent{
					UserID:         user.ID,
					SentAt:         now.Unix(),
					ItemType:       v.ItemType,
					ItemID:         v.ItemID,
					Amount:         int(v.Amount),
					PresentMessage: v.PresentMessage,
					CreatedAt:      now.Unix(),
					UpdatedAt:      now.Unix(),
				})
			}
		}

		// 個数のチェック
		if err := IsuAssert(len(expectedUserPresents), len(validatePostUserRes.UpdatedResources.UserPresents), Hint(PostUser, "userPresents の個数")); err != nil {
			return err
		}

		for _, v := range expectedUserPresents {
			for _, w := range validatePostUserRes.UpdatedResources.UserPresents {
				//プレゼントメッセージの一致で判断
				if v.PresentMessage != w.PresentMessage {
					continue
				}
				if err := Diff(&v, &w, Hint(PostUser, "updatedResources.userPresents."), IgnoreWhat("id", "deletedAt")...); err != nil {
					AdminLogger.Printf("expectedUserPresents:%#v\n actualUserPresents:%#v\n", v, w)
					return err
				}
			}
		}

		return nil
	}
}

func validateLoginFailBody(validateLoginFailRes *validateFailResponse) ResponseValidator {
	return func(r *http.Response) error {

		// HTTP status codeのチェックで失敗したら続行しない
		if err := IsuAssertStatus(LoginFailStatusCode, r.StatusCode, Hint(PostLogin, "")); err != nil {
			return err
		}

		if err := parseJsonBody(r, validateLoginFailRes); err != nil {
			AdminLogger.Printf("validateLoginFailRes:%#v\n", validateLoginFailRes)
			return failure.NewError(ValidationErrInvalidResponseBody, fmt.Errorf("%s: %v ", PostLogin+" のBodyの Json decodeに失敗しました", err))
		}

		expected := &validateFailResponse{
			StatusCode: LoginFailStatusCode,
			Message:    LoginFailMessage,
		}

		if err := Diff(expected, validateLoginFailRes, Hint(PostLogin, "")); err != nil {
			return err
		}
		return nil
	}
}

func validateLoginSecondUser(validateLoginRes *validateLoginResponse, user *ValidationUser, now time.Time, loginBonusRewardMasters []LoginBonusRewardMaster) ResponseValidator {
	return func(r *http.Response) error {

		// HTTP status codeのチェックで失敗したら続行しない
		if err := IsuAssertStatus(200, r.StatusCode, Hint(PostLogin, "")); err != nil {
			return err
		}

		if err := parseJsonBody(r, validateLoginRes); err != nil {
			AdminLogger.Printf("validateLoginRes:%#v\n", validateLoginRes)
			return failure.NewError(ValidationErrInvalidResponseBody, fmt.Errorf("%s: %v ", PostLogin+" のBodyの Json decodeに失敗しました", err))
		}

		// LoginBonusの存在チェック
		if validateLoginRes.UpdatedResources.UserLoginBonuses == nil {
			return failure.NewError(ValidationErrInvalidResponseBody, fmt.Errorf("%s ", PostLogin+" のBodyの updatedResources.userLoginBonuses がありません"))
		}

		//JSONのデータはログイン前のデータなので１回のログイン分を加算する
		if user.UserLoginBonuses[0].LoginBonusID == 1 {
			// lastLoginBonusSequenceはLoginBonusID == 1 の時のLoop最後のsequence番号
			// LoopCountをカウントアップして、sequenceを１にする
			if user.UserLoginBonuses[0].LastRewardSequence == LastLoginBonusSequence {
				user.UserLoginBonuses[0].LastRewardSequence = 1
				user.UserLoginBonuses[0].LoopCount += 1
			} else {
				user.UserLoginBonuses[0].LastRewardSequence += 1
			}
		}

		//個数のチェク
		if err := IsuAssert(len(user.UserLoginBonuses), len(validateLoginRes.UpdatedResources.UserLoginBonuses), Hint(PostLogin, "userLoginBonuses の個数")); err != nil {
			return err
		}

		// LoginBonusのチェック
		for i, v := range validateLoginRes.UpdatedResources.UserLoginBonuses {
			for _, w := range user.UserLoginBonuses {
				if w.ID != v.ID {
					continue
				}
				w.UpdatedAt = now.Unix()

				if err := Diff(&w, &v, Hint(PostLogin+fmt.Sprintf(" userLoginBonuses[%d].id:%d", i, v.ID), "updatedResources.userLoginBonuses.")); err != nil {
					AdminLogger.Printf("expecteLogiBonus:%#v\n actualLoginBonus:%#v\n", w, v)
					return err
				}
				break
			}
		}

		//ログインボーナス分の加算処理
		loginBonusID := user.UserLoginBonuses[0].LoginBonusID
		loginBonusRewardSequence := user.UserLoginBonuses[0].LastRewardSequence
		var loginRewardItem LoginBonusRewardMaster
		for _, v := range loginBonusRewardMasters {
			if v.LoginBonusID == loginBonusID && v.RewardSequence == loginBonusRewardSequence {
				loginRewardItem = v
				break
			}
		}
		//ログインボーナスのisucoin付与を加算(ItemType == 1の時Isucoin)
		if loginRewardItem.ItemType == 1 {
			user.JsonUser.IsuCoin += loginRewardItem.Amount
		}

		// Userの存在チェック
		if validateLoginRes.UpdatedResources.User == (JsonUser{}) {
			return failure.NewError(ValidationErrInvalidResponseBody, fmt.Errorf("%s ", PostLogin+" のBodyの updatedResources.user がありません"))
		}

		// Userのチェック
		if err := Diff(&user.JsonUser, &validateLoginRes.UpdatedResources.User, Hint(PostLogin+fmt.Sprintf(" id:%d ", user.ID), "updatedResources.user."), IgnoreWhat("lastActivatedAt", "updatedAt")...); err != nil {
			AdminLogger.Printf("expecteUser:%#v\n actualUser:%#v\n", user.JsonUser, validateLoginRes.UpdatedResources.User)
			return err
		}

		return nil
	}
}

func validateLoginUser(validateLoginRes *validateLoginResponse, user *ValidationUser, now time.Time, loginBonusRewardMasters []LoginBonusRewardMaster) ResponseValidator {
	return func(r *http.Response) error {

		// HTTP status codeのチェックで失敗したら続行しない
		if err := IsuAssertStatus(200, r.StatusCode, Hint(PostLogin, "")); err != nil {
			return err
		}

		if err := parseJsonBody(r, validateLoginRes); err != nil {
			AdminLogger.Printf("validateLoginRes:%#v\n", validateLoginRes)
			return failure.NewError(ValidationErrInvalidResponseBody, fmt.Errorf("%s: %v ", PostLogin+" のBodyの Json decodeに失敗しました", err))
		}

		// LoginBonusの存在チェック
		if validateLoginRes.UpdatedResources.UserLoginBonuses == nil {
			return failure.NewError(ValidationErrInvalidResponseBody, fmt.Errorf("%s ", PostLogin+" のBodyの updatedResources.userLoginBonuses がありません"))
		}

		//JSONのデータはログイン前のデータなので１回のログイン分を加算する
		if user.UserLoginBonuses[0].LoginBonusID == 1 {
			// lastLoginBonusSequenceはLoginBonusID == 1 の時のLoop最後のsequence番号
			// LoopCountをカウントアップして、sequenceを１にする
			if user.UserLoginBonuses[0].LastRewardSequence == LastLoginBonusSequence {
				user.UserLoginBonuses[0].LastRewardSequence = 1
				user.UserLoginBonuses[0].LoopCount += 1
			} else {
				user.UserLoginBonuses[0].LastRewardSequence += 1
			}
		}

		//個数のチェク
		if err := IsuAssert(len(user.UserLoginBonuses), len(validateLoginRes.UpdatedResources.UserLoginBonuses), Hint(PostLogin, "userLoginBonuses の個数")); err != nil {
			return err
		}

		// LoginBonusのチェック
		for i, v := range validateLoginRes.UpdatedResources.UserLoginBonuses {
			for _, w := range user.UserLoginBonuses {
				if w.ID != v.ID {
					continue
				}
				w.UpdatedAt = now.Unix()

				if err := Diff(&w, &v, Hint(PostLogin+fmt.Sprintf(" userLoginBonuses[%d].id:%d", i, v.ID), "updatedResources.userLoginBonuses.")); err != nil {
					AdminLogger.Printf("expecteLoginBonus:%#v\n actualLoginBonus:%#v\n", w, v)
					return err
				}
				break
			}
		}

		//ログインボーナス分の加算処理
		loginBonusID := user.UserLoginBonuses[0].LoginBonusID
		loginBonusRewardSequence := user.UserLoginBonuses[0].LastRewardSequence
		var loginRewardItem LoginBonusRewardMaster
		for _, v := range loginBonusRewardMasters {
			if v.LoginBonusID == loginBonusID && v.RewardSequence == loginBonusRewardSequence {
				loginRewardItem = v
				break
			}
		}
		//ログインボーナスのisucoin付与を加算(ItemType == 1の時Isucoin)
		if loginRewardItem.ItemType == 1 {
			user.JsonUser.IsuCoin += loginRewardItem.Amount
		}

		// Userの存在チェック
		if validateLoginRes.UpdatedResources.User == (JsonUser{}) {
			return failure.NewError(ValidationErrInvalidResponseBody, fmt.Errorf("%s ", PostLogin+" のBodyの updatedResources.user がありません"))
		}

		// Userのチェック
		if err := Diff(&user.JsonUser, &validateLoginRes.UpdatedResources.User, Hint(PostLogin+fmt.Sprintf(" id:%d ", user.ID), "updatedResources.user."), IgnoreWhat("lastActivatedAt", "updatedAt")...); err != nil {
			AdminLogger.Printf("expecteUser:%#v\n actualUser:%#v\n", user.JsonUser, validateLoginRes.UpdatedResources.User)
			return err
		}

		// UserPresentsの存在チェック
		if validateLoginRes.UpdatedResources.UserPresents == nil {
			return failure.NewError(ValidationErrInvalidResponseBody, fmt.Errorf("%s ", PostLogin+" のBodyの updatedResources.userPresents がありません"))
		}

		//個数のチェク
		if err := IsuAssert(len(user.UserLoginAppendPresents), len(validateLoginRes.UpdatedResources.UserPresents), Hint(PostLogin, "userPresents の個数")); err != nil {
			return err
		}

		// LoginAppendPresentsのチェック
		for i, v := range user.UserLoginAppendPresents {
			receiveFlag := false
			expectMessage := v.PresentMessage
			for _, w := range validateLoginRes.UpdatedResources.UserPresents {
				if v.PresentMessage != w.PresentMessage {
					continue
				}
				receiveFlag = true
				v.SentAt = now.Unix()
				v.CreatedAt = now.Unix()
				v.UpdatedAt = now.Unix()
				if err := Diff(&v, &w, Hint(PostLogin+fmt.Sprintf(" userPresents[%d].id:%d ", i, v.ID), "updatedResources.userPresents."), IgnoreWhat("id")...); err != nil {
					AdminLogger.Printf("expecteLoginPresents:%#v\n actualLoginPresent:%#v\n", v, w)
					return err
				}

			}
			if !receiveFlag {
				return failure.NewError(ValidationErrInvalidResponseBody, fmt.Errorf("%s ", PostLogin+" のBodyの プレゼントメッセージ"+expectMessage+"のプレゼントが見つかりません"))
			}
		}
		return nil
	}
}

func validateLoginBan(validateLoginBanRes *validateFailResponse, user *User) ResponseValidator {
	return func(r *http.Response) error {

		// HTTP status codeのチェックで失敗したら続行しない
		if err := IsuAssertStatus(LoginBanStatusCode, r.StatusCode, Hint(PostLoginBan, "")); err != nil {
			return err
		}

		if err := parseJsonBody(r, validateLoginBanRes); err != nil {
			AdminLogger.Printf("validateLoginBanRes:%#v\n", validateLoginBanRes)
			return failure.NewError(ValidationErrInvalidResponseBody, fmt.Errorf("%s: %v ", PostLoginBan+" のBodyの Json decodeに失敗しました", err))
		}

		expected := &validateFailResponse{
			StatusCode: LoginBanStatusCode,
			Message:    LoginBanMessage,
		}

		if err := Diff(expected, validateLoginBanRes, Hint(PostLoginBan, "")); err != nil {
			return err
		}

		return nil
	}
}

func validateHomeFail(validateHomeRes *validateFailResponse, user *ValidationUser) ResponseValidator {
	return func(r *http.Response) error {
		endpoint := strings.Replace(GetUserHome, ":userId", fmt.Sprintf("%d", user.ID), 1)
		// HTTP status codeのチェックで失敗したら続行しない
		if err := IsuAssertStatus(LoginUnauthorizedStatusCode, r.StatusCode, Hint(endpoint, "")); err != nil {
			return err
		}
		if err := parseJsonBody(r, validateHomeRes); err != nil {
			AdminLogger.Printf("validateHomeRes:%#v\n", validateHomeRes)
			return failure.NewError(ValidationErrInvalidResponseBody, fmt.Errorf("%s: %v ", endpoint+" のBodyの Json decodeに失敗しました", err))
		}

		expected := &validateFailResponse{
			StatusCode: LoginUnauthorizedStatusCode,
			Message:    LoginUnauthorizedMessage,
		}

		if err := Diff(expected, validateHomeRes, Hint(endpoint, "")); err != nil {
			return err
		}

		return nil
	}
}

func validateHomeResponseCheck(validateHomeRes *validateHomeResponse, user *ValidationUser, loginBonusRewardMasters []LoginBonusRewardMaster, loginRewardItem *LoginBonusRewardMaster, now time.Time) ResponseValidator {
	return func(r *http.Response) error {
		endpoint := strings.Replace(GetUserHome, ":userId", fmt.Sprintf("%d", user.ID), 1)

		// HTTP status codeのチェックで失敗したら続行しない
		if err := IsuAssertStatus(200, r.StatusCode, Hint(endpoint, "")); err != nil {
			return err
		}

		if err := parseJsonBody(r, validateHomeRes); err != nil {
			AdminLogger.Printf("validateHomeRes:%#v\n", validateHomeRes)
			return failure.NewError(ValidationErrInvalidResponseBody, fmt.Errorf("%s: %v ", endpoint+" のBodyの Json decodeに失敗しました", err))
		}

		//ログインボーナス分の加算処理
		loginBonusID := user.UserLoginBonuses[0].LoginBonusID
		loginBonusRewardSequence := user.UserLoginBonuses[0].LastRewardSequence
		for _, v := range loginBonusRewardMasters {
			if v.LoginBonusID == loginBonusID && v.RewardSequence == loginBonusRewardSequence {
				*loginRewardItem = v
				break
			}
		}
		//ログインボーナスのisucoin付与は、すでにLogin時に加算済みのため何もしない(ItemType == 1の時Isucoin)
		//アイテムをここで付与
		if loginRewardItem.ItemType == 3 || loginRewardItem.ItemType == 4 {
			isExist := false
			for i, v := range user.GetItemList {
				if loginRewardItem.ItemID == v.ItemID {
					isExist = true
					user.GetItemList[i].Amount += int(loginRewardItem.Amount)
				}
			}
			if !isExist {
				user.GetItemList = append(user.GetItemList, UserItem{
					ID:        0,
					UserID:    user.ID,
					ItemType:  loginRewardItem.ItemType,
					ItemID:    loginRewardItem.ItemID,
					Amount:    int(loginRewardItem.Amount),
					CreatedAt: now.Unix(),
					UpdatedAt: now.Unix(),
				})
			}
		}

		//userの存在チェック
		if validateHomeRes.User == (JsonUser{}) {
			return failure.NewError(ValidationErrInvalidResponseBody, fmt.Errorf("%s ", endpoint+" のBodyの user がありません"))
		}

		if err := Diff(&user.JsonUser, &validateHomeRes.User, Hint(endpoint, "user."), IgnoreWhat("lastActivatedAt", "updatedAt")...); err != nil {
			AdminLogger.Printf("expecteUser:%#v\n actualUser:%#v\n", user.JsonUser, validateHomeRes.User)
			return err
		}

		//Deckの存在チェック
		if validateHomeRes.Deck == (UserDeck{}) {
			return failure.NewError(ValidationErrInvalidResponseBody, fmt.Errorf("%s ", endpoint+" のBodyの deck がありません"))
		}

		if err := Diff(&user.UserDeck, &validateHomeRes.Deck, Hint(endpoint, "deck."), IgnoreWhat("updatedAt")...); err != nil {
			AdminLogger.Printf("expecteUserDeck:%#v\n actualUserDeck:%#v\n", user.UserDeck, validateHomeRes.Deck)
			return err
		}
		//nowのチェック
		if err := IsuAssert(now.Unix(), validateHomeRes.Now, Hint(endpoint, "now")); err != nil {
			return err
		}

		// pastTimeのチェック
		resPastTime := validateHomeRes.PastTime
		expectPastTime := now.Unix() - validateHomeRes.User.LastGetRewardAt
		if err := IsuAssert(expectPastTime, resPastTime, Hint(endpoint, "pastTime")); err != nil {
			return err
		}

		//totalAmountPerSecのチェック
		expectToatalAmountPerSec := int(user.TotalAmountPerSec)
		if err := IsuAssert(expectToatalAmountPerSec, validateHomeRes.TotalAmountPerSec, Hint(endpoint, "totalAmountPerSec")); err != nil {
			return err
		}
		return nil
	}
}

func validateOtherLogin(validateHomeFailRes *validateFailResponse, user *JsonUser) ResponseValidator {
	return func(r *http.Response) error {
		endpoint := strings.Replace(GetUserHome, ":userId", fmt.Sprintf("%d", user.ID), 1)
		// HTTP status codeのチェックで失敗したら続行しない
		if err := IsuAssertStatus(LoginBanStatusCode, r.StatusCode, Hint(endpoint, "")); err != nil {
			return err
		}
		if err := parseJsonBody(r, validateHomeFailRes); err != nil {
			AdminLogger.Printf("validateHomeFailRes:%#v\n", validateHomeFailRes)
			return failure.NewError(ValidationErrInvalidResponseBody, fmt.Errorf("%s: %v ", endpoint+" のBodyの Json decodeに失敗しました", err))
		}

		expected := &validateFailResponse{
			StatusCode: LoginBanStatusCode,
			Message:    LoginBanMessage,
		}

		if err := Diff(expected, validateHomeFailRes, Hint(endpoint, "")); err != nil {
			return err
		}
		return nil
	}
}

func validateRewardResponseBody(validateRewardRes *validateRewardResponse, user *ValidationUser, now time.Time) ResponseValidator {
	return func(r *http.Response) error {
		endpoint := strings.Replace(PostRewardGet, ":userId", fmt.Sprintf("%d", user.ID), 1)
		// HTTP status codeのチェックで失敗したら続行しない
		if err := IsuAssertStatus(200, r.StatusCode, Hint(endpoint, "")); err != nil {
			return err
		}

		if err := parseJsonBody(r, validateRewardRes); err != nil {
			AdminLogger.Printf("validateRewardRes:%#v\n", validateRewardRes)
			return failure.NewError(ValidationErrInvalidResponseBody, fmt.Errorf("%s: %v ", endpoint+" のBodyの Json decodeに失敗しました", err))
		}

		//Userの存在チェック
		if validateRewardRes.UpdatedResources.User == (JsonUser{}) {
			return failure.NewError(ValidationErrInvalidResponseBody, fmt.Errorf("%s ", endpoint+" のBodyの updatedResources.user がありません"))
		}

		expectedPastTime := validateRewardRes.UpdatedResources.Now - user.JsonUser.LastGetRewardAt
		expectedIsuCoin := user.TotalAmountPerSec*int64(expectedPastTime) + user.JsonUser.IsuCoin

		expected := JsonUser{
			ID:              user.JsonUser.ID,
			IsuCoin:         expectedIsuCoin,
			LastGetRewardAt: now.Unix(),
			LastActivatedAt: user.JsonUser.LastActivatedAt, //チェックしない
			RegisteredAt:    user.JsonUser.RegisteredAt,
			CreatedAt:       user.JsonUser.CreatedAt,
			UpdatedAt:       user.JsonUser.UpdatedAt, //チェックしない
			DeletedAt:       nil,
		}

		if err := Diff(&expected, &validateRewardRes.UpdatedResources.User, Hint(endpoint, "updatedResources.user."), IgnoreWhat("lastActivatedAt", "updatedAt")...); err != nil {
			AdminLogger.Printf("expecteUser:%#v\n actualUser:%#v\n", expected, validateRewardRes.UpdatedResources.User)
			return err
		}

		return nil
	}
}

func validatePostCardResponseBody(validatePostCardRes *validatePostCardResponse, user *ValidationUser, cardIDs []int64, now time.Time) ResponseValidator {
	return func(r *http.Response) error {
		endpoint := strings.Replace(PostCardSet, ":userId", fmt.Sprintf("%d", user.ID), 1)

		// HTTP status codeのチェックで失敗したら続行しない
		if err := IsuAssertStatus(200, r.StatusCode, Hint(endpoint, "")); err != nil {
			return err
		}

		if err := parseJsonBody(r, validatePostCardRes); err != nil {
			AdminLogger.Printf("validatePostCardRes:%#v\n", validatePostCardRes)
			return failure.NewError(ValidationErrInvalidResponseBody, fmt.Errorf("%s: %v ", endpoint+" のBodyの Json decodeに失敗しました", err))
		}

		//UserDecksの存在チェック
		if validatePostCardRes.UpdatedResources.UserDecks == nil {
			return failure.NewError(ValidationErrInvalidResponseBody, fmt.Errorf("%s ", endpoint+" のBodyの updatedResources.userDecks がありません"))
		}

		//個数のチェック(deckは１個固定)
		if err := IsuAssert(1, len(validatePostCardRes.UpdatedResources.UserDecks), Hint(endpoint, "updatedResources.userDecks の個数")); err != nil {
			return err
		}

		expected := UserDeck{
			ID:        0, //不明のため
			UserID:    user.ID,
			CardID1:   cardIDs[0],
			CardID2:   cardIDs[1],
			CardID3:   cardIDs[2],
			CreatedAt: now.Unix(),
			UpdatedAt: now.Unix(),
			DeletedAt: nil,
		}

		if err := Diff(&expected, &validatePostCardRes.UpdatedResources.UserDecks[0], Hint(endpoint, "updatedResources.userDecks."), IgnoreWhat("id")...); err != nil {
			AdminLogger.Printf("expecteUserDeck:%#v\n actualUserDeck:%#v\n", expected, validatePostCardRes.UpdatedResources.UserDecks[0])
			return err
		}

		return nil
	}
}

func validatePostCardAfterHomeResponseBody(validateHomeRes *validateHomeResponse, user *ValidationUser, userCardIds []int64, now time.Time) ResponseValidator {
	return func(r *http.Response) error {
		endpoint := strings.Replace(GetUserHome, ":userId", fmt.Sprintf("%d", user.ID), 1)

		// HTTP status codeのチェックで失敗したら続行しない
		if err := IsuAssertStatus(200, r.StatusCode, Hint(endpoint, "")); err != nil {
			return err
		}

		if err := parseJsonBody(r, validateHomeRes); err != nil {
			AdminLogger.Printf("validateHomeRes:%#v\n", validateHomeRes)
			return failure.NewError(ValidationErrInvalidResponseBody, fmt.Errorf("%s: %v ", endpoint+" のBodyの Json decodeに失敗しました", err))
		}

		//UserDeckの存在チェック
		if validateHomeRes.Deck == (UserDeck{}) {
			return failure.NewError(ValidationErrInvalidResponseBody, fmt.Errorf("%s ", endpoint+" のBodyの deck がありません"))
		}

		// Deck情報を更新
		user.UserDeck.CardID1 = userCardIds[0]
		user.UserDeck.CardID2 = userCardIds[1]
		user.UserDeck.CardID3 = userCardIds[2]

		//DeckからTotalAmountPerSecを導出
		var expetedTotalAmountPerSec int64
		for _, v := range user.UserCards {
			if v.ID == userCardIds[0] || v.ID == userCardIds[1] || v.ID == userCardIds[2] {
				expetedTotalAmountPerSec += int64(v.AmountPerSec)
			}
		}
		//TotalAmountPerSeのチェック
		if err := IsuAssert(expetedTotalAmountPerSec, int64(validateHomeRes.TotalAmountPerSec), Hint(endpoint, "totalAmountPerSec ")); err != nil {
			return err
		}

		if err := Diff(&user.UserDeck, &validateHomeRes.Deck, Hint(endpoint, "deck."), IgnoreWhat("id", "createdAt", "updatedAt")...); err != nil {
			AdminLogger.Printf("expecteUserDeck:%#v\n actualUserDeck:%#v\n", user.UserDeck, validateHomeRes.Deck)
			return err
		}

		return nil
	}
}

func validateItemListResponseBody(validateItemListRes *validateItemListResponse, user *ValidationUser, loginRewardItem *LoginBonusRewardMaster, now time.Time) ResponseValidator {
	return func(r *http.Response) error {

		endpoint := strings.Replace(GetItem, ":userId", fmt.Sprintf("%d", user.ID), 1)

		// HTTP status codeのチェックで失敗したら続行しない
		if err := IsuAssertStatus(200, r.StatusCode, Hint(endpoint, "")); err != nil {
			return err
		}

		if err := parseJsonBody(r, validateItemListRes); err != nil {
			AdminLogger.Printf("validateItemListRes:%#v\n", validateItemListRes)
			return failure.NewError(ValidationErrInvalidResponseBody, fmt.Errorf("%s: %v ", endpoint+" のBodyの Json decodeに失敗しました", err))
		}

		oneTimeToken := validateItemListRes.OneTimeToken

		if len(oneTimeToken) <= 0 {
			return failure.NewError(ValidationErrInvalidResponseBody, fmt.Errorf("%s", endpoint+" のBodyの oneTimeToken がありません"))
		}

		//item個数のチェック
		if err := IsuAssert(len(user.GetItemList), len(validateItemListRes.UserItems), Hint(endpoint, "items の個数")); err != nil {
			return err
		}

		//item
		for _, v := range user.GetItemList {
			for _, w := range validateItemListRes.UserItems {
				if v.ID != w.ID {
					continue
				}
				if err := Diff(&v, &w, Hint(endpoint, "items."), IgnoreWhat("updatedAt")...); err != nil {
					AdminLogger.Printf("expecteGetItemList:%#v\n actualGetItemList:%#v\n", v, w)
					return err
				}
			}
		}

		//cardの個数のチェック
		if err := IsuAssert(len(user.UserCards), len(validateItemListRes.UserCards), Hint(endpoint, "cards の個数")); err != nil {
			return err
		}

		//card
		for _, v := range user.UserCards {
			for _, w := range validateItemListRes.UserCards {
				if v.ID != w.ID {
					continue
				}
				if err := Diff(&v, &w, Hint(endpoint, "cards."), IgnoreWhat("updatedAt")...); err != nil {
					AdminLogger.Printf("expecteUserCards:%#v\n actualUserCards:%#v\n", v, w)
					return err
				}
			}
		}

		return nil
	}
}

func validatePostAddExpCardResponseBody(validatePostAddExpCardRes *validatePostAddExpCardResponse, user *ValidationUser, cardID int64, addCardExpItem []AddCardExpItem, expItemMasters []localExpItemMaster, cardMasters []CardMaster, loginRewardItem *LoginBonusRewardMaster, now time.Time) ResponseValidator {
	return func(r *http.Response) error {
		endpoint := strings.Replace(PostCardAddexp, ":userId", fmt.Sprintf("%d", user.ID), 1)
		endpoint = strings.Replace(endpoint, ":cardId", fmt.Sprintf("%d", cardID), 1)

		// HTTP status codeのチェックで失敗したら続行しない
		if err := IsuAssertStatus(200, r.StatusCode, Hint(endpoint, "")); err != nil {
			return err
		}

		if err := parseJsonBody(r, validatePostAddExpCardRes); err != nil {
			AdminLogger.Printf("validatePostAddExpCardRes:%#v\n", validatePostAddExpCardRes)
			return failure.NewError(ValidationErrInvalidResponseBody, fmt.Errorf("%s: %v ", endpoint+" のBodyの Json decodeに失敗しました", err))
		}

		//UserItemの個数のチェック(使ったアイテムは１個のため)
		if err := IsuAssert(1, len(validatePostAddExpCardRes.UpdatedResources.UserItems), Hint(endpoint, "userItems の個数")); err != nil {
			return err
		}

		userItems := validatePostAddExpCardRes.UpdatedResources.UserItems
		userCards := validatePostAddExpCardRes.UpdatedResources.UserCards

		//使用するアイテムの特定、GainedExpの取得
		var addExpItem localExpItemMaster
		for _, v := range expItemMasters {
			if userItems[0].ItemID == v.ID {
				addExpItem = localExpItemMaster{
					v.ID,
					v.GainedExp,
				}
				break
			}
		}
		//カードの特定
		var addExpCard UserCard
		for _, v := range user.UserCards {
			if v.ID == cardID {
				addExpCard = v
			}
		}
		//カードマスタの取得
		var masterCard CardMaster
		for _, v := range cardMasters {
			if v.CardID == int(addExpCard.CardID) {
				masterCard = v
			}
		}
		// 経験値をカードに付与
		addExpCard.TotalExp += int64(addExpItem.GainedExp) * 1

		// lvup判定(lv upしたら生産性を加算)
		for {
			nextLvThreshold := int64(float64(masterCard.BaseExpPerLevel) * math.Pow(1.2, float64(addExpCard.Level-1)))
			if nextLvThreshold > addExpCard.TotalExp {
				break
			}

			// lv up処理
			addExpCard.Level += 1
			addExpCard.AmountPerSec += (masterCard.MaxAmountPerSec - masterCard.BaseAmountPerSec) / (masterCard.MaxLevel - 1)
		}

		//UserCard
		if err := Diff(&addExpCard, &userCards[0], Hint(endpoint, "updatedResources.userCards."), IgnoreWhat("updatedAt")...); err != nil {
			AdminLogger.Printf("expecteUserCar:%#v\n actualUserCard:%#v\n", addExpCard, userCards[0])
			return err
		}

		//UserItems
		var expectedItem UserItem
		for _, v := range user.GetItemList {
			if v.ID == addCardExpItem[0].ID {
				expectedItem = v
				// 使用したアイテムの個数を引いておく
				expectedItem.Amount -= addCardExpItem[0].Amount
				expectedItem.UpdatedAt = now.Unix()
				break
			}
		}
		if err := Diff(&expectedItem, &userItems[0], Hint(endpoint, "updatedResources.userItems.")); err != nil {
			AdminLogger.Printf("expecteUserItem:%#v\n actualUserItem:%#v\n", expectedItem, userItems[0])
			return err
		}

		return nil
	}
}

func validatePostAddExpCardFailBody(validatePostAddExpFail *validateFailResponse, user *ValidationUser) ResponseValidator {
	return func(r *http.Response) error {
		endpoint := strings.Replace(PostCardAddexp, ":userId", fmt.Sprintf("%d", user.ID), 1)

		// HTTP status codeのチェックで失敗したら続行しない
		if err := IsuAssertStatus(BadRequest, r.StatusCode, Hint(endpoint, "")); err != nil {
			return err
		}
		if err := parseJsonBody(r, validatePostAddExpFail); err != nil {
			AdminLogger.Printf("validatePostAddExpFail:%#v\n", validatePostAddExpFail)
			return failure.NewError(ValidationErrInvalidResponseBody, fmt.Errorf("%s: %v ", endpoint+" のBodyの Json decodeに失敗しました", err))
		}

		expected := &validateFailResponse{
			StatusCode: BadRequest,
			Message:    InvalidToken,
		}

		if err := Diff(expected, validatePostAddExpFail, Hint(endpoint, "")); err != nil {
			return err
		}

		return nil
	}
}

func validatePresentListResponse(listPresentRes *ListPresentResponse, user *ValidationUser) ResponseValidator {
	return func(r *http.Response) error {
		endpoint := strings.Replace(GetPresent, ":userId", fmt.Sprintf("%d", user.ID), 1)

		// HTTP status codeのチェックで失敗したら続行しない
		if err := IsuAssertStatus(200, r.StatusCode, Hint(endpoint, "")); err != nil {
			return err
		}

		if err := parseJsonBody(r, listPresentRes); err != nil {
			AdminLogger.Printf("listPresentRes:%#v\n", listPresentRes)
			return failure.NewError(ValidationErrInvalidResponseBody, fmt.Errorf("%s: %v ", endpoint+" のBodyの Json decodeに失敗しました", err))
		}

		// 個数のチェック
		// user.Presents + user.presentall > 100 以上だったら、100
		expectedLength := len(user.UserPresents) + len(user.UserLoginAppendPresents)
		isNext := false
		if expectedLength > 100 {
			expectedLength = 100
			isNext = true
		}

		if err := IsuAssert(expectedLength, len(listPresentRes.Presents), Hint(endpoint, "presents の個数")); err != nil {
			return err
		}

		// isNextのチェック
		if err := IsuAssert(isNext, listPresentRes.IsNext, Hint(endpoint, "isNext")); err != nil {
			return err
		}

		// ログイン時のプレゼントを順番関係なくプレゼントに積まれているかチェック(メッセージの一致で探す)
		for _, v := range user.UserLoginAppendPresents {
			for _, w := range listPresentRes.Presents[0 : len(user.UserLoginAppendPresents)-1] {
				if v.PresentMessage != w.PresentMessage {
					continue
				}
				if err := Diff(&v, w, Hint(endpoint, "presents."), IgnoreWhat("id", "sentAt", "createdAt", "updatedAt")...); err != nil {
					AdminLogger.Printf("expectLoginPresnts:%#v\n actualLoginPresents:%#v\n", v, w)
					return err
				}
			}
		}

		// user.presntall以降のプレゼントを順番含めてチェック
		// id,userId,sentAt,itemType,itemId,presentMessage,createdAt,updatedAt
		offset := len(user.UserLoginAppendPresents)
		// user.UserPresentsをcreated_atの降順でソートする
		sort.Slice(user.UserPresents, func(i, j int) bool { return user.UserPresents[i].CreatedAt-user.UserPresents[j].CreatedAt > 0 })

		for i, v := range user.UserPresents {
			if err := Diff(&v, listPresentRes.Presents[i+offset], Hint(endpoint, "")); err != nil {
				AdminLogger.Printf("expectUserPresnts:%#v\n actualUserPresents:%#v\n", v, listPresentRes.Presents[i+offset])
				return err
			}
		}
		return nil
	}
}

func validatePostReceivePresent(receivePresentRes *ReceivePresentResponse, user *ValidationUser, receiveUserPresents *[]UserPresent, now time.Time) ResponseValidator {
	return func(r *http.Response) error {

		endpoint := strings.Replace(PostPresentReceive, ":userId", fmt.Sprintf("%d", user.ID), 1)

		// HTTP status codeのチェックで失敗したら続行しない
		if err := IsuAssertStatus(200, r.StatusCode, Hint(endpoint, "")); err != nil {
			return err
		}

		if err := parseJsonBody(r, receivePresentRes); err != nil {
			AdminLogger.Printf("receivePresentRes:%#v\n", receivePresentRes)
			return failure.NewError(ValidationErrInvalidResponseBody, fmt.Errorf("%s: %v ", endpoint+" のBodyの Json decodeに失敗しました", err))
		}

		//個数のチェック
		if err := IsuAssert(len(*receiveUserPresents), len(receivePresentRes.UpdatedResources.UserPresents), Hint(endpoint, "updatedResources.userPresents の個数")); err != nil {
			return err
		}

		isExists := make([]bool, len(*receiveUserPresents))
		for i, v := range *receiveUserPresents {
			for _, w := range receivePresentRes.UpdatedResources.UserPresents {
				if v.ID != w.ID {
					continue
				}
				isExists[i] = true
				v.UpdatedAt = now.Unix()
				if err := Diff(&v, &w, Hint(endpoint, "updatedResources.userPresents."), IgnoreWhat("deletedAt")...); err != nil {
					AdminLogger.Printf("expectUserPresnts:%#v\n actualUserPresents:%#v\n", v, w)
					return err
				}

				//deletedAt
				if w.DeletedAt == nil {
					return failure.NewError(
						ValidationErrInvalidResponseBody,
						fmt.Errorf("%s : expected(%s) != actual(%s)", endpoint+fmt.Sprintf(" updatedResources.userPresents[%d].id:%d", i, w.ID)+" deletedAt が違います",
							"not null",
							"null",
						),
					)
				}
			}
		}
		// 全部あるかチェック
		for i, v := range isExists {
			if !v {
				return failure.NewError(
					ValidationErrInvalidResponseBody,
					fmt.Errorf("%s : id(%d)", endpoint+" updatedResources.userPresents に見つからないプレゼントがあります",
						(*receiveUserPresents)[i].ID,
					),
				)
			}
		}
		return nil
	}
}

func validateItemListAfterReceivePresentResponseBody(validateItemListRes *validateItemListResponse, user *ValidationUser, receiveUserPresents *[]UserPresent, now time.Time) ResponseValidator {
	return func(r *http.Response) error {

		endpoint := strings.Replace(GetItem, ":userId", fmt.Sprintf("%d", user.ID), 1)

		// HTTP status codeのチェックで失敗したら続行しない
		if err := IsuAssertStatus(200, r.StatusCode, Hint(endpoint, "")); err != nil {
			return err
		}

		if err := parseJsonBody(r, validateItemListRes); err != nil {
			AdminLogger.Printf("validateItemListRes:%#v\n", validateItemListRes)
			return failure.NewError(ValidationErrInvalidResponseBody, fmt.Errorf("%s: %v ", endpoint+" のBodyの Json decodeに失敗しました", err))
		}

		for _, v := range *receiveUserPresents {
			if v.ItemType != 1 && v.ItemType != 2 && v.ItemType != 3 && v.ItemType != 4 {
				AdminLogger.Printf("receiveUserPresents:%#v\n", receiveUserPresents)
				return failure.NewError(ValidationErrInvalidResponseBody, fmt.Errorf("%s ", endpoint+" のItemTypeが不正です"))
			}
			if v.ItemType == 1 || v.ItemType == 2 {
				continue
			}
			for _, w := range validateItemListRes.UserItems {
				if v.ItemID != w.ItemID {
					continue
				}
				expected := UserItem{
					ItemType: v.ItemType,
					ItemID:   v.ItemID,
					UserID:   v.UserID,
					Amount:   v.Amount,
				}

				if err := Diff(&expected, &w, Hint(endpoint, "updatedResources.userItems."), IgnoreWhat("id", "amount", "createdAt", "updatedAt", "deletedAt")...); err != nil {
					AdminLogger.Printf("expecteUserItem:%#v\n actualUserItem:%#v\n", expected, w)
					return err
				}

				if v.Amount > w.Amount {
					AdminLogger.Printf("expecteUserItemAmount:%#v\n <= actualUserItemAmout:%#v\n", v.Amount, w.Amount)
					return failure.NewError(ValidationErrInvalidResponseBody, fmt.Errorf("%s: %v ", endpoint+" のBodyの updatedResources.userItems.Amountが違います", v.Amount))
				}
			}
		}
		return nil
	}
}

func validateAfterRecievePresentListResponse(listPresentRes *ListPresentResponse, user *ValidationUser) ResponseValidator {
	return func(r *http.Response) error {

		endpoint := strings.Replace(GetPresent, ":userId", fmt.Sprintf("%d", user.ID), 1)

		// HTTP status codeのチェックで失敗したら続行しない
		if err := IsuAssertStatus(200, r.StatusCode, Hint(endpoint, "")); err != nil {
			return err
		}

		if err := parseJsonBody(r, listPresentRes); err != nil {
			AdminLogger.Printf("listPresentRes:%#v\n", listPresentRes)
			return failure.NewError(ValidationErrInvalidResponseBody, fmt.Errorf("%s: %v ", endpoint+" のBodyの Json decodeに失敗しました", err))
		}

		//個数のチェック  １ページ目は全て受け取り済み
		expectedLength := len(user.UserPresents) + len(user.UserLoginAppendPresents) - 100
		if expectedLength > 100 {
			expectedLength = 100
		} else if expectedLength < 0 {
			expectedLength = 0
		}

		if err := IsuAssert(expectedLength, len(listPresentRes.Presents), Hint(endpoint, "Presents の個数")); err != nil {
			return err
		}

		return nil
	}
}

func validateGachaListResponse(listGachaRes *ListGachaResponse, user *ValidationUser, gachaAllMasters []GachaData, now time.Time) ResponseValidator {
	return func(r *http.Response) error {

		endpoint := strings.Replace(GetGachaIndex, ":userId", fmt.Sprintf("%d", user.ID), 1)

		// HTTP status codeのチェックで失敗したら続行しない
		if err := IsuAssertStatus(200, r.StatusCode, Hint(endpoint, "")); err != nil {
			return err
		}

		if err := parseJsonBody(r, listGachaRes); err != nil {
			AdminLogger.Printf("listGachaRes:%#v\n", listGachaRes)
			return failure.NewError(ValidationErrInvalidResponseBody, fmt.Errorf("%s: %v ", endpoint+" のBodyの Json decodeに失敗しました", err))
		}

		nowUnix := now.Unix()
		var gachaData []GachaData
		for _, v := range gachaAllMasters {
			if v.Gacha.StartAt < nowUnix && nowUnix < v.Gacha.EndAt {
				gachaData = append(gachaData, GachaData{
					Gacha:     v.Gacha,
					GachaItem: v.GachaItem,
				})
			}
		}
		sort.Slice(gachaData, func(i, j int) bool { return gachaData[i].Gacha.DisplayOrder < gachaData[j].Gacha.DisplayOrder })

		//ガチャの開催個数の確認
		if err := IsuAssert(len(gachaData), len(listGachaRes.Gachas), Hint(endpoint, "gachas の個数")); err != nil {
			return err
		}

		for i, v := range gachaData {
			//ガチャの順番の確認
			if err := Diff(&v.Gacha, &listGachaRes.Gachas[i].Gacha, Hint(endpoint, "gachas.gacha.")); err != nil {
				AdminLogger.Printf("expectGacha:%#v\n actualGacha:%#v\n", v.Gacha, listGachaRes.Gachas[i].Gacha)
				return err
			}

			//ガチャの排出アイテム数の個数の確認
			if err := IsuAssert(len(v.GachaItem), len(listGachaRes.Gachas[i].GachaItem), Hint(endpoint, "gachaItemList の個数")); err != nil {
				return err
			}

			//ガチャの排出確率の確認
			// マスタから計算
			expectedTotalWeight := 0
			for _, w := range v.GachaItem {
				expectedTotalWeight += w.Weight
			}
			// レスポンスから計算
			responseTotalWeight := 0
			for _, w := range listGachaRes.Gachas[i].GachaItem {
				responseTotalWeight += w.Weight
			}
			if err := IsuAssert(expectedTotalWeight, responseTotalWeight, Hint(endpoint, "gachaItemList の総排出確率が違います")); err != nil {
				return err
			}

			// 個々の排出アイテムのチェック
			// ID順にソート
			sort.Slice(v.GachaItem, func(i, j int) bool { return v.GachaItem[i].ID < v.GachaItem[j].ID })

			for j, w := range v.GachaItem {
				if err := Diff(&w, &listGachaRes.Gachas[i].GachaItem[j], Hint(endpoint, "gachas.gachaItemList.")); err != nil {
					AdminLogger.Printf("expectGachaItem:%#v\n actualGachaItem:%#v\n", w, listGachaRes.Gachas[i].GachaItem[j])
					return err
				}
			}
		}
		return nil
	}
}

func validateGachaDrawResponse(drawGachaRes *DrawGachaResponse, user *ValidationUser, gachaData GachaData, now time.Time) ResponseValidator {
	return func(r *http.Response) error {

		endpoint := strings.Replace(PostGachaDraw, ":userId", fmt.Sprintf("%d", user.ID), 1)

		// HTTP status codeのチェックで失敗したら続行しない
		if err := IsuAssertStatus(200, r.StatusCode, Hint(endpoint, "")); err != nil {
			return err
		}

		if err := parseJsonBody(r, drawGachaRes); err != nil {
			AdminLogger.Printf("drawGachaRes:%#v\n", drawGachaRes)
			return failure.NewError(ValidationErrInvalidResponseBody, fmt.Errorf("%s: %v ", endpoint+" のBodyの Json decodeに失敗しました", err))
		}

		//個数のチェック(10連引いたので)
		if err := IsuAssert(10, len(drawGachaRes.Present), Hint(endpoint, "presents の個数")); err != nil {
			return err
		}

		var isExist bool
		for i, v := range drawGachaRes.Present {
			isExist = false
			for _, w := range gachaData.GachaItem {
				if v.ItemID != w.ItemID {
					continue
				}
				isExist = true
				expected := UserPresent{
					ID:             0, //不明なため
					UserID:         user.ID,
					SentAt:         now.Unix(),
					ItemType:       w.ItemType,
					ItemID:         w.ItemID,
					Amount:         w.Amount,
					PresentMessage: "", //不明なため
					CreatedAt:      now.Unix(),
					UpdatedAt:      now.Unix(),
					DeletedAt:      nil,
				}
				if err := Diff(&expected, &v, Hint(endpoint, "presents."), IgnoreWhat("id", "presentMessage")...); err != nil {
					AdminLogger.Printf("expectUserPresents:%#v\n actualUserPresents:%#v\n", expected, v)
					return err
				}
				break
			}
			if !isExist {
				return failure.NewError(
					ValidationErrInvalidResponseBody,
					fmt.Errorf("%s : actual(%d)", endpoint+" のBodyの presents["+strconv.Itoa(i)+"].itemId はガチャから排出されません",
						v.ItemID,
					),
				)
			}
		}
		return nil
	}
}

func validatePostRedeemGachaFailBody(validatePostReadeemFail *validateFailResponse, user *ValidationUser) ResponseValidator {
	return func(r *http.Response) error {
		endpoint := strings.Replace(PostGachaDraw, ":userId", fmt.Sprintf("%d", user.ID), 1)

		// HTTP status codeのチェックで失敗したら続行しない
		if err := IsuAssertStatus(BadRequest, r.StatusCode, Hint(endpoint, "")); err != nil {
			return err
		}
		if err := parseJsonBody(r, validatePostReadeemFail); err != nil {
			AdminLogger.Printf("validatePostReadeemFail:%#v\n", validatePostReadeemFail)
			return failure.NewError(ValidationErrInvalidResponseBody, fmt.Errorf("%s: %v ", endpoint+" のBodyの Json decodeに失敗しました", err))
		}

		expected := &validateFailResponse{
			StatusCode: BadRequest,
			Message:    InvalidToken,
		}

		if err := Diff(expected, validatePostReadeemFail, Hint(endpoint, "")); err != nil {
			return err
		}

		return nil
	}
}
