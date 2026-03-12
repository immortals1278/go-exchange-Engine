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

	e.Book.AddOrder(order)

	e.match()
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

			if len(askLevel.Orders) == 0 {
				delete(e.Book.Asks, bestAsk)
				e.Book.AskPrices = e.Book.AskPrices[1:]
			}

		} else {

			sellOrder.Quantity -= buyOrder.Quantity
			bidLevel.Orders = bidLevel.Orders[1:]

			if len(bidLevel.Orders) == 0 {
				delete(e.Book.Bids, bestBid)
				e.Book.BidPrices = e.Book.BidPrices[1:]
			}
		}
	}
}