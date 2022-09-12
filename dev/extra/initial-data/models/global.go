package models

import (
	"fmt"
	"log"

	_ "github.com/go-sql-driver/mysql"
	"github.com/jmoiron/sqlx"
)

var (
	Db *sqlx.DB
)

func init() {
	// connect to DB
	var err error
	tmp, err := sqlx.Open("mysql", "isucon:isucon@tcp(127.0.0.1:3306)/isucon?parseTime=true&loc=Asia%2FTokyo")
	if err != nil {
		fmt.Println(err)
		log.Fatal(err)
	}
	Db = tmp
}
