package service

import (
	"context"
	"time"

	"github.com/course-go-autumn-2026/practice6-variant2-booking/domain"
)

type BookingService interface {
	Create(ctx context.Context, in CreateInput) (*domain.Booking, error)
	Get(id int64) (*domain.Booking, error)
	UpdateTime(id int64, start, end string) (*domain.Booking, error)
}

type BookingRepository interface {
	Get(id int64) (*domain.Booking, error)
	FindOverlapping(roomID int64, start, end time.Time) (*domain.Booking, error)
	Create(booking *domain.Booking) error
	Save(booking *domain.Booking) error
}

type EmployeeRepository interface {
	Get(id int64) domain.Employee
}

type RoomAvailability interface {
	Check(ctx context.Context, roomID int64, start, end string) (bool, error)
}

type Notifier interface {
	Send(ctx context.Context, bookingID int64) error
}
