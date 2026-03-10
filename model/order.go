package model

type Side string

const (
	Buy  Side = "buy"
	Sell Side = "sell"
)

type Order struct {
	ID       string
	UserID   string
	Side     Side
	Price    float64
	Quantity float64
}

//定义订单