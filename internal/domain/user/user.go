// Package user defines user domain entities and roles.
package user

import "time"

const (
	RoleMerchant = "MERCHANT"
	RoleAdmin    = "ADMIN"
	RoleUser     = "USER"
)

type User struct {
	ID           string
	Name         string
	Email        string
	PasswordHash string
	Role         string
	CreatedAt    time.Time
}

func (u *User) ChangeEmail(newEmail string) {
	u.Email = newEmail
}
