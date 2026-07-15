// Package database contains in-memory and SQL repositories.
package database

import (
	"context"
	"sort"
	"strings"
	"sync"

	appInvoice "github.com/gonszalito/go-ddd-architecture/internal/application/invoice"
	domainInvoice "github.com/gonszalito/go-ddd-architecture/internal/domain/invoice"
)

type InMemoryInvoiceRepository struct {
	mu       sync.RWMutex
	invoices map[string]*domainInvoice.Invoice
}

func NewInMemoryInvoiceRepository(seed []*domainInvoice.Invoice) *InMemoryInvoiceRepository {
	repo := &InMemoryInvoiceRepository{invoices: map[string]*domainInvoice.Invoice{}}
	for _, inv := range seed {
		copy := *inv
		repo.invoices[inv.ID] = &copy
	}
	return repo
}

func (r *InMemoryInvoiceRepository) Create(_ context.Context, inv *domainInvoice.Invoice) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	copy := *inv
	r.invoices[inv.ID] = &copy
	return nil
}

func (r *InMemoryInvoiceRepository) FindByID(_ context.Context, id string) (*domainInvoice.Invoice, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	inv, ok := r.invoices[id]
	if !ok {
		return nil, appInvoice.ErrNotFound
	}
	copy := *inv
	return &copy, nil
}

func (r *InMemoryInvoiceRepository) FindByInvoiceNumber(_ context.Context, invoiceNumber string) (*domainInvoice.Invoice, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, inv := range r.invoices {
		if inv.InvoiceNumber == invoiceNumber {
			copy := *inv
			return &copy, nil
		}
	}
	return nil, appInvoice.ErrNotFound
}

func (r *InMemoryInvoiceRepository) Save(_ context.Context, inv *domainInvoice.Invoice) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	copy := *inv
	r.invoices[inv.ID] = &copy
	return nil
}

func (r *InMemoryInvoiceRepository) Delete(_ context.Context, id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.invoices[id]; !ok {
		return appInvoice.ErrNotFound
	}
	delete(r.invoices, id)
	return nil
}

func (r *InMemoryInvoiceRepository) FindByPaymentToken(_ context.Context, paymentToken string) (*domainInvoice.Invoice, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, inv := range r.invoices {
		if inv.PaymentToken == paymentToken {
			copy := *inv
			return &copy, nil
		}
	}
	return nil, appInvoice.ErrNotFound
}

func (r *InMemoryInvoiceRepository) List(_ context.Context, filter appInvoice.ListFilter) ([]*domainInvoice.Invoice, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	items := make([]*domainInvoice.Invoice, 0, len(r.invoices))
	for _, inv := range r.invoices {
		if filter.MerchantID != "" && inv.MerchantID != filter.MerchantID {
			continue
		}
		if filter.Status != "" && !strings.EqualFold(inv.Status, filter.Status) {
			continue
		}
		if filter.StartDate != nil && inv.CreatedAt.Before(*filter.StartDate) {
			continue
		}
		if filter.EndDate != nil && inv.CreatedAt.After(*filter.EndDate) {
			continue
		}
		copy := *inv
		items = append(items, &copy)
	}

	sort.Slice(items, func(i, j int) bool {
		return items[i].CreatedAt.After(items[j].CreatedAt)
	})

	start := (filter.Page - 1) * filter.PageSize
	if start >= len(items) {
		return []*domainInvoice.Invoice{}, nil
	}
	end := start + filter.PageSize
	if end > len(items) {
		end = len(items)
	}
	return items[start:end], nil
}
