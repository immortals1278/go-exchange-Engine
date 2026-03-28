package engine

import (
	"go-exchange/model"
	"go-exchange/orderbook"
	"go-exchange/storage"
	"go-exchange/account"
	"strings"

	"github.com/shopspring/decimal"
)

const SystemUserID = "system"
const FeeRate = 0.001 // 0.1% 手续费率


type MatchingEngine struct {
	Books map[string]*orderbook.OrderBook
	OrderIndex map[string]string// ID -> Symbol
}

func NewMatchingEngine() *MatchingEngine {
	return &MatchingEngine{
		Books:      make(map[string]*orderbook.OrderBook),
		OrderIndex: make(map[string]string),
	}
}

func (e *MatchingEngine) getBook(symbol string) *orderbook.OrderBook{
	book, ok := e.Books[symbol]
	if !ok {
		e.Books[symbol] = orderbook.NewOrderBook()
		book = e.Books[symbol]
	}
	return book
}

// getAssets 根据交易对获取基础资产和报价资产
func (e *MatchingEngine) getAssets(symbol string) (string, string) {
	parts := strings.Split(symbol, "/")
	if len(parts) != 2 {
		return "", ""
	}
	return parts[0], parts[1]
}

func (e *MatchingEngine) PlaceOrder(order *model.Order) {
	baseAsset, quoteAsset := e.getAssets(order.Symbol)
	
	//冻结逻辑
	var ok bool
	if order.Side == model.Buy {
		// 买基础资产，冻结报价资产
		cost := decimal.NewFromFloat(order.Price * order.Quantity)
		ok = account.Freeze(order.UserID, quoteAsset, cost)
	} else {
		// 卖基础资产，冻结基础资产
		amount := decimal.NewFromFloat(order.Quantity)
		ok = account.Freeze(order.UserID, baseAsset, amount)
	}
	if !ok {
		return // 余额不足
	}

	order.ID = GenerateOrderID()
	order.Status = model.Open
	book := e.getBook(order.Symbol)
	book.AddOrder(order)

	e.match(order.Symbol)

	storage.SaveOrder(order.ID, order.UserID, string(order.Side), order.Price, order.Quantity)

	e.OrderIndex[order.ID] = order.Symbol
}

func (e *MatchingEngine) match(symbol string) {
	book := e.getBook(symbol)
	baseAsset, quoteAsset := e.getAssets(symbol)
	
	for {
		if len(book.BidPrices) == 0 || len(book.AskPrices) == 0 {
			return errors.New("order book is empty")
		}

		bestBid := book.BidPrices[0]
		bestAsk := book.AskPrices[0]

		if bestBid < bestAsk {
			return errors.New("best bid %f less than best ask %f, no cross", bestBid, bestAsk)
		}

		bidLevel := book.Bids[bestBid]
		askLevel := book.Asks[bestAsk]

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
		
		// 买方：扣除冻结的报价资产，获得基础资产（减fee）
		account.DeductFrozen(buyOrder.UserID, quoteAsset, tradeAmount)
		account.AddBalance(buyOrder.UserID, baseAsset, tradeQtyDec.Sub(buyerFee))
		account.ChangeBalance(buyOrder.UserID, baseAsset, buyerFee, "fee", buyOrder.ID, "")
		
		// 卖方：扣除冻结的基础资产，获得报价资产（减fee）
		account.DeductFrozen(sellOrder.UserID, baseAsset, tradeQtyDec)
		account.AddBalance(sellOrder.UserID, quoteAsset, tradeAmount.Sub(fee))
		account.ChangeBalance(sellOrder.UserID, quoteAsset, fee, "fee", sellOrder.ID, "")
		
		// 系统收取手续费（两种资产）
		account.ChangeBalance(SystemUserID, baseAsset, buyerFee, "fee_credit", "", "")
		account.ChangeBalance(SystemUserID, quoteAsset, fee, "fee_credit", "", "")

		if buyOrder.Quantity >= sellOrder.Quantity {
			buyOrder.Quantity -= sellOrder.Quantity
			askLevel.Orders = askLevel.Orders[1:]
			storage.SaveTrade(buyOrder.ID, sellOrder.ID, sellOrder.Price, sellOrder.Quantity)

			if len(askLevel.Orders) == 0 {
				delete(book.Asks, bestAsk)
				book.AskPrices = book.AskPrices[1:]
			}
		} else {
			sellOrder.Quantity -= buyOrder.Quantity
			bidLevel.Orders = bidLevel.Orders[1:]
			storage.SaveTrade(buyOrder.ID, sellOrder.ID, sellOrder.Price, buyOrder.Quantity)

			if len(bidLevel.Orders) == 0 {
				delete(book.Bids, bestBid)
				book.BidPrices = book.BidPrices[1:]
			}
		}
	}
}

func (e *MatchingEngine) CancelOrder(orderID string) bool {
	// 从订单索引中获取交易对
	symbol, exists := e.OrderIndex[orderID]
	if !exists {
		return false
	}
	
	book := e.getBook(symbol)
	order := book.GetOrder(orderID)
	
	if order == nil {
		return false
	}
	
	baseAsset, quoteAsset := e.getAssets(symbol)
	
	// 解冻资产
	if order.Side == model.Buy {
		amount := decimal.NewFromFloat(order.Price * order.Quantity)
		account.Unfreeze(order.UserID, quoteAsset, amount)
	} else {
		amount := decimal.NewFromFloat(order.Quantity)
		account.Unfreeze(order.UserID, baseAsset, amount)
	}

	// 从订单簿中移除订单
	ok := book.RemoveOrder(orderID)

	if ok {
		storage.UpdateOrderStatus(orderID, "canceled")
		delete(e.OrderIndex, orderID)
	}

	return ok
}