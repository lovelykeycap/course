package service

import (
	"context"

	"github.com/course-go-autumn-2026/practice6-variant2-booking/domain"
)

type bookingService struct {
	repo         BookingRepository
	employees    EmployeeRepository
	availability RoomAvailability
	notifier     Notifier
}

func NewBookingService(repo BookingRepository, employees EmployeeRepository, availability RoomAvailability, notifier Notifier) BookingService {
	return &bookingService{repo: repo, employees: employees, availability: availability, notifier: notifier}
}

type CreateInput struct {
	RoomID     int64
	EmployeeID int64
	StartAt    string
	EndAt      string
}

func (s *bookingService) Create(ctx context.Context, in CreateInput) (*domain.Booking, error) {
	startAt, endAt, err := domain.ParseInterval(in.StartAt, in.EndAt)
	if err != nil {
		return nil, err
	}
	employee := s.employees.Get(in.EmployeeID)
	overlapping, err := s.repo.FindOverlapping(in.RoomID, startAt, endAt)
	if err != nil {
		return nil, err
	}
	if overlapping != nil {
		if err := overlapping.Cancel(employee); err != nil {
			return nil, err
		}
		if err := s.repo.Save(overlapping); err != nil {
			return nil, err
		}
	}

	roomAvailable, avErr := s.availability.Check(ctx, in.RoomID, in.StartAt, in.EndAt)
	if avErr != nil {
		roomAvailable = false
	}
	booking, err := domain.NewBooking(in.RoomID, in.EmployeeID, startAt, endAt)
	if err != nil {
		return nil, err
	}
	if err := s.repo.Create(booking); err != nil {
		return nil, err
	}
	if roomAvailable {
		if err := booking.Confirm(); err != nil {
			return nil, err
		}
		if err := s.repo.Save(booking); err != nil {
			return nil, err
		}
	}
	_ = s.notifier.Send(ctx, booking.ID())
	return booking, nil
}

func (s *bookingService) Get(id int64) (*domain.Booking, error) {
	return s.repo.Get(id)
}

func (s *bookingService) UpdateTime(id int64, start, end string) (*domain.Booking, error) {
	newStart, newEnd, err := domain.ParseInterval(start, end)
	if err != nil {
		return nil, err
	}
	booking, err := s.repo.Get(id)
	if err != nil {
		return nil, err
	}
	if err := booking.Reschedule(newStart, newEnd); err != nil {
		return nil, err
	}
	if err := s.repo.Save(booking); err != nil {
		return nil, err
	}
	return booking, nil
}
