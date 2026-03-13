package storage

import (
	"context"

	"github.com/redis/go-redis/v9"
)

var RDB *redis.Client
var ctx = context.Background()

func InitRedis() {

	RDB = redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})
}

func SaveDepth(bids interface{}, asks interface{}) {

	RDB.HSet(ctx, "orderbook",
		"bids", bids,
		"asks", asks,
	)
}