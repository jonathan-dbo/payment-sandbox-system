package invoice_test

import (
	"context"
	"errors"
	"testing"
	"time"

	appInvoice "github.com/gonszalito/go-ddd-architecture/internal/application/invoice"
	"github.com/gonszalito/go-ddd-architecture/internal/domain/invoice"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// errorInjectingInvoiceRepo wraps an in-memory-like map with fine-grained
// error injection hooks, to exercise invoice_usecase.go branches that the
// InMemoryInvoiceRepository cannot trigger on its own (repository failures,
// duplicate-generation collisions, and refreshExpiry propagation).
type errorInjectingInvoiceRepo struct {
	invoices map[string]*invoice.Invoice

	failOnCreate            error
	failOnList              error
	failOnSave              error
	failOnFindByID          error
	failOnFindByPaymentTok  error
	invoiceNumberCollisions int // number of times FindByInvoiceNumber should report "found" before returning not-found
	paymentTokenCollisions  int
}

func newErrorInjectingInvoiceRepo() *errorInjectingInvoiceRepo {
	return &errorInjectingInvoiceRepo{invoices: map[string]*invoice.Invoice{}}
}

func (r *errorInjectingInvoiceRepo) Create(_ context.Context, inv *invoice.Invoice) error {
	if r.failOnCreate != nil {
		return r.failOnCreate
	}
	copy := *inv
	r.invoices[inv.ID] = &copy
	return nil
}

func (r *errorInjectingInvoiceRepo) FindByID(_ context.Context, id string) (*invoice.Invoice, error) {
	if r.failOnFindByID != nil {
		return nil, r.failOnFindByID
	}
	inv, ok := r.invoices[id]
	if !ok {
		return nil, appInvoice.ErrNotFound
	}
	copy := *inv
	return &copy, nil
}

func (r *errorInjectingInvoiceRepo) FindByInvoiceNumber(_ context.Context, _ string) (*invoice.Invoice, error) {
	if r.invoiceNumberCollisions > 0 {
		r.invoiceNumberCollisions--
		return &invoice.Invoice{ID: "collided"}, nil
	}
	return nil, appInvoice.ErrNotFound
}

func (r *errorInjectingInvoiceRepo) FindByPaymentToken(_ context.Context, token string) (*invoice.Invoice, error) {
	if r.failOnFindByPaymentTok != nil {
		return nil, r.failOnFindByPaymentTok
	}
	if r.paymentTokenCollisions > 0 {
		r.paymentTokenCollisions--
		return &invoice.Invoice{ID: "collided"}, nil
	}
	for _, inv := range r.invoices {
		if inv.PaymentToken == token {
			copy := *inv
			return &copy, nil
		}
	}
	return nil, appInvoice.ErrNotFound
}

func (r *errorInjectingInvoiceRepo) Save(_ context.Context, inv *invoice.Invoice) error {
	if r.failOnSave != nil {
		return r.failOnSave
	}
	copy := *inv
	r.invoices[inv.ID] = &copy
	return nil
}

func (r *errorInjectingInvoiceRepo) Delete(_ context.Context, id string) error {
	if _, ok := r.invoices[id]; !ok {
		return appInvoice.ErrNotFound
	}
	delete(r.invoices, id)
	return nil
}

func (r *errorInjectingInvoiceRepo) List(_ context.Context, _ appInvoice.ListFilter) ([]*invoice.Invoice, error) {
	if r.failOnList != nil {
		return nil, r.failOnList
	}
	items := make([]*invoice.Invoice, 0, len(r.invoices))
	for _, inv := range r.invoices {
		copy := *inv
		items = append(items, &copy)
	}
	return items, nil
}

var errRepoBoom = errors.New("repository boom")

func TestInvoiceCreate_RepoCreateErrorPropagates(t *testing.T) {
	repo := newErrorInjectingInvoiceRepo()
	repo.failOnCreate = errRepoBoom
	svc := appInvoice.NewService(repo)

	_, err := svc.Create(context.Background(), appInvoice.CreateInvoiceInput{MerchantID: "m1", Amount: 100, Currency: "USD"})
	require.Error(t, err)
	assert.True(t, errors.Is(err, errRepoBoom))
}

func TestInvoiceGenerateUniqueInvoiceNumber_RetriesOnCollisionThenSucceeds(t *testing.T) {
	repo := newErrorInjectingInvoiceRepo()
	repo.invoiceNumberCollisions = 3
	svc := appInvoice.NewService(repo)

	inv, err := svc.Create(context.Background(), appInvoice.CreateInvoiceInput{MerchantID: "m1", Amount: 100, Currency: "USD"})
	require.NoError(t, err)
	assert.NotEmpty(t, inv.InvoiceNumber)
}

func TestInvoiceGenerateUniqueInvoiceNumber_ExhaustsRetriesReturnsError(t *testing.T) {
	repo := newErrorInjectingInvoiceRepo()
	repo.invoiceNumberCollisions = 10 // always collides for all 10 attempts
	svc := appInvoice.NewService(repo)

	_, err := svc.Create(context.Background(), appInvoice.CreateInvoiceInput{MerchantID: "m1", Amount: 100, Currency: "USD"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unable to generate unique invoice number")
}

func TestInvoiceGenerateUniquePaymentToken_RetriesOnCollisionThenSucceeds(t *testing.T) {
	repo := newErrorInjectingInvoiceRepo()
	repo.paymentTokenCollisions = 3
	svc := appInvoice.NewService(repo)

	inv, err := svc.Create(context.Background(), appInvoice.CreateInvoiceInput{MerchantID: "m1", Amount: 100, Currency: "USD"})
	require.NoError(t, err)
	assert.NotEmpty(t, inv.PaymentToken)
}

func TestInvoiceGenerateUniquePaymentToken_ExhaustsRetriesReturnsError(t *testing.T) {
	repo := newErrorInjectingInvoiceRepo()
	repo.paymentTokenCollisions = 10
	svc := appInvoice.NewService(repo)

	_, err := svc.Create(context.Background(), appInvoice.CreateInvoiceInput{MerchantID: "m1", Amount: 100, Currency: "USD"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unable to generate unique payment token")
}

func TestInvoiceList_RepoListErrorPropagates(t *testing.T) {
	repo := newErrorInjectingInvoiceRepo()
	repo.failOnList = errRepoBoom
	svc := appInvoice.NewService(repo)

	_, err := svc.List(context.Background(), appInvoice.ListFilter{})
	require.Error(t, err)
	assert.True(t, errors.Is(err, errRepoBoom))
}

func TestInvoiceList_RefreshExpirySaveErrorPropagates(t *testing.T) {
	repo := newErrorInjectingInvoiceRepo()
	past := time.Now().UTC().Add(-1 * time.Hour)
	repo.invoices["inv-1"] = &invoice.Invoice{
		ID: "inv-1", MerchantID: "m1", Status: invoice.StatusPending, DueDate: past, Amount: 100,
	}
	svc := appInvoice.NewService(repo)
	repo.failOnSave = errRepoBoom

	_, err := svc.List(context.Background(), appInvoice.ListFilter{})
	require.Error(t, err)
	assert.True(t, errors.Is(err, errRepoBoom))
}

func TestInvoiceResolvePaymentToken_RepoErrorPropagates(t *testing.T) {
	repo := newErrorInjectingInvoiceRepo()
	repo.failOnFindByPaymentTok = errRepoBoom
	svc := appInvoice.NewService(repo)

	_, err := svc.ResolvePaymentToken(context.Background(), "some-token")
	require.Error(t, err)
	assert.True(t, errors.Is(err, errRepoBoom))
}

func TestInvoiceResolvePaymentToken_RefreshExpirySaveErrorPropagates(t *testing.T) {
	repo := newErrorInjectingInvoiceRepo()
	past := time.Now().UTC().Add(-1 * time.Hour)
	repo.invoices["inv-1"] = &invoice.Invoice{
		ID: "inv-1", MerchantID: "m1", Status: invoice.StatusPending, DueDate: past, PaymentToken: "tok-1", Amount: 100,
	}
	svc := appInvoice.NewService(repo)
	repo.failOnSave = errRepoBoom

	_, err := svc.ResolvePaymentToken(context.Background(), "tok-1")
	require.Error(t, err)
	assert.True(t, errors.Is(err, errRepoBoom))
}

func TestInvoiceGetByID_RepoErrorPropagates(t *testing.T) {
	repo := newErrorInjectingInvoiceRepo()
	repo.failOnFindByID = errRepoBoom
	svc := appInvoice.NewService(repo)

	_, err := svc.GetByID(context.Background(), "inv-1")
	require.Error(t, err)
	assert.True(t, errors.Is(err, errRepoBoom))
}

func TestInvoiceGetByID_RefreshExpirySaveErrorPropagates(t *testing.T) {
	repo := newErrorInjectingInvoiceRepo()
	past := time.Now().UTC().Add(-1 * time.Hour)
	repo.invoices["inv-1"] = &invoice.Invoice{
		ID: "inv-1", MerchantID: "m1", Status: invoice.StatusPending, DueDate: past, Amount: 100,
	}
	svc := appInvoice.NewService(repo)
	repo.failOnSave = errRepoBoom

	_, err := svc.GetByID(context.Background(), "inv-1")
	require.Error(t, err)
	assert.True(t, errors.Is(err, errRepoBoom))
}
