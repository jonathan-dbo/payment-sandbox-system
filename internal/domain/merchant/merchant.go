package merchant

import (
	"fmt"
	"time"

	"github.com/google/uuid"
)

// Merchant is the aggregate root for merchant accounts.
type Merchant struct {
	id         string
	name       string
	email      string
	status     MerchantStatus
	apiKeyHash string
	createdAt  time.Time
	updatedAt  time.Time
}

// NewMerchant creates a new merchant in the PENDING state.
func NewMerchant(name, email, apiKeyHash string) (*Merchant, error) {
	if name == "" {
		return nil, fmt.Errorf("invalid merchant: name must not be empty")
	}
	if email == "" {
		return nil, fmt.Errorf("invalid merchant: email must not be empty")
	}
	if apiKeyHash == "" {
		return nil, fmt.Errorf("invalid merchant: api_key_hash must not be empty")
	}

	now := time.Now().UTC()
	return &Merchant{
		id:         uuid.New().String(),
		name:       name,
		email:      email,
		status:     StatusPending,
		apiKeyHash: apiKeyHash,
		createdAt:  now,
		updatedAt:  now,
	}, nil
}

// RestoreMerchant restores a merchant from persistence.
func RestoreMerchant(id, name, email string, status MerchantStatus, apiKeyHash string, createdAt, updatedAt time.Time) (*Merchant, error) {
	if !status.IsValid() {
		return nil, fmt.Errorf("invalid merchant: unknown status %q", status)
	}

	return &Merchant{
		id:         id,
		name:       name,
		email:      email,
		status:     status,
		apiKeyHash: apiKeyHash,
		createdAt:  createdAt,
		updatedAt:  updatedAt,
	}, nil
}

// ID returns the merchant identifier.
func (m *Merchant) ID() string { return m.id }

// Name returns the merchant display name.
func (m *Merchant) Name() string { return m.name }

// Email returns the merchant email address.
func (m *Merchant) Email() string { return m.email }

// Status returns the current merchant status.
func (m *Merchant) Status() MerchantStatus { return m.status }

// CreatedAt returns the creation timestamp.
func (m *Merchant) CreatedAt() time.Time { return m.createdAt }

// UpdatedAt returns the last update timestamp.
func (m *Merchant) UpdatedAt() time.Time { return m.updatedAt }

// GetAPIKeyHash returns the stored API key hash.
func (m *Merchant) GetAPIKeyHash() string { return m.apiKeyHash }

// Activate transitions the merchant to ACTIVE.
func (m *Merchant) Activate() error {
	if !m.status.CanTransitionTo(StatusActive) {
		return fmt.Errorf("invalid state transition: cannot activate merchant in %q state", m.status)
	}

	m.status = StatusActive
	m.updatedAt = time.Now().UTC()
	return nil
}

// Suspend transitions the merchant to SUSPENDED.
func (m *Merchant) Suspend() error {
	if !m.status.CanTransitionTo(StatusSuspended) {
		return fmt.Errorf("invalid state transition: cannot suspend merchant in %q state", m.status)
	}

	m.status = StatusSuspended
	m.updatedAt = time.Now().UTC()
	return nil
}
