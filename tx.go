package sqlxtx

import (
	"fmt"

	"github.com/jmoiron/sqlx"
)

// TxFunc defines a function type that operates within a database transaction.
// It receives a transaction handle and returns a result and error.
type TxFunc func(tx *sqlx.Tx) (any, error)

// Execute runs the provided function within a database transaction.
// It handles transaction begin, commit, and rollback automatically.
// If the function returns an error or panics, the transaction is rolled back.
// Otherwise, the transaction is committed.
func Execute(db *sqlx.DB, txFunc TxFunc) (any, error) {
	var (
		tx  *sqlx.Tx
		err error
		res any
	)

	tx, err = db.Beginx()
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}

	if _, deallocErr := tx.Exec("DEALLOCATE ALL"); deallocErr != nil {
		_ = tx.Rollback()
		return nil, fmt.Errorf("failed to deallocate prepared statements: %w", deallocErr)
	}

	defer func() {
		if p := recover(); p != nil {
			_ = tx.Rollback()
			panic(p)
		} else if err != nil {
			if rollbackErr := tx.Rollback(); rollbackErr != nil {
				err = fmt.Errorf("transaction rollback failed: %v (original error: %w)", rollbackErr, err)
			}
		} else {
			err = tx.Commit()
			if err != nil {
				err = fmt.Errorf("failed to commit transaction: %w", err)
			}
		}
	}()

	res, err = txFunc(tx)
	return res, err
}
