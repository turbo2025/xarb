package monitor

import (
	"fmt"
	"sort"
	"strings"

	dsvc "xarb/internal/domain/service"
)

const (
	ansiReset    = "\033[0m"
	ansiRed      = "\033[31m"
	ansiGreen    = "\033[32m"
	ansiYellow   = "\033[33m"
	ansiDim      = "\033[2m"
	ansiClearEOL = "\033[K"
)

func colorize(s, c string) string { return c + s + ansiReset }

type Formatter struct {
	DeltaThreshold float64
	Exchanges      []string // 要显示的交易所列表（如 ["BINANCE", "BYBIT", "OKX"]），默认为所有交易所
}

func NewFormatter(threshold float64) *Formatter {
	return &Formatter{DeltaThreshold: threshold}
}

// NewFormatterWithExchanges 创建 Formatter，指定要显示的交易所
func NewFormatterWithExchanges(threshold float64, exchanges []string) *Formatter {
	return &Formatter{
		DeltaThreshold: threshold,
		Exchanges:      exchanges,
	}
}

type RenderMode int

const (
	RenderLive RenderMode = iota
	RenderSnapshot
)

func (f *Formatter) Render(st *State, mode RenderMode) string {
	snap := st.Snapshot()
	symbols := st.Symbols()

	var sb strings.Builder
	if mode == RenderLive {
		sb.WriteString("\r")
	}

	sb.WriteString(colorize("[XARB] ", ansiDim))

	for i, sym := range symbols {
		if i > 0 {
			sb.WriteString(colorize("  ||  ", ansiDim))
		}
		ss := snap[sym]

		// 获取该交易对的所有交易所
		exchanges := f.getExchangesToDisplay(ss)
		if len(exchanges) == 0 {
			sb.WriteString(sym)
			sb.WriteString(" ")
			sb.WriteString(colorize("--", ansiYellow))
			continue
		}

		// 显示所有交易所的价格
		sb.WriteString(sym)
		sb.WriteString(" ")
		for j, ex := range exchanges {
			if j > 0 {
				sb.WriteString("/")
			}
			ps := ss.exchanges[ex]
			if ps == nil || !ps.seen {
				sb.WriteString(colorize(ex[:1]+":--", ansiYellow))
				continue
			}

			priceStr := "--"
			if ps.str != "" {
				priceStr = ps.str
			}
			col := ansiYellow
			if ps.parse {
				switch ps.dir {
				case DirUp:
					col = ansiGreen
				case DirDown:
					col = ansiRed
				default:
					col = ansiYellow
				}
			}
			sb.WriteString(colorize(ex[:1]+":"+priceStr, col))
		}

		// 计算最大价差
		if len(exchanges) >= 2 {
			sb.WriteString(" ")
			maxDelta, maxExPair := f.getMaxDelta(ss, exchanges)
			deltaStr := "Δ=--"
			dCol := ansiYellow
			if maxDelta != nil {
				deltaStr = fmt.Sprintf("Δ=%+.2f(%s)", maxDelta.value, maxDelta.pair)
				switch dsvc.DeltaColor(maxDelta.value, f.DeltaThreshold) {
				case +1:
					dCol = ansiGreen
				case -1:
					dCol = ansiRed
				default:
					dCol = ansiYellow
				}
			}
			_ = maxExPair // 暂时未使用
			sb.WriteString(colorize(deltaStr, dCol))
		}
	}

	if mode == RenderLive {
		sb.WriteString(ansiClearEOL)
	}
	return sb.String()
}

// getExchangesToDisplay 获取要显示的交易所列表
func (f *Formatter) getExchangesToDisplay(ss symState) []string {
	if len(f.Exchanges) > 0 {
		// 使用指定的交易所，但只显示存在的
		var result []string
		for _, ex := range f.Exchanges {
			if _, ok := ss.exchanges[ex]; ok {
				result = append(result, ex)
			}
		}
		return result
	}

	// 如果没有指定，返回所有交易所（按字母序，确保一致性）
	result := make([]string, 0, len(ss.exchanges))
	for ex := range ss.exchanges {
		result = append(result, ex)
	}
	// 排序以确保显示顺序一致
	sort.Strings(result)
	return result
}

// deltaInfo 表示两个交易所间的价差信息
type deltaInfo struct {
	value float64
	pair  string // "EX1-EX2"
}

// getMaxDelta 找出两两比较中价差绝对值最大的
func (f *Formatter) getMaxDelta(ss symState, exchanges []string) (*deltaInfo, string) {
	if len(exchanges) < 2 {
		return nil, ""
	}

	var maxDelta *deltaInfo
	var maxAbsDelta float64

	// 两两比较
	for i := 0; i < len(exchanges)-1; i++ {
		for j := i + 1; j < len(exchanges); j++ {
			ps1 := ss.exchanges[exchanges[i]]
			ps2 := ss.exchanges[exchanges[j]]

			if ps1 == nil || ps2 == nil || !ps1.parse || !ps2.parse || !ps1.has || !ps2.has {
				continue
			}

			d := ps2.num - ps1.num
			absDelta := d
			if absDelta < 0 {
				absDelta = -absDelta
			}

			if absDelta > maxAbsDelta {
				maxAbsDelta = absDelta
				maxDelta = &deltaInfo{
					value: d,
					pair:  exchanges[i][:1] + "-" + exchanges[j][:1],
				}
			}
		}
	}

	return maxDelta, ""
}
