package orderbook

import "go-exchange/model"

type OrderBook struct {
	BuyOrders  []*model.Order
	SellOrders []*model.Order
}

func NewOrderBook() *OrderBook {
	return &OrderBook{
		BuyOrders:  []*model.Order{},
		SellOrders: []*model.Order{},
	}
}