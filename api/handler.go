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

func NewHandler(e *engine.MatchingEngine) *Handler {
	return &Handler{Engine: e}
}

func (h *Handler) PlaceOrder(w http.ResponseWriter, r *http.Request) {

	var order model.Order

	json.NewDecoder(r.Body).Decode(&order)

	h.Engine.PlaceOrder(&order)

	w.Write([]byte("ok"))
}