// Copyright (c) 2026 Lerian Studio. All rights reserved.
// Use of this source code is governed by the Elastic License 2.0
// that can be found in the LICENSE file.

//go:build e2e

package e2e

import (
	"context"
	"fmt"
	"net"
	"net/url"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"

	"github.com/LerianStudio/flowker/internal/bootstrap"
)

var (
	suite      *testSuite
	mongoC     testcontainers.Container
	pgC        testcontainers.Container
	mongoURI   string
	serverAddr string
)

type testSuite struct {
	service *bootstrap.Service
}

func TestMain(m *testing.M) {
	// If E2E_BASE_URL is set, skip bootstrapping — use external server
	if v := os.Getenv("E2E_BASE_URL"); v != "" {
		u, err := url.Parse(v)
		if err != nil || u.Host == "" {
			fmt.Printf("invalid E2E_BASE_URL %q: %v\n", v, err)
			os.Exit(1)
		}
		serverAddr = u.Host
		os.Exit(m.Run())
	}

	ctx := context.Background()

	// Start Mongo container
	uri, c, err := startMongo(ctx)
	if err != nil {
		fmt.Printf("failed to start mongo container: %v\n", err)
		os.Exit(1)
	}
	mongoC = c
	mongoURI = uri

	// Start Postgres container for audit
	pgHost, pgPort, pc, err := startPostgres(ctx)
	if err != nil {
		fmt.Printf("failed to start postgres container: %v\n", err)
		terminate(ctx)
		os.Exit(1)
	}
	pgC = pc

	// Choose free port for app
	port, err := freePort()
	if err != nil {
		fmt.Printf("failed to allocate port: %v\n", err)
		terminate(ctx)
		os.Exit(1)
	}
	serverAddr = fmt.Sprintf("127.0.0.1:%d", port)

	// Set environment
	setEnvForApp(serverAddr, mongoURI, pgHost, pgPort)

	// Start service
	svc, err := bootstrap.InitServers()
	if err != nil {
		fmt.Printf("failed to init servers: %v\n", err)
		terminate(ctx)
		os.Exit(1)
	}
	suite = &testSuite{service: svc}

	// Create indexes before running tests
	dbManager := bootstrap.NewDatabaseManagerWithConfig(&bootstrap.MongoConfig{
		URI:      mongoURI,
		Database: "flowker_e2e",
	})
	if err := dbManager.Connect(ctx); err != nil {
		fmt.Printf("failed to connect to mongo for indexes: %v\n", err)
		terminate(ctx)
		os.Exit(1)
	}
	indexManager := bootstrap.NewIndexManager(dbManager)
	if err := indexManager.CreateIndexes(ctx); err != nil {
		fmt.Printf("failed to create indexes: %v\n", err)
		terminate(ctx)
		os.Exit(1)
	}
	if err := dbManager.Disconnect(ctx); err != nil {
		fmt.Printf("warning: failed to disconnect index manager: %v\n", err)
	}

	go suite.service.Run()

	if err := waitForServer(fmt.Sprintf("http://%s/health", serverAddr), 30*time.Second); err != nil {
		fmt.Printf("server not ready: %v\n", err)
		terminate(ctx)
		os.Exit(1)
	}

	code := m.Run()
	terminate(ctx)
	os.Exit(code)
}

func terminate(ctx context.Context) {
	if mongoC != nil {
		_ = mongoC.Terminate(ctx)
	}
	if pgC != nil {
		_ = pgC.Terminate(ctx)
	}
}

func startMongo(ctx context.Context) (string, testcontainers.Container, error) {
	req := testcontainers.ContainerRequest{
		Image:        "mongo:7.0",
		ExposedPorts: []string{"27017/tcp"},
		WaitingFor:   wait.ForListeningPort("27017/tcp").WithStartupTimeout(60 * time.Second),
	}
	c, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		return "", nil, err
	}
	host, err := c.Host(ctx)
	if err != nil {
		_ = c.Terminate(ctx)
		return "", nil, err
	}
	port, err := c.MappedPort(ctx, "27017/tcp")
	if err != nil {
		_ = c.Terminate(ctx)
		return "", nil, err
	}
	uri := fmt.Sprintf("mongodb://%s:%s", host, port.Port())
	return uri, c, nil
}

func freePort() (int, error) {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return 0, err
	}
	defer l.Close()
	return l.Addr().(*net.TCPAddr).Port, nil
}

func startPostgres(ctx context.Context) (string, string, testcontainers.Container, error) {
	req := testcontainers.ContainerRequest{
		Image:        "postgres:16-alpine",
		ExposedPorts: []string{"5432/tcp"},
		Env: map[string]string{
			"POSTGRES_USER":     "flowker_audit",
			"POSTGRES_PASSWORD": "flowker_audit",
			"POSTGRES_DB":       "flowker_audit",
		},
		WaitingFor: wait.ForListeningPort("5432/tcp").WithStartupTimeout(60 * time.Second),
	}
	c, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		return "", "", nil, err
	}
	host, err := c.Host(ctx)
	if err != nil {
		_ = c.Terminate(ctx)
		return "", "", nil, err
	}
	port, err := c.MappedPort(ctx, "5432/tcp")
	if err != nil {
		_ = c.Terminate(ctx)
		return "", "", nil, err
	}
	return host, port.Port(), c, nil
}

func setEnvForApp(addr, mongoURI, pgHost, pgPort string) {
	os.Setenv("ENV_NAME", "development")
	os.Setenv("SERVER_ADDRESS", addr)
	os.Setenv("LOG_LEVEL", "error")
	os.Setenv("ENABLE_TELEMETRY", "false")
	os.Setenv("OTEL_LIBRARY_NAME", "flowker-test")
	os.Setenv("MONGO_URI", mongoURI)
	os.Setenv("MONGO_DB_NAME", "flowker_e2e")
	os.Setenv("API_KEY_ENABLED", "false")
	os.Setenv("PLUGIN_AUTH_ENABLED", "false")
	os.Setenv("SWAGGER_TITLE", "Flowker API")
	os.Setenv("SWAGGER_DESCRIPTION", "Flowker E2E tests")
	os.Setenv("SWAGGER_VERSION", "1.0.0")
	os.Setenv("SWAGGER_HOST", addr)
	os.Setenv("SWAGGER_BASE_PATH", "/")
	os.Setenv("SWAGGER_LEFT_DELIM", "{{")
	os.Setenv("SWAGGER_RIGHT_DELIM", "}}")
	os.Setenv("SWAGGER_SCHEMES", "http")
	os.Setenv("CORS_ALLOWED_ORIGINS", "*")
	os.Setenv("FAULT_INJECTION_ENABLED", "true")
	os.Setenv("SSRF_ALLOW_PRIVATE", "true")
	os.Setenv("AUDIT_DB_HOST", pgHost)
	os.Setenv("AUDIT_DB_PORT", pgPort)
	os.Setenv("AUDIT_DB_USER", "flowker_audit")
	os.Setenv("AUDIT_DB_PASSWORD", "flowker_audit")
	os.Setenv("AUDIT_DB_NAME", "flowker_audit")
	os.Setenv("AUDIT_DB_SSL_MODE", "disable")
	os.Setenv("AUDIT_MIGRATIONS_PATH", migrationsPath())
}

func waitForServer(url string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	client := httpClient()
	for time.Now().Before(deadline) {
		resp, err := client.Get(url)
		if err == nil && resp.StatusCode == 200 {
			resp.Body.Close()
			return nil
		}
		if resp != nil {
			resp.Body.Close()
		}
		time.Sleep(500 * time.Millisecond)
	}
	return fmt.Errorf("server did not become ready at %s within %s", url, timeout)
}

func migrationsPath() string {
	abs, err := filepath.Abs("../../migrations")
	if err != nil {
		panic(fmt.Sprintf("failed to resolve migrations path: %v", err))
	}

	return abs
}
