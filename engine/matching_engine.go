package engine

import (
	"go-exchange/model"
	"go-exchange/orderbook"
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

	if order.Side == model.Buy {
		e.Book.BuyOrders = append(e.Book.BuyOrders, order)
	} else {
		e.Book.SellOrders = append(e.Book.SellOrders, order)
	}

	e.match()
}

func (e *MatchingEngine) match() {

	if len(e.Book.BuyOrders) == 0 || len(e.Book.SellOrders) == 0 {
		return
	}

	buy := e.Book.BuyOrders[0]
	sell := e.Book.SellOrders[0]

	if buy.Price >= sell.Price {

		if buy.Quantity >= sell.Quantity {
			buy.Quantity -= sell.Quantity
			e.Book.SellOrders = e.Book.SellOrders[1:]
		} else {
			sell.Quantity -= buy.Quantity
			e.Book.BuyOrders = e.Book.BuyOrders[1:]
		}
	}
}