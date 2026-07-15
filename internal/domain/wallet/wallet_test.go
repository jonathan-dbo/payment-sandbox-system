package wallet_test

import (
	"testing"

	"github.com/gonszalito/go-ddd-architecture/internal/domain/wallet"
	"github.com/stretchr/testify/assert"
)

func TestWalletFieldAssignment(t *testing.T) {
	w := &wallet.Wallet{
		ID:         "w-1",
		MerchantID: "m-1",
		Balance:    1000,
	}
	assert.Equal(t, "w-1", w.ID)
	assert.Equal(t, "m-1", w.MerchantID)
	assert.Equal(t, int64(1000), w.Balance)
}

func TestWalletZeroValue(t *testing.T) {
	var w wallet.Wallet
	assert.Equal(t, "", w.ID)
	assert.Equal(t, int64(0), w.Balance)
}

func TestWalletBalanceMutation(t *testing.T) {
	w := &wallet.Wallet{ID: "w-1", MerchantID: "m-1", Balance: 500}
	w.Balance += 250
	assert.Equal(t, int64(750), w.Balance)
	w.Balance -= 1000
	assert.Equal(t, int64(-250), w.Balance)
}
