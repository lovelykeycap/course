package domain

type Role string

const (
	RoleRegular Role = "regular"
	RoleAdmin   Role = "admin"
)

type Employee struct {
	role Role
}

func NewEmployee(role Role) Employee {
	return Employee{role: role}
}

func (e Employee) CanOverrideBookings() bool { return e.role == RoleAdmin }
