package repository

import (
	"fmt"
	"sync"
	"time"

	"github.com/course-go-autumn-2026/practice6-variant2-booking/domain"
)

type storedBooking struct {
	ID         int64
	RoomID     int64
	EmployeeID int64
	StartAt    time.Time
	EndAt      time.Time
	Status     string
	CreatedAt  time.Time
}

func storeBooking(b *domain.Booking) storedBooking {
	return storedBooking{
		ID: b.ID(), RoomID: b.RoomID(), EmployeeID: b.EmployeeID(),
		StartAt: b.StartAt(), EndAt: b.EndAt(), Status: string(b.Status()), CreatedAt: b.CreatedAt(),
	}
}

func (b storedBooking) restore() (*domain.Booking, error) {
	booking, err := domain.RestoreBooking(b.ID, b.RoomID, b.EmployeeID, b.StartAt, b.EndAt, domain.Status(b.Status), b.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("restore booking %d: %v", b.ID, err)
	}
	return booking, nil
}

type MemoryBookingRepository struct {
	mu     sync.RWMutex
	db     map[int64]storedBooking
	nextID int64
}

func NewMemoryBookingRepository() *MemoryBookingRepository {
	return &MemoryBookingRepository{db: map[int64]storedBooking{}}
}

func (r *MemoryBookingRepository) Get(id int64) (*domain.Booking, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	booking, ok := r.db[id]
	if !ok {
		return nil, domain.ErrNotFound
	}
	return booking.restore()
}

func (r *MemoryBookingRepository) FindOverlapping(roomID int64, start, end time.Time) (*domain.Booking, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, record := range r.db {
		b, err := record.restore()
		if err != nil {
			return nil, err
		}
		if b.Overlaps(roomID, start, end) {
			return b, nil
		}
	}
	return nil, nil
}

func (r *MemoryBookingRepository) Create(booking *domain.Booking) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.nextID++
	booking.AssignID(r.nextID)
	r.db[booking.ID()] = storeBooking(booking)
	return nil
}

func (r *MemoryBookingRepository) Save(booking *domain.Booking) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.db[booking.ID()]; !ok {
		return domain.ErrNotFound
	}
	r.db[booking.ID()] = storeBooking(booking)
	return nil
}

type storedEmployee struct {
	Role string
}

type MemoryEmployeeRepository struct {
	employees map[int64]storedEmployee
}

func NewMemoryEmployeeRepository() *MemoryEmployeeRepository {
	return &MemoryEmployeeRepository{employees: map[int64]storedEmployee{
		1: {Role: "regular"},
		2: {Role: "admin"},
	}}
}

func (r *MemoryEmployeeRepository) Get(id int64) domain.Employee {
	employee, ok := r.employees[id]
	if !ok {
		return domain.NewEmployee(domain.RoleRegular)
	}
	return domain.NewEmployee(domain.Role(employee.Role))
}
