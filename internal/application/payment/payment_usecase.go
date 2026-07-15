// Package payment contains payment intent application flows.
package payment

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	appInvoice "github.com/gonszalito/go-ddd-architecture/internal/application/invoice"
	"github.com/gonszalito/go-ddd-architecture/internal/domain/invoice"
	"github.com/gonszalito/go-ddd-architecture/internal/domain/paymentintent"
	"github.com/gonszalito/go-ddd-architecture/internal/shared"
	"github.com/google/uuid"
)

const (
	MethodWallet      = "WALLET"
	MethodVADummy     = "VA_DUMMY"
	MethodEWALDummy   = "EWALLET_DUMMY"
	OutcomeSuccess    = "SUCCESS"
	OutcomeFailed     = "FAILED"
	defaultDueMinutes = 15
)

var ErrInvalidMethod = errors.New("invalid payment method")

type Service struct {
	repo           Repository
	invoiceService *appInvoice.Service
	nowFn          func() time.Time
}

func NewService(repo Repository, invoiceService *appInvoice.Service) *Service {
	return &Service{repo: repo, invoiceService: invoiceService, nowFn: time.Now}
}

func (s *Service) ResolvePublicToken(ctx context.Context, token string) (*invoice.Invoice, error) {
	return s.invoiceService.ResolvePaymentToken(ctx, token)
}

func (s *Service) CreateIntent(ctx context.Context, token, method string) (*paymentintent.PaymentIntent, error) {
	if !isValidMethod(method) {
		return nil, ErrInvalidMethod
	}
	inv, err := s.invoiceService.ResolvePaymentToken(ctx, token)
	if err != nil {
		return nil, err
	}
	intent := &paymentintent.PaymentIntent{
		ID:        uuid.NewString(),
		InvoiceID: inv.ID,
		Method:    strings.ToUpper(method),
		Status:    paymentintent.StatusPending,
		DueAt:     s.nowFn().UTC().Add(time.Duration(defaultDueMinutes) * time.Minute),
		CreatedAt: s.nowFn().UTC(),
	}
	if err := s.repo.Create(ctx, intent); err != nil {
		return nil, err
	}
	shared.LogEvent("payment_intent_created", map[string]any{
		"intent_id":   intent.ID,
		"invoice_id":  intent.InvoiceID,
		"method":      intent.Method,
		"status":      intent.Status,
		"due_at":      intent.DueAt.Format(time.RFC3339),
		"created_at":  intent.CreatedAt.Format(time.RFC3339),
		"due_minutes": defaultDueMinutes,
	})
	return intent, nil
}

func (s *Service) SimulateAdminOutcome(ctx context.Context, intentID, outcome string) (*paymentintent.PaymentIntent, error) {
	intent, err := s.repo.FindByID(ctx, intentID)
	if err != nil {
		return nil, err
	}
	inv, err := s.invoiceService.GetByID(ctx, intent.InvoiceID)
	if err != nil {
		return nil, err
	}
	originalIntent := *intent
	originalInvoice := *inv

	if s.nowFn().UTC().After(intent.DueAt) {
		_ = intent.MarkFailed()
		_ = inv.MarkExpired()
		if err := s.repo.Save(ctx, intent); err != nil {
			return nil, fmt.Errorf("atomic expiry flow failed at intent save: %w", err)
		}
		if err := s.invoiceService.Save(ctx, inv); err != nil {
			if rbErr := s.repo.Save(ctx, &originalIntent); rbErr != nil {
				return nil, fmt.Errorf("atomic expiry flow failed (rollback intent failed: %v): %w", rbErr, err)
			}
			return nil, fmt.Errorf("atomic expiry flow failed at invoice save: %w", err)
		}
		return intent, nil
	}

	switch strings.ToUpper(outcome) {
	case OutcomeSuccess:
		if err := intent.MarkSuccess(); err != nil {
			return nil, err
		}
		if err := inv.MarkPaid(); err != nil {
			return nil, err
		}
	case OutcomeFailed:
		if err := intent.MarkFailed(); err != nil {
			return nil, err
		}
	default:
		return nil, errors.New("invalid simulation outcome")
	}

	if err := s.repo.Save(ctx, intent); err != nil {
		return nil, fmt.Errorf("atomic payment flow failed at intent save: %w", err)
	}
	if err := s.invoiceService.Save(ctx, inv); err != nil {
		if rbErr := s.repo.Save(ctx, &originalIntent); rbErr != nil {
			return nil, fmt.Errorf("atomic payment flow failed (rollback intent failed: %v): %w", rbErr, err)
		}
		if rbErr := s.invoiceService.Save(ctx, &originalInvoice); rbErr != nil {
			return nil, fmt.Errorf("atomic payment flow failed (rollback invoice failed: %v): %w", rbErr, err)
		}
		return nil, fmt.Errorf("atomic payment flow failed at invoice save: %w", err)
	}
	shared.LogEvent("payment_simulated", map[string]any{
		"intent_id":    intent.ID,
		"invoice_id":   intent.InvoiceID,
		"method":       intent.Method,
		"final_status": intent.Status,
		"outcome":      outcome,
	})
	return intent, nil
}

func isValidMethod(method string) bool {
	switch strings.ToUpper(method) {
	case MethodWallet, MethodVADummy, MethodEWALDummy:
		return true
	default:
		return false
	}
}
