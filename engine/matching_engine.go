package engine

import (
	"errors"
	"go-exchange/account"
	"go-exchange/model"
	"go-exchange/orderbook"
	"go-exchange/storage"
	"strings"

	"github.com/shopspring/decimal"
)

const SystemUserID = "system"
const FeeRate = 0.001 // 0.1% 手续费率

type MatchingEngine struct {
	Books      map[string]*orderbook.OrderBook
	OrderIndex map[string]string // ID -> Symbol
}

func NewMatchingEngine() *MatchingEngine {
	return &MatchingEngine{
		Books:      make(map[string]*orderbook.OrderBook),
		OrderIndex: make(map[string]string),
	}
}

func (e *MatchingEngine) getBook(symbol string) *orderbook.OrderBook {
	if _, exists := e.Books[symbol]; !exists {
		e.Books[symbol] = orderbook.NewOrderBook()
	}
	return e.Books[symbol]
}

func (e *MatchingEngine) getAssets(symbol string) (string, string) {
	parts := strings.Split(symbol, "/")
	if len(parts) != 2 {
		return "", ""
	}
	return parts[0], parts[1]
}

func (e *MatchingEngine) PlaceOrder(order *model.Order) error {
	baseAsset, quoteAsset := e.getAssets(order.Symbol)

	// 生成订单ID
	order.ID = GenerateOrderID()

	//冻结逻辑
	var ok bool
	if order.Side == model.Buy {
		// 买基础资产，冻结报价资产
		cost := decimal.NewFromFloat(order.Price * order.Quantity)
		ok = account.Freeze(order.UserID, quoteAsset, cost, order.ID)
	} else {
		// 卖基础资产，冻结基础资产
		amount := decimal.NewFromFloat(order.Quantity)
		ok = account.Freeze(order.UserID, baseAsset, amount, order.ID)
	}
	if !ok {
		return errors.New("insufficient balance") // 余额不足
	}
	order.Status = model.Open
	book := e.getBook(order.Symbol)
	book.AddOrder(order)

	e.match(order.Symbol)

	storage.SaveOrder(order.ID, order.UserID, string(order.Side), order.Symbol, order.Price, order.Quantity)

	e.OrderIndex[order.ID] = order.Symbol

	return nil
}

func (e *MatchingEngine) match(symbol string) {
	book := e.getBook(symbol)
	baseAsset, quoteAsset := e.getAssets(symbol)

	for {
		if len(book.BidPrices) == 0 || len(book.AskPrices) == 0 {
			return
		}

		bestBid := book.BidPrices[0]
		bestAsk := book.AskPrices[0]

		if bestBid < bestAsk {
			return
		}

		bidLevel := book.Bids[bestBid]
		askLevel := book.Asks[bestAsk]

		if len(bidLevel.Orders) == 0 || len(askLevel.Orders) == 0 {
			// 移除空的价格级别
			if len(bidLevel.Orders) == 0 {
				delete(book.Bids, bestBid)
				book.BidPrices = book.BidPrices[1:]
			}
			if len(askLevel.Orders) == 0 {
				delete(book.Asks, bestAsk)
				book.AskPrices = book.AskPrices[1:]
			}
			continue
		}

		buyOrder := bidLevel.Orders[0]
		sellOrder := askLevel.Orders[0]

		tradePrice := bestAsk
		tradeQty := buyOrder.Quantity
		if sellOrder.Quantity < tradeQty {
			tradeQty = sellOrder.Quantity
		}

		// 计算手续费
		tradePriceDec := decimal.NewFromFloat(tradePrice)
		tradeQtyDec := decimal.NewFromFloat(tradeQty)
		tradeAmount := tradePriceDec.Mul(tradeQtyDec)
		fee := tradeAmount.Mul(decimal.NewFromFloat(FeeRate))

		// 买方：扣钱（报价资产 + fee），收币（基础资产）
		account.DeductFrozen(buyOrder.UserID, quoteAsset, tradeAmount, buyOrder.ID)
		// 退还多冻结的资金
		refund := decimal.NewFromFloat(buyOrder.Price).Sub(tradePriceDec).Mul(tradeQtyDec)
		if refund.GreaterThan(decimal.Zero) {
			account.UnfreezeAmount(buyOrder.UserID, quoteAsset, refund, buyOrder.ID)
		}
		account.ChangeBalance(buyOrder.UserID, quoteAsset, fee, "fee_available", buyOrder.ID, "")
		account.AddBalance(buyOrder.UserID, baseAsset, tradeQtyDec)

		// 卖方：扣币（基础资产），收钱（报价资产 - fee），扣手续费（报价资产）
		account.DeductFrozen(sellOrder.UserID, baseAsset, tradeQtyDec, sellOrder.ID)
		account.AddBalance(sellOrder.UserID, quoteAsset, tradeAmount.Sub(fee))

		// 系统账户：收手续费（报价资产）
		account.ChangeBalance(SystemUserID, quoteAsset, fee, "fee_credit", "", "")

		storage.SaveTrade(buyOrder.ID, sellOrder.ID, tradePrice, tradeQty)

		buyOrder.Quantity -= tradeQty
		sellOrder.Quantity -= tradeQty

		if buyOrder.Quantity == 0 {
			bidLevel.Orders = bidLevel.Orders[1:]
			if len(bidLevel.Orders) == 0 {
				delete(book.Bids, bestBid)
				book.BidPrices = book.BidPrices[1:]
			}
			buyOrder.Status = model.Filled
			storage.UpdateOrderStatus(buyOrder.ID, "filled")
			// 解冻订单的所有剩余冻结资金
			account.Unfreeze(buyOrder.UserID, quoteAsset, buyOrder.ID)
			// 从订单索引中移除
			delete(e.OrderIndex, buyOrder.ID)
		}

		if sellOrder.Quantity == 0 {
			askLevel.Orders = askLevel.Orders[1:]
			if len(askLevel.Orders) == 0 {
				delete(book.Asks, bestAsk)
				book.AskPrices = book.AskPrices[1:]
			}
			sellOrder.Status = model.Filled
			storage.UpdateOrderStatus(sellOrder.ID, "filled")
			// 解冻订单的所有剩余冻结资金
			account.Unfreeze(sellOrder.UserID, baseAsset, sellOrder.ID)
			// 从订单索引中移除
			delete(e.OrderIndex, sellOrder.ID)
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

	// 解冻资产
	baseAsset, quoteAsset := e.getAssets(symbol)
	if order.Side == model.Buy {
		// 买单，解冻报价资产
		account.Unfreeze(order.UserID, quoteAsset, order.ID)
	} else {
		// 卖单，解冻基础资产
		account.Unfreeze(order.UserID, baseAsset, order.ID)
	}

	// 从订单簿中移除
	book.RemoveOrder(orderID)

	// 更新数据库状态
	storage.UpdateOrderStatus(orderID, "canceled")

	// 从索引中移除
	delete(e.OrderIndex, orderID)

	return true
}
