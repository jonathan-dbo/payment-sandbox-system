// Package database contains in-memory and SQL repositories.
package database

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	appPayment "github.com/gonszalito/go-ddd-architecture/internal/application/payment"
	domainPayment "github.com/gonszalito/go-ddd-architecture/internal/domain/paymentintent"
)

type PostgresPaymentRepository struct {
	db *sql.DB
}

func NewPostgresPaymentRepository(db *sql.DB) *PostgresPaymentRepository {
	return &PostgresPaymentRepository{db: db}
}

func (r *PostgresPaymentRepository) Create(ctx context.Context, intent *domainPayment.PaymentIntent) error {
	exec := executorFromContext(ctx, r.db)
	_, err := exec.ExecContext(ctx, "INSERT INTO payment_intents(id, invoice_id, method, status, due_at, created_at) VALUES($1, $2, $3, $4, $5, $6)",
		intent.ID, intent.InvoiceID, intent.Method, intent.Status, intent.DueAt, intent.CreatedAt)
	return translateSQLError(err)
}

func (r *PostgresPaymentRepository) FindByID(ctx context.Context, id string) (*domainPayment.PaymentIntent, error) {
	exec := executorFromContext(ctx, r.db)
	row := exec.QueryRowContext(ctx, "SELECT id, invoice_id, method, status, due_at, created_at FROM payment_intents WHERE id = $1", id)
	var model domainPayment.PaymentIntent
	if err := row.Scan(&model.ID, &model.InvoiceID, &model.Method, &model.Status, &model.DueAt, &model.CreatedAt); err != nil {
		err = translateSQLError(err)
		if err != nil && err.Error() == "not_found" {
			return nil, appPayment.ErrNotFound
		}
		return nil, err
	}
	return &model, nil
}

func (r *PostgresPaymentRepository) Save(ctx context.Context, intent *domainPayment.PaymentIntent) error {
	exec := executorFromContext(ctx, r.db)
	_, err := exec.ExecContext(ctx, "UPDATE payment_intents SET status = $1 WHERE id = $2", intent.Status, intent.ID)
	return translateSQLError(err)
}

func (r *PostgresPaymentRepository) List(ctx context.Context, filter appPayment.ListFilter) ([]*domainPayment.PaymentIntent, error) {
	exec := executorFromContext(ctx, r.db)
	query := "SELECT id, invoice_id, method, status, due_at, created_at FROM payment_intents"
	clauses := make([]string, 0, 3)
	args := make([]any, 0, 3)
	if filter.InvoiceID != "" {
		clauses = append(clauses, "invoice_id = $1")
		args = append(args, filter.InvoiceID)
	}
	if filter.StartDate != nil {
		clauses = append(clauses, "created_at >= $"+itoa(len(args)+1))
		args = append(args, *filter.StartDate)
	}
	if filter.EndDate != nil {
		clauses = append(clauses, "created_at <= $"+itoa(len(args)+1))
		args = append(args, *filter.EndDate)
	}
	if len(clauses) > 0 {
		query += " WHERE " + strings.Join(clauses, " AND ")
	}
	query += " ORDER BY created_at ASC"
	rows, err := exec.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, translateSQLError(err)
	}
	defer rows.Close()
	out := make([]*domainPayment.PaymentIntent, 0)
	for rows.Next() {
		var model domainPayment.PaymentIntent
		if err := rows.Scan(&model.ID, &model.InvoiceID, &model.Method, &model.Status, &model.DueAt, &model.CreatedAt); err != nil {
			return nil, translateSQLError(err)
		}
		out = append(out, &model)
	}
	return out, nil
}

func itoa(v int) string {
	return fmt.Sprintf("%d", v)
}
