package database

import (
	"errors"
	"strings"
	"sync"

	domUser "github.com/gonszalito/go-ddd-architecture/internal/domain/user"
)

// InMemoryUserRepository is a simple map-backed repository for local testing
type InMemoryUserRepository struct {
	mu               sync.RWMutex
	users            map[string]*domUser.User
	merchantByUserID map[string]string
}

func NewInMemoryUserRepository(seed []*domUser.User) *InMemoryUserRepository {
	repo := &InMemoryUserRepository{
		users:            make(map[string]*domUser.User),
		merchantByUserID: make(map[string]string),
	}
	for _, u := range seed {
		copy := *u // avoid external mutation
		repo.users[u.ID] = &copy
		if u.Role == domUser.RoleMerchant {
			repo.merchantByUserID[u.ID] = u.ID
		}
	}
	return repo
}

func (r *InMemoryUserRepository) FindByID(id string) (*domUser.User, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if u, ok := r.users[id]; ok {
		copy := *u
		return &copy, nil
	}
	return nil, errors.New("user not found")
}

func (r *InMemoryUserRepository) FindByEmail(email string) (*domUser.User, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, u := range r.users {
		if strings.EqualFold(u.Email, email) {
			copy := *u
			return &copy, nil
		}
	}
	return nil, errors.New("user not found")
}

func (r *InMemoryUserRepository) Create(u *domUser.User) error {
	return r.Save(u)
}

func (r *InMemoryUserRepository) CreateWithMerchant(u *domUser.User) error {
	if err := r.Save(u); err != nil {
		return err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if u.Role == domUser.RoleMerchant {
		r.merchantByUserID[u.ID] = u.ID
	}
	return nil
}

func (r *InMemoryUserRepository) Save(u *domUser.User) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	copy := *u
	r.users[u.ID] = &copy
	return nil
}

func (r *InMemoryUserRepository) MerchantExistsForUser(userID string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	_, ok := r.merchantByUserID[userID]
	return ok
}

