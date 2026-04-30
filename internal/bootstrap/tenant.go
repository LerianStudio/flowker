// Copyright (c) 2026 Lerian Studio. All rights reserved.
// Use of this source code is governed by the Elastic License 2.0
// that can be found in the LICENSE file.

package bootstrap

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/LerianStudio/flowker/pkg/constant"
	libLog "github.com/LerianStudio/lib-commons/v5/commons/log"
	"github.com/LerianStudio/lib-commons/v5/commons/tenant-manager/client"
	"github.com/LerianStudio/lib-commons/v5/commons/tenant-manager/event"
	"github.com/LerianStudio/lib-commons/v5/commons/tenant-manager/middleware"
	tmmongo "github.com/LerianStudio/lib-commons/v5/commons/tenant-manager/mongo"
	tmpostgres "github.com/LerianStudio/lib-commons/v5/commons/tenant-manager/postgres"
	tmredis "github.com/LerianStudio/lib-commons/v5/commons/tenant-manager/redis"
	"github.com/gofiber/fiber/v2"
	"github.com/redis/go-redis/v9"
)

// TenantInfrastructure holds all multi-tenant managers and middleware.
// When MULTI_TENANT_ENABLED=true, this struct provides per-tenant database
// connection resolution via lib-commons/v5 tenant-manager sub-packages.
type TenantInfrastructure struct {
	// Client is the Tenant Manager HTTP client with circuit breaker.
	Client *client.Client

	// MongoManager manages per-tenant MongoDB connections (primary data).
	MongoManager *tmmongo.Manager

	// PostgresManager manages per-tenant PostgreSQL connections (audit trail).
	PostgresManager *tmpostgres.Manager

	// Middleware is the Fiber handler that extracts tenantId from JWT and resolves
	// tenant-specific database connections.
	Middleware fiber.Handler

	// EventListener subscribes to Redis Pub/Sub for tenant lifecycle events.
	// May be nil if Redis is not configured (MULTI_TENANT_REDIS_HOST not set).
	EventListener *event.TenantEventListener

	// RedisClient is the underlying Redis client for Pub/Sub.
	// May be nil if Redis is not configured.
	RedisClient redis.UniversalClient
}

// NewTenantInfrastructure creates all multi-tenant components when MULTI_TENANT_ENABLED=true.
// Returns nil when multi-tenant mode is disabled (single-tenant passthrough).
//
// Components created:
//   - Tenant Manager HTTP client with circuit breaker and service API key authentication
//   - MongoDB Manager for per-tenant primary database connections
//   - PostgreSQL Manager for per-tenant audit database connections
//   - TenantMiddleware with WithPG/WithMB for single-module services
//   - Optional: Redis Pub/Sub client and event listener for tenant lifecycle events
func NewTenantInfrastructure(ctx context.Context, cfg *Config, logger libLog.Logger) (*TenantInfrastructure, error) {
	// Single-tenant mode: return nil (middleware will be nil, passthrough)
	if !cfg.MultiTenantEnabled || cfg.MultiTenantURL == "" {
		return nil, nil
	}

	// Validate required configuration
	if cfg.MultiTenantServiceAPIKey == "" {
		return nil, errors.New("MULTI_TENANT_SERVICE_API_KEY is required when MULTI_TENANT_ENABLED=true")
	}

	// Create Tenant Manager HTTP client with circuit breaker and service API key
	tmClient, err := client.NewClient(
		cfg.MultiTenantURL,
		logger,
		client.WithTimeout(time.Duration(cfg.MultiTenantTimeout)*time.Second),
		client.WithCircuitBreaker(
			cfg.MultiTenantCircuitBreakerThreshold,
			time.Duration(cfg.MultiTenantCircuitBreakerTimeoutSec)*time.Second,
		),
		client.WithServiceAPIKey(cfg.MultiTenantServiceAPIKey),
		client.WithCacheTTL(time.Duration(cfg.MultiTenantCacheTTLSec)*time.Second),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create tenant manager client: %w", err)
	}

	// Create MongoDB Manager for per-tenant primary databases
	mongoMgr := tmmongo.NewManager(
		tmClient,
		constant.ApplicationName,
		tmmongo.WithModule(constant.ModuleManager),
		tmmongo.WithLogger(logger),
		tmmongo.WithMaxTenantPools(cfg.MultiTenantMaxTenantPools),
		tmmongo.WithIdleTimeout(time.Duration(cfg.MultiTenantIdleTimeoutSec)*time.Second),
	)

	// Create PostgreSQL Manager for per-tenant audit databases
	pgMgr := tmpostgres.NewManager(
		tmClient,
		constant.ApplicationName,
		tmpostgres.WithModule(constant.ModuleManager),
		tmpostgres.WithLogger(logger),
		tmpostgres.WithMaxTenantPools(cfg.MultiTenantMaxTenantPools),
		tmpostgres.WithIdleTimeout(time.Duration(cfg.MultiTenantIdleTimeoutSec)*time.Second),
		tmpostgres.WithConnectionsCheckInterval(time.Duration(cfg.MultiTenantConnectionsCheckIntervalSec)*time.Second),
	)

	// Create TenantMiddleware with both managers (single-module, no module name argument)
	// Single-module services use unnamed WithPG/WithMB for backward-compatible context keys.
	tenantMW := middleware.NewTenantMiddleware(
		middleware.WithPG(pgMgr),
		middleware.WithMB(mongoMgr),
	)

	ti := &TenantInfrastructure{
		Client:          tmClient,
		MongoManager:    mongoMgr,
		PostgresManager: pgMgr,
		Middleware:      tenantMW.WithTenantDB,
	}

	// Optional: Create Redis client and event listener for tenant lifecycle events
	if cfg.MultiTenantRedisHost != "" {
		redisClient, eventListener, err := createRedisEventListener(ctx, cfg, logger, mongoMgr, pgMgr)
		if err != nil {
			// Log warning but continue - service can operate without event-driven updates
			logger.Log(ctx, libLog.LevelWarn, "Failed to create Redis event listener, operating without event-driven tenant discovery",
				libLog.String("error.message", err.Error()),
			)
		} else {
			ti.RedisClient = redisClient
			ti.EventListener = eventListener

			// Start listening for tenant events in background
			if err := eventListener.Start(ctx); err != nil {
				logger.Log(ctx, libLog.LevelWarn, "Failed to start tenant event listener",
					libLog.String("error.message", err.Error()),
				)
			}
		}
	}

	logger.Log(ctx, libLog.LevelInfo, "Multi-tenant infrastructure initialized",
		libLog.String("tenant_manager_url", cfg.MultiTenantURL),
		libLog.Int("max_tenant_pools", cfg.MultiTenantMaxTenantPools),
		libLog.Bool("redis_enabled", cfg.MultiTenantRedisHost != ""),
	)

	return ti, nil
}

// createRedisEventListener creates the Redis client and event listener for tenant lifecycle events.
func createRedisEventListener(
	ctx context.Context,
	cfg *Config,
	logger libLog.Logger,
	mongoMgr *tmmongo.Manager,
	pgMgr *tmpostgres.Manager,
) (redis.UniversalClient, *event.TenantEventListener, error) {
	redisClient, err := tmredis.NewTenantPubSubRedisClient(ctx, tmredis.TenantPubSubRedisConfig{
		Host:     cfg.MultiTenantRedisHost,
		Port:     cfg.MultiTenantRedisPort,
		Password: cfg.MultiTenantRedisPassword,
		TLS:      cfg.MultiTenantRedisTLS,
	})
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create Redis client: %w", err)
	}

	// Create event handler that invalidates cached connections on tenant lifecycle events
	handler := createTenantEventHandler(logger, mongoMgr, pgMgr)

	eventListener, err := event.NewTenantEventListener(
		redisClient,
		handler,
		event.WithListenerLogger(logger),
		event.WithService(constant.ApplicationName),
	)
	if err != nil {
		_ = redisClient.Close()
		return nil, nil, fmt.Errorf("failed to create event listener: %w", err)
	}

	return redisClient, eventListener, nil
}

// createTenantEventHandler returns an event handler that invalidates cached connections
// when tenant lifecycle events are received (suspend, purge, etc.).
func createTenantEventHandler(
	logger libLog.Logger,
	mongoMgr *tmmongo.Manager,
	pgMgr *tmpostgres.Manager,
) event.EventHandler {
	return func(ctx context.Context, evt event.TenantLifecycleEvent) error {
		// On tenant suspend/purge events, close cached connections
		switch evt.EventType {
		case event.EventTenantSuspended, event.EventTenantServiceSuspended, event.EventTenantServicePurged, event.EventTenantDeleted:
			logger.Log(ctx, libLog.LevelInfo, "Tenant lifecycle event received, closing cached connections",
				libLog.String("tenant_id", evt.TenantID),
				libLog.String("event_type", evt.EventType),
			)

			// Close MongoDB connection for this tenant
			if mongoMgr != nil {
				if err := mongoMgr.CloseConnection(ctx, evt.TenantID); err != nil {
					logger.Log(ctx, libLog.LevelWarn, "Failed to close MongoDB connection for tenant",
						libLog.String("tenant_id", evt.TenantID),
						libLog.String("error.message", err.Error()),
					)
				}
			}

			// Close PostgreSQL connection for this tenant
			if pgMgr != nil {
				if err := pgMgr.CloseConnection(ctx, evt.TenantID); err != nil {
					logger.Log(ctx, libLog.LevelWarn, "Failed to close PostgreSQL connection for tenant",
						libLog.String("tenant_id", evt.TenantID),
						libLog.String("error.message", err.Error()),
					)
				}
			}

		case event.EventTenantActivated, event.EventTenantCreated, event.EventTenantServiceAssociated:
			// On activate/create, connections will be established on first request
			logger.Log(ctx, libLog.LevelDebug, "Tenant lifecycle event received",
				libLog.String("tenant_id", evt.TenantID),
				libLog.String("event_type", evt.EventType),
			)
		}

		return nil
	}
}

// Close gracefully shuts down all tenant infrastructure components.
// Safe to call on nil receiver (single-tenant mode).
func (ti *TenantInfrastructure) Close(ctx context.Context) error {
	if ti == nil {
		return nil
	}

	var errs []error

	// Stop event listener first (prevents new events during shutdown)
	if ti.EventListener != nil {
		if err := ti.EventListener.Stop(); err != nil {
			errs = append(errs, fmt.Errorf("failed to stop event listener: %w", err))
		}
	}

	// Close Redis client
	if ti.RedisClient != nil {
		if err := ti.RedisClient.Close(); err != nil {
			errs = append(errs, fmt.Errorf("failed to close Redis client: %w", err))
		}
	}

	// Close MongoDB manager (closes all tenant connections)
	if ti.MongoManager != nil {
		if err := ti.MongoManager.Close(ctx); err != nil {
			errs = append(errs, fmt.Errorf("failed to close MongoDB manager: %w", err))
		}
	}

	// Close PostgreSQL manager (closes all tenant connections)
	if ti.PostgresManager != nil {
		if err := ti.PostgresManager.Close(ctx); err != nil {
			errs = append(errs, fmt.Errorf("failed to close PostgreSQL manager: %w", err))
		}
	}

	// Close Tenant Manager HTTP client
	if ti.Client != nil {
		if err := ti.Client.Close(); err != nil {
			errs = append(errs, fmt.Errorf("failed to close tenant manager client: %w", err))
		}
	}

	return errors.Join(errs...)
}
