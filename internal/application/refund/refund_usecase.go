// Package refund contains refund workflow application flows.
package refund

import (
	"context"
	"errors"
	"fmt"
	"time"

	appInvoice "github.com/gonszalito/go-ddd-architecture/internal/application/invoice"
	appWallet "github.com/gonszalito/go-ddd-architecture/internal/application/wallet"
	domainInvoice "github.com/gonszalito/go-ddd-architecture/internal/domain/invoice"
	domainRefund "github.com/gonszalito/go-ddd-architecture/internal/domain/refund"
	domainWallet "github.com/gonszalito/go-ddd-architecture/internal/domain/wallet"
	"github.com/gonszalito/go-ddd-architecture/internal/shared"
	"github.com/google/uuid"
)

type Service struct {
	repo       Repository
	walletRepo appWallet.Repository
	invoiceSvc *appInvoice.Service
}

func NewService(repo Repository, walletRepo appWallet.Repository, invoiceSvc *appInvoice.Service) *Service {
	return &Service{repo: repo, walletRepo: walletRepo, invoiceSvc: invoiceSvc}
}

func (s *Service) RequestRefund(ctx context.Context, invoiceID, merchantID string, amount int64) (*domainRefund.Refund, error) {
	if s.invoiceSvc == nil {
		return nil, errors.New("invoice service is required")
	}
	inv, err := s.invoiceSvc.GetByID(ctx, invoiceID)
	if err != nil {
		return nil, err
	}
	if inv.MerchantID != merchantID {
		return nil, errors.New("merchant does not own this invoice")
	}
	if inv.Status != domainInvoice.StatusPaid {
		return nil, errors.New("refund allowed only for paid invoice")
	}
	existing, err := s.repo.ListByInvoiceID(ctx, invoiceID)
	if err != nil {
		return nil, err
	}
	var alreadyRefunded int64
	for _, rf := range existing {
		if rf.Status == domainRefund.StatusSuccess {
			alreadyRefunded += rf.Amount
		}
	}
	maxRefundable := inv.Amount - alreadyRefunded
	if maxRefundable <= 0 {
		return nil, errors.New("invoice is fully refunded")
	}
	if amount > maxRefundable {
		return nil, fmt.Errorf("refund amount exceeds max refundable amount: %d", maxRefundable)
	}

	model := &domainRefund.Refund{
		ID:         uuid.NewString(),
		InvoiceID:  invoiceID,
		MerchantID: merchantID,
		Amount:     amount,
		Status:     domainRefund.StatusRequested,
		History:    []string{"REQUESTED"},
		CreatedAt:  time.Now().UTC(),
	}
	if err := s.repo.Create(ctx, model); err != nil {
		return nil, err
	}
	shared.LogEvent("refund_requested", map[string]any{
		"refund_id":   model.ID,
		"invoice_id":  model.InvoiceID,
		"merchant_id": model.MerchantID,
		"amount":      model.Amount,
		"status":      model.Status,
	})
	return model, nil
}


func (s *Service) Approve(ctx context.Context, refundID string) (*domainRefund.Refund, error) {
	model, err := s.repo.FindByID(ctx, refundID)
	if err != nil {
		return nil, err
	}
	if err := model.Approve(); err != nil {
		return nil, err
	}
	if err := s.repo.Save(ctx, model); err != nil {
		return nil, err
	}
	shared.LogEvent("refund_approved", map[string]any{
		"refund_id":   model.ID,
		"merchant_id": model.MerchantID,
		"status":      model.Status,
	})
	return model, nil
}

func (s *Service) Reject(ctx context.Context, refundID string) (*domainRefund.Refund, error) {
	model, err := s.repo.FindByID(ctx, refundID)
	if err != nil {
		return nil, err
	}
	if err := model.Reject(); err != nil {
		return nil, err
	}
	if err := s.repo.Save(ctx, model); err != nil {
		return nil, err
	}
	shared.LogEvent("refund_rejected", map[string]any{
		"refund_id":   model.ID,
		"merchant_id": model.MerchantID,
		"status":      model.Status,
	})
	return model, nil
}

func (s *Service) Process(ctx context.Context, refundID string, success bool) (*domainRefund.Refund, error) {
	model, err := s.repo.FindByID(ctx, refundID)
	if err != nil {
		return nil, err
	}
	if success {
		originalStatus := model.Status
		w, err := s.walletRepo.FindByMerchantID(ctx, model.MerchantID)
		createdWallet := false
		if err == appWallet.ErrNotFound {
			w = &domainWallet.Wallet{ID: uuid.NewString(), MerchantID: model.MerchantID, Balance: 0}
			if err := s.walletRepo.Create(ctx, w); err != nil {
				return nil, err
			}
			createdWallet = true
		} else if err != nil {
			return nil, err
		}
		originalBalance := w.Balance
		if err := model.MarkSuccess(); err != nil {
			return nil, err
		}
		w.Balance += model.Amount
		if err := s.walletRepo.Save(ctx, w); err != nil {
			return nil, fmt.Errorf("atomic refund flow failed at wallet save: %w", err)
		}
		if err := s.repo.Save(ctx, model); err != nil {
			model.Status = originalStatus
			w.Balance = originalBalance
			_ = s.walletRepo.Save(ctx, w)
			if createdWallet {
				w.Balance = 0
				_ = s.walletRepo.Save(ctx, w)
			}
			return nil, fmt.Errorf("atomic refund flow failed at refund save: %w", err)
		}
	} else {
		if err := model.MarkFailed(); err != nil {
			return nil, err
		}
		if err := s.repo.Save(ctx, model); err != nil {
			return nil, err
		}
	}
	shared.LogEvent("refund_processed", map[string]any{
		"refund_id":   model.ID,
		"merchant_id": model.MerchantID,
		"status":      model.Status,
		"success":     success,
	})
	return model, nil
}

func (s *Service) History(ctx context.Context, merchantID string) ([]*domainRefund.Refund, error) {
	return s.repo.List(ctx, ListFilter{MerchantID: merchantID})
}
