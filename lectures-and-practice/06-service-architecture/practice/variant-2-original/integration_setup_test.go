package main

import "net/http"

func newTestHandler(availabilityURL, notificationURL string, client *http.Client) http.Handler {
	db = map[int64]*storedBooking{}
	nextID = 0
	roomAvailabilityURL = availabilityURL
	notifyURL = notificationURL
	http.DefaultClient = client
	mux := http.NewServeMux()
	mux.HandleFunc("POST /bookings", createBookingHandler)
	mux.HandleFunc("GET /bookings/{id}", getBookingHandler)
	mux.HandleFunc("PATCH /bookings/{id}/time", updateTimeHandler)
	return mux
}
