// Package database contains in-memory and SQL repositories.
package database

import (
	"context"
	"sort"
	"sync"

	appRefund "github.com/gonszalito/go-ddd-architecture/internal/application/refund"
	domainRefund "github.com/gonszalito/go-ddd-architecture/internal/domain/refund"
)

type InMemoryRefundRepository struct {
	mu      sync.RWMutex
	refunds map[string]*domainRefund.Refund
}

func NewInMemoryRefundRepository(seed []*domainRefund.Refund) *InMemoryRefundRepository {
	repo := &InMemoryRefundRepository{refunds: map[string]*domainRefund.Refund{}}
	for _, model := range seed {
		copy := *model
		repo.refunds[copy.ID] = &copy
	}
	return repo
}

func (r *InMemoryRefundRepository) Create(_ context.Context, model *domainRefund.Refund) error {
	return r.Save(context.Background(), model)
}

func (r *InMemoryRefundRepository) FindByID(_ context.Context, id string) (*domainRefund.Refund, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	model, ok := r.refunds[id]
	if !ok {
		return nil, appRefund.ErrNotFound
	}
	copy := *model
	return &copy, nil
}

func (r *InMemoryRefundRepository) Save(_ context.Context, model *domainRefund.Refund) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	copy := *model
	r.refunds[model.ID] = &copy
	return nil
}

func (r *InMemoryRefundRepository) List(_ context.Context, filter appRefund.ListFilter) ([]*domainRefund.Refund, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := []*domainRefund.Refund{}
	for _, model := range r.refunds {
		if filter.MerchantID != "" && model.MerchantID != filter.MerchantID {
			continue
		}
		if filter.StartDate != nil && model.CreatedAt.Before(*filter.StartDate) {
			continue
		}
		if filter.EndDate != nil && model.CreatedAt.After(*filter.EndDate) {
			continue
		}
		copy := *model
		out = append(out, &copy)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out, nil
}

func (r *InMemoryRefundRepository) ListByInvoiceID(_ context.Context, invoiceID string) ([]*domainRefund.Refund, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := []*domainRefund.Refund{}
	for _, model := range r.refunds {
		if model.InvoiceID != invoiceID {
			continue
		}
		copy := *model
		out = append(out, &copy)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out, nil
}
