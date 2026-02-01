package monitor

import (
	"fmt"
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
}

func NewFormatter(threshold float64) *Formatter {
	return &Formatter{DeltaThreshold: threshold}
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

		// binance
		bp := "--"
		if ss.b.seen && ss.b.str != "" {
			bp = ss.b.str
		}
		bCol := ansiYellow
		if ss.b.parse {
			switch ss.b.dir {
			case DirUp:
				bCol = ansiGreen
			case DirDown:
				bCol = ansiRed
			default:
				bCol = ansiYellow
			}
		}

		// bybit
		yp := "--"
		if ss.y.seen && ss.y.str != "" {
			yp = ss.y.str
		}
		yCol := ansiYellow
		if ss.y.parse {
			switch ss.y.dir {
			case DirUp:
				yCol = ansiGreen
			case DirDown:
				yCol = ansiRed
			default:
				yCol = ansiYellow
			}
		}

		// delta (bybit - binance)
		deltaStr := "Δ=--"
		dCol := ansiYellow
		if ss.b.parse && ss.y.parse && ss.b.has && ss.y.has {
			d := dsvc.Delta(ss.y.num, ss.b.num)
			deltaStr = fmt.Sprintf("Δ=%+.2f", d)
			switch dsvc.DeltaColor(d, f.DeltaThreshold) {
			case +1:
				dCol = ansiGreen
			case -1:
				dCol = ansiRed
			default:
				dCol = ansiYellow
			}
		}

		sb.WriteString(sym)
		sb.WriteString(" ")
		sb.WriteString(colorize("B:"+bp, bCol))
		sb.WriteString(" ")
		sb.WriteString(colorize("Y:"+yp, yCol))
		sb.WriteString(" ")
		sb.WriteString(colorize(deltaStr, dCol))
	}

	if mode == RenderLive {
		sb.WriteString(ansiClearEOL)
	}
	return sb.String()
}
