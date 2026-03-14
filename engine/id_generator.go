package engine

import (
	"strconv"
	"sync/atomic"
	"time"
)

var counter uint64

func GenerateOrderID() string {

	ts := time.Now().UnixNano()
	c := atomic.AddUint64(&counter, 1)

	return strconv.FormatInt(ts, 10) + "-" + strconv.FormatUint(c, 10)
}