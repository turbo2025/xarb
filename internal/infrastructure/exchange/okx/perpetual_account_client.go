package okx

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"xarb/internal/domain/service"

	"github.com/rs/zerolog/log"
)

// PerpetualAccountClient OKX perpetual account query client
type PerpetualAccountClient struct {
	*APIClient
}

// NewPerpetualAccountClient creates perpetual account client
func NewPerpetualAccountClient(client *APIClient) *PerpetualAccountClient {
	return &PerpetualAccountClient{APIClient: client}
}

// perpetualAccountResponse OKX perpetual account info API response structure
type perpetualAccountResponse struct {
	Code string `json:"code"`
	Msg  string `json:"msg"`
	Data []struct {
		AdjEq   string `json:"adjEq"` // 调整后的权益
		Details []struct {
			Ccy           string `json:"ccy"`           // 币种（通常是 USDT）
			Eq            string `json:"eq"`            // 权益
			CashBal       string `json:"cashBal"`       // 现金余额
			UsdEq         string `json:"usdEq"`         // USD 权益值
			AvailEq       string `json:"availEq"`       // 可用权益
			FixedBal      string `json:"fixedBal"`      // 固定余额
			TwapBal       string `json:"twapBal"`       // TWAP 余额
			IsoEq         string `json:"isoEq"`         // 隔离权益
			Mgnratio      string `json:"mgnratio"`      // 保证金率
			Imr           string `json:"imr"`           // 初始保证金率
			Mmr           string `json:"mmr"`           // 维持保证金率
			NotionalUsd   string `json:"notionalUsd"`   // USD 名义价值
			Ordfrz        string `json:"ordfrz"`        // 订单冻结
			UnrealisedPnl string `json:"unrealisedPnl"` // 未实现盈亏
			TotalPnl      string `json:"totalPnl"`      // 总损益
			EqUsd         string `json:"eqUsd"`         // 权益 USD 值
			BaseBal       string `json:"baseBal"`       // 基本余额
			DisEq         string `json:"disEq"`         // 折扣权益
		} `json:"details"`
		Imr                string `json:"imr"`                // 初始保证金总值
		Mmr                string `json:"mmr"`                // 维持保证金总值
		TotalEq            string `json:"totalEq"`            // 总权益
		OrderFrz           string `json:"orderFrz"`           // 订单冻结总额
		TotalMgnratio      string `json:"totalMgnratio"`      // 总保证金率
		TotalInitialMargin string `json:"totalInitialMargin"` // 初始保证金总额 (deprecated, use Imr)
		TotalMaintMargin   string `json:"totalMaintMargin"`   // 维持保证金总额 (deprecated, use Mmr)
	} `json:"data"`
}

// GetAccount gets perpetual account info
func (c *PerpetualAccountClient) GetAccount(ctx context.Context) (*service.AccountInfo, error) {
	resp, err := c.fetchAccount(ctx)
	if err != nil {
		return nil, err
	}

	if resp.Code != "0" {
		return nil, fmt.Errorf("okx api error: %s", resp.Msg)
	}

	if len(resp.Data) == 0 {
		return nil, fmt.Errorf("no account data returned")
	}

	acct := resp.Data[0]

	// 解析关键数值
	totalEq, _ := strconv.ParseFloat(acct.TotalEq, 64)
	totalInitialMargin, _ := strconv.ParseFloat(acct.Imr, 64)
	orderFrz, _ := strconv.ParseFloat(acct.OrderFrz, 64)

	// 计算可用保证金 = 总权益 - 初始保证金 - 挂单冻结
	availMargin := totalEq - totalInitialMargin - orderFrz
	if availMargin < 0 {
		availMargin = 0
	}

	// 计算已用保证金
	usedMargin := totalInitialMargin

	return &service.AccountInfo{
		Exchange:    "OKX",
		TotalMargin: totalEq,
		AvailMargin: availMargin,
		UsedMargin:  usedMargin,
		Positions:   make(map[string]*service.PositionInfo),
		OpenOrders:  make(map[string]*service.OpenOrderInfo),
		UpdatedAt:   time.Now(),
	}, nil
}

// GetMarginInfo 获取保证金信息
func (c *PerpetualAccountClient) GetMarginInfo(ctx context.Context) (map[string]interface{}, error) {
	resp, err := c.fetchAccount(ctx)
	if err != nil {
		return nil, err
	}

	if resp.Code != "0" {
		return nil, fmt.Errorf("okx api error: %s", resp.Msg)
	}

	if len(resp.Data) == 0 {
		return nil, fmt.Errorf("no account data returned")
	}

	acct := resp.Data[0]

	// 解析保证金数据
	imr, _ := strconv.ParseFloat(acct.Imr, 64)
	mmr, _ := strconv.ParseFloat(acct.Mmr, 64)
	totalEq, _ := strconv.ParseFloat(acct.TotalEq, 64)
	totalMgnratio, _ := strconv.ParseFloat(acct.TotalMgnratio, 64)

	result := map[string]interface{}{
		"initialMargin":     imr,
		"maintenanceMargin": mmr,
		"totalEquity":       totalEq,
		"marginRatio":       totalMgnratio,
	}

	// 添加各币种的保证金信息
	details := make([]map[string]interface{}, 0, len(acct.Details))
	for _, detail := range acct.Details {
		eq, _ := strconv.ParseFloat(detail.EqUsd, 64)
		imr, _ := strconv.ParseFloat(detail.Imr, 64)
		mmr, _ := strconv.ParseFloat(detail.Mmr, 64)

		details = append(details, map[string]interface{}{
			"currency":          detail.Ccy,
			"equity":            eq,
			"initialMargin":     imr,
			"maintenanceMargin": mmr,
			"unrealizedPnl":     detail.UnrealisedPnl,
		})
	}

	result["details"] = details
	return result, nil
}

// fetchAccount calls OKX perpetual account API
func (c *PerpetualAccountClient) fetchAccount(ctx context.Context) (*perpetualAccountResponse, error) {
	// OKX API: GET /api/v5/account/balance
	path := "/api/v5/account/balance"

	body, err := c.signedQueryRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch okx perpetual account: %w", err)
	}

	var resp perpetualAccountResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal okx perpetual account: %w", err)
	}

	if resp.Code != "0" {
		log.Warn().Str("msg", resp.Msg).Msg("OKX API returned error")
	}

	return &resp, nil
}
