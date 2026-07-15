// Package topup contains wallet top-up workflow application flows.
package topup

import (
	"context"
	"fmt"

	appWallet "github.com/gonszalito/go-ddd-architecture/internal/application/wallet"
	domainTopUp "github.com/gonszalito/go-ddd-architecture/internal/domain/topup"
	domainWallet "github.com/gonszalito/go-ddd-architecture/internal/domain/wallet"
	"github.com/gonszalito/go-ddd-architecture/internal/shared"
	"github.com/google/uuid"
)

type Service struct {
	repo       Repository
	walletRepo appWallet.Repository
}

func NewService(repo Repository, walletRepo appWallet.Repository) *Service {
	return &Service{repo: repo, walletRepo: walletRepo}
}

func (s *Service) Request(ctx context.Context, merchantID string, amount int64, requestKey string) (*domainTopUp.TopUp, error) {
	if requestKey != "" {
		existing, err := s.repo.FindByRequestKey(ctx, merchantID, requestKey)
		if err == nil && existing != nil {
			return existing, nil
		}
	}
	model := &domainTopUp.TopUp{
		ID:         uuid.NewString(),
		MerchantID: merchantID,
		Amount:     amount,
		Status:     domainTopUp.StatusPending,
		RequestKey: requestKey,
	}
	if err := s.repo.Create(ctx, model); err != nil {
		return nil, err
	}
	shared.LogEvent("topup_requested", map[string]any{
		"topup_id":    model.ID,
		"merchant_id": model.MerchantID,
		"amount":      model.Amount,
		"status":      model.Status,
	})
	return model, nil
}

func (s *Service) AdminUpdate(ctx context.Context, topupID string, success bool) (*domainTopUp.TopUp, error) {
	model, err := s.repo.FindByID(ctx, topupID)
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
			return nil, fmt.Errorf("atomic topup flow failed at wallet save: %w", err)
		}
		if err := s.repo.Save(ctx, model); err != nil {
			model.Status = originalStatus
			w.Balance = originalBalance
			_ = s.walletRepo.Save(ctx, w)
			if createdWallet {
				w.Balance = 0
				_ = s.walletRepo.Save(ctx, w)
			}
			return nil, fmt.Errorf("atomic topup flow failed at topup save: %w", err)
		}
	} else {
		if err := model.MarkFailed(); err != nil {
			return nil, err
		}
		if err := s.repo.Save(ctx, model); err != nil {
			return nil, err
		}
	}
	shared.LogEvent("topup_admin_updated", map[string]any{
		"topup_id":    model.ID,
		"merchant_id": model.MerchantID,
		"amount":      model.Amount,
		"status":      model.Status,
		"success":     success,
	})
	return model, nil
}

func (s *Service) History(ctx context.Context, merchantID string) ([]*domainTopUp.TopUp, error) {
	return s.repo.ListByMerchantID(ctx, merchantID)
}
