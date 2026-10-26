package domain

import "time"

func ParseInterval(start, end string) (time.Time, time.Time, error) {
	startAt, err := time.Parse(time.RFC3339, start)
	if err != nil {
		return time.Time{}, time.Time{}, ErrInvalidStart
	}
	endAt, err := time.Parse(time.RFC3339, end)
	if err != nil {
		return time.Time{}, time.Time{}, ErrInvalidEnd
	}
	if !endAt.After(startAt) {
		return time.Time{}, time.Time{}, ErrInvalidInterval
	}
	return startAt, endAt, nil
}
