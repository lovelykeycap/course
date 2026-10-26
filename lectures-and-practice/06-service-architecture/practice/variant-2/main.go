package main

import (
	"log"
	"net/http"

	"github.com/course-go-autumn-2026/practice6-variant2-booking/handler"
	"github.com/course-go-autumn-2026/practice6-variant2-booking/repository"
	"github.com/course-go-autumn-2026/practice6-variant2-booking/service"
)

func main() {
	repo := repository.NewMemoryBookingRepository()
	employees := repository.NewMemoryEmployeeRepository()
	availability := repository.NewHTTPRoomAvailability("http://room-availability.internal/check", http.DefaultClient)
	notifier := repository.NewHTTPNotifier("http://notify.internal/send", http.DefaultClient)
	svc := service.NewBookingService(repo, employees, availability, notifier)
	h := handler.NewBookingHandler(svc)

	mux := handler.NewRouter(h)

	log.Println("listening on :8080")
	log.Fatal(http.ListenAndServe(":8080", mux))
}
