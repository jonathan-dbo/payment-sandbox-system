package user

import "github.com/gonszalito/go-ddd-architecture/internal/domain/user"

// UserRepository defines storage operations for User entities
type UserRepository interface {
	FindByID(id string) (*user.User, error)
	FindByEmail(email string) (*user.User, error)
	Create(u *user.User) error
	CreateWithMerchant(u *user.User) error
	Save(u *user.User) error
}
