// Package database contains in-memory and SQL repositories.
package database

import (
	"context"
	"database/sql"

	appRefund "github.com/gonszalito/go-ddd-architecture/internal/application/refund"
	domainRefund "github.com/gonszalito/go-ddd-architecture/internal/domain/refund"
)

type PostgresRefundRepository struct {
	db *sql.DB
}

func NewPostgresRefundRepository(db *sql.DB) *PostgresRefundRepository {
	return &PostgresRefundRepository{db: db}
}

func (r *PostgresRefundRepository) Create(ctx context.Context, model *domainRefund.Refund) error {
	exec := executorFromContext(ctx, r.db)
	_, err := exec.ExecContext(ctx, "INSERT INTO refunds(id, invoice_id, merchant_id, amount, status, created_at) VALUES($1, $2, $3, $4, $5, $6)",
		model.ID, model.InvoiceID, model.MerchantID, model.Amount, model.Status, model.CreatedAt)
	return translateSQLError(err)
}

func (r *PostgresRefundRepository) FindByID(ctx context.Context, id string) (*domainRefund.Refund, error) {
	exec := executorFromContext(ctx, r.db)
	row := exec.QueryRowContext(ctx, "SELECT id, invoice_id, merchant_id, amount, status, created_at FROM refunds WHERE id = $1", id)
	var model domainRefund.Refund
	if err := row.Scan(&model.ID, &model.InvoiceID, &model.MerchantID, &model.Amount, &model.Status, &model.CreatedAt); err != nil {
		err = translateSQLError(err)
		if err != nil && err.Error() == "not_found" {
			return nil, appRefund.ErrNotFound
		}
		return nil, err
	}
	return &model, nil
}

func (r *PostgresRefundRepository) Save(ctx context.Context, model *domainRefund.Refund) error {
	exec := executorFromContext(ctx, r.db)
	_, err := exec.ExecContext(ctx, "UPDATE refunds SET status = $1 WHERE id = $2", model.Status, model.ID)
	return translateSQLError(err)
}

func (r *PostgresRefundRepository) List(ctx context.Context, filter appRefund.ListFilter) ([]*domainRefund.Refund, error) {
	exec := executorFromContext(ctx, r.db)
	rows, err := exec.QueryContext(ctx, "SELECT id, invoice_id, merchant_id, amount, status, created_at FROM refunds WHERE merchant_id = $1 ORDER BY id ASC", filter.MerchantID)
	if err != nil {
		return nil, translateSQLError(err)
	}
	defer rows.Close()
	out := []*domainRefund.Refund{}
	for rows.Next() {
		var model domainRefund.Refund
		if err := rows.Scan(&model.ID, &model.InvoiceID, &model.MerchantID, &model.Amount, &model.Status, &model.CreatedAt); err != nil {
			return nil, translateSQLError(err)
		}
		if filter.StartDate != nil && model.CreatedAt.Before(*filter.StartDate) {
			continue
		}
		if filter.EndDate != nil && model.CreatedAt.After(*filter.EndDate) {
			continue
		}
		out = append(out, &model)
	}
	return out, nil
}

func (r *PostgresRefundRepository) ListByInvoiceID(ctx context.Context, invoiceID string) ([]*domainRefund.Refund, error) {
	exec := executorFromContext(ctx, r.db)
	rows, err := exec.QueryContext(ctx, "SELECT id, invoice_id, merchant_id, amount, status, created_at FROM refunds WHERE invoice_id = $1 ORDER BY id ASC", invoiceID)
	if err != nil {
		return nil, translateSQLError(err)
	}
	defer rows.Close()
	out := []*domainRefund.Refund{}
	for rows.Next() {
		var model domainRefund.Refund
		if err := rows.Scan(&model.ID, &model.InvoiceID, &model.MerchantID, &model.Amount, &model.Status, &model.CreatedAt); err != nil {
			return nil, translateSQLError(err)
		}
		out = append(out, &model)
	}
	return out, nil
}
