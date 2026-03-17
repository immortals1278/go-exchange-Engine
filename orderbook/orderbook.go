package orderbook

import (
	"go-exchange/model"
	"sort"
)

type PriceLevel struct {
	Price  float64
	Orders []*model.Order
}

type OrderBook struct {
	Bids map[float64]*PriceLevel
	Asks map[float64]*PriceLevel

	BidPrices []float64
	AskPrices []float64
}

func NewOrderBook() *OrderBook {
	return &OrderBook{
		Bids: make(map[float64]*PriceLevel),
		Asks: make(map[float64]*PriceLevel),
	}
}

func (ob *OrderBook) AddOrder(order *model.Order) {

	if order.Side == model.Buy {

		level, exists := ob.Bids[order.Price]

		if !exists {
			level = &PriceLevel{Price: order.Price}
			ob.Bids[order.Price] = level
			ob.BidPrices = append(ob.BidPrices, order.Price)

			sort.Sort(sort.Reverse(sort.Float64Slice(ob.BidPrices)))
		}

		level.Orders = append(level.Orders, order)

	} else {

		level, exists := ob.Asks[order.Price]

		if !exists {
			level = &PriceLevel{Price: order.Price}
			ob.Asks[order.Price] = level
			ob.AskPrices = append(ob.AskPrices, order.Price)

			sort.Float64s(ob.AskPrices)
		}

		level.Orders = append(level.Orders, order)
	}

	

}

func (ob *OrderBook) GetOrder(orderID string) *model.Order {

	for _, level := range ob.Bids {

		for _, o := range level.Orders {

			if o.ID == orderID {
				return o
			}
		}
	}

	for _, level := range ob.Asks {

		for _, o := range level.Orders {

			if o.ID == orderID {
				return o
			}
		}
	}

	return nil
}

func (ob *OrderBook) RemoveOrder(orderID string) bool {

	for price, level := range ob.Bids {

		for i, o := range level.Orders {

			if o.ID == orderID {

				level.Orders = append(level.Orders[:i], level.Orders[i+1:]...)

				if len(level.Orders) == 0 {
					delete(ob.Bids, price)
				}

				return true
			}
		}
	}

	for price, level := range ob.Asks {

		for i, o := range level.Orders {

			if o.ID == orderID {

				level.Orders = append(level.Orders[:i], level.Orders[i+1:]...)

				if len(level.Orders) == 0 {
					delete(ob.Asks, price)
				}

				return true
			}
		}
	}

	return false
}