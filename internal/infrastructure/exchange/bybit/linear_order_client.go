package bybit

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
	"strings"
	"time"

	"github.com/rs/zerolog/log"
)

// FuturesOrderClient Bybit 期货 REST 客户端 (V5 API)
type FuturesOrderClient struct {
	*ClientFields
	baseURL string
}

// PlaceOrder 下单
func (c *FuturesOrderClient) PlaceOrder(
	ctx context.Context,
	symbol string,
	side string,
	quantity float64,
	price float64,
	isMarket bool,
) (string, error) {
	payload := map[string]interface{}{
		"category": "linear",
		"symbol":   symbol,
		"side":     side,
		"qty":      fmt.Sprintf("%.8g", quantity),
	}

	if isMarket {
		payload["orderType"] = "Market"
	} else {
		payload["orderType"] = "Limit"
		payload["price"] = fmt.Sprintf("%.8g", price)
		payload["timeInForce"] = "GTC"
	}

	body, err := c.sendRequest(ctx, http.MethodPost, "/v5/order/create", payload)
	if err != nil {
		return "", fmt.Errorf("place order failed: %w", err)
	}

	var resp PlaceOrderResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return "", fmt.Errorf("parse order response failed: %w", err)
	}

	if resp.RetCode != 0 {
		return "", fmt.Errorf("place order error: [%d] %s", resp.RetCode, resp.RetMsg)
	}

	log.Info().
		Str("exchange", "BYBIT").
		Str("symbol", symbol).
		Str("side", side).
		Float64("quantity", quantity).
		Float64("price", price).
		Str("orderID", resp.Result.OrderID).
		Msg("order placed")

	return resp.Result.OrderID, nil
}

// CancelOrder 撤销订单
func (c *FuturesOrderClient) CancelOrder(ctx context.Context, symbol string, orderId string) error {
	payload := map[string]interface{}{
		"category": "linear",
		"symbol":   symbol,
		"orderId":  orderId,
	}

	body, err := c.sendRequest(ctx, http.MethodPost, "/v5/order/cancel", payload)
	if err != nil {
		return fmt.Errorf("cancel order failed: %w", err)
	}

	var resp ApiResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return fmt.Errorf("parse cancel response failed: %w", err)
	}

	if resp.RetCode != 0 {
		return fmt.Errorf("cancel order error: [%d] %s", resp.RetCode, resp.RetMsg)
	}

	log.Info().
		Str("exchange", "BYBIT").
		Str("symbol", symbol).
		Str("orderId", orderId).
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
	params.Set("category", "linear")
	params.Set("symbol", symbol)
	params.Set("orderId", orderId)

	body, err := c.sendRequestWithQuery(ctx, http.MethodGet, "/v5/order/realtime", params)
	if err != nil {
		return nil, fmt.Errorf("get order status failed: %w", err)
	}

	var resp GetOrderResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("parse order status failed: %w", err)
	}

	if resp.RetCode != 0 {
		return nil, fmt.Errorf("get order status error: [%d] %s", resp.RetCode, resp.RetMsg)
	}

	if len(resp.Result.List) == 0 {
		return nil, fmt.Errorf("order not found")
	}

	order := resp.Result.List[0]
	qty, _ := strconv.ParseFloat(order.Qty, 64)
	cumExecQty, _ := strconv.ParseFloat(order.CumExecQty, 64)
	price, _ := strconv.ParseFloat(order.Price, 64)
	avgPrice, _ := strconv.ParseFloat(order.AvgPrice, 64)

	status := &OrderStatus{
		OrderID:          orderId,
		Symbol:           symbol,
		Side:             order.Side,
		Quantity:         qty,
		ExecutedQuantity: cumExecQty,
		Price:            price,
		AvgExecutedPrice: avgPrice,
		Status:           order.OrderStatus,
		CreatedAt:        order.CreatedTime,
		UpdatedAt:        order.UpdatedTime,
	}

	return status, nil
}

// GetFundingRate 获取融资费率
func (c *FuturesOrderClient) GetFundingRate(ctx context.Context, symbol string) (float64, error) {
	params := url.Values{}
	params.Set("category", "linear")
	params.Set("symbol", symbol)

	body, err := c.sendRequestWithQuery(ctx, http.MethodGet, "/v5/market/funding/history", params)
	if err != nil {
		return 0, fmt.Errorf("get funding rate failed: %w", err)
	}

	var resp FundingRateListResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return 0, fmt.Errorf("parse funding rate failed: %w", err)
	}

	if resp.RetCode != 0 || len(resp.Result.List) == 0 {
		return 0, fmt.Errorf("get funding rate error: [%d] %s", resp.RetCode, resp.RetMsg)
	}

	rate, _ := strconv.ParseFloat(resp.Result.List[0].FundingRate, 64)
	return rate, nil
}

// GetAccount 查询账户信息
func (c *FuturesOrderClient) GetAccount(ctx context.Context) (*AccountInfo, error) {
	params := url.Values{}
	params.Set("accountType", "UNIFIED")

	body, err := c.sendRequestWithQuery(ctx, http.MethodGet, "/v5/account/wallet-balance", params)
	if err != nil {
		return nil, fmt.Errorf("get account failed: %w", err)
	}

	var resp AccountResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("parse account response failed: %w", err)
	}

	if resp.RetCode != 0 {
		return nil, fmt.Errorf("get account error: [%d] %s", resp.RetCode, resp.RetMsg)
	}

	account := &AccountInfo{
		CanTrade:    true,
		CanDeposit:  true,
		CanWithdraw: true,
		Positions:   make([]*Position, 0),
	}

	// 解析钱包信息
	if len(resp.Result.List) > 0 {
		wallet := resp.Result.List[0]
		totalWalletBalance, _ := strconv.ParseFloat(wallet.TotalWalletBalance, 64)
		totalMarginBalance, _ := strconv.ParseFloat(wallet.TotalMarginBalance, 64)
		totalAvailableBalance, _ := strconv.ParseFloat(wallet.TotalAvailableBalance, 64)

		account.TotalWalletBalance = totalWalletBalance
		account.AvailableBalance = totalAvailableBalance
		account.TotalMarginRequired = totalMarginBalance - totalAvailableBalance
	}

	// 查询持仓信息
	positions, _ := c.GetOpenPositions(ctx)
	account.Positions = positions

	return account, nil
}

// GetOpenPositions 查询所有开仓持仓
func (c *FuturesOrderClient) GetOpenPositions(ctx context.Context) ([]*Position, error) {
	params := url.Values{}
	params.Set("category", "linear")

	body, err := c.sendRequestWithQuery(ctx, http.MethodGet, "/v5/position/list", params)
	if err != nil {
		return nil, fmt.Errorf("get positions failed: %w", err)
	}

	var resp PositionListResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("parse positions response failed: %w", err)
	}

	if resp.RetCode != 0 {
		return nil, fmt.Errorf("get positions error: [%d] %s", resp.RetCode, resp.RetMsg)
	}

	var positions []*Position
	for _, posDetail := range resp.Result.List {
		// 只返回有仓位的持仓
		if posDetail.Size == "0" {
			continue
		}

		size, _ := strconv.ParseFloat(posDetail.Size, 64)
		entryPrice, _ := strconv.ParseFloat(posDetail.AvgPrice, 64)
		markPrice, _ := strconv.ParseFloat(posDetail.MarkPrice, 64)
		unPnL, _ := strconv.ParseFloat(posDetail.UnrealisedPnl, 64)

		positions = append(positions, &Position{
			Symbol:           posDetail.Symbol,
			PositionAmount:   size,
			EntryPrice:       entryPrice,
			MarkPrice:        markPrice,
			UnrealizedProfit: unPnL,
			Leverage:         posDetail.Leverage,
			IsAutoAddMargin:  posDetail.AutoAddMargin == "1",
		})
	}

	return positions, nil
}

// sendRequest 发送 POST 请求
func (c *FuturesOrderClient) sendRequest(
	ctx context.Context,
	method string,
	path string,
	payload interface{},
) ([]byte, error) {
	// 序列化 payload
	jsonBody, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	// 构建请求
	reqURL := c.baseURL + path
	req, err := http.NewRequestWithContext(ctx, method, reqURL, strings.NewReader(string(jsonBody)))
	if err != nil {
		return nil, err
	}

	// 添加签名
	timestamp := strconv.FormatInt(time.Now().UnixMilli(), 10)
	signature := c.sign(timestamp, string(jsonBody))

	req.Header.Set("X-BAPI-API-KEY", c.ApiKey)
	req.Header.Set("X-BAPI-TIMESTAMP", timestamp)
	req.Header.Set("X-BAPI-SIGN", signature)
	req.Header.Set("Content-Type", "application/json")

	// 执行请求
	resp, err := c.HttpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("http %d: %s", resp.StatusCode, string(body))
	}

	return body, nil
}

// sendRequestWithQuery 发送 GET 请求
func (c *FuturesOrderClient) sendRequestWithQuery(
	ctx context.Context,
	method string,
	path string,
	params url.Values,
) ([]byte, error) {
	reqURL := c.baseURL + path + "?" + params.Encode()
	req, err := http.NewRequestWithContext(ctx, method, reqURL, nil)
	if err != nil {
		return nil, err
	}

	// 添加签名
	timestamp := strconv.FormatInt(time.Now().UnixMilli(), 10)
	signature := c.sign(timestamp, params.Encode())

	req.Header.Set("X-BAPI-API-KEY", c.ApiKey)
	req.Header.Set("X-BAPI-TIMESTAMP", timestamp)
	req.Header.Set("X-BAPI-SIGN", signature)

	// 执行请求
	resp, err := c.HttpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("http %d: %s", resp.StatusCode, string(body))
	}

	return body, nil
}

// sign 生成签名
func (c *FuturesOrderClient) sign(timestamp, message string) string {
	h := hmac.New(sha256.New, []byte(c.ApiSecret))
	h.Write([]byte(timestamp + message))
	return hex.EncodeToString(h.Sum(nil))
}

// ===== Response Models =====

// PlaceOrderResponse 下单响应
type PlaceOrderResponse struct {
	RetCode int    `json:"retCode"`
	RetMsg  string `json:"retMsg"`
	Result  struct {
		OrderID     string `json:"orderId"`
		OrderLinkID string `json:"orderLinkId"`
		OrderStatus string `json:"orderStatus"`
		Symbol      string `json:"symbol"`
		Side        string `json:"side"`
		OrderType   string `json:"orderType"`
		Qty         string `json:"qty"`
		Price       string `json:"price"`
		TimeInForce string `json:"timeInForce"`
		CreatedTime string `json:"createdTime"`
	} `json:"result"`
}

// GetOrderResponse 查询订单响应
type GetOrderResponse struct {
	RetCode int    `json:"retCode"`
	RetMsg  string `json:"retMsg"`
	Result  struct {
		List []struct {
			OrderID     string `json:"orderId"`
			OrderLinkID string `json:"orderLinkId"`
			Symbol      string `json:"symbol"`
			Side        string `json:"side"`
			OrderType   string `json:"orderType"`
			OrderStatus string `json:"orderStatus"`
			Qty         string `json:"qty"`
			Price       string `json:"price"`
			AvgPrice    string `json:"avgPrice"`
			CumExecQty  string `json:"cumExecQty"`
			CreatedTime int64  `json:"createdTime,string"`
			UpdatedTime int64  `json:"updatedTime,string"`
			TimeInForce string `json:"timeInForce"`
		} `json:"list"`
	} `json:"result"`
}

// FundingRateListResponse 融资费率响应
type FundingRateListResponse struct {
	RetCode int    `json:"retCode"`
	RetMsg  string `json:"retMsg"`
	Result  struct {
		List []struct {
			Symbol      string `json:"symbol"`
			FundingRate string `json:"fundingRate"`
			FundingTime string `json:"fundingTime"`
		} `json:"list"`
	} `json:"result"`
}

// AccountResponse 账户信息响应
type AccountResponse struct {
	RetCode int    `json:"retCode"`
	RetMsg  string `json:"retMsg"`
	Result  struct {
		List []struct {
			TotalWalletBalance    string `json:"totalWalletBalance"`
			TotalMarginBalance    string `json:"totalMarginBalance"`
			TotalAvailableBalance string `json:"totalAvailableBalance"`
			TotalPerpUPL          string `json:"totalPerpUPL"`
			TotalInitialMargin    string `json:"totalInitialMargin"`
			TotalMaintMargin      string `json:"totalMaintMargin"`
		} `json:"list"`
	} `json:"result"`
}

// PositionListResponse 持仓列表响应
type PositionListResponse struct {
	RetCode int    `json:"retCode"`
	RetMsg  string `json:"retMsg"`
	Result  struct {
		List []struct {
			Symbol        string `json:"symbol"`
			Leverage      string `json:"leverage"`
			AvgPrice      string `json:"avgPrice"`
			LiqPrice      string `json:"liqPrice"`
			MarkPrice     string `json:"markPrice"`
			Size          string `json:"size"`
			PositionIM    string `json:"positionIM"`
			PositionMM    string `json:"positionMM"`
			Side          string `json:"side"`
			UnrealisedPnl string `json:"unrealisedPnl"`
			AutoAddMargin string `json:"autoAddMargin"`
			CreatedTime   string `json:"createdTime"`
			UpdatedTime   string `json:"updatedTime"`
		} `json:"list"`
	} `json:"result"`
}

// ApiResponse 通用 API 响应
type ApiResponse struct {
	RetCode int         `json:"retCode"`
	RetMsg  string      `json:"retMsg"`
	Result  interface{} `json:"result"`
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
	CanDeposit            bool
	CanTrade              bool
	CanWithdraw           bool
	TotalWalletBalance    float64 // 总钱包余额
	TotalUnrealizedProfit float64 // 总未实现盈亏
	TotalMarginRequired   float64 // 总所需保证金
	AvailableBalance      float64 // 可用余额
	Positions             []*Position
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
