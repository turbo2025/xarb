package monitor

import "xarb/internal/application/port"

type Repository = port.Repository

// for repos needing Close()
type RepositoryCloser interface {
	port.Repository
	Close() error
}
