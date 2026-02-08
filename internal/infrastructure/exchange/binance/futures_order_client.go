package binance

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/rs/zerolog/log"
)

// FuturesOrderClient Binance 期货 REST 客户端
type FuturesOrderClient struct {
	*clientFields
	baseURL string
}

// NewFuturesOrderClient 创建 Binance 期货客户端
func NewFuturesOrderClient(apiKey, secretKey string) *FuturesOrderClient {
	return &FuturesOrderClient{
		clientFields: &clientFields{
			apiKey:    apiKey,
			apiSecret: secretKey,
			client: &http.Client{
				Timeout: time.Second * 30,
			},
		},
		baseURL: "https://fapi.binance.com",
	}
}

// PlaceOrder 下单
// side: "BUY" 或 "SELL"
// quantity: 交易数量
// price: 价格（市价单为 0）
// isMarket: 是否市价单
func (c *FuturesOrderClient) PlaceOrder(
	ctx context.Context,
	symbol string,
	side string,
	quantity float64,
	price float64,
	isMarket bool,
) (string, error) {
	params := url.Values{}
	params.Set("symbol", symbol)
	params.Set("side", side)
	params.Set("quantity", fmt.Sprintf("%.8g", quantity))

	if isMarket {
		params.Set("type", "MARKET")
	} else {
		params.Set("type", "LIMIT")
		params.Set("timeInForce", "GTC") // Good Till Cancel
		params.Set("price", fmt.Sprintf("%.8g", price))
	}

	body, err := c.sendRequest(ctx, http.MethodPost, "/fapi/v1/order", params)
	if err != nil {
		return "", fmt.Errorf("place order failed: %w", err)
	}

	var resp OrderResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return "", fmt.Errorf("parse order response failed: %w", err)
	}

	if resp.OrderID == 0 {
		return "", fmt.Errorf("order failed: %s", string(body))
	}

	log.Info().
		Str("exchange", "BINANCE").
		Str("symbol", symbol).
		Str("side", side).
		Float64("quantity", quantity).
		Float64("price", price).
		Int64("orderID", resp.OrderID).
		Str("status", resp.Status).
		Msg("order placed")

	return strconv.FormatInt(resp.OrderID, 10), nil
}

// CancelOrder 撤销订单
func (c *FuturesOrderClient) CancelOrder(ctx context.Context, symbol string, orderId string) error {
	params := url.Values{}
	params.Set("symbol", symbol)
	params.Set("orderId", orderId)

	body, err := c.sendRequest(ctx, http.MethodDelete, "/fapi/v1/order", params)
	if err != nil {
		return fmt.Errorf("cancel order failed: %w", err)
	}

	var resp OrderResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return fmt.Errorf("parse cancel response failed: %w", err)
	}

	log.Info().
		Str("exchange", "BINANCE").
		Str("symbol", symbol).
		Str("orderId", orderId).
		Str("status", resp.Status).
		Msg("order cancelled")

	return nil
}

// GetOrderStatus 查询订单状态
func (c *FuturesOrderClient) GetOrderStatus(
	ctx context.Context,
	symbol string,
	orderId string,
) (*OrderStatus, error) {
	params := url.Values{}
	params.Set("symbol", symbol)
	params.Set("orderId", orderId)

	body, err := c.sendRequest(ctx, http.MethodGet, "/fapi/v1/openOrder", params)
	if err != nil {
		return nil, fmt.Errorf("get order status failed: %w", err)
	}

	var resp OrderResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("parse order status failed: %w", err)
	}

	executedQty, _ := strconv.ParseFloat(resp.ExecutedQty, 64)
	avgPrice, _ := strconv.ParseFloat(resp.AvgPrice, 64)

	status := &OrderStatus{
		OrderID:          orderId,
		Symbol:           symbol,
		Side:             resp.Side,
		Quantity:         resp.OrigQty,
		ExecutedQuantity: executedQty,
		Price:            resp.Price,
		AvgExecutedPrice: avgPrice,
		Status:           resp.Status,
		CreatedAt:        resp.Time,
		UpdatedAt:        resp.UpdateTime,
	}

	return status, nil
}

// GetFundingRate 获取融资费率
func (c *FuturesOrderClient) GetFundingRate(ctx context.Context, symbol string) (float64, error) {
	params := url.Values{}
	params.Set("symbol", symbol)

	body, err := c.sendRequest(ctx, http.MethodGet, "/fapi/v1/fundingRate", params)
	if err != nil {
		return 0, fmt.Errorf("get funding rate failed: %w", err)
	}

	var resp FundingRateResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return 0, fmt.Errorf("parse funding rate failed: %w", err)
	}

	rate, _ := strconv.ParseFloat(resp.FundingRate, 64)
	return rate, nil
}

// GetAccount 查询账户信息
func (c *FuturesOrderClient) GetAccount(ctx context.Context) (*AccountInfo, error) {
	body, err := c.sendRequest(ctx, http.MethodGet, "/fapi/v2/account", url.Values{})
	if err != nil {
		return nil, fmt.Errorf("get account failed: %w", err)
	}

	var resp AccountResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("parse account response failed: %w", err)
	}

	// 解析账户数据
	account := &AccountInfo{
		CanDeposit:  resp.CanDeposit,
		CanTrade:    resp.CanTrade,
		CanWithdraw: resp.CanWithdraw,
		FeeTier:     resp.FeeTier,
		Positions:   make([]*Position, 0),
	}

	// 解析持仓
	for _, pos := range resp.Positions {
		posQty, _ := strconv.ParseFloat(pos.PositionAmt, 64)
		entryPrice, _ := strconv.ParseFloat(pos.EntryPrice, 64)
		unPnL, _ := strconv.ParseFloat(pos.UnrealizedProfit, 64)
		markPrice, _ := strconv.ParseFloat(pos.MarkPrice, 64)

		account.Positions = append(account.Positions, &Position{
			Symbol:           pos.Symbol,
			PositionAmount:   posQty,
			EntryPrice:       entryPrice,
			MarkPrice:        markPrice,
			UnrealizedProfit: unPnL,
			Leverage:         pos.Leverage,
			IsAutoAddMargin:  pos.AutoAddMargin == "true",
		})
	}

	// 解析资产
	totalWalletBalance, _ := strconv.ParseFloat(resp.TotalWalletBalance, 64)
	totalUnrealizedProfit, _ := strconv.ParseFloat(resp.TotalUnrealizedProfit, 64)
	totalMarginRequired, _ := strconv.ParseFloat(resp.TotalMarginRequired, 64)
	totalOpenOrderInitialMargin, _ := strconv.ParseFloat(resp.TotalOpenOrderInitialMargin, 64)

	account.TotalWalletBalance = totalWalletBalance
	account.TotalUnrealizedProfit = totalUnrealizedProfit
	account.TotalMarginRequired = totalMarginRequired
	account.TotalOpenOrderInitialMargin = totalOpenOrderInitialMargin
	account.AvailableBalance = totalWalletBalance - totalMarginRequired

	return account, nil
}

// GetOpenPositions 查询所有开仓持仓
func (c *FuturesOrderClient) GetOpenPositions(ctx context.Context) ([]*Position, error) {
	account, err := c.GetAccount(ctx)
	if err != nil {
		return nil, err
	}

	// 过滤出有仓位的持仓
	var openPositions []*Position
	for _, pos := range account.Positions {
		if pos.PositionAmount != 0 {
			openPositions = append(openPositions, pos)
		}
	}

	return openPositions, nil
}

// sendRequest 发送 HTTP 请求
func (c *FuturesOrderClient) sendRequest(
	ctx context.Context,
	method string,
	path string,
	params url.Values,
) ([]byte, error) {
	// 添加时间戳
	params.Set("timestamp", strconv.FormatInt(time.Now().UnixMilli(), 10))

	// 签名
	signature := c.sign(params.Encode())
	params.Set("signature", signature)

	// 构建请求URL
	reqURL := c.baseURL + path + "?" + params.Encode()

	req, err := http.NewRequestWithContext(ctx, method, reqURL, nil)
	if err != nil {
		return nil, err
	}

	// 添加 API Key 头
	req.Header.Set("X-MBX-APIKEY", c.apiKey)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	// 执行请求
	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	// 检查 HTTP 状态码
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("http %d: %s", resp.StatusCode, string(body))
	}

	return body, nil
}

// sign 生成签名
func (c *FuturesOrderClient) sign(data string) string {
	h := hmac.New(sha256.New, []byte(c.apiSecret))
	h.Write([]byte(data))
	return hex.EncodeToString(h.Sum(nil))
}

// ===== Response Models =====

// OrderResponse 订单响应
type OrderResponse struct {
	OrderID       int64   `json:"orderId"`
	Symbol        string  `json:"symbol"`
	Status        string  `json:"status"`
	ClientOrderID string  `json:"clientOrderId"`
	Side          string  `json:"side"`
	Type          string  `json:"type"`
	TimeInForce   string  `json:"timeInForce"`
	OrigQty       float64 `json:"origQty,string"`
	ExecutedQty   string  `json:"executedQty"`
	AvgPrice      string  `json:"avgPrice"`
	Price         float64 `json:"price,string"`
	Time          int64   `json:"time"`
	UpdateTime    int64   `json:"updateTime"`
}

// AccountResponse 账户响应
type AccountResponse struct {
	FeeTier                     int         `json:"feeTier"`
	CanTrade                    bool        `json:"canTrade"`
	CanDeposit                  bool        `json:"canDeposit"`
	CanWithdraw                 bool        `json:"canWithdraw"`
	UpdateTime                  int64       `json:"updateTime"`
	TotalInitialMargin          string      `json:"totalInitialMargin"`
	TotalMaintMargin            string      `json:"totalMaintMargin"`
	TotalWalletBalance          string      `json:"totalWalletBalance"`
	TotalUnrealizedProfit       string      `json:"totalUnrealizedProfit"`
	TotalMarginRequired         string      `json:"totalMarginRequired"`
	TotalOpenOrderInitialMargin string      `json:"totalOpenOrderInitialMargin"`
	TotalCrossWalletBalance     string      `json:"totalCrossWalletBalance"`
	TotalCrossUnPnl             string      `json:"totalCrossUnPnl"`
	AvailableBalance            string      `json:"availableBalance"`
	MaxWithdrawAmount           string      `json:"maxWithdrawAmount"`
	Assets                      []Asset     `json:"assets"`
	Positions                   []PosDetail `json:"positions"`
}

// Asset 资产信息
type Asset struct {
	Asset                  string `json:"asset"`
	WalletBalance          string `json:"walletBalance"`
	UnrealizedProfit       string `json:"unrealizedProfit"`
	MaintMargin            string `json:"maintMargin"`
	InitialMargin          string `json:"initialMargin"`
	PositionInitialMargin  string `json:"positionInitialMargin"`
	OpenOrderInitialMargin string `json:"openOrderInitialMargin"`
	CrossWalletBalance     string `json:"crossWalletBalance"`
	CrossUnPnl             string `json:"crossUnPnl"`
	AvailableBalance       string `json:"availableBalance"`
	MaxWithdrawAmount      string `json:"maxWithdrawAmount"`
	MarginAvailable        bool   `json:"marginAvailable"`
}

// PosDetail 持仓详情
type PosDetail struct {
	Symbol           string `json:"symbol"`
	InitialMargin    string `json:"initialMargin"`
	MaintMargin      string `json:"maintMargin"`
	UnrealizedProfit string `json:"unrealizedProfit"`
	PositionAmt      string `json:"positionAmt"`
	MarkPrice        string `json:"markPrice"`
	EntryPrice       string `json:"entryPrice"`
	Leverage         string `json:"leverage"`
	AutoAddMargin    string `json:"autoAddMargin"`
	IsolatedCreated  bool   `json:"isolatedCreated"`
	PositionSide     string `json:"positionSide"`
	MaxNotionalValue string `json:"maxNotionalValue"`
	BidNotional      string `json:"bidNotional"`
	AskNotional      string `json:"askNotional"`
	UpdateTime       int64  `json:"updateTime"`
}

// FundingRateResponse 融资费率响应
type FundingRateResponse struct {
	Symbol      string `json:"symbol"`
	FundingRate string `json:"fundingRate"`
	FundingTime int64  `json:"fundingTime"`
}

// ===== Domain Models =====

// Position 持仓信息
type Position struct {
	Symbol           string
	PositionAmount   float64 // 持仓数量
	EntryPrice       float64 // 开仓价格
	MarkPrice        float64 // 标记价格
	UnrealizedProfit float64 // 未实现盈亏
	Leverage         string
	IsAutoAddMargin  bool
}

// AccountInfo 账户信息
type AccountInfo struct {
	CanDeposit                  bool
	CanTrade                    bool
	CanWithdraw                 bool
	FeeTier                     int
	TotalWalletBalance          float64 // 总钱包余额
	TotalUnrealizedProfit       float64 // 总未实现盈亏
	TotalMarginRequired         float64 // 总所需保证金
	TotalOpenOrderInitialMargin float64 // 总未成交订单初始保证金
	AvailableBalance            float64 // 可用余额
	Positions                   []*Position
}

// OrderStatus 订单状态
type OrderStatus struct {
	OrderID          string
	Symbol           string
	Side             string
	Quantity         float64
	ExecutedQuantity float64
	Price            float64
	AvgExecutedPrice float64
	Status           string
	CreatedAt        int64
	UpdatedAt        int64
}
