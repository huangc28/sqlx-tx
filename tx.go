package sqlxtx

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/jmoiron/sqlx"
)

// TxFunc defines a function type that operates within a database transaction
type TxFunc[T any] func(tx *sqlx.Tx) (T, error)

// Config holds configuration options for transaction execution
type Config struct {
	TxOptions     *sql.TxOptions
	DeallocateAll bool // PostgreSQL specific
}

// ConfigOption is a function that modifies Config
type ConfigOption func(*Config)

// WithDeallocateAll enables PostgreSQL-specific prepared statement cleanup
func WithDeallocateAll() ConfigOption {
	return func(c *Config) {
		c.DeallocateAll = true
	}
}

// WithTxOptions sets custom transaction options
func WithTxOptions(opts *sql.TxOptions) ConfigOption {
	return func(c *Config) {
		c.TxOptions = opts
	}
}

// WithIsolationLevel sets the transaction isolation level
func WithIsolationLevel(level sql.IsolationLevel) ConfigOption {
	return func(c *Config) {
		if c.TxOptions == nil {
			c.TxOptions = &sql.TxOptions{}
		}
		c.TxOptions.Isolation = level
	}
}

// WithReadOnly sets the transaction to read-only mode
func WithReadOnly() ConfigOption {
	return func(c *Config) {
		if c.TxOptions == nil {
			c.TxOptions = &sql.TxOptions{}
		}
		c.TxOptions.ReadOnly = true
	}
}

// Execute runs a function within a transaction with default settings
func Execute[T any](db *sqlx.DB, txFunc TxFunc[T]) (T, error) {
	return ExecuteContext(context.Background(), db, txFunc)
}

// ExecuteContext runs a function within a transaction with context support and optional configuration
func ExecuteContext[T any](ctx context.Context, db *sqlx.DB, txFunc TxFunc[T], options ...ConfigOption) (result T, err error) {
	// Apply configuration options
	config := &Config{}
	for _, option := range options {
		option(config)
	}

	tx, err := db.BeginTxx(ctx, config.TxOptions)
	if err != nil {
		return result, fmt.Errorf("failed to begin transaction: %w", err)
	}

	// PostgreSQL-specific cleanup (optional)
	if config.DeallocateAll {
		if _, deallocErr := tx.ExecContext(ctx, "DEALLOCATE ALL"); deallocErr != nil {
			_ = tx.Rollback()
			return result, fmt.Errorf("failed to deallocate prepared statements: %w", deallocErr)
		}
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

	result, err = txFunc(tx)
	return result, err
}
