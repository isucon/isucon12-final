package main

import (
	"flag"
	"fmt"
	"log"
	"math/rand"
	"os"
	"strconv"
	"time"

	mapset "github.com/deckarep/golang-set/v2"

	"github.com/google/uuid"
	"github.com/isucon/isucon12-final/dev/extra/initial-data/models"
	"github.com/isucon/isucon12-final/dev/extra/initial-data/random"
)

const (
	DefaultOutDir = "."
	DefaultDSN    = "isucon:isucon@tcp(127.0.0.1:3306)/isucon?parseTime=true&loc=Asia%2FTokyo"
)

type Option struct {
	OutDir      string
	DatabaseDSN string
}

func init() {
	loc, _ := time.LoadLocation("Asia/Tokyo")
	time.Local = loc
	t, _ := time.Parse(time.RFC3339, "2022-08-27T00:00:00+09:00")
	rand.Seed(t.UnixNano())
}

func createRoyalUser(userID int64) models.User {
	var user models.User
	registeredTime := random.OldUserTime().Unix()   // 2-3年前
	activateTime := random.NearOneWeekTime().Unix() // 1週間以内
	isuCoin := rand.Int63n(10000)
	//debug
	if registeredTime > activateTime {
		fmt.Printf("createRoyalUser registeredTime:%d, activateTime:%d\n", registeredTime, activateTime)
		os.Exit(2)
	}
	user = models.NewUser(userID, isuCoin, registeredTime, activateTime)
	user.Create()

	return user
}

func createCombackUser(userID int64) models.User {
	var user models.User

	registeredTime := random.OldUserTime().Unix() // 2-3年前
	activateTime := registeredTime + 60*24*365    //最終アクセスが１年以上前(1-2年前)
	isuCoin := rand.Int63n(10000)
	//debug
	if registeredTime > activateTime {
		fmt.Printf("createCombackUser registeredTime:%d, activateTime:%d\n", registeredTime, activateTime)
		os.Exit(2)
	}

	user = models.NewUser(userID, isuCoin, registeredTime, activateTime)
	user.Create()

	return user
}

func createOneYearlUser(userID int64) models.User {
	var user models.User

	registeredTime := random.OneYearUserTime().Unix() //ユーザー登録１年以内
	activateTime := random.NearOneWeekTime().Unix()
	isuCoin := rand.Int63n(10000)
	//debug
	if registeredTime > activateTime {
		fmt.Printf("createOneYearlUser registeredTime:%d, activateTime:%d\n", registeredTime, activateTime)
		os.Exit(2)
	}

	user = models.NewUser(userID, isuCoin, registeredTime, activateTime)
	user.Create()

	return user
}

func createUserDevice(userDeviceID int64, user models.User) ([]models.UserDevice, int) {
	max := 30000000
	min := 100000
	var userDevices []models.UserDevice
	platformID := max - rand.Intn(max-min)
	userDevice := models.NewUserDevice(userDeviceID, user, strconv.Itoa(platformID))
	userDevice.Create()

	userDevices = append(userDevices, userDevice)

	if rand.Int()%2 == 0 {
		platformUuid, _ := uuid.NewRandom()
		platformType := rand.Intn(2) + 2
		userOtherDeviceID := int64(rand.Int31())
		userOtherDevice := models.NewUserDeviceOther(userOtherDeviceID, user, platformUuid.String(), platformType)
		userOtherDevice.Create()
		userDevices = append(userDevices, userOtherDevice)
	}
	return userDevices, platformID
}

func createuserLoginBonus(bonusID int, user models.User, loginBonusMasters []models.LoginBonusMaster) models.UserLoginBonus {
again:
	// login_bonus_masterのType 1のデータ
	userLoginBonusID := generateID()
	var rewardSequence int
	var loopCount int

	if bonusID == 1 {
		// (registerdTime - activateTime)/24%(column_count) +1 = sequence
		rewardSequence = int(user.LastActivatedAt-user.RegisteredAt)/60/24%loginBonusMasters[0].ColumnCount + 1
		// (registerdTime - activateTime)/24/(column_count) = loop_count
		loopCount = int(user.LastActivatedAt-user.RegisteredAt) / 60 / 24 / loginBonusMasters[0].ColumnCount
	} else if bonusID == 2 {
		rewardSequence = 7
		loopCount = 1
	} else if bonusID == 4 {
		rewardSequence = 7
		loopCount = 1
	} else {
		fmt.Println("createuserLoginBonus: undefined bonusID", bonusID)
		return models.UserLoginBonus{}
	}
	userLoginBonus := models.NewUserLoginBonus(
		userLoginBonusID,
		user,
		bonusID,
		loopCount,
		rewardSequence)
	if err := userLoginBonus.Create(); err != nil {
		fmt.Println("retry:createuserLoginBonus")
		goto again
	}

	return userLoginBonus

}

func createUserPresentReceivedBulk(user models.User, presentCount int) []models.UserPresent {
again:

	var userPresents []models.UserPresent
	for j := 0; j < presentCount; j++ {
		//TODO sentAtからGachaIDをきめる
		gachaID := 1
		gachaResult := models.DrawGacha(gachaID)
		sentAt := random.OldUserTime().Unix()
		presentMessage := "ガチャ取得アイテムです"
		userPresentID := generateID()
		userPresent := models.NewUserPresent(
			userPresentID,
			user,
			sentAt,
			gachaResult.ItemType,
			gachaResult.ItemID,
			gachaResult.Amount,
			presentMessage)
		userPresents = append(userPresents, userPresent)
	}
	if err := models.PresentReceivedBulkCreate(&userPresents); err != nil {
		fmt.Println("retry:PresentReceivedBulkCreate")
		goto again
	}
	return userPresents
}

func createUserPresentBulk(user models.User, presentCount int) []models.UserPresent {
again:
	var userPresents []models.UserPresent
	for j := 0; j < presentCount; j++ {
		//TODO sentAtからGachaIDをきめる
		gachaID := 37
		gachaResult := models.DrawGacha(gachaID)
		userPresentID := generateID()
		sentAt := random.NearOneWeekTime().Unix()
		presentMessage := "ガチャ取得アイテムです"
		userPresent := models.NewUserPresent(
			userPresentID,
			user,
			sentAt,
			gachaResult.ItemType,
			gachaResult.ItemID,
			gachaResult.Amount,
			presentMessage)
		userPresents = append(userPresents, userPresent)
	}
	if err := models.PresentBulkCreate(&userPresents); err != nil {
		fmt.Println("retry:PresentBulkCreate")
		goto again
	}

	return userPresents
}

func createUserCardBulk(userPresentsAll []models.UserPresent, user models.User) ([]models.UserCard, int64) {
again:
	var userCards []models.UserCard
	var totalAmountPerSec int64

	totalAmountPerSec = 0
	//初期必須データの作成
	for i := 0; i < 3; i++ {
		userCardID := generateID()
		cardID := 2
		totalExp := rand.Int63n(10000) + 10
		cardMaster := models.GetCardMaster(cardID)
		level, amountPerSec := models.GetCardLevelAndAmountPerSec(cardMaster, totalExp)

		userCard := models.NewUserCard(
			userCardID,
			user,
			cardID,
			amountPerSec,
			level,
			totalExp)
		userCards = append(userCards, userCard)
		totalAmountPerSec = totalAmountPerSec + int64(amountPerSec)

	}
	//プレゼントからの受け取り分
	for _, v := range userPresentsAll {
		if v.ItemType == 2 {
			userCardID := generateID()
			cardID := v.ItemID
			totalExp := rand.Int63n(10000) + 10

			cardMaster := models.GetCardMaster(cardID)
			level, amountPerSec := models.GetCardLevelAndAmountPerSec(cardMaster, totalExp)
			userCard := models.NewUserCard(
				userCardID,
				user,
				cardID,
				amountPerSec,
				level,
				totalExp)
			userCards = append(userCards, userCard)
		}
	}
	if err := models.UserCardBulkCreate(&userCards); err != nil {
		fmt.Println("retry:UserCardBulkCreate")
		goto again
	}

	return userCards, totalAmountPerSec
}

func createUserItem(userPresentsAll []models.UserPresent, user models.User) []models.UserItem {
again:
	//return用
	var userItems []models.UserItem
	//sum処理
	userKyokaItems := map[int]int{}
	userJitanItems := map[int]int{}
	for _, v := range userPresentsAll {
		if v.ItemType == 3 {
			userKyokaItems[v.ItemID] += v.Amount
		} else if v.ItemType == 4 {
			userJitanItems[v.ItemID] += v.Amount
		}
	}

	for key, value := range userKyokaItems {
		userItemID := generateID()
		itemType := 3
		itemID := key
		amount := value
		userItem := models.NewUserItem(
			userItemID,
			user,
			itemType,
			itemID,
			amount,
			user.RegisteredAt)
		userItems = append(userItems, userItem)
	}
	for key, value := range userJitanItems {
		userItemID := generateID()
		itemType := 4
		itemID := key
		amount := value
		userItem := models.NewUserItem(
			userItemID,
			user,
			itemType,
			itemID,
			amount,
			user.RegisteredAt)
		userItems = append(userItems, userItem)
	}
	if err := models.UserItemBulkCreate(&userItems); err != nil {
		fmt.Println("retry:UserItemBulkCreate")
		goto again
	}

	return userItems
}

func createUserPresentAllReceivedHistoryBulk(user models.User) {
again:
	var userPresentAllReceivedHistories []models.UserPresentAllReceivedHistory
	for _, v := range models.UserPresentAllMasters {
		userPresentAllID := generateID()
		userPresentAllReceivedHistory := models.NewUserPresentAllReceivedHistory(
			userPresentAllID,
			user,
			int64(v.ID),
			v.CreatedAt,
			v.CreatedAt)
		userPresentAllReceivedHistories = append(userPresentAllReceivedHistories, userPresentAllReceivedHistory)
	}
	if err := models.UserPresentAllReceivedHistoryBulkCreate(&userPresentAllReceivedHistories); err != nil {
		fmt.Println("retry:UserPresentAllReceivedHistoryBulkCreate")
		goto again
	}
}

func createUserPresentHalfReceivedHistoryBulk(user models.User) ([]models.UserPresentAllReceivedHistory, []models.UserPresent) {
again:
	var appendUserPresents []models.UserPresent
	var userPresentAllReceivedHistories []models.UserPresentAllReceivedHistory
	for _, v := range models.UserPresentAllMasters {
		if v.ID > len(models.UserPresentAllMasters)/2 {
			appendUserPresents = append(appendUserPresents,
				models.NewUserPresent(
					0,
					user,
					time.Now().Unix(),
					v.ItemType,
					v.ItemID,
					v.Amount,
					v.PresentMessage))
		} else {
			userPresentAllID := generateID()
			userPresentAllReceivedHistory := models.NewUserPresentAllReceivedHistory(
				userPresentAllID,
				user,
				int64(v.ID),
				v.CreatedAt,
				v.CreatedAt)
			userPresentAllReceivedHistories = append(userPresentAllReceivedHistories, userPresentAllReceivedHistory)
		}
	}
	if err := models.UserPresentAllReceivedHistoryBulkCreate(&userPresentAllReceivedHistories); err != nil {
		fmt.Println("retry:UserPresentAllReceivedHistoryBulkCreate")
		goto again
	}

deleteAgain:
	// deletedAtがあるデータを作成
	userPresentAllReceivedHistory := userPresentAllReceivedHistories[0]
	userPresentAllReceivedHistory.ID += -1
	userPresentAllReceivedHistory.DeletedAt = &userPresentAllReceivedHistory.UpdatedAt
	if err := userPresentAllReceivedHistory.CreateDeleted(); err != nil {
		fmt.Println("retry:userPresentAllReceivedHistory.CreateDeleted")
		goto deleteAgain
	}

	userPresentAllReceivedHistories = append(userPresentAllReceivedHistories, userPresentAllReceivedHistory)
	return userPresentAllReceivedHistories, appendUserPresents
}

func createUserBan(user models.User) {
	userBanID := generateID()
	createdAt := user.CreatedAt + 10*60
	userBan := models.NewUserBan(
		userBanID,
		user.ID,
		createdAt,
	)
	userBan.Create()
}

func generateID() int64 {
	max := int64(99999999999)
	min := int64(1000000) //100万以上
	return max - rand.Int63n(max-min)
}

func generateUserID() int64 {
	max := int64(9999999999)
	min := int64(2000000) //200万以上
	return max - rand.Int63n(max-min)
}

func generateUserDeviceID() int64 {
	max := int64(9999999999)
	min := int64(2000000) //200万以上
	return max - rand.Int63n(max-min)
}

func main() {
	option := Option{}
	flag.StringVar(&option.OutDir, "out-dir", DefaultOutDir, "Output directory")
	flag.Parse()

	var makeCount int
	var err error
	if len(flag.Args()) != 1 {
		makeCount = 33
	} else {
		makeCount, err = strconv.Atoi(flag.Args()[0])
		if err != nil {
			fmt.Println("Invalid number")
			return
		}
	}

	jsonArray := models.JsonArray{}

	loginBonusMasters := models.GetLoginBonusMasters()

	models.InitCardMaster()
	models.InitUserPresentAllMaster()
	models.InitAllGachaItemMaster()

	createRoyalUserCount := makeCount
	createCombackUserCount := makeCount
	createOneYearUserCount := makeCount
	createBanUserCount := makeCount
	createValidateUserCount := makeCount / 3

	//ロイヤルユーザの作成
	for i := 0; i < createRoyalUserCount; i++ {
		userID := generateUserID()
		userDeviceID := generateUserDeviceID()
		var userPresentsAll []models.UserPresent
		var platformID int

		// user作成
		user := createRoyalUser(userID)

		// userdevice作成
		_, platformID = createUserDevice(userDeviceID, user)

		//user_login_bonuses
		var bonusID int
		bonusID = 1
		createuserLoginBonus(bonusID, user, loginBonusMasters)

		// login_bonus_masterのType 2のデータ
		// すべてうけとりずみ
		bonusID = 2
		createuserLoginBonus(bonusID, user, loginBonusMasters)

		// login_bonus_masterのType 4のデータ
		// すべてうけとりずみ
		bonusID = 4
		createuserLoginBonus(bonusID, user, loginBonusMasters)

		//user_presents
		// login_bonuseのデータをうけとりずみ（deletedAt済み）でいれる
		//受け取り済み
		var presentCount int

		presentCount = rand.Intn(30) + 600
		userPresentsAll = createUserPresentReceivedBulk(user, presentCount)

		//未受け取り
		presentCount = rand.Intn(5) + 20
		createUserPresentBulk(user, presentCount)

		//上記データをもとに下記のデータを作成
		//user_cards
		var userCards []models.UserCard

		userCards, _ = createUserCardBulk(userPresentsAll, user)

		//user_items
		createUserItem(userPresentsAll, user)

		//user_decks
		userDeck := models.NewUserDeck(
			userID,
			user,
			userCards[0].ID,
			userCards[1].ID,
			userCards[2].ID,
			user.RegisteredAt)
		userDeck.Create()

		//user_preesnt_all_receive_history　全部受け取っている状態
		createUserPresentAllReceivedHistoryBulk(user)

		jsonData := models.Json{
			UserType: "ロイヤル",
			UserID:   userID,
			ViewerID: strconv.Itoa(platformID),
		}
		jsonArray = append(jsonArray, &jsonData)
	}
	if err := jsonArray.Commit(option.OutDir); err != nil {
		log.Fatal("file output error!")
	}

	if err := jsonArray.Rename("royalUserInitialize.json"); err != nil {
		log.Fatal("file rename error!")
	}

	//カムバックユーザの作成
	jsonArray = models.JsonArray{}
	for i := 0; i < createCombackUserCount; i++ {

		userID := generateUserID()
		userDeviceID := generateUserDeviceID()
		var userPresentsAll []models.UserPresent
		var platformID int

		// user作成
		user := createCombackUser(userID)
		// userdevice作成
		_, platformID = createUserDevice(userDeviceID, user)

		//user_login_bonuses
		var bonusID int
		bonusID = 1
		createuserLoginBonus(bonusID, user, loginBonusMasters)

		// login_bonus_masterのType 2のデータ
		// すべてうけとりずみ
		bonusID = 2
		createuserLoginBonus(bonusID, user, loginBonusMasters)

		// login_bonus_masterのType 4のデータ
		// すべてうけとりずみ
		bonusID = 4
		createuserLoginBonus(bonusID, user, loginBonusMasters)

		//user_presents
		// login_bonuseのデータをうけとりずみ（deletedAt済み）でいれる
		//受け取り済み
		var presentCount int

		presentCount = rand.Intn(30) + 300
		userPresentsAll = createUserPresentReceivedBulk(user, presentCount)

		//未受け取り
		presentCount = rand.Intn(5) + 20
		createUserPresentBulk(user, presentCount)

		//上記データをもとに下記のデータを作成
		//user_cards
		var userCards []models.UserCard

		userCards, _ = createUserCardBulk(userPresentsAll, user)

		//user_items
		createUserItem(userPresentsAll, user)

		//user_decks
		userDeck := models.NewUserDeck(
			userID,
			user,
			userCards[0].ID,
			userCards[1].ID,
			userCards[2].ID,
			user.RegisteredAt)
		userDeck.Create()

		//user_preesnt_all_receive_history 半分だけ受け取っている状態
		createUserPresentHalfReceivedHistoryBulk(user)

		jsonData := models.Json{
			UserType: "カムバック",
			UserID:   userID,
			ViewerID: strconv.Itoa(platformID),
		}
		jsonArray = append(jsonArray, &jsonData)
	}
	if err := jsonArray.Commit(option.OutDir); err != nil {
		log.Fatal("file output error!")
	}

	if err := jsonArray.Rename("combackUserInitialize.json"); err != nil {
		log.Fatal("file rename error!")
	}

	//一年以内のユーザ
	jsonArray = models.JsonArray{}
	for i := 0; i < createOneYearUserCount; i++ {

		userID := generateUserID()
		userDeviceID := generateUserDeviceID()
		var userPresentsAll []models.UserPresent
		var platformID int

		//新規ユーザ
		// user作成
		user := createOneYearlUser(userID)
		// userdevice作成
		_, platformID = createUserDevice(userDeviceID, user)

		//user_login_bonuses
		var bonusID int
		bonusID = 1
		createuserLoginBonus(bonusID, user, loginBonusMasters)

		// login_bonus_masterのType 2のデータ
		// すべてうけとりずみ
		bonusID = 2
		createuserLoginBonus(bonusID, user, loginBonusMasters)

		// login_bonus_masterのType 4のデータ
		// すべてうけとりずみ
		bonusID = 4
		createuserLoginBonus(bonusID, user, loginBonusMasters)

		//user_presents
		// login_bonuseのデータをうけとりずみ（deletedAt済み）でいれる
		//受け取り済み
		var presentCount int

		presentCount = rand.Intn(10) + 100
		userPresentsAll = createUserPresentReceivedBulk(user, presentCount)

		//未受け取り
		presentCount = rand.Intn(3) + 10
		createUserPresentBulk(user, presentCount)

		//上記データをもとに下記のデータを作成
		//user_cards
		var userCards []models.UserCard

		userCards, _ = createUserCardBulk(userPresentsAll, user)

		//user_items
		createUserItem(userPresentsAll, user)

		//user_decks
		{
			userDeck := models.NewUserDeck(
				userID,
				user,
				userCards[0].ID,
				userCards[1].ID,
				userCards[2].ID,
				user.RegisteredAt)
			userDeck.Create()
		}

		//user_preesnt_all_receive_history
		createUserPresentAllReceivedHistoryBulk(user)

		jsonData := models.Json{
			UserType: "一年以内のユーザ",
			UserID:   userID,
			ViewerID: strconv.Itoa(platformID),
		}
		jsonArray = append(jsonArray, &jsonData)
	}

	if err := jsonArray.Commit(option.OutDir); err != nil {
		log.Fatal("file output error!")
	}

	if err := jsonArray.Rename("oneYearUserInitialize.json"); err != nil {
		log.Fatal("file rename error!")
	}

	//banユーザの作成
	jsonArray = models.JsonArray{}
	for i := 0; i < createBanUserCount; i++ {
		userID := generateUserID()
		userDeviceID := generateUserDeviceID()
		//新規ユーザ
		var user models.User
		var platformID int
		// user作成
		user = createOneYearlUser(userID)
		// userdevice作成
		_, platformID = createUserDevice(userDeviceID, user)
		// user_banの作成
		createUserBan(user)

		jsonData := models.Json{
			UserType: "banユーザ",
			UserID:   userID,
			ViewerID: strconv.Itoa(platformID),
		}
		jsonArray = append(jsonArray, &jsonData)
	}
	if err := jsonArray.Commit(option.OutDir); err != nil {
		log.Fatal("file output error!")
	}

	if err := jsonArray.Rename("banUserInitialize.json"); err != nil {
		log.Fatal("file rename error!")
	}

	//Validationユーザの作成
	jsonValidates := models.JsonValidates{}
	for i := 0; i < createValidateUserCount; i++ {
		userID := generateUserID()
		userDeviceID := generateUserDeviceID()
		// var userAllPresents []models.UserPresent
		var platformID int
		var user models.User
		var userDevices []models.UserDevice

		// user作成
		// user = createRoyalUser(userID)
		user = createCombackUser(userID)

		// userdevice作成
		userDevices, platformID = createUserDevice(userDeviceID, user)

		//user_login_bonuses
		var bonusID int
		bonusID = 1
		validateLoginBonus := createuserLoginBonus(bonusID, user, loginBonusMasters)

		// login_bonus_masterのType 2のデータ
		// すべてうけとりずみ
		bonusID = 2
		createuserLoginBonus(bonusID, user, loginBonusMasters)

		// login_bonus_masterのType 4のデータ
		// すべてうけとりずみ
		bonusID = 4
		createuserLoginBonus(bonusID, user, loginBonusMasters)

		//user_presents
		// login_bonuseのデータをうけとりずみ（deletedAt済み）でいれる
		//受け取り済み
		var presentCount int

		presentCount = rand.Intn(10) + 100
		userAllPresents := createUserPresentReceivedBulk(user, presentCount)

		//未受け取り
		presentCount = rand.Intn(10) + 50
		userPresents := createUserPresentBulk(user, presentCount)

		//上記データをもとに下記のデータを作成
		//user_cards
		var userCards []models.UserCard
		var totalAmountPerSec int64
		var userItems []models.UserItem

		userCards, totalAmountPerSec = createUserCardBulk(userAllPresents, user)

		//user_items
		userItems = createUserItem(userAllPresents, user)

		//user_decks
		userDeck := models.NewUserDeck(
			userID,
			user,
			userCards[0].ID,
			userCards[1].ID,
			userCards[2].ID,
			user.RegisteredAt)
		userDeck.Create()

		//user_preesnt_all_receive_history 半分だけ受け取っている状態
		userPresentAllReceiveHistories, appendUserPresents := createUserPresentHalfReceivedHistoryBulk(user)

		//受け取り済みに変更する
		for i := range userAllPresents {
			userAllPresents[i].DeletedAt = &userAllPresents[i].UpdatedAt
		}

		//未受け取り、受け取り済みを含む全てのプレゼントを格納
		userAllPresents = append(userAllPresents, userPresents...)

		jsonData := models.JsonValidate{
			UserType:                       "validate",
			UserID:                         userID,
			ViewerID:                       strconv.Itoa(platformID),
			UserLoginBonuses:               []models.UserLoginBonus{validateLoginBonus},
			UserLoginAppendPresents:        appendUserPresents,
			JsonUser:                       user,
			UserDeck:                       userDeck,
			UserDevices:                    userDevices,
			TotalAmountPerSec:              totalAmountPerSec,
			GetItemList:                    userItems,
			UserCards:                      userCards,
			UserPresents:                   userPresents,
			UserAllPresents:                userAllPresents,
			UserPresentAllReceiveHistories: userPresentAllReceiveHistories,
		}
		jsonValidates = append(jsonValidates, &jsonData)
	}
	if err := jsonValidates.Commit(option.OutDir); err != nil {
		log.Fatal("file output error!")
	}

	if err := jsonValidates.Rename("validateUserInitialize.json"); err != nil {
		log.Fatal("file rename error!")
	}

	//ExpItemMasterの書き出し
	expItemMasters := models.GetExpItemMaster()
	if err := expItemMasters.Commit(option.OutDir); err != nil {
		log.Fatal("file output error!")
	}

	if err := expItemMasters.Rename("expItemMaster.json"); err != nil {
		log.Fatal("file rename error!")
	}

	cardMasters := models.GetCardMasters()
	//cardMasterの書き出し
	if err := cardMasters.Commit(option.OutDir); err != nil {
		log.Fatal("file output error!")
	}

	if err := cardMasters.Rename("cardMaster.json"); err != nil {
		log.Fatal("file rename error!")
	}

	loginBonusRewardMasters := models.GetLoginBonusRewardMaster()
	//cardMasterの書き出し
	if err := loginBonusRewardMasters.Commit(option.OutDir); err != nil {
		log.Fatal("file output error!")
	}

	if err := loginBonusRewardMasters.Rename("loginBonusRewardMaster.json"); err != nil {
		log.Fatal("file rename error!")
	}

	gachaAllMasters := models.GetGachaAllItemMasters()
	//gahcaItemsMastersの書き出し
	if err := gachaAllMasters.Commit(option.OutDir); err != nil {
		log.Fatal("file output error!")
	}

	if err := gachaAllMasters.Rename("gachaAllItemMaster.json"); err != nil {
		log.Fatal("file rename error!")
	}
	presentAllMasters := models.GetPresentAllMasters()
	//presentAllMastersの書き出し
	if err := presentAllMasters.Commit(option.OutDir); err != nil {
		log.Fatal("file output error!")
	}

	if err := presentAllMasters.Rename("presentAllMaster.json"); err != nil {
		log.Fatal("file rename error!")
	}

	// register用のViewerIDを作成 //2000000 以下
	count := 50000
	orgPlatformIds := mapset.NewSet[int64]()
	for i := 0; i < count; i++ {
		n := rand.Int63n(1999999)
		orgPlatformIds.Add(n)
	}
	var platforms models.JsonPlatform
	for _, v := range orgPlatformIds.ToSlice() {
		platforms = append(platforms, &models.Platform{
			ID:   v,
			Type: 1,
		})
	}
	// register用のViewerIDの書き出し
	if err := platforms.Commit(option.OutDir); err != nil {
		log.Fatal("file output error!")
	}

	if err := platforms.Rename("platforms.json"); err != nil {
		log.Fatal("file rename error!")
	}

}
