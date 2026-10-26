package handler

import "net/http"

func NewRouter(h *BookingHandler) *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /bookings", h.Create)
	mux.HandleFunc("GET /bookings/{id}", h.Get)
	mux.HandleFunc("PATCH /bookings/{id}/time", h.UpdateTime)
	return mux
}
