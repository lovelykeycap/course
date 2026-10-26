package main

import (
	"net/http"

	"github.com/course-go-autumn-2026/practice6-variant2-booking/handler"
	"github.com/course-go-autumn-2026/practice6-variant2-booking/repository"
	"github.com/course-go-autumn-2026/practice6-variant2-booking/service"
)

func newTestHandler(availabilityURL, notifyURL string, client *http.Client) http.Handler {
	repo := repository.NewMemoryBookingRepository()
	employees := repository.NewMemoryEmployeeRepository()
	availability := repository.NewHTTPRoomAvailability(availabilityURL, client)
	notifier := repository.NewHTTPNotifier(notifyURL, client)
	svc := service.NewBookingService(repo, employees, availability, notifier)
	return handler.NewRouter(handler.NewBookingHandler(svc))
}
