package bybit

import (
	"context"
	"fmt"
)

// PerpetualPositionClient Bybit 永续合约持仓客户端
type PerpetualPositionClient struct {
	*APIClient
}

// NewPerpetualPositionClient 创建永续合约持仓客户端
func NewPerpetualPositionClient(client *APIClient) *PerpetualPositionClient {
	return &PerpetualPositionClient{APIClient: client}
}

// PerpetualPositionResponse 永续合约持仓响应（包含详细的持仓数据）
type PerpetualPositionResponse struct {
	RetCode int    `json:"retCode"`
	RetMsg  string `json:"retMsg"`
	Result  struct {
		List []struct {
			PositionIdx     int    `json:"positionIdx"`
			TradeMode       int    `json:"tradeMode"`
			RiskID          int    `json:"riskId"`
			RiskLimitValue  string `json:"riskLimitValue"`
			Symbol          string `json:"symbol"`
			Side            string `json:"side"`
			Size            string `json:"size"`
			PositionValue   string `json:"positionValue"`
			EntryPrice      string `json:"entryPrice"`
			Leverage        string `json:"leverage"`
			PosLeverage     string `json:"posLeverage"`
			MarkPrice       string `json:"markPrice"`
			LiqPrice        string `json:"liqPrice"`
			BustPrice       string `json:"bustPrice"`
			IM              string `json:"im"`
			MM              string `json:"mm"`
			RealisedPnl     string `json:"realisedPnl"`
			UnrealisedPnl   string `json:"unrealisedPnl"`
			CumRealisedPnl  string `json:"cumRealisedPnl"`
			SessionAvgPrice string `json:"sessionAvgPrice"`
			Opm             string `json:"opm"`
			Tp              string `json:"tp"`
			Sl              string `json:"sl"`
			TpslMode        string `json:"tpslMode"`
			CreatedTime     string `json:"createdTime"`
			UpdatedTime     string `json:"updatedTime"`
			IsUsed          bool   `json:"isUsed"`
		} `json:"list"`
	} `json:"result"`
}

// GetPositions 获取永续合约持仓
func (c *PerpetualPositionClient) GetPositions(ctx context.Context) (interface{}, error) {
	// TODO: 实现 GET /v5/position/list?category=linear
	return nil, fmt.Errorf("not implemented")
}

// GetPosition 获取单个交易对的持仓
func (c *PerpetualPositionClient) GetPosition(ctx context.Context, symbol string) (interface{}, error) {
	// TODO: 实现 GET /v5/position/list?category=linear&symbol=BTCUSDT
	return nil, fmt.Errorf("not implemented")
}

// ClosePosition 平仓（发送反向订单）
func (c *PerpetualPositionClient) ClosePosition(ctx context.Context, symbol string) error {
	// TODO: 实现平仓逻辑
	return fmt.Errorf("not implemented")
}

// GetLiquidationPrice 获取清算价格
func (c *PerpetualPositionClient) GetLiquidationPrice(ctx context.Context, symbol string) (float64, error) {
	// TODO: 从持仓数据计算清算价格
	return 0, fmt.Errorf("not implemented")
}
