package bybit

import (
	"context"
	"fmt"

	"xarb/internal/domain/service"
)

// SpotAccountClient Bybit 现货账户查询客户端
type SpotAccountClient struct {
	*ClientFields
}

// spotAccountResponse 现货账户响应结构
type spotAccountResponse struct {
	RetCode int    `json:"retCode"`
	RetMsg  string `json:"retMsg"`
	Result  struct {
		Coin []struct {
			Coin       string `json:"coin"`
			CoinId     string `json:"coinId"`
			WalletType string `json:"walletType"` // SPOT, FUNDING, etc
			Transfer   string `json:"transfer"`
			Status     string `json:"status"`
			ChainType  string `json:"chainType"`
			Chains     []struct {
				Chain                 string `json:"chain"`
				ChainType             string `json:"chainType"`
				Confirmation          string `json:"confirmation"`
				WithdrawFee           string `json:"withdrawFee"`
				DepositMin            string `json:"depositMin"`
				WithdrawMin           string `json:"withdrawMin"`
				WithdrawPercentageFee string `json:"withdrawPercentageFee"`
				DepositPercentageFee  string `json:"depositPercentageFee"`
				AvgProcessTime        string `json:"avgProcessTime"`
			} `json:"chains"`
		} `json:"coin"`
		Members []struct {
			MemberId string `json:"memberId"`
			Assets   []struct {
				Coin          string `json:"coin"`
				WalletBalance string `json:"walletBalance"`
				Transferable  string `json:"transferable"`
			} `json:"assets"`
		} `json:"members"`
	} `json:"result"`
}

// GetAccount 获取现货账户信息
func (c *SpotAccountClient) GetAccount(ctx context.Context) (*service.AccountInfo, error) {
	// TODO: 实现 GET /v5/account/wallet-balance?accountType=SPOT
	// Bybit 现货账户 API: https://bybit-exchange.cn/zh-CN/help-center/article/WALLET_API
	return nil, fmt.Errorf("not implemented")
}

// GetPositions 获取现货持仓（实际是代币余额）
func (c *SpotAccountClient) GetPositions(ctx context.Context) ([]*service.PositionInfo, error) {
	// TODO: 实现 GET /v5/account/wallet-balance?accountType=SPOT 并返回非零余额作为 "持仓"
	// 现货没有真正的持仓概念，只有钱包余额
	return nil, fmt.Errorf("not implemented")
}

// GetOpenOrders 获取现货挂单
func (c *SpotAccountClient) GetOpenOrders(ctx context.Context, symbol string) ([]*service.OpenOrderInfo, error) {
	// TODO: 实现 GET /v5/order/realtime?category=spot&symbol=BTCUSDT
	return nil, fmt.Errorf("not implemented")
}

// GetOrderHistory 获取现货订单历史
func (c *SpotAccountClient) GetOrderHistory(ctx context.Context, symbol string, limit int) ([]*service.OrderLog, error) {
	// TODO: 实现 GET /v5/order/history?category=spot&symbol=BTCUSDT&limit=100
	return nil, fmt.Errorf("not implemented")
}

// GetBalance 获取现货总余额
func (c *SpotAccountClient) GetBalance(ctx context.Context) (float64, error) {
	// TODO: 实现 GET /v5/account/wallet-balance?accountType=SPOT 并计算总 USDT 价值
	return 0, fmt.Errorf("not implemented")
}
