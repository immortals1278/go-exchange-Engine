package account

import (
	"go-exchange/storage"
	"log"

	"github.com/shopspring/decimal"
)

// FrozenBalance 订单级别的冻结资金
type FrozenBalance struct {
	UserID  string
	Asset   string
	OrderID string
	Amount  decimal.Decimal
}

// 内部存储：userID -> asset -> orderID -> amount
var frozenBalances = make(map[string]map[string]map[string]decimal.Decimal)

func init() {
	// 初始化存储结构
	frozenBalances = make(map[string]map[string]map[string]decimal.Decimal)
}

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

func GetAllBalances(userID string) map[string]string {
	rows, err := storage.DB.Query(
		"SELECT asset, available FROM balances WHERE user_id=?",
		userID,
	)
	if err != nil {
		return nil
	}
	defer rows.Close()

	balances := make(map[string]string)
	for rows.Next() {
		var asset, available string
		if err := rows.Scan(&asset, &available); err != nil {
			continue
		}
		balances[asset] = available
	}

	return balances
}

func ChangeBalance(userID, asset string, delta decimal.Decimal, entryType string, refOrderID, refTradeID string) bool {
	// 开始事务
	tx, err := storage.DB.Begin()
	if err != nil {
		log.Println("事务开始失败:", err)
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
		log.Printf("查询余额失败: userID=%s, asset=%s, error=%v", userID, asset, err)
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
			log.Printf("余额不足: available=%s, delta=%s", availDec.String(), delta.String())
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
	case "fee_available":
		if availDec.LessThan(delta) {
			tx.Rollback()
			return false
		}
		newAvailable = availDec.Sub(delta)
		newFrozen = frozenDec
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

// Freeze 冻结资金（订单级别）
func Freeze(userID, asset string, amount decimal.Decimal, orderID string) bool {
	// 初始化存储结构
	if frozenBalances[userID] == nil {
		frozenBalances[userID] = make(map[string]map[string]decimal.Decimal)
	}
	if frozenBalances[userID][asset] == nil {
		frozenBalances[userID][asset] = make(map[string]decimal.Decimal)
	}

	// 冻结资金
	if !ChangeBalance(userID, asset, amount, "freeze", orderID, "") {
		return false
	}

	// 记录订单冻结
	frozenBalances[userID][asset][orderID] = frozenBalances[userID][asset][orderID].Add(amount)
	return true
}

// Unfreeze 解冻订单的所有冻结资金
func Unfreeze(userID, asset string, orderID string) {
	// 获取订单冻结金额
	amount := GetFrozenByOrder(userID, asset, orderID)
	if amount.IsZero() {
		return
	}

	// 解冻资金
	ChangeBalance(userID, asset, amount, "unfreeze", orderID, "")

	// 清除订单冻结记录
	if frozenBalances[userID] != nil && frozenBalances[userID][asset] != nil {
		delete(frozenBalances[userID][asset], orderID)
	}
}

func UnfreezeAmount(userID, asset string, amount decimal.Decimal, orderID string) bool {
	if amount.LessThanOrEqual(decimal.Zero) {
		return true
	}

	frozen := GetFrozenByOrder(userID, asset, orderID)
	if frozen.LessThan(amount) {
		log.Printf("订单冻结资金不足，无法部分解冻: orderID=%s, frozen=%s, amount=%s", orderID, frozen.String(), amount.String())
		return false
	}

	if !ChangeBalance(userID, asset, amount, "unfreeze", orderID, "") {
		return false
	}

	newAmount := frozen.Sub(amount)
	if newAmount.IsZero() {
		if frozenBalances[userID] != nil && frozenBalances[userID][asset] != nil {
			delete(frozenBalances[userID][asset], orderID)
		}
		return true
	}

	frozenBalances[userID][asset][orderID] = newAmount
	return true
}

// DeductFrozen 扣除订单的冻结资金
func DeductFrozen(userID, asset string, amount decimal.Decimal, orderID string) bool {
	// 检查订单冻结金额
	frozen := GetFrozenByOrder(userID, asset, orderID)
	if frozen.LessThan(amount) {
		log.Printf("订单冻结资金不足: orderID=%s, frozen=%s, amount=%s", orderID, frozen.String(), amount.String())
		return false
	}

	// 扣除冻结资金
	if !ChangeBalance(userID, asset, amount, "trade_deduct", orderID, "") {
		return false
	}

	// 更新订单冻结记录
	newAmount := frozenBalances[userID][asset][orderID].Sub(amount)
	if newAmount.IsZero() {
		delete(frozenBalances[userID][asset], orderID)
		return true
	}
	frozenBalances[userID][asset][orderID] = newAmount
	return true
}

// GetFrozenByOrder 获取订单的冻结资金
func GetFrozenByOrder(userID, asset, orderID string) decimal.Decimal {
	if frozenBalances[userID] == nil || frozenBalances[userID][asset] == nil {
		return decimal.Zero
	}
	return frozenBalances[userID][asset][orderID]
}

func AddBalance(userID, asset string, amount decimal.Decimal) {
	ChangeBalance(userID, asset, amount, "trade_add", "", "")
}
