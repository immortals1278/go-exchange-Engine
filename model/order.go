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
	ID       string
	UserID   string
	Side     Side
	Price    float64
	Quantity float64
	Status Status
}

//定义订单