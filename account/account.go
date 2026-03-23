package account

import (
	"go-exchange/storage"
	"log"

	"github.com/shopspring/decimal"
)

func GetBalance(userID, asset string) (decimal.Decimal, decimal.Decimal) {

	var available, frozen string

	err := storage.DB.QueryRow(
		"SELECT available,frozen FROM balances WHERE user_id=? AND asset=?",
		userID, asset,
	).Scan(&available, &frozen)

	if err != nil {
		return decimal.Zero, decimal.Zero
	}

	availDec, _ := decimal.NewFromString(available)
	frozenDec, _ := decimal.NewFromString(frozen)

	return availDec, frozenDec
}

func ChangeBalance(userID, asset string, delta decimal.Decimal, entryType string, refOrderID, refTradeID string) bool {
	// 开始事务
	tx, err := storage.DB.Begin()
	if err != nil {
		log.Println(err)
		return false
	}
	
	// 获取当前余额
	var available, frozen string
	err = tx.QueryRow(
		"SELECT available,frozen FROM balances WHERE user_id=? AND asset=?",
		userID, asset,
	).Scan(&available, &frozen)
	
	if err != nil {
		tx.Rollback()
		log.Println(err)
		return false
	}
	
	availDec, _ := decimal.NewFromString(available)
	frozenDec, _ := decimal.NewFromString(frozen)
	
	// 根据操作类型更新余额
	var newAvailable, newFrozen decimal.Decimal
	var change decimal.Decimal
	
	switch entryType {
	case "freeze":
		if availDec.LessThan(delta) {
			tx.Rollback()
			return false
		}
		newAvailable = availDec.Sub(delta)
		newFrozen = frozenDec.Add(delta)
		change = delta.Neg()
	case "unfreeze":
		if frozenDec.LessThan(delta) {
			tx.Rollback()
			return false
		}
		newAvailable = availDec.Add(delta)
		newFrozen = frozenDec.Sub(delta)
		change = delta
	case "trade_deduct":
		if frozenDec.LessThan(delta) {
			tx.Rollback()
			return false
		}
		newAvailable = availDec
		newFrozen = frozenDec.Sub(delta)
		change = delta.Neg()
	case "trade_add":
		newAvailable = availDec.Add(delta)
		newFrozen = frozenDec
		change = delta
	case "fee":
		if frozenDec.LessThan(delta) {
			tx.Rollback()
			return false
		}
		newAvailable = availDec
		newFrozen = frozenDec.Sub(delta)
		change = delta.Neg()
	case "fee_credit":
		newAvailable = availDec.Add(delta)
		newFrozen = frozenDec
		change = delta
	default:
		tx.Rollback()
		return false
	}
	
	// 更新余额
	_, err = tx.Exec(
		`UPDATE balances
		 SET available = ?, frozen = ?
		 WHERE user_id=? AND asset=?`,
		newAvailable.String(), newFrozen.String(), userID, asset,
	)
	
	if err != nil {
		tx.Rollback()
		log.Println(err)
		return false
	}
	
	// 记录 ledger 条目
	_, err = tx.Exec(
		`INSERT INTO ledger_entries(user_id, asset, change_amount, balance_after, entry_type, ref_order_id, ref_trade_id, created_at)
		 VALUES(?, ?, ?, ?, ?, ?, ?, NOW())`,
		userID, asset, change.String(), newAvailable.String(), entryType, refOrderID, refTradeID,
	)
	
	if err != nil {
		tx.Rollback()
		log.Println(err)
		return false
	}
	
	// 提交事务
	return tx.Commit() == nil
}

func Freeze(userID, asset string, amount float64) bool {
	delta := decimal.NewFromFloat(amount)
	return ChangeBalance(userID, asset, delta, "freeze", "", "")
}

func Unfreeze(userID, asset string, amount float64) {
	delta := decimal.NewFromFloat(amount)
	ChangeBalance(userID, asset, delta, "unfreeze", "", "")
}

func DeductFrozen(userID, asset string, amount float64) {
	delta := decimal.NewFromFloat(amount)
	ChangeBalance(userID, asset, delta, "trade_deduct", "", "")
}

func AddBalance(userID, asset string, amount float64) {
	delta := decimal.NewFromFloat(amount)
	ChangeBalance(userID, asset, delta, "trade_add", "", "")
}