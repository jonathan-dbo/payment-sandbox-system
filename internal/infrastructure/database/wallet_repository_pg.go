// Package database contains in-memory and SQL repositories.
package database

import (
	"context"
	"database/sql"

	appWallet "github.com/gonszalito/go-ddd-architecture/internal/application/wallet"
	domainWallet "github.com/gonszalito/go-ddd-architecture/internal/domain/wallet"
)

type PostgresWalletRepository struct {
	db *sql.DB
}

func NewPostgresWalletRepository(db *sql.DB) *PostgresWalletRepository {
	return &PostgresWalletRepository{db: db}
}

func (r *PostgresWalletRepository) Create(ctx context.Context, model *domainWallet.Wallet) error {
	exec := executorFromContext(ctx, r.db)
	_, err := exec.ExecContext(ctx, "INSERT INTO wallets(id, merchant_id, balance) VALUES($1, $2, $3)", model.ID, model.MerchantID, model.Balance)
	return translateSQLError(err)
}

func (r *PostgresWalletRepository) FindByMerchantID(ctx context.Context, merchantID string) (*domainWallet.Wallet, error) {
	exec := executorFromContext(ctx, r.db)
	row := exec.QueryRowContext(ctx, "SELECT id, merchant_id, balance FROM wallets WHERE merchant_id = $1", merchantID)
	var model domainWallet.Wallet
	if err := row.Scan(&model.ID, &model.MerchantID, &model.Balance); err != nil {
		err = translateSQLError(err)
		if err != nil && err.Error() == "not_found" {
			return nil, appWallet.ErrNotFound
		}
		return nil, err
	}
	return &model, nil
}

func (r *PostgresWalletRepository) Save(ctx context.Context, model *domainWallet.Wallet) error {
	exec := executorFromContext(ctx, r.db)
	_, err := exec.ExecContext(ctx, "UPDATE wallets SET balance = $1 WHERE merchant_id = $2", model.Balance, model.MerchantID)
	return translateSQLError(err)
}
