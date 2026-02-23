package notification

import (
	"fmt"
	"sync"
	"time"

	"xarb/internal/application/port"
	"xarb/internal/infrastructure/config"

	"github.com/rs/zerolog/log"
)

// Sink 复合 Sink，支持同时输出到 console 和飞书
type Sink struct {
	mu           sync.RWMutex
	consoleSink  port.Sink
	clients      map[string]*Client
	lastSnapshot time.Time
	snapshotBuf  string
}

// NewSink 创建飞书 Sink
func NewSink(consoleSink port.Sink, feishuConfigs []config.FeishuConfig) *Sink {
	s := &Sink{
		consoleSink: consoleSink,
		clients:     make(map[string]*Client),
	}

	// 初始化飞书客户端
	for _, cfg := range feishuConfigs {
		channel := cfg.Channel
		if channel == "" {
			channel = "market" // 默认频道
		}
		s.clients[channel] = NewClient(cfg.Webhook, cfg.Secret, channel)
		log.Info().Str("channel", channel).Msg("feishu client initialized")
	}

	return s
}

// WriteLive 实时更新行（输出到 console）
func (s *Sink) WriteLive(line string) error {
	if s.consoleSink != nil {
		return s.consoleSink.WriteLive(line)
	}
	return nil
}

// WriteSnapshot 快照行（输出到 console，并定时发送到飞书）
func (s *Sink) WriteSnapshot(ts time.Time, line string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// 先写入 console
	if s.consoleSink != nil {
		if err := s.consoleSink.WriteSnapshot(ts, line); err != nil {
			return err
		}
	}

	// 保存快照内容，用于定时推送
	s.lastSnapshot = ts
	s.snapshotBuf = line

	return nil
}

// NewLine 普通换行（仅输出到 console）
func (s *Sink) NewLine() error {
	if s.consoleSink != nil {
		return s.consoleSink.NewLine()
	}
	return nil
}

// SendSignal 发送套利信号到飞书（signal 频道）
func (s *Sink) SendSignal(symbol string, delta float64, payload string) error {
	client, ok := s.clients["signal"]
	if !ok {
		log.Warn().Msg("no signal channel configured in feishu")
		return nil
	}

	title := "🎯 Arbitrage Signal Detected [" + time.Now().Format("2006-01-02 15:04:05") + "]"
	// 清理 payload 中的 ANSI 颜色编码
	cleanPayload := stripANSI(payload)
	content := []string{
		"",
		symbol,
		"Delta: " + formatFloat(delta),
		"",
		"Details:",
		cleanPayload,
	}

	return client.SendPost(title, content)
}

// SendOrder 发送订单通知到飞书（order 频道）
func (s *Sink) SendOrder(orderInfo string) error {
	client, ok := s.clients["order"]
	if !ok {
		log.Warn().Msg("no order channel configured in feishu")
		return nil
	}

	title := "📋 Order Executed [" + time.Now().Format("2006-01-02 15:04:05") + "]"
	content := []string{orderInfo}

	return client.SendPost(title, content)
}

// SendPnL 发送盈亏信息到飞书（pnl 频道）
func (s *Sink) SendPnL(pnlInfo string) error {
	client, ok := s.clients["pnl"]
	if !ok {
		log.Warn().Msg("no pnl channel configured in feishu")
		return nil
	}

	title := "💰 P&L Update [" + time.Now().Format("2006-01-02 15:04:05") + "]"
	content := []string{pnlInfo}

	return client.SendPost(title, content)
}

// SendMarket 发送市场信息到飞书（market 频道）
func (s *Sink) SendMarket(marketInfo string) error {
	client, ok := s.clients["market"]
	if !ok {
		log.Warn().Msg("no market channel configured in feishu")
		return nil
	}

	title := "📈 Market Update [" + time.Now().Format("2006-01-02 15:04:05") + "]"
	content := []string{marketInfo}

	return client.SendPost(title, content)
}

// SendText 发送纯文本消息到指定频道
func (s *Sink) SendText(channel, text string) error {
	client, ok := s.clients[channel]
	if !ok {
		log.Warn().Str("channel", channel).Msg("channel not configured in feishu")
		return nil
	}

	return client.SendText(text)
}

func formatFloat(f float64) string {
	if f == 0 {
		return "0.00%"
	}
	sign := "+"
	if f < 0 {
		sign = ""
	}
	return sign + fmt.Sprintf("%.2f", f) + "%"
}

// stripANSI 移除字符串中的 ANSI 颜色编码
func stripANSI(s string) string {
	// 移除所有 ANSI escape 序列: \033[...m 或 [XXXm
	var result []rune
	inEscape := false

	for _, r := range s {
		if r == '\033' || (inEscape && r == '[') {
			inEscape = true
			continue
		}
		if inEscape && r == 'm' {
			inEscape = false
			continue
		}
		if inEscape {
			continue
		}
		result = append(result, r)
	}

	return string(result)
}
