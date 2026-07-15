// Package database contains in-memory and SQL repositories.
package database

import (
	"context"
	"sort"
	"sync"

	appTopUp "github.com/gonszalito/go-ddd-architecture/internal/application/topup"
	domainTopUp "github.com/gonszalito/go-ddd-architecture/internal/domain/topup"
)

type InMemoryTopUpRepository struct {
	mu     sync.RWMutex
	topups map[string]*domainTopUp.TopUp
}

func NewInMemoryTopUpRepository(seed []*domainTopUp.TopUp) *InMemoryTopUpRepository {
	repo := &InMemoryTopUpRepository{topups: map[string]*domainTopUp.TopUp{}}
	for _, model := range seed {
		copy := *model
		repo.topups[copy.ID] = &copy
	}
	return repo
}

func (r *InMemoryTopUpRepository) Create(_ context.Context, model *domainTopUp.TopUp) error {
	return r.Save(context.Background(), model)
}

func (r *InMemoryTopUpRepository) FindByID(_ context.Context, id string) (*domainTopUp.TopUp, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	model, ok := r.topups[id]
	if !ok {
		return nil, appTopUp.ErrNotFound
	}
	copy := *model
	return &copy, nil
}

func (r *InMemoryTopUpRepository) FindByRequestKey(_ context.Context, merchantID, requestKey string) (*domainTopUp.TopUp, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, model := range r.topups {
		if model.MerchantID == merchantID && model.RequestKey == requestKey && requestKey != "" {
			copy := *model
			return &copy, nil
		}
	}
	return nil, appTopUp.ErrNotFound
}

func (r *InMemoryTopUpRepository) ListByMerchantID(_ context.Context, merchantID string) ([]*domainTopUp.TopUp, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := []*domainTopUp.TopUp{}
	for _, model := range r.topups {
		if merchantID != "" && model.MerchantID != merchantID {
			continue
		}
		copy := *model
		out = append(out, &copy)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out, nil
}

func (r *InMemoryTopUpRepository) Save(_ context.Context, model *domainTopUp.TopUp) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	copy := *model
	r.topups[model.ID] = &copy
	return nil
}
