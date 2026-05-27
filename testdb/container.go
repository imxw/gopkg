package testdb

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
)

// ContainerConfig configures the test PostgreSQL container.
type ContainerConfig struct {
	Image    string
	Database string
	Username string
	Password string
}

// DefaultContainerConfig returns sensible defaults for a test Postgres container.
func DefaultContainerConfig() ContainerConfig {
	return ContainerConfig{
		Image:    "postgres:16-alpine",
		Database: "testdb",
		Username: "test",
		Password: "test",
	}
}

// StartContainer launches a PostgreSQL test container and returns a connection
// pool configured for test use (max 5 conns, 30s lifetime).
func StartContainer(ctx context.Context, cfg ContainerConfig) (*pgxpool.Pool, func(), error) {
	c, err := postgres.Run(ctx, cfg.Image,
		postgres.WithDatabase(cfg.Database),
		postgres.WithUsername(cfg.Username),
		postgres.WithPassword(cfg.Password),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(30*time.Second),
		),
	)
	if err != nil {
		return nil, nil, fmt.Errorf("start test container: %w", err)
	}

	host, err := c.Host(ctx)
	if err != nil {
		return nil, nil, fmt.Errorf("get container host: %w", err)
	}
	port, err := c.MappedPort(ctx, "5432")
	if err != nil {
		return nil, nil, fmt.Errorf("get container port: %w", err)
	}

	dsn := fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable",
		cfg.Username, cfg.Password, host, port.Port(), cfg.Database,
	)

	poolCfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, nil, fmt.Errorf("parse dsn: %w", err)
	}
	poolCfg.MaxConns = 5
	poolCfg.MaxConnLifetime = 30 * time.Second

	pool, err := pgxpool.NewWithConfig(ctx, poolCfg)
	if err != nil {
		return nil, nil, fmt.Errorf("create pool: %w", err)
	}

	cleanup := func() {
		pool.Close()
		_ = c.Terminate(ctx)
	}
	return pool, cleanup, nil
}
