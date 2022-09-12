package main

import (
	"bytes"
	"database/sql"
	"encoding/csv"
	"io"
	"net/http"
	"strings"

	"github.com/labstack/echo/v4"
	"golang.org/x/crypto/bcrypt"
)

// //////////////////////////////////////
// admin

// adminSessionCheckMiddleware 管理者ツール向けのセッション確認middleware
func (h *Handler) adminSessionCheckMiddleware(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		sessID := c.Request().Header.Get("x-session")

		adminSession := new(Session)
		query := "SELECT * FROM admin_sessions WHERE session_id=? AND deleted_at IS NULL"
		if err := h.DB.Get(adminSession, query, sessID); err != nil {
			if err == sql.ErrNoRows {
				return errorResponse(c, http.StatusUnauthorized, ErrUnauthorized)
			}
			return errorResponse(c, http.StatusInternalServerError, err)
		}

		requestAt, err := getRequestTime(c)
		if err != nil {
			return errorResponse(c, http.StatusInternalServerError, ErrGetRequestTime)
		}

		if adminSession.ExpiredAt < requestAt {
			query = "UPDATE admin_sessions SET deleted_at=? WHERE session_id=?"
			if _, err = h.DB.Exec(query, requestAt, sessID); err != nil {
				return errorResponse(c, http.StatusInternalServerError, err)
			}
			return errorResponse(c, http.StatusUnauthorized, ErrExpiredSession)
		}

		if err := next(c); err != nil {
			c.Error(err)
		}
		return nil
	}
}

// adminLogin 管理者権限ログイン
// POST /admin/login
func (h *Handler) adminLogin(c echo.Context) error {
	defer c.Request().Body.Close()
	req := new(AdminLoginRequest)
	if err := parseRequestBody(c, req); err != nil {
		return errorResponse(c, http.StatusBadRequest, err)
	}

	requestAt, err := getRequestTime(c)
	if err != nil {
		return errorResponse(c, http.StatusInternalServerError, ErrGetRequestTime)
	}

	tx, err := h.DB.Beginx()
	if err != nil {
		return errorResponse(c, http.StatusInternalServerError, err)
	}
	defer tx.Rollback() //nolint:errcheck

	query := "SELECT * FROM admin_users WHERE id=?"
	user := new(AdminUser)
	if err = tx.Get(user, query, req.UserID); err != nil {
		if err == sql.ErrNoRows {
			return errorResponse(c, http.StatusNotFound, ErrUserNotFound)
		}
		return errorResponse(c, http.StatusInternalServerError, err)
	}

	if err = verifyPassword(user.Password, req.Password); err != nil {
		return errorResponse(c, http.StatusUnauthorized, err)
	}

	query = "UPDATE admin_users SET last_activated_at=?, updated_at=? WHERE id=?"
	if _, err = tx.Exec(query, requestAt, requestAt, req.UserID); err != nil {
		return errorResponse(c, http.StatusInternalServerError, err)
	}

	query = "UPDATE admin_sessions SET deleted_at=? WHERE user_id=? AND deleted_at IS NULL"
	if _, err = tx.Exec(query, requestAt, req.UserID); err != nil {
		return errorResponse(c, http.StatusInternalServerError, err)
	}

	sID, err := h.generateID()
	if err != nil {
		return errorResponse(c, http.StatusInternalServerError, err)
	}
	sessID, err := generateUUID()
	if err != nil {
		return errorResponse(c, http.StatusInternalServerError, err)
	}
	sess := &Session{
		ID:        sID,
		UserID:    req.UserID,
		SessionID: sessID,
		CreatedAt: requestAt,
		UpdatedAt: requestAt,
		ExpiredAt: requestAt + 86400,
	}

	query = "INSERT INTO admin_sessions(id, user_id, session_id, created_at, updated_at, expired_at) VALUES (?, ?, ?, ?, ?, ?)"
	if _, err = tx.Exec(query, sess.ID, sess.UserID, sess.SessionID, sess.CreatedAt, sess.UpdatedAt, sess.ExpiredAt); err != nil {
		return errorResponse(c, http.StatusInternalServerError, err)
	}

	err = tx.Commit()
	if err != nil {
		return errorResponse(c, http.StatusInternalServerError, err)
	}

	return successResponse(c, &AdminLoginResponse{
		AdminSession: sess,
	})
}

type AdminLoginRequest struct {
	UserID   int64  `json:"userId"`
	Password string `json:"password"`
}

type AdminLoginResponse struct {
	AdminSession *Session `json:"session"`
}

// adminLogout 管理者権限ログアウト
// DELETE /admin/logout
func (h *Handler) adminLogout(c echo.Context) error {
	sessID := c.Request().Header.Get("x-session")

	requestAt, err := getRequestTime(c)
	if err != nil {
		return errorResponse(c, http.StatusInternalServerError, ErrGetRequestTime)
	}

	query := "UPDATE admin_sessions SET deleted_at=? WHERE session_id=? AND deleted_at IS NULL"
	if _, err = h.DB.Exec(query, requestAt, sessID); err != nil {
		return errorResponse(c, http.StatusInternalServerError, err)
	}

	return noContentResponse(c, http.StatusNoContent)
}

// adminListMaster マスタデータ閲覧
// GET /admin/master
func (h *Handler) adminListMaster(c echo.Context) error {
	masterVersions := make([]*VersionMaster, 0)
	if err := h.DB.Select(&masterVersions, "SELECT * FROM version_masters"); err != nil {
		return errorResponse(c, http.StatusInternalServerError, err)
	}

	items := make([]*ItemMaster, 0)
	if err := h.DB.Select(&items, "SELECT * FROM item_masters"); err != nil {
		return errorResponse(c, http.StatusInternalServerError, err)
	}

	gachas := make([]*GachaMaster, 0)
	if err := h.DB.Select(&gachas, "SELECT * FROM gacha_masters"); err != nil {
		return errorResponse(c, http.StatusInternalServerError, err)
	}

	gachaItems := make([]*GachaItemMaster, 0)
	if err := h.DB.Select(&gachaItems, "SELECT * FROM gacha_item_masters"); err != nil {
		return errorResponse(c, http.StatusInternalServerError, err)
	}

	presentAlls := make([]*PresentAllMaster, 0)
	if err := h.DB.Select(&presentAlls, "SELECT * FROM present_all_masters"); err != nil {
		return errorResponse(c, http.StatusInternalServerError, err)

	}

	loginBonuses := make([]*LoginBonusMaster, 0)
	if err := h.DB.Select(&loginBonuses, "SELECT * FROM login_bonus_masters"); err != nil {
		return errorResponse(c, http.StatusInternalServerError, err)

	}

	loginBonusRewards := make([]*LoginBonusRewardMaster, 0)
	if err := h.DB.Select(&loginBonusRewards, "SELECT * FROM login_bonus_reward_masters"); err != nil {
		return errorResponse(c, http.StatusInternalServerError, err)
	}

	return successResponse(c, &AdminListMasterResponse{
		VersionMaster:     masterVersions,
		Items:             items,
		Gachas:            gachas,
		GachaItems:        gachaItems,
		PresentAlls:       presentAlls,
		LoginBonuses:      loginBonuses,
		LoginBonusRewards: loginBonusRewards,
	})
}

type AdminListMasterResponse struct {
	VersionMaster     []*VersionMaster          `json:"versionMaster"`
	Items             []*ItemMaster             `json:"items"`
	Gachas            []*GachaMaster            `json:"gachas"`
	GachaItems        []*GachaItemMaster        `json:"gachaItems"`
	PresentAlls       []*PresentAllMaster       `json:"presentAlls"`
	LoginBonusRewards []*LoginBonusRewardMaster `json:"loginBonusRewards"`
	LoginBonuses      []*LoginBonusMaster       `json:"loginBonuses"`
}

// adminUpdateMaster マスタデータ更新
// PUT /admin/master
func (h *Handler) adminUpdateMaster(c echo.Context) error {
	tx, err := h.DB.Beginx()
	if err != nil {
		return errorResponse(c, http.StatusInternalServerError, err)
	}
	defer tx.Rollback() //nolint:errcheck

	// version master
	versionMasterRecs, err := readFormFileToCSV(c, "versionMaster")
	if err != nil {
		if err != ErrNoFormFile {
			return errorResponse(c, http.StatusBadRequest, err)
		}
	}
	if versionMasterRecs != nil {
		data := []map[string]interface{}{}
		for i, v := range versionMasterRecs {
			if i == 0 {
				continue
			}
			data = append(data, map[string]interface{}{
				"id":             v[0],
				"status":         v[1],
				"master_version": v[2],
			})
		}

		query := "INSERT INTO version_masters(id, status, master_version) VALUES (:id, :status, :master_version) ON DUPLICATE KEY UPDATE status=VALUES(status), master_version=VALUES(master_version)"
		if _, err = tx.NamedExec(query, data); err != nil {
			return errorResponse(c, http.StatusInternalServerError, err)
		}
	} else {
		c.Logger().Debug("Skip Update Master: versionMaster")
	}

	// item
	itemMasterRecs, err := readFormFileToCSV(c, "itemMaster")
	if err != nil {
		if err != ErrNoFormFile {
			return errorResponse(c, http.StatusBadRequest, err)
		}
	}
	if itemMasterRecs != nil {
		data := []map[string]interface{}{}
		for i, v := range itemMasterRecs {
			if i == 0 {
				continue
			}
			data = append(data, map[string]interface{}{
				"id":                 v[0],
				"item_type":          v[1],
				"name":               v[2],
				"description":        v[3],
				"amount_per_sec":     v[4],
				"max_level":          v[5],
				"max_amount_per_sec": v[6],
				"base_exp_per_level": v[7],
				"gained_exp":         v[8],
				"shortening_min":     v[9],
			})
		}

		query := strings.Join([]string{
			"INSERT INTO item_masters(id, item_type, name, description, amount_per_sec, max_level, max_amount_per_sec, base_exp_per_level, gained_exp, shortening_min)",
			"VALUES (:id, :item_type, :name, :description, :amount_per_sec, :max_level, :max_amount_per_sec, :base_exp_per_level, :gained_exp, :shortening_min)",
			"ON DUPLICATE KEY UPDATE item_type=VALUES(item_type), name=VALUES(name), description=VALUES(description), amount_per_sec=VALUES(amount_per_sec), max_level=VALUES(max_level), max_amount_per_sec=VALUES(max_amount_per_sec), base_exp_per_level=VALUES(base_exp_per_level), gained_exp=VALUES(gained_exp), shortening_min=VALUES(shortening_min)",
		}, " ")
		if _, err = tx.NamedExec(query, data); err != nil {
			return errorResponse(c, http.StatusInternalServerError, err)
		}
	} else {
		c.Logger().Debug("Skip Update Master: itemMaster")
	}

	// gacha
	gachaRecs, err := readFormFileToCSV(c, "gachaMaster")
	if err != nil {
		if err != ErrNoFormFile {
			return errorResponse(c, http.StatusBadRequest, err)
		}
	}
	if gachaRecs != nil {
		data := []map[string]interface{}{}
		for i, v := range gachaRecs {
			if i == 0 {
				continue
			}
			data = append(data, map[string]interface{}{
				"id":            v[0],
				"name":          v[1],
				"start_at":      v[2],
				"end_at":        v[3],
				"display_order": v[4],
				"created_at":    v[5],
			})
		}

		query := strings.Join([]string{
			"INSERT INTO gacha_masters(id, name, start_at, end_at, display_order, created_at)",
			"VALUES (:id, :name, :start_at, :end_at, :display_order, :created_at)",
			"ON DUPLICATE KEY UPDATE name=VALUES(name), start_at=VALUES(start_at), end_at=VALUES(end_at), display_order=VALUES(display_order), created_at=VALUES(created_at)",
		}, " ")
		if _, err = tx.NamedExec(query, data); err != nil {
			return errorResponse(c, http.StatusInternalServerError, err)
		}
	} else {
		c.Logger().Debug("Skip Update Master: gachaMaster")
	}

	// gacha item
	gachaItemRecs, err := readFormFileToCSV(c, "gachaItemMaster")
	if err != nil {
		if err != ErrNoFormFile {
			return errorResponse(c, http.StatusBadRequest, err)
		}
	}
	if gachaItemRecs != nil {
		data := []map[string]interface{}{}
		for i, v := range gachaItemRecs {
			if i == 0 {
				continue
			}
			data = append(data, map[string]interface{}{
				"id":         v[0],
				"gacha_id":   v[1],
				"item_type":  v[2],
				"item_id":    v[3],
				"amount":     v[4],
				"weight":     v[5],
				"created_at": v[6],
			})
		}

		query := strings.Join([]string{
			"INSERT INTO gacha_item_masters(id, gacha_id, item_type, item_id, amount, weight, created_at)",
			"VALUES (:id, :gacha_id, :item_type, :item_id, :amount, :weight, :created_at)",
			"ON DUPLICATE KEY UPDATE gacha_id=VALUES(gacha_id), item_type=VALUES(item_type), item_id=VALUES(item_id), amount=VALUES(amount), weight=VALUES(weight), created_at=VALUES(created_at)",
		}, " ")
		if _, err = tx.NamedExec(query, data); err != nil {
			return errorResponse(c, http.StatusInternalServerError, err)
		}
	} else {
		c.Logger().Debug("Skip Update Master: gachaItemMaster")
	}

	// present all
	presentAllRecs, err := readFormFileToCSV(c, "presentAllMaster")
	if err != nil {
		if err != ErrNoFormFile {
			return errorResponse(c, http.StatusBadRequest, err)
		}
	}
	if presentAllRecs != nil {
		data := []map[string]interface{}{}
		for i, v := range presentAllRecs {
			if i == 0 {
				continue
			}
			data = append(data, map[string]interface{}{
				"id":                  v[0],
				"registered_start_at": v[1],
				"registered_end_at":   v[2],
				"item_type":           v[3],
				"item_id":             v[4],
				"amount":              v[5],
				"present_message":     v[6],
				"created_at":          v[7],
			})
		}

		query := strings.Join([]string{
			"INSERT INTO present_all_masters(id, registered_start_at, registered_end_at, item_type, item_id, amount, present_message, created_at)",
			"VALUES (:id, :registered_start_at, :registered_end_at, :item_type, :item_id, :amount, :present_message, :created_at)",
			"ON DUPLICATE KEY UPDATE registered_start_at=VALUES(registered_start_at), registered_end_at=VALUES(registered_end_at), item_type=VALUES(item_type), item_id=VALUES(item_id), amount=VALUES(amount), present_message=VALUES(present_message), created_at=VALUES(created_at)",
		}, " ")
		if _, err = tx.NamedExec(query, data); err != nil {
			return errorResponse(c, http.StatusInternalServerError, err)
		}
	} else {
		c.Logger().Debug("Skip Update Master: presentAllMaster")
	}

	// login bonuses
	loginBonusRecs, err := readFormFileToCSV(c, "loginBonusMaster")
	if err != nil {
		if err != ErrNoFormFile {
			return errorResponse(c, http.StatusBadRequest, err)
		}
	}
	if loginBonusRecs != nil {
		data := []map[string]interface{}{}
		for i, v := range loginBonusRecs {
			if i == 0 {
				continue
			}
			looped := 0
			if v[4] == "TRUE" {
				looped = 1
			}
			data = append(data, map[string]interface{}{
				"id":           v[0],
				"start_at":     v[1],
				"end_at":       v[2],
				"column_count": v[3],
				"looped":       looped,
				"created_at":   v[5],
			})
		}

		query := strings.Join([]string{
			"INSERT INTO login_bonus_masters(id, start_at, end_at, column_count, looped, created_at)",
			"VALUES (:id, :start_at, :end_at, :column_count, :looped, :created_at)",
			"ON DUPLICATE KEY UPDATE start_at=VALUES(start_at), end_at=VALUES(end_at), column_count=VALUES(column_count), looped=VALUES(looped), created_at=VALUES(created_at)",
		}, " ")
		if _, err = tx.NamedExec(query, data); err != nil {
			return errorResponse(c, http.StatusInternalServerError, err)
		}
	} else {
		c.Logger().Debug("Skip Update Master: loginBonusMaster")
	}

	// login bonus rewards
	loginBonusRewardRecs, err := readFormFileToCSV(c, "loginBonusRewardMaster")
	if err != nil {
		if err != ErrNoFormFile {
			return errorResponse(c, http.StatusBadRequest, err)
		}
	}
	if loginBonusRewardRecs != nil {
		data := []map[string]interface{}{}
		for i, v := range loginBonusRewardRecs {
			if i == 0 {
				continue
			}
			data = append(data, map[string]interface{}{
				"id":              v[0],
				"login_bonus_id":  v[1],
				"reward_sequence": v[2],
				"item_type":       v[3],
				"item_id":         v[4],
				"amount":          v[5],
				"created_at":      v[6],
			})
		}

		query := strings.Join([]string{
			"INSERT INTO login_bonus_reward_masters(id, login_bonus_id, reward_sequence, item_type, item_id, amount, created_at)",
			"VALUES (:id, :login_bonus_id, :reward_sequence, :item_type, :item_id, :amount, :created_at)",
			"ON DUPLICATE KEY UPDATE login_bonus_id=VALUES(login_bonus_id), reward_sequence=VALUES(reward_sequence), item_type=VALUES(item_type), item_id=VALUES(item_id), amount=VALUES(amount), created_at=VALUES(created_at)",
		}, " ")
		if _, err = tx.NamedExec(query, data); err != nil {
			return errorResponse(c, http.StatusInternalServerError, err)
		}
	} else {
		c.Logger().Debug("Skip Update Master: loginBonusRewardMaster")
	}

	activeMaster := new(VersionMaster)
	if err = tx.Get(activeMaster, "SELECT * FROM version_masters WHERE status=1"); err != nil {
		return errorResponse(c, http.StatusInternalServerError, err)
	}

	err = tx.Commit()
	if err != nil {
		return errorResponse(c, http.StatusInternalServerError, err)
	}

	return successResponse(c, &AdminUpdateMasterResponse{
		VersionMaster: activeMaster,
	})
}

type AdminUpdateMasterResponse struct {
	VersionMaster *VersionMaster `json:"versionMaster"`
}

// readFromFileToCSV ファイルからcsvレコードを取得する
func readFormFileToCSV(c echo.Context, name string) ([][]string, error) {
	file, err := c.FormFile(name)
	if err != nil {
		return nil, ErrNoFormFile
	}

	src, err := file.Open()
	if err != nil {
		return nil, err
	}
	defer src.Close()

	buf := new(bytes.Buffer)
	if _, err = io.Copy(buf, src); err != nil {
		return nil, err
	}

	csvReader := csv.NewReader(bytes.NewReader(buf.Bytes()))
	records, err := csvReader.ReadAll()
	if err != nil {
		return nil, err
	}

	return records, nil
}

// adminUser ユーザの詳細画面
// GET /admin/user/{userID}
func (h *Handler) adminUser(c echo.Context) error {
	userID, err := getUserID(c)
	if err != nil {
		return errorResponse(c, http.StatusBadRequest, err)
	}

	query := "SELECT * FROM users WHERE id=?"
	user := new(User)
	if err = h.DB.Get(user, query, userID); err != nil {
		if err == sql.ErrNoRows {
			return errorResponse(c, http.StatusNotFound, ErrUserNotFound)
		}
		return errorResponse(c, http.StatusInternalServerError, err)
	}

	query = "SELECT * FROM user_devices WHERE user_id=?"
	devices := make([]*UserDevice, 0)
	if err = h.DB.Select(&devices, query, userID); err != nil {
		return errorResponse(c, http.StatusInternalServerError, err)
	}

	query = "SELECT * FROM user_cards WHERE user_id=?"
	cards := make([]*UserCard, 0)
	if err = h.DB.Select(&cards, query, userID); err != nil {
		return errorResponse(c, http.StatusInternalServerError, err)
	}

	query = "SELECT * FROM user_decks WHERE user_id=?"
	decks := make([]*UserDeck, 0)
	if err = h.DB.Select(&decks, query, userID); err != nil {
		return errorResponse(c, http.StatusInternalServerError, err)
	}

	query = "SELECT * FROM user_items WHERE user_id=?"
	items := make([]*UserItem, 0)
	if err = h.DB.Select(&items, query, userID); err != nil {
		return errorResponse(c, http.StatusInternalServerError, err)
	}

	query = "SELECT * FROM user_login_bonuses WHERE user_id=?"
	loginBonuses := make([]*UserLoginBonus, 0)
	if err = h.DB.Select(&loginBonuses, query, userID); err != nil {
		return errorResponse(c, http.StatusInternalServerError, err)
	}

	query = "SELECT * FROM user_presents WHERE user_id=?"
	presents := make([]*UserPresent, 0)
	if err = h.DB.Select(&presents, query, userID); err != nil {
		return errorResponse(c, http.StatusInternalServerError, err)
	}

	query = "SELECT * FROM user_present_all_received_history WHERE user_id=?"
	presentHistory := make([]*UserPresentAllReceivedHistory, 0)
	if err = h.DB.Select(&presentHistory, query, userID); err != nil {
		return errorResponse(c, http.StatusInternalServerError, err)
	}

	return successResponse(c, &AdminUserResponse{
		User:                          user,
		UserDevices:                   devices,
		UserCards:                     cards,
		UserDecks:                     decks,
		UserItems:                     items,
		UserLoginBonuses:              loginBonuses,
		UserPresents:                  presents,
		UserPresentAllReceivedHistory: presentHistory,
	})
}

type AdminUserResponse struct {
	User *User `json:"user"`

	UserDevices                   []*UserDevice                    `json:"userDevices"`
	UserCards                     []*UserCard                      `json:"userCards"`
	UserDecks                     []*UserDeck                      `json:"userDecks"`
	UserItems                     []*UserItem                      `json:"userItems"`
	UserLoginBonuses              []*UserLoginBonus                `json:"userLoginBonuses"`
	UserPresents                  []*UserPresent                   `json:"userPresents"`
	UserPresentAllReceivedHistory []*UserPresentAllReceivedHistory `json:"userPresentAllReceivedHistory"`
}

// adminBanUser ユーザBAN処理
// POST /admin/user/{userId}/ban
func (h *Handler) adminBanUser(c echo.Context) error {
	userID, err := getUserID(c)
	if err != nil {
		return errorResponse(c, http.StatusBadRequest, err)
	}

	requestAt, err := getRequestTime(c)
	if err != nil {
		return errorResponse(c, http.StatusInternalServerError, ErrGetRequestTime)
	}

	query := "SELECT * FROM users WHERE id=?"
	user := new(User)
	if err = h.DB.Get(user, query, userID); err != nil {
		if err == sql.ErrNoRows {
			return errorResponse(c, http.StatusBadRequest, ErrUserNotFound)
		}
		return errorResponse(c, http.StatusInternalServerError, err)
	}

	banID, err := h.generateID()
	if err != nil {
		return errorResponse(c, http.StatusInternalServerError, err)
	}
	query = "INSERT user_bans(id, user_id, created_at, updated_at) VALUES (?, ?, ?, ?) ON DUPLICATE KEY UPDATE updated_at = ?"
	if _, err = h.DB.Exec(query, banID, userID, requestAt, requestAt, requestAt); err != nil {
		return errorResponse(c, http.StatusInternalServerError, err)
	}

	return successResponse(c, &AdminBanUserResponse{
		User: user,
	})
}

type AdminBanUserResponse struct {
	User *User `json:"user"`
}

// hashPassword パスワードをハッシュ化する
//nolint:deadcode,unused
func hashPassword(pw string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(pw), bcrypt.DefaultCost)
	if err != nil {
		return "", ErrGeneratePassword
	}
	return string(hash), nil
}

// verifyPassword ハッシュ化されたパスワードを検証する
func verifyPassword(hash, pw string) error {
	if err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(pw)); err != nil {
		return ErrUnauthorized
	}
	return nil
}

type AdminUser struct {
	ID              int64  `db:"id"`
	Password        string `db:"password"`
	LastActivatedAt int64  `db:"last_activated_at"`
	CreatedAt       int64  `db:"created_at"`
	UpdatedAt       int64  `db:"updated_at"`
	DeletedAt       *int64 `db:"deleted_at"`
}
