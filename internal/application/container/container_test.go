package container

import (
	"context"
	"os"
	"testing"

	"xarb/internal/application/service"
	"xarb/internal/infrastructure/config"
	infracontainer "xarb/internal/infrastructure/container"
)

func TestContainerWithSQLite(t *testing.T) {
	dbPath := "test_container.db"
	defer os.Remove(dbPath)

	cfg := &config.Config{}
	cfg.SQLite.Enabled = true
	cfg.SQLite.Path = dbPath

	c, err := infracontainer.New(cfg)
	if err != nil {
		t.Fatalf("failed to create container: %v", err)
	}
	defer c.Close()

	repo := c.SQLiteRepo()
	if repo == nil {
		t.Errorf("expected SQLiteRepo, got nil")
	}
}

func TestContainerServiceWorkflow(t *testing.T) {
	dbPath := "test_workflow.db"
	defer os.Remove(dbPath)

	cfg := &config.Config{}
	cfg.SQLite.Enabled = true
	cfg.SQLite.Path = dbPath

	c, err := infracontainer.New(cfg)
	if err != nil {
		t.Fatalf("failed to create container: %v", err)
	}
	defer c.Close()

	ctx := context.Background()
	ts := int64(1234567890)

	repo := c.SQLiteRepo()
	priceService := service.NewPriceService(repo)
	positionService := service.NewPositionService(repo)

	err = priceService.UpdatePrice(ctx, "binance", "BTC/USDT", 45000.0, ts)
	if err != nil {
		t.Fatalf("UpdatePrice failed: %v", err)
	}

	err = positionService.UpdatePosition(ctx, "binance", "BTC/USDT", 1.5, 40000.0, ts)
	if err != nil {
		t.Fatalf("UpdatePosition failed: %v", err)
	}

	positions, err := positionService.ListAllPositions(ctx)
	if err != nil {
		t.Fatalf("ListAllPositions failed: %v", err)
	}

	if len(positions) != 1 {
		t.Errorf("expected 1 position, got %d", len(positions))
	}
}
