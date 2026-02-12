package bybit

import (
	"context"
	"fmt"

	"xarb/internal/domain/service"
)

// SpotAccountClient Bybit 现货账户查询客户端
type SpotAccountClient struct {
	*APIClient
}

// NewSpotAccountClient 创建现货账户客户端
func NewSpotAccountClient(client *APIClient) *SpotAccountClient {
	return &SpotAccountClient{APIClient: client}
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

// GetBalance 获取现货总余额
func (c *SpotAccountClient) GetBalance(ctx context.Context) (float64, error) {
	// TODO: 实现 GET /v5/account/wallet-balance?accountType=SPOT 并计算总 USDT 价值
	return 0, fmt.Errorf("not implemented")
}
