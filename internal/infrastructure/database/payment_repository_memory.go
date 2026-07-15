// Package database contains in-memory and SQL repositories.
package database

import (
	"context"
	"sort"
	"sync"

	appPayment "github.com/gonszalito/go-ddd-architecture/internal/application/payment"
	domainPayment "github.com/gonszalito/go-ddd-architecture/internal/domain/paymentintent"
)

type InMemoryPaymentRepository struct {
	mu      sync.RWMutex
	intents map[string]*domainPayment.PaymentIntent
}

func NewInMemoryPaymentRepository(seed []*domainPayment.PaymentIntent) *InMemoryPaymentRepository {
	repo := &InMemoryPaymentRepository{intents: map[string]*domainPayment.PaymentIntent{}}
	for _, model := range seed {
		copy := *model
		repo.intents[copy.ID] = &copy
	}
	return repo
}

func (r *InMemoryPaymentRepository) Create(_ context.Context, intent *domainPayment.PaymentIntent) error {
	return r.Save(context.Background(), intent)
}

func (r *InMemoryPaymentRepository) FindByID(_ context.Context, id string) (*domainPayment.PaymentIntent, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	model, ok := r.intents[id]
	if !ok {
		return nil, appPayment.ErrNotFound
	}
	copy := *model
	return &copy, nil
}

func (r *InMemoryPaymentRepository) Save(_ context.Context, intent *domainPayment.PaymentIntent) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	copy := *intent
	r.intents[intent.ID] = &copy
	return nil
}

func (r *InMemoryPaymentRepository) List(_ context.Context, filter appPayment.ListFilter) ([]*domainPayment.PaymentIntent, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	items := make([]*domainPayment.PaymentIntent, 0, len(r.intents))
	for _, model := range r.intents {
		if filter.InvoiceID != "" && model.InvoiceID != filter.InvoiceID {
			continue
		}
		if filter.StartDate != nil && model.CreatedAt.Before(*filter.StartDate) {
			continue
		}
		if filter.EndDate != nil && model.CreatedAt.After(*filter.EndDate) {
			continue
		}
		copy := *model
		items = append(items, &copy)
	}
	sort.Slice(items, func(i, j int) bool {
		return items[i].CreatedAt.Before(items[j].CreatedAt)
	})
	return items, nil
}
