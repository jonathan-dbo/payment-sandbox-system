// Package database contains in-memory and SQL repositories.
package database

import (
	"context"
	"database/sql"

	appTopUp "github.com/gonszalito/go-ddd-architecture/internal/application/topup"
	domainTopUp "github.com/gonszalito/go-ddd-architecture/internal/domain/topup"
)

type PostgresTopUpRepository struct {
	db *sql.DB
}

func NewPostgresTopUpRepository(db *sql.DB) *PostgresTopUpRepository {
	return &PostgresTopUpRepository{db: db}
}

func (r *PostgresTopUpRepository) Create(ctx context.Context, model *domainTopUp.TopUp) error {
	exec := executorFromContext(ctx, r.db)
	_, err := exec.ExecContext(ctx, "INSERT INTO top_ups(id, merchant_id, amount, status, request_key) VALUES($1, $2, $3, $4, $5)",
		model.ID, model.MerchantID, model.Amount, model.Status, model.RequestKey)
	return translateSQLError(err)
}

func (r *PostgresTopUpRepository) FindByID(ctx context.Context, id string) (*domainTopUp.TopUp, error) {
	exec := executorFromContext(ctx, r.db)
	row := exec.QueryRowContext(ctx, "SELECT id, merchant_id, amount, status, request_key FROM top_ups WHERE id = $1", id)
	var model domainTopUp.TopUp
	if err := row.Scan(&model.ID, &model.MerchantID, &model.Amount, &model.Status, &model.RequestKey); err != nil {
		err = translateSQLError(err)
		if err != nil && err.Error() == "not_found" {
			return nil, appTopUp.ErrNotFound
		}
		return nil, err
	}
	return &model, nil
}

func (r *PostgresTopUpRepository) FindByRequestKey(ctx context.Context, merchantID, requestKey string) (*domainTopUp.TopUp, error) {
	exec := executorFromContext(ctx, r.db)
	row := exec.QueryRowContext(ctx, "SELECT id, merchant_id, amount, status, request_key FROM top_ups WHERE merchant_id = $1 AND request_key = $2", merchantID, requestKey)
	var model domainTopUp.TopUp
	if err := row.Scan(&model.ID, &model.MerchantID, &model.Amount, &model.Status, &model.RequestKey); err != nil {
		err = translateSQLError(err)
		if err != nil && err.Error() == "not_found" {
			return nil, appTopUp.ErrNotFound
		}
		return nil, err
	}
	return &model, nil
}

func (r *PostgresTopUpRepository) ListByMerchantID(ctx context.Context, merchantID string) ([]*domainTopUp.TopUp, error) {
	exec := executorFromContext(ctx, r.db)
	rows, err := exec.QueryContext(ctx, "SELECT id, merchant_id, amount, status, request_key FROM top_ups WHERE merchant_id = $1 ORDER BY id ASC", merchantID)
	if err != nil {
		return nil, translateSQLError(err)
	}
	defer rows.Close()
	out := []*domainTopUp.TopUp{}
	for rows.Next() {
		var model domainTopUp.TopUp
		if err := rows.Scan(&model.ID, &model.MerchantID, &model.Amount, &model.Status, &model.RequestKey); err != nil {
			return nil, translateSQLError(err)
		}
		out = append(out, &model)
	}
	return out, nil
}

func (r *PostgresTopUpRepository) Save(ctx context.Context, model *domainTopUp.TopUp) error {
	exec := executorFromContext(ctx, r.db)
	_, err := exec.ExecContext(ctx, "UPDATE top_ups SET status = $1 WHERE id = $2", model.Status, model.ID)
	return translateSQLError(err)
}
