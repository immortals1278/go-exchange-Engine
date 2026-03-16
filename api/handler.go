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

func (h *Handler) CancelOrder(w http.ResponseWriter, r *http.Request) {

	id := r.URL.Query().Get("id")

	ok := h.Engine.CancelOrder(id)

	if ok {
		w.Write([]byte("canceled"))
	} else {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte("order not found"))
	}
}