package handler

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/course-go-autumn-2026/practice6-variant2-booking/domain"
	"github.com/course-go-autumn-2026/practice6-variant2-booking/service"
)

type BookingHandler struct {
	service service.BookingService
}

func NewBookingHandler(s service.BookingService) *BookingHandler {
	return &BookingHandler{service: s}
}

type createBookingRequest struct {
	RoomID     int64  `json:"room_id"`
	EmployeeID int64  `json:"employee_id"`
	StartAt    string `json:"start_at"`
	EndAt      string `json:"end_at"`
}

type updateTimeRequest struct {
	StartAt string `json:"start_at"`
	EndAt   string `json:"end_at"`
}

func (h *BookingHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req createBookingRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	booking, err := h.service.Create(r.Context(), service.CreateInput{
		RoomID: req.RoomID, EmployeeID: req.EmployeeID, StartAt: req.StartAt, EndAt: req.EndAt,
	})
	if err != nil {
		writeError(w, err)
		return
	}
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]any{"id": booking.ID(), "status": booking.Status()})
}

func (h *BookingHandler) Get(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	booking, err := h.service.Get(id)
	if err != nil {
		writeError(w, err)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(bookingResponseFrom(booking))
}

func (h *BookingHandler) UpdateTime(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	var req updateTimeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	booking, err := h.service.UpdateTime(id, req.StartAt, req.EndAt)
	if err != nil {
		writeError(w, err)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(bookingResponseFrom(booking))
}

type bookingResponse struct {
	ID         int64
	RoomID     int64
	EmployeeID int64
	StartAt    time.Time
	EndAt      time.Time
	Status     domain.Status
	CreatedAt  time.Time
}

func bookingResponseFrom(b *domain.Booking) bookingResponse {
	return bookingResponse{
		ID: b.ID(), RoomID: b.RoomID(), EmployeeID: b.EmployeeID(),
		StartAt: b.StartAt(), EndAt: b.EndAt(), Status: b.Status(), CreatedAt: b.CreatedAt(),
	}
}
