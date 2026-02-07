package service

import (
	"context"
	"testing"
)

func TestPositionServiceUpdatePosition(t *testing.T) {
	mock := &mockRepository{
		priceUpdates: make(map[string]float64),
	}
	svc := NewPositionService(mock)

	ctx := context.Background()
	err := svc.UpdatePosition(ctx, "binance", "BTC/USDT", 1.5, 40000.0, 1234567890)

	if err != nil {
		t.Fatalf("UpdatePosition failed: %v", err)
	}
}

func TestPositionServiceListPositions(t *testing.T) {
	mock := &mockRepository{
		priceUpdates: make(map[string]float64),
	}
	svc := NewPositionService(mock)

	ctx := context.Background()
	positions, err := svc.ListAllPositions(ctx)

	if err != nil {
		t.Fatalf("ListAllPositions failed: %v", err)
	}

	if positions == nil {
		t.Errorf("expected positions slice, got nil")
	}
}
