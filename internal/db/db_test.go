package db

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"testing"
)

func TestOpen(t *testing.T) {
	t.Run("creates database file", func(t *testing.T) {
		tmpDir := t.TempDir()
		dbPath := filepath.Join(tmpDir, "test.db")

		database, err := Open(dbPath)
		if err != nil {
			t.Fatalf("Open() error = %v", err)
		}
		defer func() { _ = database.Close() }() //nolint:errcheck // Intentionally ignoring close error in test cleanup //nolint:errcheck // Intentionally ignoring close error in test cleanup

		if _, err := os.Stat(dbPath); os.IsNotExist(err) {
			t.Error("database file was not created")
		}
	})

	t.Run("creates parent directories", func(t *testing.T) {
		tmpDir := t.TempDir()
		dbPath := filepath.Join(tmpDir, "nested", "dir", "test.db")

		database, err := Open(dbPath)
		if err != nil {
			t.Fatalf("Open() error = %v", err)
		}
		defer func() { _ = database.Close() }() //nolint:errcheck // Intentionally ignoring close error in test cleanup //nolint:errcheck // Intentionally ignoring close error in test cleanup

		if _, err := os.Stat(dbPath); os.IsNotExist(err) {
			t.Error("database file was not created in nested directory")
		}
	})

	t.Run("runs migrations", func(t *testing.T) {
		tmpDir := t.TempDir()
		dbPath := filepath.Join(tmpDir, "test.db")

		database, err := Open(dbPath)
		if err != nil {
			t.Fatalf("Open() error = %v", err)
		}
		defer func() { _ = database.Close() }() //nolint:errcheck // Intentionally ignoring close error in test cleanup

		// Check that sessions table exists
		var tableName string
		err = database.QueryRowContext(context.Background(),
			"SELECT name FROM sqlite_master WHERE type='table' AND name='sessions'").Scan(&tableName)
		if err != nil {
			t.Fatalf("sessions table not created: %v", err)
		}

		// Check that messages table exists
		err = database.QueryRowContext(context.Background(),
			"SELECT name FROM sqlite_master WHERE type='table' AND name='messages'").Scan(&tableName)
		if err != nil {
			t.Fatalf("messages table not created: %v", err)
		}
	})

	t.Run("enables WAL mode", func(t *testing.T) {
		tmpDir := t.TempDir()
		dbPath := filepath.Join(tmpDir, "test.db")

		database, err := Open(dbPath)
		if err != nil {
			t.Fatalf("Open() error = %v", err)
		}
		defer func() { _ = database.Close() }() //nolint:errcheck // Intentionally ignoring close error in test cleanup

		var journalMode string
		err = database.QueryRowContext(context.Background(), "PRAGMA journal_mode").Scan(&journalMode)
		if err != nil {
			t.Fatalf("failed to get journal_mode: %v", err)
		}

		if journalMode != "wal" {
			t.Errorf("journal_mode = %q, want %q", journalMode, "wal")
		}
	})

	t.Run("enables foreign keys", func(t *testing.T) {
		tmpDir := t.TempDir()
		dbPath := filepath.Join(tmpDir, "test.db")

		database, err := Open(dbPath)
		if err != nil {
			t.Fatalf("Open() error = %v", err)
		}
		defer func() { _ = database.Close() }() //nolint:errcheck // Intentionally ignoring close error in test cleanup

		var foreignKeys int
		err = database.QueryRowContext(context.Background(), "PRAGMA foreign_keys").Scan(&foreignKeys)
		if err != nil {
			t.Fatalf("failed to get foreign_keys: %v", err)
		}

		if foreignKeys != 1 {
			t.Errorf("foreign_keys = %d, want 1", foreignKeys)
		}
	})
}

func TestDB_Path(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	database, err := Open(dbPath)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer func() { _ = database.Close() }() //nolint:errcheck // Intentionally ignoring close error in test cleanup

	if got := database.Path(); got != dbPath {
		t.Errorf("Path() = %q, want %q", got, dbPath)
	}
}

func TestDB_Conn(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	database, err := Open(dbPath)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer func() { _ = database.Close() }() //nolint:errcheck // Intentionally ignoring close error in test cleanup

	conn := database.Conn()
	if conn == nil {
		t.Error("Conn() returned nil")
	}

	// Verify connection is usable
	if err := conn.PingContext(context.Background()); err != nil {
		t.Errorf("connection ping failed: %v", err)
	}
}

func TestDB_WithTx(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	database, err := Open(dbPath)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer func() { _ = database.Close() }() //nolint:errcheck // Intentionally ignoring close error in test cleanup

	t.Run("commits on success", func(t *testing.T) {
		ctx := context.Background()

		err := database.WithTx(ctx, func(tx *sql.Tx) error {
			_, err := tx.ExecContext(ctx, `INSERT INTO sessions (id, title, created_at, updated_at) VALUES ('tx-test', 'Test', 0, 0)`)
			return err
		})
		if err != nil {
			t.Fatalf("WithTx() error = %v", err)
		}

		// Verify row exists
		var id string
		err = database.QueryRowContext(ctx, "SELECT id FROM sessions WHERE id = 'tx-test'").Scan(&id)
		if err != nil {
			t.Errorf("committed row not found: %v", err)
		}
	})

	t.Run("rolls back on error", func(t *testing.T) {
		ctx := context.Background()

		err := database.WithTx(ctx, func(tx *sql.Tx) error {
			_, err := tx.ExecContext(ctx, `INSERT INTO sessions (id, title, created_at, updated_at) VALUES ('rollback-test', 'Test', 0, 0)`)
			if err != nil {
				return err
			}
			return context.Canceled // Simulate error
		})
		if err == nil {
			t.Fatal("WithTx() expected error, got nil")
		}

		// Verify row does not exist
		var id string
		err = database.QueryRowContext(ctx, "SELECT id FROM sessions WHERE id = 'rollback-test'").Scan(&id)
		if err == nil {
			t.Error("rolled back row should not exist")
		}
	})
}

func TestDB_Close(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	database, err := Open(dbPath)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}

	if err := database.Close(); err != nil {
		t.Errorf("Close() error = %v", err)
	}

	// Verify connection is closed
	if err := database.Conn().PingContext(context.Background()); err == nil {
		t.Error("connection should be closed")
	}
}
