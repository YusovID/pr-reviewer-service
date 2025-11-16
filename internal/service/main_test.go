package service

import (
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/require"
)

func newMockDBAndTx(t *testing.T) (*sqlx.DB, *sqlx.Tx, sqlmock.Sqlmock) {
	t.Helper()
	mockDB, smock, err := sqlmock.New()
	require.NoError(t, err)
	sqlxDB := sqlx.NewDb(mockDB, "sqlmock")
	smock.ExpectBegin()
	tx, err := sqlxDB.Beginx()
	require.NoError(t, err)
	return sqlxDB, tx, smock
}
