package fixtures

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestPaymentFixture_Default tests that the default payment fixture is properly initialized.
func TestPaymentFixture_Default(t *testing.T) {
	fixture := NewPaymentFixture()
	payment := fixture.DefaultPayment()

	assert.NotEmpty(t, payment.ID)
	assert.NotEmpty(t, payment.MerchantID)
	assert.Equal(t, int64(5000), payment.Amount)
	assert.Equal(t, "USD", payment.Currency)
	assert.Equal(t, "PENDING", payment.Status)
	assert.Equal(t, int64(0), payment.RefundedAmount)
	assert.NotNil(t, payment.CreatedAt)
	assert.NotNil(t, payment.UpdatedAt)
}

// TestPaymentFixture_Builder tests the builder pattern for payment fixtures.
func TestPaymentFixture_Builder(t *testing.T) {
	fixture := NewPaymentFixture()
	payment := fixture.DefaultPayment().
		WithMerchantID("merchant_custom").
		WithAmount(10000).
		WithCurrency("EUR").
		WithStatus("CAPTURED").
		WithRefundedAmount(2000)

	assert.Equal(t, "merchant_custom", payment.MerchantID)
	assert.Equal(t, int64(10000), payment.Amount)
	assert.Equal(t, "EUR", payment.Currency)
	assert.Equal(t, "CAPTURED", payment.Status)
	assert.Equal(t, int64(2000), payment.RefundedAmount)
}

// TestMerchantFixture_Default tests that the default merchant fixture is properly initialized.
func TestMerchantFixture_Default(t *testing.T) {
	fixture := NewMerchantFixture()
	merchant := fixture.DefaultMerchant()

	assert.NotEmpty(t, merchant.ID)
	assert.Equal(t, "Test Merchant", merchant.Name)
	assert.NotEmpty(t, merchant.Email)
	assert.Equal(t, "ACTIVE", merchant.Status)
	assert.NotNil(t, merchant.CreatedAt)
	assert.NotNil(t, merchant.UpdatedAt)
}

// TestMerchantFixture_Builder tests the builder pattern for merchant fixtures.
func TestMerchantFixture_Builder(t *testing.T) {
	fixture := NewMerchantFixture()
	merchant := fixture.DefaultMerchant().
		WithName("Custom Merchant").
		WithEmail("custom@example.com").
		WithStatus("SUSPENDED")

	assert.Equal(t, "Custom Merchant", merchant.Name)
	assert.Equal(t, "custom@example.com", merchant.Email)
	assert.Equal(t, "SUSPENDED", merchant.Status)
}

// TestTransactionFixture_Default tests that the default transaction fixture is properly initialized.
func TestTransactionFixture_Default(t *testing.T) {
	fixture := NewTransactionFixture()
	transaction := fixture.DefaultTransaction()

	assert.NotEmpty(t, transaction.ID)
	assert.NotEmpty(t, transaction.PaymentID)
	assert.Equal(t, "AUTHORIZATION", transaction.Type)
	assert.Equal(t, "SUCCESS", transaction.Status)
	assert.Equal(t, int64(5000), transaction.Amount)
	assert.NotNil(t, transaction.Timestamp)
}

// TestTransactionFixture_Builder tests the builder pattern for transaction fixtures.
func TestTransactionFixture_Builder(t *testing.T) {
	fixture := NewTransactionFixture()
	transaction := fixture.DefaultTransaction().
		WithPaymentID("payment_123").
		WithType("CAPTURE").
		WithAmount(3000)

	assert.Equal(t, "payment_123", transaction.PaymentID)
	assert.Equal(t, "CAPTURE", transaction.Type)
	assert.Equal(t, int64(3000), transaction.Amount)
}

// TestCommonTestHelper_ContextTimeout tests the common test helper for context creation.
func TestCommonTestHelper_ContextTimeout(t *testing.T) {
	helper := NewCommonTestHelper()
	ctx, cancel := helper.CreateContextWithTimeout(helper.DefaultTimeout())
	defer cancel()

	assert.NotNil(t, ctx)
	// Verify context is not immediately cancelled
	select {
	case <-ctx.Done():
		t.Fatal("context should not be cancelled immediately")
	default:
		// Expected behavior
	}
}
