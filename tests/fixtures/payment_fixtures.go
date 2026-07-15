package fixtures

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// PaymentFixture provides factory methods for creating test Payment objects.
// Using fixtures ensures consistent test data and reduces boilerplate in tests.
type PaymentFixture struct{}

// Payment represents a payment test fixture with all fields.
type Payment struct {
	ID             string
	MerchantID     string
	Amount         int64
	Currency       string
	Status         string
	RefundedAmount int64
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

// NewPaymentFixture returns a new payment fixture builder.
func NewPaymentFixture() *PaymentFixture {
	return &PaymentFixture{}
}

// DefaultPayment returns a payment fixture with sensible defaults.
func (f *PaymentFixture) DefaultPayment() Payment {
	now := time.Now().UTC()
	return Payment{
		ID:             uuid.New().String(),
		MerchantID:     "merchant_" + uuid.New().String()[:8],
		Amount:         5000,
		Currency:       "USD",
		Status:         "PENDING",
		RefundedAmount: 0,
		CreatedAt:      now,
		UpdatedAt:      now,
	}
}

// WithMerchantID sets the merchant ID and returns the modified fixture.
func (p Payment) WithMerchantID(merchantID string) Payment {
	p.MerchantID = merchantID
	return p
}

// WithAmount sets the amount and returns the modified fixture.
func (p Payment) WithAmount(amount int64) Payment {
	p.Amount = amount
	return p
}

// WithCurrency sets the currency and returns the modified fixture.
func (p Payment) WithCurrency(currency string) Payment {
	p.Currency = currency
	return p
}

// WithStatus sets the status and returns the modified fixture.
func (p Payment) WithStatus(status string) Payment {
	p.Status = status
	return p
}

// WithRefundedAmount sets the refunded amount and returns the modified fixture.
func (p Payment) WithRefundedAmount(amount int64) Payment {
	p.RefundedAmount = amount
	return p
}

// MerchantFixture provides factory methods for creating test Merchant objects.
type MerchantFixture struct{}

// Merchant represents a merchant test fixture with all fields.
type Merchant struct {
	ID        string
	Name      string
	Email     string
	Status    string
	CreatedAt time.Time
	UpdatedAt time.Time
}

// NewMerchantFixture returns a new merchant fixture builder.
func NewMerchantFixture() *MerchantFixture {
	return &MerchantFixture{}
}

// DefaultMerchant returns a merchant fixture with sensible defaults.
func (f *MerchantFixture) DefaultMerchant() Merchant {
	now := time.Now().UTC()
	return Merchant{
		ID:        uuid.New().String(),
		Name:      "Test Merchant",
		Email:     "test+" + uuid.New().String()[:8] + "@example.com",
		Status:    "ACTIVE",
		CreatedAt: now,
		UpdatedAt: now,
	}
}

// WithName sets the name and returns the modified fixture.
func (m Merchant) WithName(name string) Merchant {
	m.Name = name
	return m
}

// WithEmail sets the email and returns the modified fixture.
func (m Merchant) WithEmail(email string) Merchant {
	m.Email = email
	return m
}

// WithStatus sets the status and returns the modified fixture.
func (m Merchant) WithStatus(status string) Merchant {
	m.Status = status
	return m
}

// TransactionFixture provides factory methods for creating test Transaction objects.
type TransactionFixture struct{}

// Transaction represents a transaction test fixture.
type Transaction struct {
	ID        string
	PaymentID string
	Type      string
	Status    string
	Amount    int64
	Timestamp time.Time
}

// NewTransactionFixture returns a new transaction fixture builder.
func NewTransactionFixture() *TransactionFixture {
	return &TransactionFixture{}
}

// DefaultTransaction returns a transaction fixture with sensible defaults.
func (f *TransactionFixture) DefaultTransaction() Transaction {
	return Transaction{
		ID:        uuid.New().String(),
		PaymentID: uuid.New().String(),
		Type:      "AUTHORIZATION",
		Status:    "SUCCESS",
		Amount:    5000,
		Timestamp: time.Now().UTC(),
	}
}

// WithPaymentID sets the payment ID and returns the modified fixture.
func (t Transaction) WithPaymentID(paymentID string) Transaction {
	t.PaymentID = paymentID
	return t
}

// WithType sets the transaction type and returns the modified fixture.
func (t Transaction) WithType(txnType string) Transaction {
	t.Type = txnType
	return t
}

// WithAmount sets the amount and returns the modified fixture.
func (t Transaction) WithAmount(amount int64) Transaction {
	t.Amount = amount
	return t
}

// CommonTestHelper provides common test utilities for integration tests.
type CommonTestHelper struct{}

// NewCommonTestHelper returns a new test helper.
func NewCommonTestHelper() *CommonTestHelper {
	return &CommonTestHelper{}
}

// CreateContextWithTimeout creates a context with the given timeout duration.
func (h *CommonTestHelper) CreateContextWithTimeout(timeout time.Duration) (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), timeout)
}

// DefaultTimeout returns the default timeout for tests (5 seconds).
func (h *CommonTestHelper) DefaultTimeout() time.Duration {
	return 5 * time.Second
}
