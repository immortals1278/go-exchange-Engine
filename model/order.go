package model

type Side string

const (
	Buy  Side = "buy"
	Sell Side = "sell"
)

type Status string

const (
	Open      Status = "open"
	Filled    Status = "filled"
	Canceled  Status = "canceled"
)

type Order struct {
	ID       string  `json:"id"`
	UserID   string  `json:"user_id"`
	Symbol   string  `json:"symbol"`
	Side     Side    `json:"side"`
	Price    float64 `json:"price"`
	Quantity float64 `json:"quantity"`
	Status   Status  `json:"status"`
}

//定义订单
