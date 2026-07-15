package database

import (
	"database/sql"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRepositoryErrorTranslation(t *testing.T) {
	err := translateSQLError(sql.ErrNoRows)
	assert.EqualError(t, err, "not_found")
	assert.NoError(t, translateSQLError(nil))
}
