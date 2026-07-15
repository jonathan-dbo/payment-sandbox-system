package database

import (
	"database/sql"

	"github.com/gonszalito/go-ddd-architecture/internal/domain/user"
)

type PostgresUserRepository struct {
	db *sql.DB
}

func NewPostgresUserRepository(db *sql.DB) *PostgresUserRepository {
	return &PostgresUserRepository{db: db}
}

func (r *PostgresUserRepository) FindByID(id string) (*user.User, error) {
	row := r.db.QueryRow("SELECT id, name, email, password_hash, role FROM users WHERE id = $1", id)
	var u user.User
	err := row.Scan(&u.ID, &u.Name, &u.Email, &u.PasswordHash, &u.Role)
	if err != nil {
		return nil, translateSQLError(err)
	}
	return &u, nil
}

func (r *PostgresUserRepository) FindByEmail(email string) (*user.User, error) {
	row := r.db.QueryRow("SELECT id, name, email, password_hash, role FROM users WHERE email = $1", email)
	var u user.User
	err := row.Scan(&u.ID, &u.Name, &u.Email, &u.PasswordHash, &u.Role)
	if err != nil {
		return nil, translateSQLError(err)
	}
	return &u, nil
}

func (r *PostgresUserRepository) Create(u *user.User) error {
	_, err := r.db.Exec(
		"INSERT INTO users(id, name, email, password_hash, role) VALUES($1, $2, $3, $4, $5)",
		u.ID, u.Name, u.Email, u.PasswordHash, u.Role,
	)
	return translateSQLError(err)
}

func (r *PostgresUserRepository) CreateWithMerchant(u *user.User) error {
	tx, err := r.db.Begin()
	if err != nil {
		return translateSQLError(err)
	}

	if _, err := tx.Exec(
		"INSERT INTO users(id, name, email, password_hash, role) VALUES($1, $2, $3, $4, $5)",
		u.ID, u.Name, u.Email, u.PasswordHash, u.Role,
	); err != nil {
		_ = tx.Rollback()
		return translateSQLError(err)
	}

	if u.Role == user.RoleMerchant {
		if _, err := tx.Exec(
			"INSERT INTO merchants(id, user_id, name, email) VALUES($1, $2, $3, $4)",
			u.ID, u.ID, u.Name, u.Email,
		); err != nil {
			_ = tx.Rollback()
			return translateSQLError(err)
		}
	}

	if err := tx.Commit(); err != nil {
		_ = tx.Rollback()
		return translateSQLError(err)
	}
	return nil
}

func (r *PostgresUserRepository) Save(u *user.User) error {
	_, err := r.db.Exec("UPDATE users SET email = $1, password_hash = $2, role = $3 WHERE id = $4", u.Email, u.PasswordHash, u.Role, u.ID)
	return translateSQLError(err)
}
