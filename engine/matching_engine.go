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
	Books map[string]*orderbook.OderBook
}

func NewMatchingEngine() *MatchingEngine {
	return &MatchingEngine{
		Books: make(map[string]*orderbook.OrderBook),
	}
}

func (e *MatchingEngine) getBook(symbol string) *orderbook.OrderBook{
	book,ok := e.books[symbol]
	if !ok {
		e.books[symbol] = orderbook.NewOrderBook()
		book = e.books[symbol]
	}
	return book
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
	Book := e.getBook(order.Symbol)
	Book.AddOrder(order)

	e.match(synbol)

	storage.SaveOrder(order.ID, order.UserID, string(order.Side), order.Price, order.Quantity)
}

func (e *MatchingEngine) match(symbol string) {
	Book := e.getBook(symbol)
	for {

		if len(Book.BidPrices) == 0 || len(Book.AskPrices) == 0 {
			return
		}

		bestBid := Book.BidPrices[0]
		bestAsk := Book.AskPrices[0]

		if bestBid < bestAsk {
			return
		}

		bidLevel := Book.Bids[bestBid]
		askLevel := Book.Asks[bestAsk]

		buyOrder := bidLevel.Orders[0]
		sellOrder := askLevel.Orders[0]

		tradeQtyDec := decimal.NewFromFloat(sellOrder.Quantity)
		if buyOrder.Quantity < sellOrder.Quantity {
			tradeQtyDec = decimal.NewFromFloat(buyOrder.Quantity)
		}

		tradePriceDec := decimal.NewFromFloat(sellOrder.Price)
		tradeAmount := tradePriceDec.Mul(tradeQtyDec)
		
		// 计算手续费
		feeRate := decimal.NewFromFloat(FeeRate)
		fee := tradeAmount.Mul(feeRate)
		buyerFee := tradeQtyDec.Mul(feeRate)
		
		// 买方：扣除冻结的USDT，获得BTC（减fee）
		account.DeductFrozen(buyOrder.UserID, "USDT", tradeAmount.InexactFloat64())
		account.AddBalance(buyOrder.UserID, "BTC", tradeQtyDec.Sub(buyerFee).InexactFloat64())
		account.ChangeBalance(buyOrder.UserID, "BTC", buyerFee, "fee", buyOrder.ID, "")
		
		// 卖方：扣除冻结的BTC，获得USDT（减fee）
		account.DeductFrozen(sellOrder.UserID, "BTC", tradeQtyDec.InexactFloat64())
		account.AddBalance(sellOrder.UserID, "USDT", tradeAmount.Sub(fee).InexactFloat64())
		account.ChangeBalance(sellOrder.UserID, "USDT", fee, "fee", sellOrder.ID, "")
		
		// 系统收取手续费（两种资产）
		account.ChangeBalance(SystemUserID, "BTC", buyerFee, "fee_credit", "", "")
		account.ChangeBalance(SystemUserID, "USDT", fee, "fee_credit", "", "")

		if buyOrder.Quantity >= sellOrder.Quantity {

			buyOrder.Quantity -= sellOrder.Quantity
			askLevel.Orders = askLevel.Orders[1:]
			storage.SaveTrade(buyOrder.ID, sellOrder.ID, sellOrder.Price, sellOrder.Quantity)

			if len(askLevel.Orders) == 0 {
				delete(Book.Asks, bestAsk)
				Book.AskPrices = Book.AskPrices[1:]
			}

		} else {

			sellOrder.Quantity -= buyOrder.Quantity
			bidLevel.Orders = bidLevel.Orders[1:]
			storage.SaveTrade(buyOrder.ID, sellOrder.ID, sellOrder.Price, buyOrder.Quantity)

			if len(bidLevel.Orders) == 0 {
				delete(Book.Bids, bestBid)
				Book.BidPrices = Book.BidPrices[1:]
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