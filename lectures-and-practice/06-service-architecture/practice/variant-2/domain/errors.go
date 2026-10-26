package domain

import "errors"

var (
	ErrInvalidStatusTransition = errors.New("invalid booking status transition")
	ErrInvalidStart            = errors.New("start_at must be a valid timestamp")
	ErrInvalidEnd              = errors.New("end_at must be a valid timestamp")
	ErrInvalidInterval         = errors.New("end_at must be after start_at")
	ErrAlreadyBooked           = errors.New("room is already booked for this time")
	ErrCancelled               = errors.New("cannot reschedule cancelled booking")
	ErrNotFound                = errors.New("booking not found")
)
