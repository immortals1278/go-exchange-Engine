package main

import (
	"go-exchange/api"
	"go-exchange/engine"
	"go-exchange/storage"
	"net/http"
)

func main() {
	
	storage.InitMySQL()
	// Redis 已移除

	engine := engine.NewMatchingEngine()

	handler := api.NewHandler(engine)

	http.HandleFunc("/api/login", handler.LogIn)
	http.HandleFunc("/api/order", handler.PlaceOrder)
	http.HandleFunc("/api/cancel", handler.CancelOrder)
	http.HandleFunc("/api/balance", handler.GetBalance)

	http.ListenAndServe(":8080", nil)
}