package main

// リクエストを送る動作 "Action" を中心に集約しているファイル

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/isucon/isucandar/agent"
)

const (
	VersionMasterPath    string = "./resource/version_master.csv"
	PresentAllMasterPath string = "./resource/present_all_master.csv"
)

// POST /initialize にリクエストを送る
func PostInitializeAction(ctx context.Context, agent *agent.Agent) (*http.Response, error) {
	req, err := agent.POST("/initialize", nil)
	if err != nil {
		return nil, err
	}

	setContentType(req)

	return agent.Do(ctx, req)
}

// POST /login にリクエストを送る
func PostLoginAction(ctx context.Context, agent *agent.Agent, userID int64, viewerID string, masterVersion string, xIsuDate time.Time) (*http.Response, error) {
	json, err := json.Marshal(LoginRequest{UserID: userID, ViewerID: viewerID})
	if err != nil {
		// このエラーは実装上の問題でエラーになるはずなので、もし送出される場合は何かがおかしい。
		return nil, err
	}

	req, err := agent.POST("/login", bytes.NewBuffer(json))
	setMasterVersion(req, masterVersion)
	if err != nil {
		return nil, err
	}
	setIsuDate(req, masterVersion, xIsuDate)
	setContentType(req)

	return agent.Do(ctx, req)
}

// POST /user にリクエストを送る
func PostUserAction(ctx context.Context, agent *agent.Agent, platform *Platform, masterVersion string, xIsuDate time.Time) (*http.Response, error) {
	json, err := json.Marshal(CreateUserRequest{
		ViewerID:     strconv.FormatInt(platform.ID, 10),
		PlatformType: platform.Type,
	})
	if err != nil {
		return nil, err
	}

	req, err := agent.POST("/user", bytes.NewBuffer(json))
	if err != nil {
		return nil, err
	}
	setMasterVersion(req, masterVersion)
	setIsuDate(req, masterVersion, xIsuDate)
	setContentType(req)

	return agent.Do(ctx, req)
}

// POST /user/{userId}/reward にリクエストを送る
func PostReward(ctx context.Context, agent *agent.Agent, userID int64, masterVersion string, xIsuDate time.Time, login *Login) (*http.Response, error) {
	json, err := json.Marshal(RewardRequest{
		ViewerID: login.ViewerID,
	})
	if err != nil {
		return nil, err
	}

	req, err := agent.POST("/user/"+strconv.FormatInt(userID, 10)+"/reward", bytes.NewBuffer(json))
	if err != nil {
		return nil, err
	}
	setMasterVersion(req, masterVersion)
	setSession(req, login.SessionID)
	setIsuDate(req, masterVersion, xIsuDate)
	setContentType(req)

	return agent.Do(ctx, req)
}

// GET /user/:userId/home にリクエストを送る
func GetHome(ctx context.Context, agent *agent.Agent, userID int64, masterVersion string, xIsuDate time.Time, login *Login) (*http.Response, error) {
	req, err := agent.GET("/user/" + fmt.Sprint(userID) + "/home")
	setMasterVersion(req, masterVersion)
	setSession(req, login.SessionID)
	if err != nil {
		return nil, err
	}
	setIsuDate(req, masterVersion, xIsuDate)
	setContentType(req)

	return agent.Do(ctx, req)
}

// GET /user/:userId/item にリクエストを送る
func GetItemList(ctx context.Context, agent *agent.Agent, userID int64, masterVersion string, xIsuDate time.Time, login *Login) (*http.Response, error) {
	req, err := agent.GET("/user/" + fmt.Sprint(userID) + "/item")
	setMasterVersion(req, masterVersion)
	setSession(req, login.SessionID)
	if err != nil {
		return nil, err
	}
	setIsuDate(req, masterVersion, xIsuDate)
	setContentType(req)

	return agent.Do(ctx, req)
}

// GET /user/:userId/present/index/:n にリクエストを送る
func GetPresentList(ctx context.Context, agent *agent.Agent, userID int64, masterVersion string, xIsuDate time.Time, login *Login) (*http.Response, error) {
	req, err := agent.GET("/user/" + fmt.Sprint(userID) + "/present/index/" + fmt.Sprint(1))
	if err != nil {
		return nil, err
	}
	setMasterVersion(req, masterVersion)
	setSession(req, login.SessionID)
	setIsuDate(req, masterVersion, xIsuDate)
	setContentType(req)

	return agent.Do(ctx, req)
}

// POST /user/:userId/card にリクエストを送る
func PostCard(ctx context.Context, agent *agent.Agent, userID int64, cardIds []int64, masterVersion string, xIsuDate time.Time, login *Login) (*http.Response, error) {
	json, err := json.Marshal(PostCardRequest{
		CardIDs:  cardIds,
		ViewerID: login.ViewerID,
	})
	if err != nil {
		// このエラーは実装上の問題でエラーになるはずなので、もし送出される場合は何かがおかしい。
		return nil, err
	}

	req, err := agent.POST("/user/"+fmt.Sprint(userID)+"/card", bytes.NewBuffer(json))
	if err != nil {
		return nil, err
	}
	setMasterVersion(req, masterVersion)
	setSession(req, login.SessionID)
	setIsuDate(req, masterVersion, xIsuDate)
	setContentType(req)

	return agent.Do(ctx, req)
}

// POST /user/:userId/card/addexp/:cardId にリクエストを送る
func PostAddExpCard(ctx context.Context, agent *agent.Agent, userID int64, cardID int64, addCardExpItem []AddCardExpItem, oneTimeToken string, masterVersion string, xIsuDate time.Time, login *Login) (*http.Response, error) {
	json, err := json.Marshal(PostAddCardExpRequest{
		Items:        addCardExpItem,
		OneTimeToken: oneTimeToken,
		ViewerID:     login.ViewerID,
	})
	if err != nil {
		// このエラーは実装上の問題でエラーになるはずなので、もし送出される場合は何かがおかしい。
		return nil, err
	}

	req, err := agent.POST("/user/"+fmt.Sprint(userID)+"/card/addexp/"+fmt.Sprint(cardID), bytes.NewBuffer(json))
	if err != nil {
		return nil, err
	}
	setMasterVersion(req, masterVersion)
	setSession(req, login.SessionID)
	setIsuDate(req, masterVersion, xIsuDate)
	setContentType(req)

	return agent.Do(ctx, req)
}

// POST /user/:userId/present/receive にリクエストを送る
func PostReceivePresent(ctx context.Context, agent *agent.Agent, userID int64, presentIDs []int64, masterVersion string, xIsuDate time.Time, login *Login) (*http.Response, error) {
	json, err := json.Marshal(ReceivePresentRequest{
		ViewerID:   login.ViewerID,
		PresentIDs: presentIDs,
	})
	if err != nil {
		// このエラーは実装上の問題でエラーになるはずなので、もし送出される場合は何かがおかしい。
		return nil, err
	}

	req, err := agent.POST("/user/"+fmt.Sprint(userID)+"/present/receive", bytes.NewBuffer(json))
	if err != nil {
		return nil, err
	}
	setMasterVersion(req, masterVersion)
	setSession(req, login.SessionID)
	setIsuDate(req, masterVersion, xIsuDate)
	setContentType(req)

	return agent.Do(ctx, req)
}

// GET /user/:userId/gacha/index にリクエストを送る
func GetGachaList(ctx context.Context, agent *agent.Agent, userID int64, masterVersion string, xIsuDate time.Time, login *Login) (*http.Response, error) {
	req, err := agent.GET("/user/" + fmt.Sprint(userID) + "/gacha/index")
	if err != nil {
		return nil, err
	}
	setMasterVersion(req, masterVersion)
	setSession(req, login.SessionID)
	setIsuDate(req, masterVersion, xIsuDate)
	setContentType(req)

	return agent.Do(ctx, req)
}

// POST /user/:userId/gacha/draw/:gachaTypeId にリクエストを送る
func PostRedeemGacha(ctx context.Context, agent *agent.Agent, userID int64, masterVersion string, xIsuDate time.Time, gachaTypeID int, oneTimeToken string, login *Login) (*http.Response, error) {
	json, err := json.Marshal(DrawGachaRequest{
		ViewerID:     login.ViewerID,
		OneTimeToken: oneTimeToken,
	})
	if err != nil {
		// このエラーは実装上の問題でエラーになるはずなので、もし送出される場合は何かがおかしい。
		return nil, err
	}

	req, err := agent.POST("/user/"+fmt.Sprint(userID)+"/gacha/draw/"+fmt.Sprint(gachaTypeID)+"/10", bytes.NewBuffer(json))
	if err != nil {
		return nil, err
	}
	setMasterVersion(req, masterVersion)
	setSession(req, login.SessionID)
	setIsuDate(req, masterVersion, xIsuDate)
	setContentType(req)

	return agent.Do(ctx, req)
}

func PostAdminLoginAction(ctx context.Context, agent *agent.Agent, adminUser *AdminUser, masterVersion string) (*http.Response, error) {
	json, err := json.Marshal(AdminUserLoginRequest{
		UserID:   adminUser.ID,
		Password: adminUser.Password,
	})
	if err != nil {
		return nil, err
	}

	req, err := agent.POST("/admin/login", bytes.NewBuffer(json))
	if err != nil {
		return nil, err
	}
	setMasterVersion(req, masterVersion)
	setContentType(req)

	return agent.Do(ctx, req)
}

func PutRefreshMasterData(ctx context.Context, agent *agent.Agent, sessionID string, masterVersion string) (*http.Response, error) {
	body := new(bytes.Buffer)

	versionMasterFile, err := os.Open(VersionMasterPath)
	if err != nil {
		ContestantLogger.Printf("マスター更新に必要な「%s」がありません。エラー: %v", VersionMasterPath, err)
		AdminLogger.Printf("マスター更新に必要な「%s」がありません。ベンチマーカー側の異常なので、運営で確認が必要です。エラー: %+v", VersionMasterPath, err)
		// この場合、運営の Slack 等に強制的に通知したいので処理全体を異常終了させる。
		os.Exit(1)
	}
	defer versionMasterFile.Close()

	presentAllMasterFile, err := os.Open(PresentAllMasterPath)
	if err != nil {
		ContestantLogger.Printf("マスター更新に必要な「%s」がありません。エラー: %v", PresentAllMasterPath, err)
		AdminLogger.Printf("マスター更新に必要な「%s」がありません。ベンチマーカー側の異常なので、運営で確認が必要です。エラー: %+v", PresentAllMasterPath, err)
		// この場合、運営の Slack 等に強制的に通知したいので処理全体を異常終了させる。
		os.Exit(1)
	}
	defer presentAllMasterFile.Close()

	mw := multipart.NewWriter(body)

	part, err := mw.CreateFormFile("versionMaster", filepath.Base(versionMasterFile.Name()))
	if err != nil {
		return nil, err
	}
	_, err = io.Copy(part, versionMasterFile)
	if err != nil {
		return nil, err
	}

	part, err = mw.CreateFormFile("presentAllMaster", filepath.Base(presentAllMasterFile.Name()))
	if err != nil {
		return nil, err
	}
	_, err = io.Copy(part, presentAllMasterFile)
	if err != nil {
		return nil, err
	}

	boundary := mw.FormDataContentType()

	mw.Close()

	req, err := agent.PUT("/admin/master", body)
	if err != nil {
		return nil, err
	}

	req.Header.Add("Content-Type", boundary)
	setMasterVersion(req, masterVersion)
	setSession(req, sessionID)

	return agent.Do(ctx, req)
}

func DeleteAdminLogoutAction(ctx context.Context, agent *agent.Agent, sessionID string, masterVersion string) (*http.Response, error) {
	req, err := agent.DELETE("/admin/logout", nil)
	if err != nil {
		return nil, err
	}

	setMasterVersion(req, masterVersion)
	setSession(req, sessionID)
	setContentType(req)

	return agent.Do(ctx, req)
}

func PostAdminUserBanAction(ctx context.Context, agent *agent.Agent, sessionID string, masterVersion string, userID int64) (*http.Response, error) {
	req, err := agent.POST("/admin/user/"+fmt.Sprint(userID)+"/ban", nil)

	if err != nil {
		return nil, err
	}
	setMasterVersion(req, masterVersion)
	setSession(req, sessionID)
	setContentType(req)

	return agent.Do(ctx, req)
}

func GetAdminMasterAction(ctx context.Context, agent *agent.Agent, sessionID string, masterVersion string) (*http.Response, error) {
	req, err := agent.GET("/admin/master")

	if err != nil {
		return nil, err
	}
	setMasterVersion(req, masterVersion)
	setSession(req, sessionID)
	setContentType(req)

	return agent.Do(ctx, req)
}

func GetAdminUserAction(ctx context.Context, agent *agent.Agent, sessionID string, masterVersion string, userID int64) (*http.Response, error) {
	req, err := agent.GET("/admin/user/" + strconv.FormatInt(userID, 10))

	if err != nil {
		return nil, err
	}
	setMasterVersion(req, masterVersion)
	setSession(req, sessionID)
	setContentType(req)

	return agent.Do(ctx, req)
}

func setMasterVersion(req *http.Request, masterVersion string) {
	req.Header.Set("x-master-version", masterVersion)
}

func setSession(req *http.Request, sessionID string) {
	req.Header.Set("x-session", sessionID)
}

func setContentType(req *http.Request) {
	req.Header.Set("Content-Type", "application/json")
}

func setIsuDate(req *http.Request, masterVersion string, xIsuDate time.Time) {
	if xIsuDate == *new(time.Time) {
		xIsuDate = time.Now()
	}
	intMasterVersion, err := strconv.Atoi(masterVersion)
	if err == nil {
		if intMasterVersion >= 2 {
			xIsuDate = xIsuDate.Add(24 * time.Hour) // １日加算する
		}
	}
	req.Header.Set("x-isu-date", xIsuDate.Format(time.RFC1123))
}
