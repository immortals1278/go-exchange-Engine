package main

import (
	"go-exchange/api"
	"go-exchange/engine"
	"net/http"
)

func main() {
	
	storage.InitMySQL()
	storage.InitRedis()

	engine := engine.NewMatchingEngine()

	handler := api.NewHandler(engine)

	http.HandleFunc("/order", handler.PlaceOrder)

	http.HandleFunc("/cancel", handler.CancelOrder)

	http.ListenAndServe(":8080", nil)
}