package invoice

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/gonszalito/go-ddd-architecture/internal/domain/invoice"
	"github.com/gonszalito/go-ddd-architecture/internal/shared"
	"github.com/google/uuid"
)

var ErrNotFound = errors.New("invoice not found")

type Service struct {
	repo InvoiceRepository
}

type CreateInvoiceInput struct {
	MerchantID string
	Amount     int64
	Currency   string
	DueDate    *time.Time
}

type UpdateInvoiceInput struct {
	Amount   int64
	Currency string
	DueDate  *time.Time
}

func NewService(repo InvoiceRepository) *Service {
	return &Service{repo: repo}
}

func (s *Service) Create(ctx context.Context, input CreateInvoiceInput) (*invoice.Invoice, error) {
	invoiceNumber, err := s.generateUniqueInvoiceNumber(ctx)
	if err != nil {
		return nil, err
	}

	paymentToken, err := s.generateUniquePaymentToken(ctx)
	if err != nil {
		return nil, err
	}

	inv := &invoice.Invoice{
		ID:            uuid.NewString(),
		InvoiceNumber: invoiceNumber,
		MerchantID:    input.MerchantID,
		Amount:        input.Amount,
		Currency:      strings.ToUpper(strings.TrimSpace(input.Currency)),
		Status:        invoice.StatusPending,
		PaymentToken:  paymentToken,
		CreatedAt:     time.Now().UTC(),
	}
	if inv.Currency == "" {
		inv.Currency = "USD"
	}
	if input.DueDate != nil {
		inv.DueDate = input.DueDate.UTC()
	}

	if err := s.repo.Create(ctx, inv); err != nil {
		return nil, err
	}
	shared.LogEvent("invoice_created", map[string]any{
		"invoice_id":     inv.ID,
		"invoice_number": inv.InvoiceNumber,
		"merchant_id":    inv.MerchantID,
		"amount":         inv.Amount,
		"currency":       inv.Currency,
	})
	return inv, nil
}

func (s *Service) List(ctx context.Context, filter ListFilter) ([]*invoice.Invoice, error) {
	if filter.Page <= 0 {
		filter.Page = 1
	}
	if filter.PageSize <= 0 {
		filter.PageSize = 10
	}
	if filter.PageSize > 100 {
		filter.PageSize = 100
	}
	items, err := s.repo.List(ctx, filter)
	if err != nil {
		return nil, err
	}
	for _, inv := range items {
		if err := s.refreshExpiry(ctx, inv); err != nil {
			return nil, err
		}
	}
	return items, nil
}

func (s *Service) ResolvePaymentToken(ctx context.Context, token string) (*invoice.Invoice, error) {
	inv, err := s.repo.FindByPaymentToken(ctx, token)
	if err != nil {
		return nil, err
	}
	if err := s.refreshExpiry(ctx, inv); err != nil {
		return nil, err
	}
	return inv, nil
}

func (s *Service) GetByID(ctx context.Context, id string) (*invoice.Invoice, error) {
	inv, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if err := s.refreshExpiry(ctx, inv); err != nil {
		return nil, err
	}
	return inv, nil
}

func (s *Service) Save(ctx context.Context, inv *invoice.Invoice) error {
	return s.repo.Save(ctx, inv)
}

func (s *Service) Update(ctx context.Context, id string, input UpdateInvoiceInput) (*invoice.Invoice, error) {
	inv, err := s.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if inv.Status != invoice.StatusPending {
		return nil, errors.New("only pending invoice can be updated")
	}
	inv.Amount = input.Amount
	inv.Currency = strings.ToUpper(strings.TrimSpace(input.Currency))
	if inv.Currency == "" {
		inv.Currency = "USD"
	}
	if input.DueDate != nil {
		inv.DueDate = input.DueDate.UTC()
	}
	if err := s.repo.Save(ctx, inv); err != nil {
		return nil, err
	}
	shared.LogEvent("invoice_updated", map[string]any{
		"invoice_id":  inv.ID,
		"merchant_id": inv.MerchantID,
		"amount":      inv.Amount,
		"currency":    inv.Currency,
		"status":      inv.Status,
	})
	return inv, nil
}

func (s *Service) Delete(ctx context.Context, id string) error {
	inv, err := s.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if inv.Status != invoice.StatusPending {
		return errors.New("only pending invoice can be deleted")
	}
	if err := s.repo.Delete(ctx, id); err != nil {
		return err
	}
	shared.LogEvent("invoice_deleted", map[string]any{
		"invoice_id":  inv.ID,
		"merchant_id": inv.MerchantID,
	})
	return nil
}

func (s *Service) refreshExpiry(ctx context.Context, inv *invoice.Invoice) error {
	if inv == nil || inv.DueDate.IsZero() || inv.Status != invoice.StatusPending {
		return nil
	}
	if time.Now().UTC().After(inv.DueDate) {
		if err := inv.MarkExpired(); err != nil {
			return err
		}
		return s.repo.Save(ctx, inv)
	}
	return nil
}

func (s *Service) generateUniqueInvoiceNumber(ctx context.Context) (string, error) {
	for range 10 {
		buf := make([]byte, 3)
		if _, err := rand.Read(buf); err != nil {
			return "", err
		}
		candidate := fmt.Sprintf("INV-%s-%s", time.Now().UTC().Format("20060102"), strings.ToUpper(hex.EncodeToString(buf)))
		_, err := s.repo.FindByInvoiceNumber(ctx, candidate)
		if errors.Is(err, ErrNotFound) {
			return candidate, nil
		}
		if err != nil {
			return "", err
		}
	}
	return "", errors.New("unable to generate unique invoice number")
}

func (s *Service) generateUniquePaymentToken(ctx context.Context) (string, error) {
	for range 10 {
		buf := make([]byte, 16)
		if _, err := rand.Read(buf); err != nil {
			return "", err
		}
		candidate := hex.EncodeToString(buf)
		_, err := s.repo.FindByPaymentToken(ctx, candidate)
		if errors.Is(err, ErrNotFound) {
			return candidate, nil
		}
		if err != nil {
			return "", err
		}
	}
	return "", errors.New("unable to generate unique payment token")
}
