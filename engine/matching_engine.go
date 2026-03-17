package engine

import (
	"go-exchange/model"
	"go-exchange/orderbook"
	"go-exchange/storage"
	"go-exchange/account"
)

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
			storage.SaveTrade(buyOrder.ID, sellOrder.ID, sellOrder.Price, sellOrder.Quantity)

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