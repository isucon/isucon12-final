package random

import (
	"math/rand"
	"time"
)

var (
	// 初期データ生成時に BaseTime を利用するため、初期データは BaseTime より前の出来事が DB に INSERT された結果である
	// ベンチマークシナリオ実行時に BaseTime を利用するため、ベンチマークシナリオは BaseTime 以降に起こる出来事である
	jst      = time.FixedZone("Asia/Tokyo", 9*60*60)
	BaseTime = time.Date(2022, 6, 4, 0, 0, 0, 0, jst) // 競技の日と被るように
)

func Time() time.Time {
	subFrom := BaseTime.Unix()
	subValue := rand.Int63n(60 * 60 * 24 * 365 * 3) // 0 ~ 3年
	return time.Unix(subFrom-subValue, 0)
}

func OldUserTime() time.Time {
	subFrom := BaseTime.Unix()
	subValue := rand.Int63n(60 * 60 * 24 * 365 * 1)        // 0 ~ 1年
	return time.Unix(subFrom-subValue-(60*60*24*365*2), 0) // 2-3年前を返す
}

func OneYearUserTime() time.Time {
	subFrom := BaseTime.Unix()
	subValue := rand.Int63n(60 * 60 * 24 * 365 * 1)    // 0 ~ 1年
	return time.Unix(subFrom-subValue-(60*60*24*8), 0) //一年以内 - 8日
}

func NearOneWeekTime() time.Time {
	subFrom := BaseTime.Unix()
	subValue := rand.Int63n(60 * 60 * 24 * 7) // 0 〜 7日
	return time.Unix(subFrom-subValue, 0)
}

func TimeAfterArg(t time.Time) time.Time {
	createdAtUnix := t.Unix()
	baseTimeUnix := BaseTime.Unix()
	return time.Unix(createdAtUnix+rand.Int63n(baseTimeUnix-createdAtUnix), 0)
}
