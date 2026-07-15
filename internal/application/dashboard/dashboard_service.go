// Package dashboard contains admin statistics aggregation use cases.
package dashboard

import (
	"context"
	"time"

	appInvoice "github.com/gonszalito/go-ddd-architecture/internal/application/invoice"
	appPayment "github.com/gonszalito/go-ddd-architecture/internal/application/payment"
	appRefund "github.com/gonszalito/go-ddd-architecture/internal/application/refund"
	"github.com/gonszalito/go-ddd-architecture/internal/domain/invoice"
	"github.com/gonszalito/go-ddd-architecture/internal/domain/paymentintent"
	"github.com/gonszalito/go-ddd-architecture/internal/domain/refund"
)

type Filter struct {
	MerchantID string
	StartDate  *time.Time
	EndDate    *time.Time
}

type Stats struct {
	TotalInvoices          int   `json:"totalInvoices"`
	PaidCount              int   `json:"paidCount"`
	FailedCount            int   `json:"failedCount"`
	ExpiredCount           int   `json:"expiredCount"`
	TotalTransactionAmount int64 `json:"totalTransactionAmount"`
	TotalRefundAmount      int64 `json:"totalRefundAmount"`
}

type Service struct {
	invoiceRepo appInvoice.InvoiceRepository
	paymentRepo appPayment.Repository
	refundRepo  appRefund.Repository
}

func NewService(invoiceRepo appInvoice.InvoiceRepository, paymentRepo appPayment.Repository, refundRepo appRefund.Repository) *Service {
	return &Service{invoiceRepo: invoiceRepo, paymentRepo: paymentRepo, refundRepo: refundRepo}
}

func (s *Service) GetStats(ctx context.Context, filter Filter) (*Stats, error) {
	invoices, err := s.invoiceRepo.List(ctx, appInvoice.ListFilter{
		MerchantID: filter.MerchantID,
		StartDate:  filter.StartDate,
		EndDate:    filter.EndDate,
		Page:       1,
		PageSize:   100000,
	})
	if err != nil {
		return nil, err
	}
	paymentIntents, err := s.paymentRepo.List(ctx, appPayment.ListFilter{
		StartDate: filter.StartDate,
		EndDate:   filter.EndDate,
	})
	if err != nil {
		return nil, err
	}
	refunds, err := s.refundRepo.List(ctx, appRefund.ListFilter{
		MerchantID: filter.MerchantID,
		StartDate:  filter.StartDate,
		EndDate:    filter.EndDate,
	})
	if err != nil {
		return nil, err
	}

	out := &Stats{TotalInvoices: len(invoices)}
	for _, inv := range invoices {
		switch inv.Status {
		case invoice.StatusPaid:
			out.PaidCount++
			out.TotalTransactionAmount += inv.Amount
		case invoice.StatusExpired:
			out.ExpiredCount++
		}
	}
	for _, intent := range paymentIntents {
		if intent.Status == paymentintent.StatusFailed {
			out.FailedCount++
		}
	}
	for _, rf := range refunds {
		if rf.Status == refund.StatusSuccess {
			out.TotalRefundAmount += rf.Amount
		}
	}
	return out, nil
}
