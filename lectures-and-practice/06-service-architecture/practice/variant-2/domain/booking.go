package domain

import "time"

type Status string

const (
	StatusPendingConfirmation Status = "pending_confirmation"
	StatusConfirmed           Status = "confirmed"
	StatusCancelled           Status = "cancelled"
	StatusRescheduling        Status = "rescheduling"
	rescheduleConfirmWindow          = 2 * time.Hour
)

type Booking struct {
	id         int64
	roomID     int64
	employeeID int64
	startAt    time.Time
	endAt      time.Time
	status     Status
	createdAt  time.Time
}

func NewBooking(roomID, employeeID int64, startAt, endAt time.Time) (*Booking, error) {
	if !endAt.After(startAt) {
		return nil, ErrInvalidInterval
	}
	return &Booking{
		roomID: roomID, employeeID: employeeID, startAt: startAt, endAt: endAt,
		status: StatusPendingConfirmation, createdAt: time.Now(),
	}, nil
}

func RestoreBooking(id, roomID, employeeID int64, startAt, endAt time.Time, status Status, createdAt time.Time) (*Booking, error) {
	if !endAt.After(startAt) {
		return nil, ErrInvalidInterval
	}
	switch status {
	case StatusPendingConfirmation, StatusConfirmed, StatusCancelled, StatusRescheduling:
	default:
		return nil, ErrInvalidStatusTransition
	}
	return &Booking{
		id: id, roomID: roomID, employeeID: employeeID,
		startAt: startAt, endAt: endAt, status: status, createdAt: createdAt,
	}, nil
}

func (b *Booking) ID() int64            { return b.id }
func (b *Booking) RoomID() int64        { return b.roomID }
func (b *Booking) EmployeeID() int64    { return b.employeeID }
func (b *Booking) StartAt() time.Time   { return b.startAt }
func (b *Booking) EndAt() time.Time     { return b.endAt }
func (b *Booking) Status() Status       { return b.status }
func (b *Booking) CreatedAt() time.Time { return b.createdAt }
func (b *Booking) AssignID(id int64)    { b.id = id }

func (b *Booking) Confirm() error {
	if b.status != StatusPendingConfirmation {
		return ErrInvalidStatusTransition
	}
	b.status = StatusConfirmed
	return nil
}

func (b *Booking) Cancel(employee Employee) error {
	if !employee.CanOverrideBookings() {
		return ErrAlreadyBooked
	}
	b.status = StatusCancelled
	return nil
}

func (b *Booking) Reschedule(startAt, endAt time.Time) error {
	if !endAt.After(startAt) {
		return ErrInvalidInterval
	}
	if b.status == StatusCancelled {
		return ErrCancelled
	}
	b.startAt = startAt
	b.endAt = endAt
	if b.status == StatusConfirmed && time.Until(startAt) < rescheduleConfirmWindow {
		b.status = StatusRescheduling
	}
	return nil
}

func (b *Booking) Overlaps(roomID int64, startAt, endAt time.Time) bool {
	return b.roomID == roomID && b.status != StatusCancelled &&
		startAt.Before(b.endAt) && b.startAt.Before(endAt)
}
