package main

import (
	"github.com/isucon/isucandar"
	"github.com/isucon/isucandar/score"
)

// スコアタグ情報を集約するファイル

const (
	// ユーザーログインシナリオ
	ScoreLoginSuccess           score.ScoreTag = "POST /login"
	ScoreShowHomeViewSuccess    score.ScoreTag = "GET /user/:userId/home"
	ScoreShowPresentListSuccess score.ScoreTag = "GET /user/:userId/present/index/:n"
	ScoreReceivePresentSuccess  score.ScoreTag = "POST /user/:userId/present/receive"
	ScoreRedeemRewardSuccess    score.ScoreTag = "POST /user/:userId/reward"
	ScoreShowGachaListSuccess   score.ScoreTag = "GET /user/:userId/gacha/index"
	ScoreRedeemGachaSuccess     score.ScoreTag = "POST /user/:userId/gacha/draw/:gachaId"

	// ユーザー作成シナリオ
	ScoreCreateUser  score.ScoreTag = "POST /user"
	ScoreShowItems   score.ScoreTag = "GET /user/:userId/item"
	ScorePostAddCard score.ScoreTag = "POST /user/:userId/card/addexp/:cardId"
	ScoreSetDeck     score.ScoreTag = "POST /user/:userId/card"

	// ユーザーBanシナリオ
	ScoreBanLogin score.ScoreTag = "POST /login(ban)"
)

var ScoreRateTable = map[score.ScoreTag]int64{
	ScoreLoginSuccess:           3,
	ScoreShowHomeViewSuccess:    1,
	ScoreShowPresentListSuccess: 3,
	ScoreShowGachaListSuccess:   1,
	ScoreReceivePresentSuccess:  2,
	ScoreRedeemRewardSuccess:    1,
	ScoreRedeemGachaSuccess:     2,
	ScoreCreateUser:             3,
	ScoreShowItems:              1,
	ScorePostAddCard:            1,
	ScoreSetDeck:                1,
	ScoreBanLogin:               1,
}

// シナリオ中に発生したエラーは1つ15点減点する
const ErrorDeduction = 15

func MakeScoreTable(score *score.Score) *score.Score {
	for tag, rate := range ScoreRateTable {
		score.Set(tag, rate)
	}
	return score
}

func ConstructBreakdown(result *isucandar.BenchmarkResult) score.ScoreTable {
	bd := result.Score.Breakdown()
	for tag := range ScoreRateTable {
		if _, ok := bd[tag]; !ok {
			bd[tag] = int64(0)
		}
	}
	return bd
}
