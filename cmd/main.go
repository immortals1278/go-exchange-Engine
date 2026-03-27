package main

import (
	"go-exchange/api"
	"go-exchange/engine"
	"go-exchange/storage"
	"net/http"
)

func main() {
	
	storage.InitMySQL()
	storage.InitRedis()

	engine := engine.NewMatchingEngine()

	handler := api.NewHandler(engine)

	http.HandleFunc("/api/login", handler.LogIn)
	http.HandleFunc("/api/order", handler.PlaceOrder)
	http.HandleFunc("/api/cancel", handler.CancelOrder)

	http.ListenAndServe(":8080", nil)
}