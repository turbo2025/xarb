package service

import (
	"context"
	"testing"
)

type mockRepository struct {
	priceUpdates map[string]float64
}

func (m *mockRepository) UpsertLatestPrice(ctx context.Context, ex, symbol string, price float64, ts int64) error {
	key := ex + ":" + symbol
	m.priceUpdates[key] = price
	return nil
}

func (m *mockRepository) UpsertPosition(ctx context.Context, ex, symbol string, quantity, entryPrice float64, ts int64) error {
	return nil
}

func (m *mockRepository) GetPosition(ctx context.Context, ex, symbol string) (float64, float64, error) {
	return 0, 0, nil
}

func (m *mockRepository) ListPositions(ctx context.Context) ([]map[string]interface{}, error) {
	return nil, nil
}

func (m *mockRepository) InsertSnapshot(ctx context.Context, ts int64, payload string) error {
	return nil
}

func (m *mockRepository) InsertSignal(ctx context.Context, ts int64, symbol string, delta float64, payload string) error {
	return nil
}

func (m *mockRepository) Close() error {
	return nil
}

func TestPriceServiceUpdatePrice(t *testing.T) {
	mock := &mockRepository{priceUpdates: make(map[string]float64)}
	svc := NewPriceService(mock)

	ctx := context.Background()
	err := svc.UpdatePrice(ctx, "binance", "BTC/USDT", 45000.0, 1234567890)

	if err != nil {
		t.Fatalf("UpdatePrice failed: %v", err)
	}

	key := "binance:BTC/USDT"
	if price, exists := mock.priceUpdates[key]; !exists || price != 45000.0 {
		t.Errorf("expected price 45000.0, got %v", price)
	}
}
