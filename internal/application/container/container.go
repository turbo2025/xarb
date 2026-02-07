package container

import (
	"xarb/internal/application/port"
	"xarb/internal/application/service"
)

type Container struct {
	repo port.Repository

	priceService    *service.PriceService
	positionService *service.PositionService
	snapshotService *service.SnapshotService
	signalService   *service.SignalService
}

func New(repo port.Repository) *Container {
	return &Container{
		repo: repo,
	}
}

func (c *Container) Repository() port.Repository {
	return c.repo
}

func (c *Container) PriceService() *service.PriceService {
	if c.priceService == nil {
		c.priceService = service.NewPriceService(c.repo)
	}
	return c.priceService
}

func (c *Container) PositionService() *service.PositionService {
	if c.positionService == nil {
		c.positionService = service.NewPositionService(c.repo)
	}
	return c.positionService
}

func (c *Container) SnapshotService() *service.SnapshotService {
	if c.snapshotService == nil {
		c.snapshotService = service.NewSnapshotService(c.repo)
	}
	return c.snapshotService
}

func (c *Container) SignalService() *service.SignalService {
	if c.signalService == nil {
		c.signalService = service.NewSignalService(c.repo)
	}
	return c.signalService
}

func (c *Container) Close() error {
	return c.repo.Close()
}
