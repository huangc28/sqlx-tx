package sqlxtx

import (
	"errors"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/jmoiron/sqlx"
)

func TestExecute_Success(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer db.Close()

	sqlxDB := sqlx.NewDb(db, "postgres")

	mock.ExpectBegin()
	mock.ExpectExec("DEALLOCATE ALL").WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectQuery("SELECT 1").WillReturnRows(sqlmock.NewRows([]string{"result"}).AddRow(1))
	mock.ExpectCommit()

	result, err := Execute(sqlxDB, func(tx *sqlx.Tx) (any, error) {
		var result int
		err := tx.QueryRow("SELECT 1").Scan(&result)
		return result, err
	})

	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}

	if result != 1 {
		t.Errorf("expected result to be 1, got %v", result)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}
}

func TestExecute_Rollback(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer db.Close()

	sqlxDB := sqlx.NewDb(db, "postgres")

	mock.ExpectBegin()
	mock.ExpectExec("DEALLOCATE ALL").WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectRollback()

	_, err = Execute(sqlxDB, func(tx *sqlx.Tx) (any, error) {
		return nil, errors.New("test error")
	})

	if err == nil {
		t.Error("expected error, got nil")
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}
}

func TestExecute_Panic(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer db.Close()

	sqlxDB := sqlx.NewDb(db, "postgres")

	mock.ExpectBegin()
	mock.ExpectExec("DEALLOCATE ALL").WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectRollback()

	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic, but function did not panic")
		}
	}()

	Execute(sqlxDB, func(tx *sqlx.Tx) (any, error) {
		panic("test panic")
	})

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}
}
