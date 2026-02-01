package presentation

import (
	"fmt"
	"strings"

	"binance-ws/domain"
)

// ANSI color codes
const (
	ansiReset    = "\033[0m"
	ansiRed      = "\033[31m"
	ansiGreen    = "\033[32m"
	ansiYellow   = "\033[33m"
	ansiDim      = "\033[2m"
	ansiClearEOL = "\033[K"
)

// Colorize applies ANSI color to a string
func Colorize(s, color string) string {
	return color + s + ansiReset
}

// Renderer handles rendering the board to terminal
type Renderer struct {
	threshold float64
}

// NewRenderer creates a new Renderer
func NewRenderer(threshold float64) *Renderer {
	return &Renderer{threshold: threshold}
}

// RenderLine renders a single board line
func (r *Renderer) RenderLine(symbols []string, snapshot map[string]*domain.SymbolState, live bool) string {
	var sb strings.Builder

	if live {
		sb.WriteString("\r")
	}

	sb.WriteString(Colorize("[XARB] ", ansiDim))

	for i, sym := range symbols {
		if i > 0 {
			sb.WriteString(Colorize("  ||  ", ansiDim))
		}

		st := snapshot[sym]
		if st == nil {
			continue
		}

		// Binance price
		bPrice := "--"
		if st.Binance.IsSeen && st.Binance.String != "" {
			bPrice = st.Binance.String
		}
		bCol := ansiYellow
		if st.Binance.IsParsed {
			switch st.Binance.Direction {
			case domain.DirectionUp:
				bCol = ansiGreen
			case domain.DirectionDown:
				bCol = ansiRed
			default:
				bCol = ansiYellow
			}
		}

		// Bybit price
		yPrice := "--"
		if st.Bybit.IsSeen && st.Bybit.String != "" {
			yPrice = st.Bybit.String
		}
		yCol := ansiYellow
		if st.Bybit.IsParsed {
			switch st.Bybit.Direction {
			case domain.DirectionUp:
				yCol = ansiGreen
			case domain.DirectionDown:
				yCol = ansiRed
			default:
				yCol = ansiYellow
			}
		}

		// Delta: Bybit - Binance
		deltaStr := "Δ=--"
		dCol := ansiYellow
		if st.Binance.IsParsed && st.Bybit.IsParsed && st.Binance.HasValue && st.Bybit.HasValue {
			d := st.Bybit.Number - st.Binance.Number
			deltaStr = fmt.Sprintf("Δ=%+.2f", d)

			if d >= r.threshold {
				dCol = ansiGreen
			} else if d <= -r.threshold {
				dCol = ansiRed
			} else {
				dCol = ansiYellow
			}
		}

		sb.WriteString(sym)
		sb.WriteString(" ")
		sb.WriteString(Colorize("B:"+bPrice, bCol))
		sb.WriteString(" ")
		sb.WriteString(Colorize("Y:"+yPrice, yCol))
		sb.WriteString(" ")
		sb.WriteString(Colorize(deltaStr, dCol))
	}

	if live {
		sb.WriteString(ansiClearEOL)
	}

	return sb.String()
}
