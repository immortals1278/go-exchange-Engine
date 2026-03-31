package api

import (
	"encoding/json"
	"go-exchange/account"
	"go-exchange/engine"
	"go-exchange/model"
	"net/http"
)

type Handler struct {
	Engine *engine.MatchingEngine
}

type Response struct {
	Code int         `json:"code"`
	Msg  string      `json:"msg"`
	Data interface{} `json:"data,omitempty"`
}

func NewHandler(e *engine.MatchingEngine) *Handler {
	return &Handler{Engine: e}
}

func (h *Handler) PlaceOrder(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

	// 处理 OPTIONS 请求
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	var order model.Order

	err := json.NewDecoder(r.Body).Decode(&order)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(Response{
			Code: 1,
			Msg:  "invalid request",
		})
		return
	}

	err = h.Engine.PlaceOrder(&order)
	if err != nil {
		json.NewEncoder(w).Encode(Response{
			Code: 2,
			Msg:  err.Error(),
		})
		return
	}

	// 获取所有资产余额
	balances := account.GetAllBalances(order.UserID)

	json.NewEncoder(w).Encode(Response{
		Code: 0,
		Msg:  "ok",
		Data: map[string]interface{}{
			"order_id": order.ID,
			"balances": balances,
		},
	})
}

func (h *Handler) CancelOrder(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

	// 处理 OPTIONS 请求
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	id := r.URL.Query().Get("id")
	userID := r.URL.Query().Get("user_id")
	if id == "" || userID == "" {
		json.NewEncoder(w).Encode(Response{
			Code: 1,
			Msg:  "id and user_id required",
		})
		return
	}

	ok := h.Engine.CancelOrder(id)

	if !ok {
		json.NewEncoder(w).Encode(Response{
			Code: 2,
			Msg:  "order not found",
		})
		return
	}

	// 获取所有资产余额
	balances := account.GetAllBalances(userID)

	json.NewEncoder(w).Encode(Response{
		Code: 0,
		Msg:  "canceled",
		Data: balances,
	})
}

type LoginRequest struct {
	UserID string `json:"user_id"`
}

func (h *Handler) LogIn(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

	// 处理 OPTIONS 请求
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	var req LoginRequest

	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil || req.UserID == "" {
		json.NewEncoder(w).Encode(Response{
			Code: 1,
			Msg:  "invalid user_id",
		})
		return
	}

	json.NewEncoder(w).Encode(Response{
		Code: 0,
		Msg:  "ok",
		Data: map[string]string{
			"user_id": req.UserID,
		},
	})
}

func (h *Handler) GetBalance(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

	// 处理 OPTIONS 请求
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	userID := r.URL.Query().Get("user_id")
	if userID == "" {
		json.NewEncoder(w).Encode(Response{
			Code: 1,
			Msg:  "user_id required",
		})
		return
	}

	// 获取所有资产余额
	balances := account.GetAllBalances(userID)

	json.NewEncoder(w).Encode(Response{
		Code: 0,
		Msg:  "ok",
		Data: balances,
	})
}
