package sqlite

import (
	"context"
	"os"
	"testing"
)

func TestSQLiteRepoUpsertPrice(t *testing.T) {
	dbPath := "test.db"
	defer os.Remove(dbPath)

	repo, err := New(dbPath)
	if err != nil {
		t.Fatalf("failed to create repo: %v", err)
	}
	defer repo.Close()

	ctx := context.Background()
	err = repo.UpsertLatestPrice(ctx, "binance", "BTC/USDT", 45000.0, 1234567890)
	if err != nil {
		t.Fatalf("UpsertLatestPrice failed: %v", err)
	}
}

func TestSQLiteRepoUpsertPosition(t *testing.T) {
	dbPath := "test_pos.db"
	defer os.Remove(dbPath)

	repo, err := New(dbPath)
	if err != nil {
		t.Fatalf("failed to create repo: %v", err)
	}
	defer repo.Close()

	ctx := context.Background()
	err = repo.UpsertPosition(ctx, "binance", "BTC/USDT", 1.5, 40000.0, 1234567890)
	if err != nil {
		t.Fatalf("UpsertPosition failed: %v", err)
	}

	quantity, entryPrice, err := repo.GetPosition(ctx, "binance", "BTC/USDT")
	if err != nil {
		t.Fatalf("GetPosition failed: %v", err)
	}

	if quantity != 1.5 || entryPrice != 40000.0 {
		t.Errorf("expected quantity=1.5, entryPrice=40000.0, got %v, %v", quantity, entryPrice)
	}
}

func TestSQLiteRepoListPositions(t *testing.T) {
	dbPath := "test_list.db"
	defer os.Remove(dbPath)

	repo, err := New(dbPath)
	if err != nil {
		t.Fatalf("failed to create repo: %v", err)
	}
	defer repo.Close()

	ctx := context.Background()
	repo.UpsertPosition(ctx, "binance", "BTC/USDT", 1.5, 40000.0, 1234567890)
	repo.UpsertPosition(ctx, "binance", "ETH/USDT", 10.0, 2000.0, 1234567890)

	positions, err := repo.ListPositions(ctx)
	if err != nil {
		t.Fatalf("ListPositions failed: %v", err)
	}

	if len(positions) != 2 {
		t.Errorf("expected 2 positions, got %d", len(positions))
	}
}

func TestSQLiteRepoInsertSnapshot(t *testing.T) {
	dbPath := "test_snapshot.db"
	defer os.Remove(dbPath)

	repo, err := New(dbPath)
	if err != nil {
		t.Fatalf("failed to create repo: %v", err)
	}
	defer repo.Close()

	ctx := context.Background()
	payload := `{"prices":{"BTC/USDT":45000}}`
	err = repo.InsertSnapshot(ctx, 1234567890, payload)
	if err != nil {
		t.Fatalf("InsertSnapshot failed: %v", err)
	}
}

func TestSQLiteRepoInsertSignal(t *testing.T) {
	dbPath := "test_signal.db"
	defer os.Remove(dbPath)

	repo, err := New(dbPath)
	if err != nil {
		t.Fatalf("failed to create repo: %v", err)
	}
	defer repo.Close()

	ctx := context.Background()
	payload := `{"reason":"price spike"}`
	err = repo.InsertSignal(ctx, 1234567890, "BTC/USDT", 0.05, payload)
	if err != nil {
		t.Fatalf("InsertSignal failed: %v", err)
	}
}
