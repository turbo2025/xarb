package binance

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"xarb/internal/domain/service"
)

// PerpetualAccountClient Binance perpetual account query client
type PerpetualAccountClient struct {
	*APIClient
}

// NewPerpetualAccountClient creates perpetual account client
func NewPerpetualAccountClient(client *APIClient) *PerpetualAccountClient {
	return &PerpetualAccountClient{APIClient: client}
}

// GetAccount gets perpetual account info
func (c *PerpetualAccountClient) GetAccount(ctx context.Context) (*service.AccountInfo, error) {
	body, err := c.signedRequest(ctx, "GET", "/fapi/v2/account", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get binance perpetual account: %w", err)
	}

	var resp AccountResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal binance perpetual account: %w", err)
	}

	// 解析保证金金额
	totalMaintMargin, _ := strconv.ParseFloat(resp.TotalMaintMargin, 64)
	totalWalletBalance, _ := strconv.ParseFloat(resp.TotalWalletBalance, 64)

	return &service.AccountInfo{
		TotalMargin: totalWalletBalance,
		UsedMargin:  totalMaintMargin,
		AvailMargin: totalWalletBalance - totalMaintMargin,
		UpdatedAt:   time.Now(),
	}, nil
}

// GetBalance 获取余额
func (c *PerpetualAccountClient) GetBalance(ctx context.Context) (float64, error) {
	body, err := c.signedRequest(ctx, "GET", "/fapi/v2/account", nil)
	if err != nil {
		return 0, fmt.Errorf("failed to get binance perpetual balance: %w", err)
	}

	var resp AccountResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return 0, fmt.Errorf("failed to unmarshal binance perpetual balance: %w", err)
	}

	balance, _ := strconv.ParseFloat(resp.TotalWalletBalance, 64)
	return balance, nil
}
