package account

import (
	"go-exchange/storage"
	"log"
)

func GetBalance(userID, asset string) (float64, float64) {

	var available, frozen float64

	err := storage.DB.QueryRow(
		"SELECT available,frozen FROM balances WHERE user_id=? AND asset=?",
		userID, asset,
	).Scan(&available, &frozen)

	if err != nil {
		return 0, 0
	}

	return available, frozen
}

func Freeze(userID, asset string, amount float64) bool {

	available, _ := GetBalance(userID, asset)

	if available < amount {
		return false
	}

	_, err := storage.DB.Exec(
		`UPDATE balances
		 SET available = available - ?, frozen = frozen + ?
		 WHERE user_id=? AND asset=?`,
		amount, amount, userID, asset,
	)

	if err != nil {
		log.Println(err)
		return false
	}

	return true
}

func Unfreeze(userID, asset string, amount float64) {

	_, err := storage.DB.Exec(
		`UPDATE balances
		 SET available = available + ?, frozen = frozen - ?
		 WHERE user_id=? AND asset=?`,
		amount, amount, userID, asset,
	)

	if err != nil {
		log.Println(err)
	}
}

func DeductFrozen(userID, asset string, amount float64) {

	_, err := storage.DB.Exec(
		`UPDATE balances
		 SET frozen = frozen - ?
		 WHERE user_id=? AND asset=?`,
		amount, userID, asset,
	)

	if err != nil {
		log.Println(err)
	}
}

func AddBalance(userID, asset string, amount float64) {

	_, err := storage.DB.Exec(
		`UPDATE balances
		 SET available = available + ?
		 WHERE user_id=? AND asset=?`,
		amount, userID, asset,
	)

	if err != nil {
		log.Println(err)
	}
}