# sqlx-tx

A simple Go package for handling database transactions with automatic rollback. It handles `commit` and `rollback` whenever an error returns from the db transaction handler.

## Installation

```bash
go get github.com/huangc28/sqlx-tx
```

## Usage

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

    result, err := sqlxtx.Execute(db, func(tx *sqlx.Tx) (any, error) {
        // Your database operations here
        var id int
        err := tx.QueryRow("INSERT INTO users (name) VALUES ($1) RETURNING id", "John").Scan(&id)
        if err != nil {
            return nil, err
        }

        // More operations...

        return id, nil
    })

    if err != nil {
        log.Fatal(err)
    }

    log.Printf("Created user with ID: %v", result)
}
```
