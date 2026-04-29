// Copyright (c) 2026 Lerian Studio. All rights reserved.
// Use of this source code is governed by the Elastic License 2.0
// that can be found in the LICENSE file.

package bootstrap

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/LerianStudio/flowker/internal/adapters/http/in"
	httpaudit "github.com/LerianStudio/flowker/internal/adapters/http/in/audit"
	inCatalog "github.com/LerianStudio/flowker/internal/adapters/http/in/catalog"
	httpdashboard "github.com/LerianStudio/flowker/internal/adapters/http/in/dashboard"
	httpexecution "github.com/LerianStudio/flowker/internal/adapters/http/in/execution"
	httpexecutorconfig "github.com/LerianStudio/flowker/internal/adapters/http/in/executor_configuration"
	httpMiddleware "github.com/LerianStudio/flowker/internal/adapters/http/in/middleware"
	httpproviderconfig "github.com/LerianStudio/flowker/internal/adapters/http/in/provider_configuration"
	httpwebhook "github.com/LerianStudio/flowker/internal/adapters/http/in/webhook"
	httpworkflow "github.com/LerianStudio/flowker/internal/adapters/http/in/workflow"
	mongodashboard "github.com/LerianStudio/flowker/internal/adapters/mongodb/dashboard"
	mongoexecution "github.com/LerianStudio/flowker/internal/adapters/mongodb/execution"
	mongoexecutorconfig "github.com/LerianStudio/flowker/internal/adapters/mongodb/executor_configuration"
	mongoproviderconfig "github.com/LerianStudio/flowker/internal/adapters/mongodb/provider_configuration"
	mongoworkflow "github.com/LerianStudio/flowker/internal/adapters/mongodb/workflow"
	pgaudit "github.com/LerianStudio/flowker/internal/adapters/postgresql/audit"
	"github.com/LerianStudio/flowker/internal/services"
	"github.com/LerianStudio/flowker/internal/services/command"
	"github.com/LerianStudio/flowker/internal/services/query"
	"github.com/LerianStudio/flowker/pkg/circuitbreaker"
	"github.com/LerianStudio/flowker/pkg/condition"
	"github.com/LerianStudio/flowker/pkg/executor"
	"github.com/LerianStudio/flowker/pkg/executors"
	"github.com/LerianStudio/flowker/pkg/model"
	"github.com/LerianStudio/flowker/pkg/templates"
	"github.com/LerianStudio/flowker/pkg/transformation"
	"github.com/LerianStudio/flowker/pkg/triggers"
	"github.com/LerianStudio/flowker/pkg/webhook"

	authMiddleware "github.com/LerianStudio/lib-auth/v2/auth/middleware"
	libCommons "github.com/LerianStudio/lib-commons/v5/commons"
	libLog "github.com/LerianStudio/lib-commons/v5/commons/log"
	libOtel "github.com/LerianStudio/lib-commons/v5/commons/opentelemetry"
	libZap "github.com/LerianStudio/lib-commons/v5/commons/zap"
)

// Config is the top level configuration struct for the entire application.
type Config struct {
	EnvName                 string `env:"ENV_NAME"`
	ServerAddress           string `env:"SERVER_ADDRESS"`
	LogLevel                string `env:"LOG_LEVEL"`
	OtelServiceName         string `env:"OTEL_RESOURCE_SERVICE_NAME"`
	OtelLibraryName         string `env:"OTEL_LIBRARY_NAME"`
	OtelServiceVersion      string `env:"OTEL_RESOURCE_SERVICE_VERSION"`
	OtelDeploymentEnv       string `env:"OTEL_RESOURCE_DEPLOYMENT_ENVIRONMENT"`
	OtelColExporterEndpoint string `env:"OTEL_EXPORTER_OTLP_ENDPOINT"`
	EnableTelemetry         bool   `env:"ENABLE_TELEMETRY"`
	// MongoDB configuration
	MongoURI       string `env:"MONGO_URI"`
	MongoDBName    string `env:"MONGO_DB_NAME"`
	MongoTLSCACert string `env:"MONGO_TLS_CA_CERT"`
	// Swagger configuration
	SwaggerTitle       string `env:"SWAGGER_TITLE"`
	SwaggerDescription string `env:"SWAGGER_DESCRIPTION"`
	SwaggerVersion     string `env:"SWAGGER_VERSION"`
	SwaggerHost        string `env:"SWAGGER_HOST"`
	SwaggerBasePath    string `env:"SWAGGER_BASE_PATH"`
	SwaggerLeftDelim   string `env:"SWAGGER_LEFT_DELIM"`
	SwaggerRightDelim  string `env:"SWAGGER_RIGHT_DELIM"`
	SwaggerSchemes     string `env:"SWAGGER_SCHEMES"`
	APIKey             string `env:"API_KEY"`
	APIKeyEnabled      bool   `env:"API_KEY_ENABLED"`
	CORSAllowedOrigins string `env:"CORS_ALLOWED_ORIGINS"`
	// Access Manager (plugin auth) configuration
	PluginAuthEnabled bool   `env:"PLUGIN_AUTH_ENABLED"`
	PluginAuthAddress string `env:"PLUGIN_AUTH_ADDRESS"`
	// Audit database configuration
	AuditDBHost string `env:"AUDIT_DB_HOST"`
	// Feature flags
	SkipLibCommonsTelemetry bool `env:"SKIP_LIB_COMMONS_TELEMETRY"`
	FaultInjectionEnabled   bool `env:"FAULT_INJECTION_ENABLED"`
}

// InitServers initiate http server.
// Returns an error if any configuration or initialization step fails.
func InitServers() (*Service, error) {
	cfg := &Config{}

	if err := libCommons.SetConfigFromEnvVars(cfg); err != nil {
		return nil, fmt.Errorf("failed to load config from env vars: %w", err)
	}

	zapLogger, err := libZap.New(libZap.Config{
		Environment:     libZap.Environment(cfg.EnvName),
		Level:           cfg.LogLevel,
		OTelLibraryName: cfg.OtelLibraryName,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to initialize logger: %w", err)
	}

	var logger libLog.Logger = zapLogger

	ctx := context.Background()

	// Validate API key configuration
	if cfg.APIKeyEnabled && cfg.APIKey == "" {
		return nil, fmt.Errorf("API_KEY_ENABLED=true requires API_KEY to be set")
	}

	// Validate Access Manager plugin configuration
	if err := ValidateAccessManagerConfig(cfg, logger); err != nil {
		return nil, err
	}

	// Init Open telemetry to control logs and flows
	if cfg.EnableTelemetry && cfg.OtelColExporterEndpoint == "" {
		return nil, fmt.Errorf("ENABLE_TELEMETRY=true requires OTEL_EXPORTER_OTLP_ENDPOINT to be set")
	}

	telemetry, err := libOtel.NewTelemetry(libOtel.TelemetryConfig{
		LibraryName:               cfg.OtelLibraryName,
		ServiceName:               cfg.OtelServiceName,
		ServiceVersion:            cfg.OtelServiceVersion,
		DeploymentEnv:             cfg.OtelDeploymentEnv,
		CollectorExporterEndpoint: cfg.OtelColExporterEndpoint,
		EnableTelemetry:           cfg.EnableTelemetry,
		Logger:                    logger,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to initialize telemetry: %w", err)
	}

	// Initialize MongoDB connection manager
	dbManager := NewDatabaseManagerWithConfig(&MongoConfig{
		URI:       cfg.MongoURI,
		Database:  cfg.MongoDBName,
		TLSCACert: cfg.MongoTLSCACert,
	})

	// Connect to MongoDB with timeout
	// The application should fail fast if database is unavailable at startup
	connectCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := dbManager.Connect(connectCtx); err != nil {
		logger.Log(ctx, libLog.LevelError, "Failed to connect to MongoDB", libLog.Any("error.message", err.Error()))
		return nil, fmt.Errorf("failed to connect to MongoDB: %w", err)
	}

	logger.Log(ctx, libLog.LevelInfo, "Successfully connected to MongoDB")

	// Initialize executor catalog with built-in executors (e.g., HTTP).
	executorCatalog := executor.NewCatalog()

	if err := executors.RegisterDefaults(executorCatalog); err != nil {
		logger.Log(ctx, libLog.LevelError, "Failed to register default executors", libLog.Any("error.message", err.Error()))
		return nil, err
	}

	if err := triggers.RegisterDefaults(executorCatalog); err != nil {
		logger.Log(ctx, libLog.LevelError, "Failed to register default triggers", libLog.Any("error.message", err.Error()))
		return nil, err
	}

	if err := templates.RegisterDefaults(executorCatalog); err != nil {
		logger.Log(ctx, libLog.LevelError, "Failed to register default templates", libLog.Any("error.message", err.Error()))
		return nil, err
	}

	// Initialize audit components (mandatory - AUDIT_DB_HOST must be configured)
	if cfg.AuditDBHost == "" {
		return nil, fmt.Errorf("AUDIT_DB_HOST is required: audit trail is mandatory for compliance")
	}

	auditHandler, auditWriter, err := initAuditComponents(logger)
	if err != nil {
		logger.Log(ctx, libLog.LevelError, "Failed to initialize audit components", libLog.Any("error.message", err.Error()))
		return nil, err
	}

	logger.Log(ctx, libLog.LevelInfo, "Audit trail components initialized successfully")

	// Initialize provider configuration components
	providerConfigHandler, providerConfigRepo, err := initProviderConfigComponents(dbManager, executorCatalog, auditWriter, logger)
	if err != nil {
		return nil, err
	}

	// Initialize webhook registry
	webhookRegistry := webhook.NewRegistry()

	// Initialize workflow components (with webhook registry)
	workflowHandler, catalogHandler, workflowRepo, err := initWorkflowComponents(dbManager, providerConfigRepo, providerConfigRepo, executorCatalog, auditWriter, logger, webhookRegistry)
	if err != nil {
		return nil, err
	}

	// Initialize executor configuration components
	executorConfigHandler, err := initExecutorConfigComponents(dbManager, logger)
	if err != nil {
		return nil, err
	}

	// Initialize execution components
	executionHandler, executeSvc, err := initExecutionComponents(dbManager, workflowRepo, providerConfigRepo, executorCatalog, auditWriter, logger)
	if err != nil {
		return nil, err
	}

	// Initialize webhook handler (uses execution service to trigger workflows)
	var webhookHandler *httpwebhook.Handler
	if executeSvc != nil {
		webhookHandler, err = httpwebhook.NewHandler(webhookRegistry, executeSvc)
		if err != nil {
			logger.Log(ctx, libLog.LevelError, "Failed to create webhook handler", libLog.Any("error.message", err.Error()))
			return nil, err
		}
	}

	// Populate webhook registry with active workflows
	populateWebhookRegistry(ctx, workflowRepo, webhookRegistry, logger)

	// Initialize dashboard components
	dashboardHandler, err := initDashboardComponents(dbManager, logger)
	if err != nil {
		return nil, err
	}

	swaggerCfg := in.SwaggerConfig{
		Title:       cfg.SwaggerTitle,
		Description: cfg.SwaggerDescription,
		Version:     cfg.SwaggerVersion,
		Host:        cfg.SwaggerHost,
		BasePath:    cfg.SwaggerBasePath,
		LeftDelim:   cfg.SwaggerLeftDelim,
		RightDelim:  cfg.SwaggerRightDelim,
		Schemes:     cfg.SwaggerSchemes,
	}

	routeCfg := &in.RouteConfig{
		CORSAllowedOrigins:      cfg.CORSAllowedOrigins,
		SkipLibCommonsTelemetry: cfg.SkipLibCommonsTelemetry,
		FaultInjectionEnabled:   cfg.FaultInjectionEnabled,
	}

	// Create Access Manager auth client and guard
	authClient := authMiddleware.NewAuthClient(cfg.PluginAuthAddress, cfg.PluginAuthEnabled, &logger)
	authGuard := httpMiddleware.NewAuthGuard(httpMiddleware.AuthGuardConfig{
		APIKey:            cfg.APIKey,
		APIKeyEnabled:     cfg.APIKeyEnabled,
		PluginAuthEnabled: cfg.PluginAuthEnabled,
		AppName:           "flowker",
	}, authClient)

	if authGuard == nil && cfg.PluginAuthEnabled {
		return nil, fmt.Errorf("failed to create auth guard: plugin auth is enabled but auth client is unavailable")
	}

	httpApp, err := in.NewRoutes(logger, telemetry, swaggerCfg, dbManager, routeCfg, workflowHandler, catalogHandler, executorConfigHandler, providerConfigHandler, executionHandler, dashboardHandler, auditHandler, webhookHandler, authGuard)
	if err != nil {
		logger.Log(ctx, libLog.LevelError, "Failed to create HTTP routes", libLog.Any("error.message", err.Error()))
		return nil, err
	}

	serverAPI := NewHTTPServer(cfg, httpApp, logger, telemetry)

	return &Service{
		HTTPServer: serverAPI,
		Logger:     logger,
	}, nil
}

// initWorkflowComponents creates all workflow-related commands, queries, services and handlers.
func initWorkflowComponents(
	dbManager *DatabaseManager,
	providerConfigRepo command.ProviderConfigReadRepository,
	providerConfigFullRepo command.ProviderConfigRepository,
	executorCatalog executor.Catalog,
	auditWriter command.AuditWriter,
	logger libLog.Logger,
	webhookRegistry *webhook.Registry,
) (*httpworkflow.Handler, *inCatalog.Handler, *mongoworkflow.MongoDBRepository, error) {
	ctx := context.Background()

	db, err := dbManager.GetDatabase(ctx)
	if err != nil {
		logger.Log(ctx, libLog.LevelError, "Failed to get database for workflow components", libLog.Any("error.message", err.Error()))
		return nil, nil, nil, err
	}

	workflowRepo := mongoworkflow.NewMongoDBRepository(db)

	transformSvc := transformation.NewService()
	transformValidator := transformSvc.Validator()

	createWorkflowCmd, err := command.NewCreateWorkflowCommand(workflowRepo, providerConfigRepo, executorCatalog, transformValidator, nil, auditWriter)
	if err != nil {
		logger.Log(ctx, libLog.LevelError, "Failed to create CreateWorkflowCommand", libLog.Any("error.message", err.Error()))
		return nil, nil, nil, err
	}

	createFromTemplateCmd, err := command.NewCreateWorkflowFromTemplateCommand(executorCatalog, createWorkflowCmd)
	if err != nil {
		logger.Log(ctx, libLog.LevelError, "Failed to create CreateWorkflowFromTemplateCommand", libLog.Any("error.message", err.Error()))
		return nil, nil, nil, err
	}

	updateWorkflowCmd, err := command.NewUpdateWorkflowCommand(workflowRepo, providerConfigRepo, executorCatalog, transformValidator, nil, auditWriter)
	if err != nil {
		logger.Log(ctx, libLog.LevelError, "Failed to create UpdateWorkflowCommand", libLog.Any("error.message", err.Error()))
		return nil, nil, nil, err
	}

	cloneWorkflowCmd, err := command.NewCloneWorkflowCommand(workflowRepo, nil)
	if err != nil {
		logger.Log(ctx, libLog.LevelError, "Failed to create CloneWorkflowCommand", libLog.Any("error.message", err.Error()))
		return nil, nil, nil, err
	}

	activateWorkflowCmd, err := command.NewActivateWorkflowCommand(workflowRepo, providerConfigRepo, executorCatalog, transformValidator, nil, auditWriter, webhookRegistry)
	if err != nil {
		logger.Log(ctx, libLog.LevelError, "Failed to create ActivateWorkflowCommand", libLog.Any("error.message", err.Error()))
		return nil, nil, nil, err
	}

	deactivateWorkflowCmd, err := command.NewDeactivateWorkflowCommand(workflowRepo, nil, auditWriter, webhookRegistry)
	if err != nil {
		logger.Log(ctx, libLog.LevelError, "Failed to create DeactivateWorkflowCommand", libLog.Any("error.message", err.Error()))
		return nil, nil, nil, err
	}

	moveToDraftCmd, err := command.NewMoveToDraftWorkflowCommand(workflowRepo, nil, auditWriter)
	if err != nil {
		logger.Log(ctx, libLog.LevelError, "Failed to create MoveToDraftWorkflowCommand", libLog.Any("error.message", err.Error()))
		return nil, nil, nil, err
	}

	deleteWorkflowCmd, err := command.NewDeleteWorkflowCommand(workflowRepo, nil, auditWriter, webhookRegistry)
	if err != nil {
		logger.Log(ctx, libLog.LevelError, "Failed to create DeleteWorkflowCommand", libLog.Any("error.message", err.Error()))
		return nil, nil, nil, err
	}

	getWorkflowQuery, err := query.NewGetWorkflowQuery(workflowRepo)
	if err != nil {
		logger.Log(ctx, libLog.LevelError, "Failed to create GetWorkflowQuery", libLog.Any("error.message", err.Error()))
		return nil, nil, nil, err
	}

	getWorkflowByNameQuery, err := query.NewGetWorkflowByNameQuery(workflowRepo)
	if err != nil {
		logger.Log(ctx, libLog.LevelError, "Failed to create GetWorkflowByNameQuery", libLog.Any("error.message", err.Error()))
		return nil, nil, nil, err
	}

	listWorkflowsQuery, err := query.NewListWorkflowsQuery(workflowRepo)
	if err != nil {
		logger.Log(ctx, libLog.LevelError, "Failed to create ListWorkflowsQuery", libLog.Any("error.message", err.Error()))
		return nil, nil, nil, err
	}

	workflowSvc, err := services.NewWorkflowService(
		createWorkflowCmd, createFromTemplateCmd, updateWorkflowCmd, cloneWorkflowCmd,
		activateWorkflowCmd, deactivateWorkflowCmd, moveToDraftCmd, deleteWorkflowCmd,
		getWorkflowQuery, getWorkflowByNameQuery, listWorkflowsQuery,
	)
	if err != nil {
		logger.Log(ctx, libLog.LevelError, "Failed to create WorkflowService", libLog.Any("error.message", err.Error()))
		return nil, nil, nil, err
	}

	workflowHandler, err := httpworkflow.NewHandler(workflowSvc, workflowSvc)
	if err != nil {
		logger.Log(ctx, libLog.LevelError, "Failed to create workflow handler", libLog.Any("error.message", err.Error()))
		return nil, nil, nil, err
	}

	configLister := &providerConfigListerAdapter{repo: providerConfigFullRepo}
	catalogHandler, err := inCatalog.NewHandler(executorCatalog, configLister)

	if err != nil {
		logger.Log(ctx, libLog.LevelError, "Failed to create catalog handler", libLog.Any("error.message", err.Error()))
		return nil, nil, nil, err
	}

	return workflowHandler, catalogHandler, workflowRepo, nil
}

// initExecutorConfigComponents creates all executor configuration commands, queries, services and handlers.
func initExecutorConfigComponents(
	dbManager *DatabaseManager,
	logger libLog.Logger,
) (*httpexecutorconfig.Handler, error) {
	ctx := context.Background()

	db, err := dbManager.GetDatabase(ctx)
	if err != nil {
		logger.Log(ctx, libLog.LevelError, "Failed to get database for executor config components", libLog.Any("error.message", err.Error()))
		return nil, err
	}

	executorConfigRepo := mongoexecutorconfig.NewMongoDBRepository(db)

	createExecutorConfigCmd, err := command.NewCreateExecutorConfigCommand(executorConfigRepo, nil)
	if err != nil {
		logger.Log(ctx, libLog.LevelError, "Failed to create CreateExecutorConfigCommand", libLog.Any("error.message", err.Error()))
		return nil, err
	}

	updateExecutorConfigCmd, err := command.NewUpdateExecutorConfigCommand(executorConfigRepo, nil)
	if err != nil {
		logger.Log(ctx, libLog.LevelError, "Failed to create UpdateExecutorConfigCommand", libLog.Any("error.message", err.Error()))
		return nil, err
	}

	markConfiguredCmd, err := command.NewMarkConfiguredCommand(executorConfigRepo, nil)
	if err != nil {
		logger.Log(ctx, libLog.LevelError, "Failed to create MarkConfiguredCommand", libLog.Any("error.message", err.Error()))
		return nil, err
	}

	markTestedCmd, err := command.NewMarkTestedCommand(executorConfigRepo, nil)
	if err != nil {
		logger.Log(ctx, libLog.LevelError, "Failed to create MarkTestedCommand", libLog.Any("error.message", err.Error()))
		return nil, err
	}

	testConnectivityCmd, err := command.NewTestExecutorConnectivityCommand(executorConfigRepo, nil)
	if err != nil {
		logger.Log(ctx, libLog.LevelError, "Failed to create TestExecutorConnectivityCommand", libLog.Any("error.message", err.Error()))
		return nil, err
	}

	activateExecutorConfigCmd, err := command.NewActivateExecutorConfigCommand(executorConfigRepo, nil)
	if err != nil {
		logger.Log(ctx, libLog.LevelError, "Failed to create ActivateExecutorConfigCommand", libLog.Any("error.message", err.Error()))
		return nil, err
	}

	disableExecutorConfigCmd, err := command.NewDisableExecutorConfigCommand(executorConfigRepo, nil)
	if err != nil {
		logger.Log(ctx, libLog.LevelError, "Failed to create DisableExecutorConfigCommand", libLog.Any("error.message", err.Error()))
		return nil, err
	}

	enableExecutorConfigCmd, err := command.NewEnableExecutorConfigCommand(executorConfigRepo, nil)
	if err != nil {
		logger.Log(ctx, libLog.LevelError, "Failed to create EnableExecutorConfigCommand", libLog.Any("error.message", err.Error()))
		return nil, err
	}

	deleteExecutorConfigCmd, err := command.NewDeleteExecutorConfigCommand(executorConfigRepo, nil)
	if err != nil {
		logger.Log(ctx, libLog.LevelError, "Failed to create DeleteExecutorConfigCommand", libLog.Any("error.message", err.Error()))
		return nil, err
	}

	getExecutorConfigQuery, err := query.NewGetExecutorConfigQuery(executorConfigRepo)
	if err != nil {
		logger.Log(ctx, libLog.LevelError, "Failed to create GetExecutorConfigQuery", libLog.Any("error.message", err.Error()))
		return nil, err
	}

	getExecutorConfigByNameQuery, err := query.NewGetExecutorConfigByNameQuery(executorConfigRepo)
	if err != nil {
		logger.Log(ctx, libLog.LevelError, "Failed to create GetExecutorConfigByNameQuery", libLog.Any("error.message", err.Error()))
		return nil, err
	}

	listExecutorConfigsQuery, err := query.NewListExecutorConfigsQuery(executorConfigRepo)
	if err != nil {
		logger.Log(ctx, libLog.LevelError, "Failed to create ListExecutorConfigsQuery", libLog.Any("error.message", err.Error()))
		return nil, err
	}

	existsExecutorConfigQuery, err := query.NewExistsExecutorConfigQuery(executorConfigRepo)
	if err != nil {
		logger.Log(ctx, libLog.LevelError, "Failed to create ExistsExecutorConfigQuery", libLog.Any("error.message", err.Error()))
		return nil, err
	}

	executorConfigSvc, err := services.NewExecutorConfigurationService(
		createExecutorConfigCmd, updateExecutorConfigCmd, markConfiguredCmd, markTestedCmd, testConnectivityCmd,
		activateExecutorConfigCmd, disableExecutorConfigCmd, enableExecutorConfigCmd, deleteExecutorConfigCmd,
		getExecutorConfigQuery, getExecutorConfigByNameQuery, listExecutorConfigsQuery, existsExecutorConfigQuery,
	)
	if err != nil {
		logger.Log(ctx, libLog.LevelError, "Failed to create ExecutorConfigurationService", libLog.Any("error.message", err.Error()))
		return nil, err
	}

	executorConfigHandler, err := httpexecutorconfig.NewHandler(executorConfigSvc, executorConfigSvc)
	if err != nil {
		logger.Log(ctx, libLog.LevelError, "Failed to create executor configuration handler", libLog.Any("error.message", err.Error()))
		return nil, err
	}

	return executorConfigHandler, nil
}

// initProviderConfigComponents creates all provider configuration commands, queries, services and handlers.
func initProviderConfigComponents(
	dbManager *DatabaseManager,
	executorCatalog executor.Catalog,
	auditWriter command.AuditWriter,
	logger libLog.Logger,
) (*httpproviderconfig.Handler, *mongoproviderconfig.MongoDBRepository, error) {
	ctx := context.Background()

	db, err := dbManager.GetDatabase(ctx)
	if err != nil {
		logger.Log(ctx, libLog.LevelError, "Failed to get database for provider config components", libLog.Any("error.message", err.Error()))
		return nil, nil, err
	}

	providerConfigRepo := mongoproviderconfig.NewMongoDBRepository(db)

	createProviderConfigCmd, err := command.NewCreateProviderConfigCommand(providerConfigRepo, executorCatalog, auditWriter)
	if err != nil {
		logger.Log(ctx, libLog.LevelError, "Failed to create CreateProviderConfigCommand", libLog.Any("error.message", err.Error()))
		return nil, nil, err
	}

	updateProviderConfigCmd, err := command.NewUpdateProviderConfigCommand(providerConfigRepo, executorCatalog, auditWriter)
	if err != nil {
		logger.Log(ctx, libLog.LevelError, "Failed to create UpdateProviderConfigCommand", libLog.Any("error.message", err.Error()))
		return nil, nil, err
	}

	deleteProviderConfigCmd, err := command.NewDeleteProviderConfigCommand(providerConfigRepo, auditWriter)
	if err != nil {
		logger.Log(ctx, libLog.LevelError, "Failed to create DeleteProviderConfigCommand", libLog.Any("error.message", err.Error()))
		return nil, nil, err
	}

	disableProviderConfigCmd, err := command.NewDisableProviderConfigCommand(providerConfigRepo)
	if err != nil {
		logger.Log(ctx, libLog.LevelError, "Failed to create DisableProviderConfigCommand", libLog.Any("error.message", err.Error()))
		return nil, nil, err
	}

	enableProviderConfigCmd, err := command.NewEnableProviderConfigCommand(providerConfigRepo)
	if err != nil {
		logger.Log(ctx, libLog.LevelError, "Failed to create EnableProviderConfigCommand", libLog.Any("error.message", err.Error()))
		return nil, nil, err
	}

	testConnectivityCmd, err := command.NewTestProviderConfigConnectivityCommand(providerConfigRepo, nil)
	if err != nil {
		logger.Log(ctx, libLog.LevelError, "Failed to create TestProviderConfigConnectivityCommand", libLog.Any("error.message", err.Error()))
		return nil, nil, err
	}

	getProviderConfigQuery, err := query.NewGetProviderConfigByIDQuery(providerConfigRepo)
	if err != nil {
		logger.Log(ctx, libLog.LevelError, "Failed to create GetProviderConfigByIDQuery", libLog.Any("error.message", err.Error()))
		return nil, nil, err
	}

	listProviderConfigsQuery, err := query.NewListProviderConfigsQuery(providerConfigRepo)
	if err != nil {
		logger.Log(ctx, libLog.LevelError, "Failed to create ListProviderConfigsQuery", libLog.Any("error.message", err.Error()))
		return nil, nil, err
	}

	providerConfigSvc, err := services.NewProviderConfigurationService(
		createProviderConfigCmd, updateProviderConfigCmd, deleteProviderConfigCmd,
		disableProviderConfigCmd, enableProviderConfigCmd, testConnectivityCmd,
		getProviderConfigQuery, listProviderConfigsQuery,
	)
	if err != nil {
		logger.Log(ctx, libLog.LevelError, "Failed to create ProviderConfigurationService", libLog.Any("error.message", err.Error()))
		return nil, nil, err
	}

	providerConfigHandler, err := httpproviderconfig.NewHandler(providerConfigSvc, providerConfigSvc)
	if err != nil {
		logger.Log(ctx, libLog.LevelError, "Failed to create provider configuration handler", libLog.Any("error.message", err.Error()))
		return nil, nil, err
	}

	return providerConfigHandler, providerConfigRepo, nil
}

// initExecutionComponents creates all execution-related commands, queries, services and handlers.
// Returns the HTTP handler and the execution service (needed by the webhook handler).
func initExecutionComponents(
	dbManager *DatabaseManager,
	workflowRepo *mongoworkflow.MongoDBRepository,
	providerConfigRepo command.ProviderConfigReadRepository,
	executorCatalog executor.Catalog,
	auditWriter command.AuditWriter,
	logger libLog.Logger,
) (*httpexecution.Handler, *services.ExecutionService, error) {
	ctx := context.Background()

	db, err := dbManager.GetDatabase(ctx)
	if err != nil {
		logger.Log(ctx, libLog.LevelError, "Failed to get database for execution components", libLog.Any("error.message", err.Error()))
		return nil, nil, err
	}

	executionRepo := mongoexecution.NewMongoDBRepository(db)

	cbManager := circuitbreaker.NewManager()
	condEvaluator := condition.NewEvaluator()
	transformSvc := transformation.NewService()

	executeWorkflowCmd, err := command.NewExecuteWorkflowCommand(
		executionRepo, workflowRepo, providerConfigRepo,
		executorCatalog, cbManager, condEvaluator, transformSvc, auditWriter,
	)
	if err != nil {
		logger.Log(ctx, libLog.LevelError, "Failed to create ExecuteWorkflowCommand", libLog.Any("error.message", err.Error()))
		return nil, nil, err
	}

	getExecutionQuery, err := query.NewGetExecutionQuery(executionRepo)
	if err != nil {
		logger.Log(ctx, libLog.LevelError, "Failed to create GetExecutionQuery", libLog.Any("error.message", err.Error()))
		return nil, nil, err
	}

	getExecutionResultsQuery, err := query.NewGetExecutionResultsQuery(executionRepo)
	if err != nil {
		logger.Log(ctx, libLog.LevelError, "Failed to create GetExecutionResultsQuery", libLog.Any("error.message", err.Error()))
		return nil, nil, err
	}

	listExecutionsQuery, err := query.NewListExecutionsQuery(executionRepo)
	if err != nil {
		logger.Log(ctx, libLog.LevelError, "Failed to create ListExecutionsQuery", libLog.Any("error.message", err.Error()))
		return nil, nil, err
	}

	executionSvc, err := services.NewExecutionService(
		executeWorkflowCmd, getExecutionQuery, getExecutionResultsQuery, listExecutionsQuery,
	)
	if err != nil {
		logger.Log(ctx, libLog.LevelError, "Failed to create ExecutionService", libLog.Any("error.message", err.Error()))
		return nil, nil, err
	}

	executionHandler, err := httpexecution.NewHandler(executionSvc, executionSvc)
	if err != nil {
		logger.Log(ctx, libLog.LevelError, "Failed to create execution handler", libLog.Any("error.message", err.Error()))
		return nil, nil, err
	}

	return executionHandler, executionSvc, nil
}

// initDashboardComponents creates all dashboard-related queries, services and handlers.
func initDashboardComponents(
	dbManager *DatabaseManager,
	logger libLog.Logger,
) (*httpdashboard.Handler, error) {
	ctx := context.Background()

	db, err := dbManager.GetDatabase(ctx)
	if err != nil {
		logger.Log(ctx, libLog.LevelError, "Failed to get database for dashboard components", libLog.Any("error.message", err.Error()))
		return nil, err
	}

	dashboardRepo, err := mongodashboard.NewMongoDBRepository(db)
	if err != nil {
		logger.Log(ctx, libLog.LevelError, "Failed to create dashboard repository", libLog.Any("error.message", err.Error()))
		return nil, err
	}

	workflowSummaryQuery, err := query.NewGetWorkflowSummaryQuery(dashboardRepo)
	if err != nil {
		logger.Log(ctx, libLog.LevelError, "Failed to create GetWorkflowSummaryQuery", libLog.Any("error.message", err.Error()))
		return nil, err
	}

	executionSummaryQuery, err := query.NewGetExecutionSummaryQuery(dashboardRepo)
	if err != nil {
		logger.Log(ctx, libLog.LevelError, "Failed to create GetExecutionSummaryQuery", libLog.Any("error.message", err.Error()))
		return nil, err
	}

	dashboardSvc, err := services.NewDashboardService(workflowSummaryQuery, executionSummaryQuery)
	if err != nil {
		logger.Log(ctx, libLog.LevelError, "Failed to create DashboardService", libLog.Any("error.message", err.Error()))
		return nil, err
	}

	dashboardHandler, err := httpdashboard.NewHandler(dashboardSvc)
	if err != nil {
		logger.Log(ctx, libLog.LevelError, "Failed to create dashboard handler", libLog.Any("error.message", err.Error()))
		return nil, err
	}

	return dashboardHandler, nil
}

// ValidateAccessManagerConfig validates the Access Manager plugin configuration.
// It logs an info message when plugin auth is disabled (expected in dev).
// It fails if plugin auth is enabled but the address is missing.
// It also warns if legacy OIDC_* env vars are still set (replaced by PLUGIN_AUTH_*).
func ValidateAccessManagerConfig(cfg *Config, logger libLog.Logger) error {
	warnDeprecatedOIDCEnvVars(logger)

	if !cfg.PluginAuthEnabled {
		logger.With(libLog.String("config", "PLUGIN_AUTH_ENABLED")).Log(context.Background(), libLog.LevelInfo, "Access Manager plugin authentication is disabled")
		return nil
	}

	if cfg.PluginAuthAddress == "" {
		return fmt.Errorf("PLUGIN_AUTH_ADDRESS must be set when PLUGIN_AUTH_ENABLED=true")
	}

	return nil
}

// warnDeprecatedOIDCEnvVars emits a warning for each legacy OIDC_* env var still
// set in the environment. These were replaced by PLUGIN_AUTH_ENABLED / PLUGIN_AUTH_ADDRESS;
// the app no longer reads them. Helps operators notice stale deployment configs.
func warnDeprecatedOIDCEnvVars(logger libLog.Logger) {
	deprecated := []string{"OIDC_ENABLED", "OIDC_ISSUER_URL", "OIDC_JWKS_URL", "OIDC_AUDIENCE"}
	for _, name := range deprecated {
		if _, ok := os.LookupEnv(name); ok {
			logger.With(libLog.String("env.var", name)).Log(context.Background(), libLog.LevelWarn, "Deprecated OIDC env var is set but no longer used; replaced by PLUGIN_AUTH_ENABLED / PLUGIN_AUTH_ADDRESS")
		}
	}
}

// providerConfigListerAdapter adapts a command.ProviderConfigRepository into a
// catalog.ProviderConfigLister, providing read-only access filtered by provider and status.
type providerConfigListerAdapter struct {
	repo command.ProviderConfigRepository
}

// initAuditComponents creates all audit-related repositories, queries, services and handlers.
// It creates its own AuditDatabaseManager, connects to PostgreSQL, and sets up the full
// audit read pipeline (queries + service facade + HTTP handler).
func initAuditComponents(
	logger libLog.Logger,
) (*httpaudit.Handler, command.AuditWriter, error) {
	ctx := context.Background()

	// Create and connect audit database
	auditDBManager := NewAuditDatabaseManager()

	connectCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := auditDBManager.Connect(connectCtx); err != nil {
		logger.Log(ctx, libLog.LevelError, "Failed to connect to audit database", libLog.Any("error.message", err.Error()))
		return nil, nil, err
	}

	// Create PostgreSQL audit repository
	auditRepo, err := pgaudit.NewPostgreSQLRepository(auditDBManager.GetPool())
	if err != nil {
		logger.Log(ctx, libLog.LevelError, "Failed to create audit repository", libLog.Any("error.message", err.Error()))
		return nil, nil, err
	}

	// Create audit writer command (used by all command handlers for mandatory audit trail)
	auditWriter, err := command.NewRecordAuditEventCommand(auditRepo)
	if err != nil {
		logger.Log(ctx, libLog.LevelError, "Failed to create RecordAuditEventCommand", libLog.Any("error.message", err.Error()))
		return nil, nil, err
	}

	// Create query services
	searchQuery, err := query.NewSearchAuditLogsQuery(auditRepo)
	if err != nil {
		logger.Log(ctx, libLog.LevelError, "Failed to create SearchAuditLogsQuery", libLog.Any("error.message", err.Error()))
		return nil, nil, err
	}

	getByIDQuery, err := query.NewGetAuditEntryByIDQuery(auditRepo)
	if err != nil {
		logger.Log(ctx, libLog.LevelError, "Failed to create GetAuditEntryByIDQuery", libLog.Any("error.message", err.Error()))
		return nil, nil, err
	}

	verifyHashQuery, err := query.NewVerifyAuditHashChainQuery(auditRepo)
	if err != nil {
		logger.Log(ctx, libLog.LevelError, "Failed to create VerifyAuditHashChainQuery", libLog.Any("error.message", err.Error()))
		return nil, nil, err
	}

	// Create audit service facade
	auditSvc, err := services.NewAuditService(searchQuery, getByIDQuery, verifyHashQuery)
	if err != nil {
		logger.Log(ctx, libLog.LevelError, "Failed to create AuditService", libLog.Any("error.message", err.Error()))
		return nil, nil, err
	}

	// Create HTTP handler
	auditHandler, err := httpaudit.NewHandler(auditSvc)
	if err != nil {
		logger.Log(ctx, libLog.LevelError, "Failed to create audit handler", libLog.Any("error.message", err.Error()))
		return nil, nil, err
	}

	return auditHandler, auditWriter, nil
}

// populateWebhookRegistry loads all active workflows from the database and
// populates the webhook registry with any webhook trigger routes.
// It paginates through all pages to ensure no workflows are missed.
// Errors are logged but do not prevent server startup.
func populateWebhookRegistry(
	ctx context.Context,
	workflowRepo *mongoworkflow.MongoDBRepository,
	webhookRegistry *webhook.Registry,
	logger libLog.Logger,
) {
	activeStatus := model.WorkflowStatusActive
	registered := 0
	cursor := ""

	for {
		filter := command.WorkflowListFilter{
			Status:    &activeStatus,
			Limit:     100,
			Cursor:    cursor,
			SortBy:    "created_at",
			SortOrder: "ASC",
		}

		result, err := workflowRepo.List(ctx, filter)
		if err != nil {
			logger.Log(ctx, libLog.LevelWarn, "Failed to load active workflows for webhook registry",
				libLog.Any("error.message", err.Error()))
			return
		}

		registered += services.PopulateRegistryFromWorkflows(webhookRegistry, result.Items)

		if !result.HasMore {
			break
		}

		cursor = result.NextCursor
	}

	if registered > 0 {
		logger.Log(ctx, libLog.LevelInfo, "Populated webhook registry from active workflows",
			libLog.Any("routes.registered", registered))
	}
}

func (a *providerConfigListerAdapter) ListActiveByProvider(ctx context.Context, providerID string) ([]*model.ProviderConfiguration, error) {
	status := model.ProviderConfigStatusActive
	filter := command.ProviderConfigListFilter{
		Status:     &status,
		ProviderID: &providerID,
		Limit:      100,
		SortBy:     "name",
		SortOrder:  "ASC",
	}

	result, err := a.repo.List(ctx, filter)
	if err != nil {
		return nil, err
	}

	return result.Items, nil
}
