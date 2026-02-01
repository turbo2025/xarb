package port

import "time"

type Sink interface {
	// Live line: overwrite last line (no newline)
	WriteLive(line string) error
	// Snapshot line: append a historical line with timestamp, and (A) leave an empty line for future live updates
	WriteSnapshot(ts time.Time, line string) error
	// Normal newline (for logs)
	NewLine() error
}
