package engine

import (
	"go-exchange/model"
	"go-exchange/orderbook"
	"go-exchange/storage"
	"go-exchange/account"

	"github.com/shopspring/decimal"
)

const SystemUserID = "system"
const FeeRate = 0.001 // 0.1% 手续费率

type MatchingEngine struct {
	Book *orderbook.OrderBook
}

func NewMatchingEngine() *MatchingEngine {
	return &MatchingEngine{
		Book: orderbook.NewOrderBook(),
	}
}

func (e *MatchingEngine) PlaceOrder(order *model.Order) {
	//冻结逻辑
	var ok bool
	if order.Side == model.Buy {
		// 买 BTC，冻结 USDT
		cost := order.Price * order.Quantity
		ok = account.Freeze(order.UserID, "USDT", cost)
	} else {
		// 卖 BTC，冻结 BTC
		ok = account.Freeze(order.UserID, "BTC", order.Quantity)
	}
	if !ok {
		return // 余额不足
	}


	order.ID = GenerateOrderID()
	order.Status = model.Open

	e.Book.AddOrder(order)

	e.match()

	storage.SaveOrder(order.ID, order.UserID, string(order.Side), order.Price, order.Quantity)
}

func (e *MatchingEngine) match() {

	for {

		if len(e.Book.BidPrices) == 0 || len(e.Book.AskPrices) == 0 {
			return
		}

		bestBid := e.Book.BidPrices[0]
		bestAsk := e.Book.AskPrices[0]

		if bestBid < bestAsk {
			return
		}

		bidLevel := e.Book.Bids[bestBid]
		askLevel := e.Book.Asks[bestAsk]

		buyOrder := bidLevel.Orders[0]
		sellOrder := askLevel.Orders[0]

		tradeQty := sellOrder.Quantity
		if buyOrder.Quantity < sellOrder.Quantity {
			tradeQty = buyOrder.Quantity
		}

		tradePrice := sellOrder.Price
		tradeAmount := tradePrice * tradeQty
		
		// 计算手续费
		feeRate := decimal.NewFromFloat(FeeRate)
		tradeAmountDec := decimal.NewFromFloat(tradeAmount)
		fee := tradeAmountDec.Mul(feeRate)
		
		// 买方：扣除冻结的USDT，获得BTC
		account.DeductFrozen(buyOrder.UserID, "USDT", tradeAmount)
		account.AddBalance(buyOrder.UserID, "BTC", tradeQty)
		
		// 卖方：扣除冻结的BTC，获得USDT（扣除手续费）
		account.DeductFrozen(sellOrder.UserID, "BTC", tradeQty)
		account.AddBalance(sellOrder.UserID, "USDT", tradeAmount-fee.InexactFloat64())
		
		// 系统收取手续费
		account.ChangeBalance(SystemUserID, "USDT", fee, "fee_credit", buyOrder.ID+"_"+sellOrder.ID, "")

		if buyOrder.Quantity >= sellOrder.Quantity {

			buyOrder.Quantity -= sellOrder.Quantity
			askLevel.Orders = askLevel.Orders[1:]
			storage.SaveTrade(buyOrder.ID, sellOrder.ID, sellOrder.Price, sellOrder.Quantity)

			if len(askLevel.Orders) == 0 {
				delete(e.Book.Asks, bestAsk)
				e.Book.AskPrices = e.Book.AskPrices[1:]
			}

		} else {

			sellOrder.Quantity -= buyOrder.Quantity
			bidLevel.Orders = bidLevel.Orders[1:]
			storage.SaveTrade(buyOrder.ID, sellOrder.ID, sellOrder.Price, buyOrder.Quantity)

			if len(bidLevel.Orders) == 0 {
				delete(e.Book.Bids, bestBid)
				e.Book.BidPrices = e.Book.BidPrices[1:]
			}
		}
	}
}

func (e *MatchingEngine) CancelOrder(orderID string) bool {
	order := e.Book.GetOrder(orderID)
	
	if order == nil {
		return false
	}
	//操作数据库
	if order.Side == model.Buy {

		account.Unfreeze(order.UserID, "USDT",
			order.Price*order.Quantity)

	} else {

		account.Unfreeze(order.UserID, "BTC",
			order.Quantity)
	}


	ok := e.Book.RemoveOrder(orderID)

	if ok {
		storage.UpdateOrderStatus(orderID, "canceled")
	}

	return ok
}