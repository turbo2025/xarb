package container

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog/log"

	"xarb/internal/infrastructure/config"
	redisrepo "xarb/internal/infrastructure/storage/redis"
	sqliterepo "xarb/internal/infrastructure/storage/sqlite"
)

// Container 包含所有应用依赖
type Container struct {
	cfg         *config.Config
	redisClient *redis.Client
	sqliteRepo  *sqliterepo.Repo
	redisRepo   *redisrepo.Repo
	closeOnce   sync.Once
	closerChain []func() error
}

// New 创建新的容器实例
func New(cfg *config.Config) (*Container, error) {
	c := &Container{
		cfg:         cfg,
		closerChain: make([]func() error, 0),
	}

	// 初始化存储层
	if cfg.Storage.Enabled {
		if err := c.initStorage(); err != nil {
			// 清理已初始化的资源
			_ = c.Close()
			return nil, err
		}
	}

	return c, nil
}

// initStorage 初始化存储层（Redis、SQLite、Postgres）
func (c *Container) initStorage() error {
	// Redis
	if c.cfg.Storage.Redis.Enabled {
		if err := c.initRedis(); err != nil {
			return fmt.Errorf("redis init failed: %w", err)
		}
	}

	// SQLite
	if c.cfg.Storage.SQLite.Enabled {
		if err := c.initSQLite(); err != nil {
			return fmt.Errorf("sqlite init failed: %w", err)
		}
	}

	// Postgres
	// if c.cfg.Storage.Postgres.Enabled {
	// 	if err := c.initPostgres(); err != nil {
	// 		return fmt.Errorf("postgres init failed: %w", err)
	// 	}
	// }

	return nil
}

// initRedis 初始化 Redis 连接
func (c *Container) initRedis() error {
	rdb := redis.NewClient(&redis.Options{
		Addr:     c.cfg.Storage.Redis.Addr,
		Password: c.cfg.Storage.Redis.Password,
		DB:       c.cfg.Storage.Redis.DB,
	})

	// 测试连接
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := rdb.Ping(ctx).Err(); err != nil {
		return fmt.Errorf("redis ping failed: %w", err)
	}

	c.redisClient = rdb
	ttl := time.Duration(c.cfg.Storage.Redis.TTLSeconds) * time.Second

	c.redisRepo = redisrepo.New(
		rdb,
		c.cfg.Storage.Redis.Prefix,
		ttl,
		c.cfg.Storage.Redis.SignalStream,
		c.cfg.Storage.Redis.SignalChannel,
	)

	// 注册关闭回调
	c.closerChain = append(c.closerChain, func() error {
		log.Info().Msg("closing redis connection")
		return rdb.Close()
	})

	log.Info().
		Str("addr", c.cfg.Storage.Redis.Addr).
		Int("db", c.cfg.Storage.Redis.DB).
		Msg("redis initialized")

	return nil
}

// initSQLite 初始化 SQLite 数据库
func (c *Container) initSQLite() error {
	repo, err := sqliterepo.New(c.cfg.Storage.SQLite.Path)
	if err != nil {
		return err
	}

	c.sqliteRepo = repo

	// 注册关闭回调
	c.closerChain = append(c.closerChain, func() error {
		log.Info().Msg("closing sqlite connection")
		return repo.Close()
	})

	log.Info().
		Str("path", c.cfg.Storage.SQLite.Path).
		Msg("sqlite initialized")

	return nil
}

// Config 获取配置
func (c *Container) Config() *config.Config {
	return c.cfg
}

// RedisClient 获取 Redis 客户端
func (c *Container) RedisClient() *redis.Client {
	return c.redisClient
}

// RedisRepo 获取 Redis 仓储
func (c *Container) RedisRepo() *redisrepo.Repo {
	return c.redisRepo
}

// SQLiteRepo 获取 SQLite 仓储
func (c *Container) SQLiteRepo() *sqliterepo.Repo {
	return c.sqliteRepo
}

// SQLiteArbitrageRepo 获取 SQLite 套利仓储
func (c *Container) SQLiteArbitrageRepo() *sqliterepo.ArbitrageRepo {
	if c.sqliteRepo == nil {
		return nil
	}
	return sqliterepo.NewArbitrageRepo(c.sqliteRepo.GetDB())
}

// Close 关闭所有资源（按后进先出顺序）
func (c *Container) Close() error {
	var err error
	c.closeOnce.Do(func() {
		for i := len(c.closerChain) - 1; i >= 0; i-- {
			if e := c.closerChain[i](); e != nil {
				log.Error().Err(e).Msg("error closing resource")
				if err == nil {
					err = e
				}
			}
		}
		log.Info().Msg("container closed")
	})
	return err
}
