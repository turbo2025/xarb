package console

import (
	"fmt"
	"time"

	"xarb/internal/application/port"
)

type Sink struct{}

func NewSink() port.Sink { return &Sink{} }

func (s *Sink) WriteLive(line string) error {
	fmt.Print(line) // no newline
	return nil
}

// A 方案：打印快照行后，留一个空行占位；不立刻重画 live，等下一次变化刷新
func (s *Sink) WriteSnapshot(ts time.Time, line string) error {
	fmt.Print("\n")
	fmt.Printf("%s %s\n", ts.Format("2006-01-02 15:04:05"), line)
	fmt.Print("\n")
	return nil
}

func (s *Sink) NewLine() error {
	fmt.Print("\n")
	return nil
}
