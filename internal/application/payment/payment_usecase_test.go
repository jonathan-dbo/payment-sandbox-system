package payment

import (
	"context"
	"errors"
	"testing"
	"time"

	appInvoice "github.com/gonszalito/go-ddd-architecture/internal/application/invoice"
	"github.com/gonszalito/go-ddd-architecture/internal/domain/invoice"
	"github.com/gonszalito/go-ddd-architecture/internal/domain/paymentintent"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPaymentIntentPublicTokenResolve(t *testing.T) {
	svc, _, _ := newPaymentServiceWithInvoice(t)
	inv, err := svc.ResolvePublicToken(context.Background(), "pay-token-1")
	require.NoError(t, err)
	assert.Equal(t, "inv-1", inv.ID)
}

func TestPaymentIntentIntentCreation(t *testing.T) {
	svc, _, _ := newPaymentServiceWithInvoice(t)
	intent, err := svc.CreateIntent(context.Background(), "pay-token-1", MethodWallet)
	require.NoError(t, err)
	assert.Equal(t, paymentintent.StatusPending, intent.Status)
	assert.Equal(t, MethodWallet, intent.Method)
}

func TestPaymentIntentAdminSimulationSuccessFailed(t *testing.T) {
	svc, _, _ := newPaymentServiceWithInvoice(t)
	intent, err := svc.CreateIntent(context.Background(), "pay-token-1", MethodVADummy)
	require.NoError(t, err)

	updated, err := svc.SimulateAdminOutcome(context.Background(), intent.ID, OutcomeSuccess)
	require.NoError(t, err)
	assert.Equal(t, paymentintent.StatusSuccess, updated.Status)

	intent2, err := svc.CreateIntent(context.Background(), "pay-token-1", MethodEWALDummy)
	require.NoError(t, err)
	updated2, err := svc.SimulateAdminOutcome(context.Background(), intent2.ID, OutcomeFailed)
	require.NoError(t, err)
	assert.Equal(t, paymentintent.StatusFailed, updated2.Status)
}

func TestPaymentIntentExpirySimulation(t *testing.T) {
	svc, _, _ := newPaymentServiceWithInvoice(t)
	intent, err := svc.CreateIntent(context.Background(), "pay-token-1", MethodWallet)
	require.NoError(t, err)

	svc.nowFn = func() time.Time { return intent.DueAt.Add(time.Second) }
	updated, err := svc.SimulateAdminOutcome(context.Background(), intent.ID, OutcomeSuccess)
	require.NoError(t, err)
	assert.Equal(t, paymentintent.StatusFailed, updated.Status)
}

func TestPaymentIntentInvoiceStatusPropagation(t *testing.T) {
	svc, _, _ := newPaymentServiceWithInvoice(t)
	intent, err := svc.CreateIntent(context.Background(), "pay-token-1", MethodWallet)
	require.NoError(t, err)

	_, err = svc.SimulateAdminOutcome(context.Background(), intent.ID, OutcomeSuccess)
	require.NoError(t, err)

	inv, err := svc.invoiceService.GetByID(context.Background(), "inv-1")
	require.NoError(t, err)
	assert.Equal(t, invoice.StatusPaid, inv.Status)
}

func TestAtomicPaymentRollbackOnInvoiceSaveFailure(t *testing.T) {
	svc, paymentRepo, invoiceRepo := newPaymentServiceWithInvoice(t)
	invoiceRepo.failOnSave = true
	intent, err := svc.CreateIntent(context.Background(), "pay-token-1", MethodWallet)
	require.NoError(t, err)

	_, err = svc.SimulateAdminOutcome(context.Background(), intent.ID, OutcomeSuccess)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "atomic payment flow failed")
	assert.True(t, errors.Is(err, errForcedSave))

	persistedIntent, err := paymentRepo.FindByID(context.Background(), intent.ID)
	require.NoError(t, err)
	assert.Equal(t, paymentintent.StatusPending, persistedIntent.Status)
	persistedInvoice, err := invoiceRepo.FindByID(context.Background(), "inv-1")
	require.NoError(t, err)
	assert.Equal(t, invoice.StatusPending, persistedInvoice.Status)
}

func TestAtomicPaymentCommitSuccess(t *testing.T) {
	svc, paymentRepo, invoiceRepo := newPaymentServiceWithInvoice(t)
	intent, err := svc.CreateIntent(context.Background(), "pay-token-1", MethodWallet)
	require.NoError(t, err)
	_, err = svc.SimulateAdminOutcome(context.Background(), intent.ID, OutcomeSuccess)
	require.NoError(t, err)

	persistedIntent, err := paymentRepo.FindByID(context.Background(), intent.ID)
	require.NoError(t, err)
	assert.Equal(t, paymentintent.StatusSuccess, persistedIntent.Status)
	persistedInvoice, err := invoiceRepo.FindByID(context.Background(), "inv-1")
	require.NoError(t, err)
	assert.Equal(t, invoice.StatusPaid, persistedInvoice.Status)
}

func newPaymentServiceWithInvoice(t *testing.T) (*Service, *paymentRepoStub, *invoiceRepoStub) {
	t.Helper()
	invoiceRepo := &invoiceRepoStub{
		byID: map[string]*invoice.Invoice{},
	}
	invoiceRepo.byID["inv-1"] = &invoice.Invoice{
		ID:            "inv-1",
		InvoiceNumber: "INV-001",
		MerchantID:    "m-1",
		Amount:        1000,
		Currency:      "USD",
		Status:        invoice.StatusPending,
		PaymentToken:  "pay-token-1",
		CreatedAt:     time.Now().UTC(),
	}
	invoiceService := appInvoice.NewService(invoiceRepo)
	paymentRepo := &paymentRepoStub{byID: map[string]*paymentintent.PaymentIntent{}}
	svc := NewService(paymentRepo, invoiceService)
	return svc, paymentRepo, invoiceRepo
}

type paymentRepoStub struct {
	byID         map[string]*paymentintent.PaymentIntent
	failOnCreate bool
	failOnSave   bool
}

func (s *paymentRepoStub) Create(_ context.Context, intent *paymentintent.PaymentIntent) error {
	if s.failOnCreate {
		return errForcedCreate
	}
	copy := *intent
	s.byID[intent.ID] = &copy
	return nil
}
func (s *paymentRepoStub) FindByID(_ context.Context, id string) (*paymentintent.PaymentIntent, error) {
	model, ok := s.byID[id]
	if !ok {
		return nil, ErrNotFound
	}
	copy := *model
	return &copy, nil
}
func (s *paymentRepoStub) Save(_ context.Context, intent *paymentintent.PaymentIntent) error {
	if s.failOnSave {
		return errForcedPaymentSave
	}
	copy := *intent
	s.byID[intent.ID] = &copy
	return nil
}
func (s *paymentRepoStub) List(_ context.Context, _ ListFilter) ([]*paymentintent.PaymentIntent, error) {
	items := make([]*paymentintent.PaymentIntent, 0, len(s.byID))
	for _, model := range s.byID {
		copy := *model
		items = append(items, &copy)
	}
	return items, nil
}

type invoiceRepoStub struct {
	byID       map[string]*invoice.Invoice
	failOnSave bool
}

func (s *invoiceRepoStub) Create(_ context.Context, inv *invoice.Invoice) error {
	return s.Save(context.Background(), inv)
}
func (s *invoiceRepoStub) FindByID(_ context.Context, id string) (*invoice.Invoice, error) {
	inv, ok := s.byID[id]
	if !ok {
		return nil, appInvoice.ErrNotFound
	}
	copy := *inv
	return &copy, nil
}
func (s *invoiceRepoStub) FindByInvoiceNumber(_ context.Context, invoiceNumber string) (*invoice.Invoice, error) {
	for _, inv := range s.byID {
		if inv.InvoiceNumber == invoiceNumber {
			copy := *inv
			return &copy, nil
		}
	}
	return nil, appInvoice.ErrNotFound
}
func (s *invoiceRepoStub) FindByPaymentToken(_ context.Context, token string) (*invoice.Invoice, error) {
	for _, inv := range s.byID {
		if inv.PaymentToken == token {
			copy := *inv
			return &copy, nil
		}
	}
	return nil, appInvoice.ErrNotFound
}
func (s *invoiceRepoStub) Save(_ context.Context, inv *invoice.Invoice) error {
	if s.failOnSave {
		return errForcedSave
	}
	copy := *inv
	s.byID[inv.ID] = &copy
	return nil
}
func (s *invoiceRepoStub) List(_ context.Context, _ appInvoice.ListFilter) ([]*invoice.Invoice, error) {
	items := []*invoice.Invoice{}
	for _, inv := range s.byID {
		copy := *inv
		items = append(items, &copy)
	}
	return items, nil
}

func (s *invoiceRepoStub) Delete(_ context.Context, id string) error {
	if _, ok := s.byID[id]; !ok {
		return appInvoice.ErrNotFound
	}
	delete(s.byID, id)
	return nil
}

var errForcedSave = errors.New("forced save failure")
var errForcedCreate = errors.New("forced payment intent create failure")
var errForcedPaymentSave = errors.New("forced payment intent save failure")
