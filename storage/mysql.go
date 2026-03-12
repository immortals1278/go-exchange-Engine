package storage

import (
	"database/sql"
	"log"

	_ "github.com/go-sql-driver/mysql"
)

var DB *sql.DB

func InitMySQL() {

	dsn := "root:password@tcp(127.0.0.1:3306)/exchange"

	db, err := sql.Open("mysql", dsn)
	if err != nil {
		log.Fatal(err)
	}

	DB = db
}

func SaveOrder(id, userID, side string, price, qty float64) {

	_, err := DB.Exec(
		"INSERT INTO orders(id,user_id,side,price,quantity,status) VALUES(?,?,?,?,?,?)",
		id, userID, side, price, qty, "open",
	)

	if err != nil {
		log.Println(err)
	}
}

func SaveOrder(id, userID, side string, price, qty float64) {

	_, err := DB.Exec(
		"INSERT INTO orders(id,user_id,side,price,quantity,status) VALUES(?,?,?,?,?,?)",
		id, userID, side, price, qty, "open",
	)

	if err != nil {
		log.Println(err)
	}
}

func SaveTrade(buyID, sellID string, price, qty float64) {

	_, err := DB.Exec(
		"INSERT INTO trades(buy_order_id,sell_order_id,price,quantity) VALUES(?,?,?,?)",
		buyID, sellID, price, qty,
	)

	if err != nil {
		log.Println(err)
	}
}