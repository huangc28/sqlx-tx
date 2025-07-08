# sqlx-tx

A robust Go package for handling database transactions with automatic rollback, context support, and type safety. It handles `commit` and `rollback` automatically based on function results and provides panic recovery.

## Features

- ✅ **Type-safe transactions** with Go generics
- ✅ **Context support** for cancellation and timeouts
- ✅ **Automatic rollback** on errors and panics
- ✅ **Configurable transaction options**
- ✅ **Database-agnostic** (works with PostgreSQL, MySQL, SQLite, etc.)
- ✅ **Optimized for serverless** (minimal overhead)

## Installation

```bash
go get github.com/huangc28/sqlx-tx
```

## Basic Usage

### Simple Transaction
```go
package main

import (
    "database/sql"
    "log"

    "github.com/jmoiron/sqlx"
    _ "github.com/lib/pq"
    "github.com/huangc28/sqlx-tx"
)

func main() {
    db, err := sqlx.Connect("postgres", "user=username dbname=mydb sslmode=disable")
    if err != nil {
        log.Fatal(err)
    }
    defer db.Close()

    // Type-safe transaction - returns int
    userID, err := sqlxtx.Execute(db, func(tx *sqlx.Tx) (int, error) {
        var id int
        err := tx.QueryRow("INSERT INTO users (name) VALUES ($1) RETURNING id", "John").Scan(&id)
        if err != nil {
            return 0, err
        }

        // More operations...

        return id, nil
    })

    if err != nil {
        log.Fatal(err)
    }

    log.Printf("Created user with ID: %d", userID)
}
```

### With Context (Recommended)
```go
import (
    "context"
    "time"
)

func createUserWithTimeout(db *sqlx.DB, name string) (int, error) {
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()

    return sqlxtx.ExecuteContext(ctx, db, nil, func(tx *sqlx.Tx) (int, error) {
        var id int
        err := tx.QueryRowContext(ctx, "INSERT INTO users (name) VALUES ($1) RETURNING id", name).Scan(&id)
        return id, err
    })
}
```

### With Custom Configuration
```go
import (
    "database/sql"
    "context"
)

func createUserWithConfig(db *sqlx.DB, name string) (int, error) {
    config := &sqlxtx.Config{
        TxOptions: &sql.TxOptions{
            Isolation: sql.LevelReadCommitted,
            ReadOnly:  false,
        },
        DeallocateAll: true, // PostgreSQL only - use with caution
    }

    return sqlxtx.ExecuteWithConfig(db, config, func(tx *sqlx.Tx) (int, error) {
        var id int
        err := tx.QueryRow("INSERT INTO users (name) VALUES ($1) RETURNING id", name).Scan(&id)
        return id, err
    })
}
```

## Advanced Usage

### Complex Transaction with Multiple Operations
```go
type UserOrder struct {
    UserID  int
    OrderID int
    Total   float64
}

func createUserAndOrder(db *sqlx.DB, userName string, orderTotal float64) (UserOrder, error) {
    return sqlxtx.Execute(db, func(tx *sqlx.Tx) (UserOrder, error) {
        // Create user
        var userID int
        err := tx.QueryRow(
            "INSERT INTO users (name) VALUES ($1) RETURNING id",
            userName,
        ).Scan(&userID)
        if err != nil {
            return UserOrder{}, err
        }

        // Create order
        var orderID int
        err = tx.QueryRow(
            "INSERT INTO orders (user_id, total) VALUES ($1, $2) RETURNING id",
            userID, orderTotal,
        ).Scan(&orderID)
        if err != nil {
            return UserOrder{}, err
        }

        return UserOrder{
            UserID:  userID,
            OrderID: orderID,
            Total:   orderTotal,
        }, nil
    })
}
```

### Error Handling and Rollback
```go
func transferMoney(db *sqlx.DB, fromAccount, toAccount int, amount float64) error {
    _, err := sqlxtx.Execute(db, func(tx *sqlx.Tx) (any, error) {
        // Debit from account
        result, err := tx.Exec(
            "UPDATE accounts SET balance = balance - $1 WHERE id = $2 AND balance >= $1",
            amount, fromAccount,
        )
        if err != nil {
            return nil, err
        }

        rowsAffected, _ := result.RowsAffected()
        if rowsAffected == 0 {
            return nil, errors.New("insufficient funds")
        }

        // Credit to account
        _, err = tx.Exec(
            "UPDATE accounts SET balance = balance + $1 WHERE id = $2",
            amount, toAccount,
        )
        if err != nil {
            return nil, err // Automatic rollback
        }

        return nil, nil
    })

    return err
}
```

## API Reference

### Functions

#### `Execute[T any](db *sqlx.DB, txFunc TxFunc[T]) (T, error)`
Executes a transaction with default settings.

#### `ExecuteWithConfig[T any](db *sqlx.DB, config *Config, txFunc TxFunc[T]) (T, error)`
Executes a transaction with custom configuration.

#### `ExecuteContext[T any](ctx context.Context, db *sqlx.DB, config *Config, txFunc TxFunc[T]) (T, error)`
Executes a transaction with context support and optional configuration.

### Types

#### `TxFunc[T any]`
```go
type TxFunc[T any] func(tx *sqlx.Tx) (T, error)
```
Function type that operates within a database transaction.

#### `Config`
```go
type Config struct {
    TxOptions     *sql.TxOptions // Custom transaction options
    DeallocateAll bool          // PostgreSQL-specific cleanup (default: false)
}
```

## Best Practices

### 1. **Use Context for Timeouts**
```go
ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
defer cancel()

result, err := sqlxtx.ExecuteContext(ctx, db, nil, txFunc)
```

### 2. **Keep Transactions Short**
```go
// ✅ Good - focused transaction
result, err := sqlxtx.Execute(db, func(tx *sqlx.Tx) (User, error) {
    return createUser(tx, userData)
})

// ❌ Avoid - long-running operations
result, err := sqlxtx.Execute(db, func(tx *sqlx.Tx) (User, error) {
    user := createUser(tx, userData)
    sendEmail(user.Email) // Don't do I/O in transactions
    return user, nil
})
```

### 3. **Use Type-Safe Returns**
```go
// ✅ Type-safe
userID, err := sqlxtx.Execute(db, func(tx *sqlx.Tx) (int, error) {
    // ...
    return id, nil
})

// ❌ Less safe
result, err := sqlxtx.Execute(db, func(tx *sqlx.Tx) (any, error) {
    // ...
    return id, nil
})
userID := result.(int) // Type assertion required
```

### 4. **Serverless/Vercel Optimization**
For serverless functions, use the default configuration (no `DeallocateAll`):
```go
// ✅ Optimized for serverless
result, err := sqlxtx.Execute(db, txFunc)

// ❌ Unnecessary overhead in serverless
config := &sqlxtx.Config{DeallocateAll: true}
result, err := sqlxtx.ExecuteWithConfig(db, config, txFunc)
```

## Database Compatibility

This package works with any database supported by `sqlx`:
- ✅ PostgreSQL
- ✅ MySQL
- ✅ SQLite
- ✅ SQL Server
- ✅ Oracle (with appropriate drivers)

**Note**: The `DeallocateAll` option is PostgreSQL-specific and should only be used with PostgreSQL databases.

## License

MIT License
