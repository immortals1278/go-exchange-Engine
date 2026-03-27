package api

import (
	"encoding/json"
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

	json.NewEncoder(w).Encode(Response{
		Code: 0,
		Msg:  "ok",
	})
}

func (h *Handler) CancelOrder(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	id := r.URL.Query().Get("id")
	if id == "" {
		json.NewEncoder(w).Encode(Response{
			Code: 1,
			Msg:  "id required",
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

	json.NewEncoder(w).Encode(Response{
		Code: 0,
		Msg:  "canceled",
	})
}

type LoginRequest struct {
	UserID string `json:"user_id"`
}

func (h *Handler) LogIn(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

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
