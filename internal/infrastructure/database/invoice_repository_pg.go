// Package database contains in-memory and SQL repositories.
package database

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	appInvoice "github.com/gonszalito/go-ddd-architecture/internal/application/invoice"
	domainInvoice "github.com/gonszalito/go-ddd-architecture/internal/domain/invoice"
)

type PostgresInvoiceRepository struct {
	db *sql.DB
}

func NewPostgresInvoiceRepository(db *sql.DB) *PostgresInvoiceRepository {
	return &PostgresInvoiceRepository{db: db}
}

func (r *PostgresInvoiceRepository) Create(ctx context.Context, inv *domainInvoice.Invoice) error {
	exec := executorFromContext(ctx, r.db)
	_, err := exec.ExecContext(ctx, "INSERT INTO invoices(id, invoice_number, merchant_id, amount, currency, status, payment_token, due_date, created_at) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9)",
		inv.ID, inv.InvoiceNumber, inv.MerchantID, inv.Amount, inv.Currency, inv.Status, inv.PaymentToken, nullableTime(inv.DueDate), inv.CreatedAt)
	return translateSQLError(err)
}

func (r *PostgresInvoiceRepository) FindByID(ctx context.Context, id string) (*domainInvoice.Invoice, error) {
	exec := executorFromContext(ctx, r.db)
	row := exec.QueryRowContext(ctx, "SELECT id, invoice_number, merchant_id, amount, currency, status, payment_token, due_date, created_at FROM invoices WHERE id = $1", id)
	var inv domainInvoice.Invoice
	var dueDate sql.NullTime
	err := row.Scan(&inv.ID, &inv.InvoiceNumber, &inv.MerchantID, &inv.Amount, &inv.Currency, &inv.Status, &inv.PaymentToken, &dueDate, &inv.CreatedAt)
	if err != nil {
		return nil, mapInvoiceErr(translateSQLError(err))
	}
	if dueDate.Valid {
		inv.DueDate = dueDate.Time
	}
	return &inv, nil
}

func (r *PostgresInvoiceRepository) FindByInvoiceNumber(ctx context.Context, invoiceNumber string) (*domainInvoice.Invoice, error) {
	exec := executorFromContext(ctx, r.db)
	row := exec.QueryRowContext(ctx, "SELECT id, invoice_number, merchant_id, amount, currency, status, payment_token, due_date, created_at FROM invoices WHERE invoice_number = $1", invoiceNumber)
	var inv domainInvoice.Invoice
	var dueDate sql.NullTime
	err := row.Scan(&inv.ID, &inv.InvoiceNumber, &inv.MerchantID, &inv.Amount, &inv.Currency, &inv.Status, &inv.PaymentToken, &dueDate, &inv.CreatedAt)
	if err != nil {
		return nil, mapInvoiceErr(translateSQLError(err))
	}
	if dueDate.Valid {
		inv.DueDate = dueDate.Time
	}
	return &inv, nil
}

func (r *PostgresInvoiceRepository) FindByPaymentToken(ctx context.Context, paymentToken string) (*domainInvoice.Invoice, error) {
	exec := executorFromContext(ctx, r.db)
	row := exec.QueryRowContext(ctx, "SELECT id, invoice_number, merchant_id, amount, currency, status, payment_token, due_date, created_at FROM invoices WHERE payment_token = $1", paymentToken)
	var inv domainInvoice.Invoice
	var dueDate sql.NullTime
	err := row.Scan(&inv.ID, &inv.InvoiceNumber, &inv.MerchantID, &inv.Amount, &inv.Currency, &inv.Status, &inv.PaymentToken, &dueDate, &inv.CreatedAt)
	if err != nil {
		return nil, mapInvoiceErr(translateSQLError(err))
	}
	if dueDate.Valid {
		inv.DueDate = dueDate.Time
	}
	return &inv, nil
}

func (r *PostgresInvoiceRepository) List(ctx context.Context, filter appInvoice.ListFilter) ([]*domainInvoice.Invoice, error) {
	exec := executorFromContext(ctx, r.db)
	where := make([]string, 0, 2)
	args := make([]any, 0, 4)
	argPos := 1

	if strings.TrimSpace(filter.MerchantID) != "" {
		where = append(where, fmt.Sprintf("merchant_id = $%d", argPos))
		args = append(args, filter.MerchantID)
		argPos++
	}
	if strings.TrimSpace(filter.Status) != "" {
		where = append(where, fmt.Sprintf("status = $%d", argPos))
		args = append(args, strings.ToUpper(strings.TrimSpace(filter.Status)))
		argPos++
	}

	query := "SELECT id, invoice_number, merchant_id, amount, currency, status, payment_token, due_date, created_at FROM invoices"
	if len(where) > 0 {
		query += " WHERE " + strings.Join(where, " AND ")
	}
	query += " ORDER BY created_at DESC"

	limit := filter.PageSize
	if limit <= 0 {
		limit = 10
	}
	offset := (filter.Page - 1) * limit
	if offset < 0 {
		offset = 0
	}
	query += fmt.Sprintf(" LIMIT $%d OFFSET $%d", argPos, argPos+1)
	args = append(args, limit, offset)

	rows, err := exec.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, translateSQLError(err)
	}
	defer rows.Close()

	out := []*domainInvoice.Invoice{}
	for rows.Next() {
		var inv domainInvoice.Invoice
		var dueDate sql.NullTime
		if err := rows.Scan(&inv.ID, &inv.InvoiceNumber, &inv.MerchantID, &inv.Amount, &inv.Currency, &inv.Status, &inv.PaymentToken, &dueDate, &inv.CreatedAt); err != nil {
			return nil, translateSQLError(err)
		}
		if dueDate.Valid {
			inv.DueDate = dueDate.Time
		}
		out = append(out, &inv)
	}
	return out, nil
}

func (r *PostgresInvoiceRepository) Save(ctx context.Context, inv *domainInvoice.Invoice) error {
	exec := executorFromContext(ctx, r.db)
	_, err := exec.ExecContext(ctx, "UPDATE invoices SET amount = $1, currency = $2, status = $3, due_date = $4 WHERE id = $5", inv.Amount, inv.Currency, inv.Status, nullableTime(inv.DueDate), inv.ID)
	return translateSQLError(err)
}

func (r *PostgresInvoiceRepository) Delete(ctx context.Context, id string) error {
	exec := executorFromContext(ctx, r.db)
	_, err := exec.ExecContext(ctx, "DELETE FROM invoices WHERE id = $1", id)
	return translateSQLError(err)
}

func nullableTime(v time.Time) interface{} {
	if v.IsZero() {
		return nil
	}
	return v
}

func mapInvoiceErr(err error) error {
	if err != nil && err.Error() == "not_found" {
		return appInvoice.ErrNotFound
	}
	return err
}
