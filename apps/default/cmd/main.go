package main

import (
	"context"
	"net/http"
	"sync"

	"github.com/pitabwire/frame"
	"github.com/pitabwire/frame/config"
	"github.com/pitabwire/frame/datastore"
	"github.com/pitabwire/frame/datastore/pool"
	"github.com/pitabwire/frame/security"
	"github.com/pitabwire/frame/security/authorizer"
	securityhttp "github.com/pitabwire/frame/security/interceptors/httptor"
	"github.com/pitabwire/util"

	appconfig "github.com/antinvestor/service-trustage/apps/default/config"
	"github.com/antinvestor/service-trustage/apps/default/service/authz"
	"github.com/antinvestor/service-trustage/apps/default/service/business"
	appcache "github.com/antinvestor/service-trustage/apps/default/service/cache"
	"github.com/antinvestor/service-trustage/apps/default/service/handlers"
	"github.com/antinvestor/service-trustage/apps/default/service/queues"
	"github.com/antinvestor/service-trustage/apps/default/service/repository"
	"github.com/antinvestor/service-trustage/apps/default/service/schedulers"
	"github.com/antinvestor/service-trustage/connector"
	"github.com/antinvestor/service-trustage/connector/adapters"
	"github.com/antinvestor/service-trustage/pkg/telemetry"
)

func main() { //nolint:funlen // main function wiring
	ctx := context.Background()

	cfg, err := config.LoadWithOIDC[appconfig.Config](ctx)
	if err != nil {
		util.Log(ctx).WithError(err).Fatal("failed to load configuration")
	}

	if cfg.Name() == "" {
		cfg.ServiceName = "trustage-api"
	}

	ctx, svc := frame.NewServiceWithContext(
		ctx,
		frame.WithName(cfg.Name()),
		frame.WithConfig(&cfg),
		frame.WithDatastore(
			pool.WithPreferSimpleProtocol(true),
			pool.WithPreparedStatements(false),
		),
	)
	defer svc.Stop(ctx)

	log := svc.Log(ctx)

	// Database setup.
	dbManager := svc.DatastoreManager()

	if cfg.DoDatabaseMigrate() {
		if migrateErr := repository.Migrate(ctx, dbManager); migrateErr != nil {
			log.WithError(migrateErr).Fatal("database migration failed")
		}
		log.Info("database migration completed successfully")
		return
	}

	if migrateErr := repository.Migrate(ctx, dbManager); migrateErr != nil {
		log.WithError(migrateErr).Fatal("database migration failed")
	}

	dbPool := dbManager.GetPool(ctx, datastore.DefaultPoolName)

	// Repositories.
	defRepo := repository.NewWorkflowDefinitionRepository(dbPool)
	instanceRepo := repository.NewWorkflowInstanceRepository(dbPool)
	execRepo := repository.NewWorkflowExecutionRepository(dbPool)
	schemaRepo := repository.NewSchemaRegistryRepository(dbPool)
	outputRepo := repository.NewWorkflowOutputRepository(dbPool)
	auditRepo := repository.NewAuditEventRepository(dbPool)
	eventLogRepo := repository.NewEventLogRepository(dbPool)
	triggerRepo := repository.NewTriggerBindingRepository(dbPool)
	retryPolicyRepo := repository.NewRetryPolicyRepository(dbPool)
	scheduleRepo := repository.NewScheduleRepository(dbPool)

	// Connector registry.
	httpClient := svc.HTTPClientManager().Client(ctx)
	registry := setupConnectorRegistry(httpClient)

	// Cache setup (Valkey with in-memory fallback).
	rawCache, cacheErr := appcache.SetupCache(cfg.ValkeyCacheURL)
	if cacheErr != nil {
		log.WithError(cacheErr).Warn("cache setup failed, using in-memory fallback")
		rawCache, _ = appcache.SetupCache("")
	}

	// Business layer.
	metrics := telemetry.NewMetrics()

	schemaReg := business.NewSchemaRegistry(schemaRepo, rawCache)
	engine := business.NewStateEngine(
		instanceRepo,
		execRepo,
		outputRepo,
		auditRepo,
		defRepo,
		retryPolicyRepo,
		schemaReg,
		metrics,
		rawCache,
	)
	eventRouter := business.NewEventRouter(triggerRepo, defRepo, instanceRepo, auditRepo, engine, metrics)
	workflowBiz := business.NewWorkflowBusiness(defRepo, schemaReg)

	sm := svc.SecurityManager()
	auth := sm.GetAuthorizer(ctx)
	authzMiddleware := authz.NewMiddleware(auth)
	tenancyAccessChecker := authorizer.NewTenancyAccessChecker(auth, authz.NamespaceTenancyAccess)

	// Schedulers (background goroutines with coordinated shutdown).
	// Schedulers process all tenants, so skip tenancy checks on BaseRepository queries.
	schedulerCtx, schedulerCancel := context.WithCancel(security.SkipTenancyChecksOnClaims(ctx))

	var schedulerWg sync.WaitGroup

	dispatchSched := schedulers.NewDispatchScheduler(execRepo, engine, svc.QueueManager(), &cfg, metrics)
	retrySched := schedulers.NewRetryScheduler(execRepo, instanceRepo, &cfg, metrics)
	timeoutSched := schedulers.NewTimeoutScheduler(execRepo, instanceRepo, retryPolicyRepo, auditRepo, &cfg, metrics)
	outboxSched := schedulers.NewOutboxScheduler(eventLogRepo, svc.QueueManager(), &cfg, metrics)

	startScheduler := func(name string, startFn func(context.Context)) {
		schedulerWg.Add(1)

		go func() {
			defer schedulerWg.Done()
			log.Info("scheduler starting", "scheduler", name)
			startFn(schedulerCtx)
			log.Info("scheduler stopped", "scheduler", name)
		}()
	}

	cleanupSched := schedulers.NewCleanupScheduler(eventLogRepo, auditRepo, &cfg)
	cronSched := schedulers.NewCronScheduler(scheduleRepo, eventLogRepo, &cfg)

	startScheduler("dispatch", dispatchSched.Start)
	startScheduler("retry", retrySched.Start)
	startScheduler("timeout", timeoutSched.Start)
	startScheduler("outbox", outboxSched.Start)
	startScheduler("cleanup", cleanupSched.Start)
	startScheduler("cron", cronSched.Start)

	// HTTP handlers.
	workflowHandler := handlers.NewWorkflowHandler(workflowBiz, authzMiddleware, metrics)
	rateLimiter := handlers.NewRateLimiter(rawCache, cfg.EventIngestRateLimit)
	eventHandler := handlers.NewEventHandler(eventLogRepo, auditRepo, authzMiddleware, metrics, rateLimiter)
	formHandler := handlers.NewFormHandler(eventLogRepo, authzMiddleware, metrics, rateLimiter)
	webhookReceiveHandler := handlers.NewWebhookReceiveHandler(eventLogRepo, authzMiddleware, metrics, rateLimiter)
	instanceHandler := handlers.NewInstanceHandler(instanceRepo, execRepo, auditRepo, authzMiddleware)
	executionHandler := handlers.NewExecutionHandler(execRepo, instanceRepo, outputRepo, auditRepo, authzMiddleware)

	protectedMux := http.NewServeMux()

	// Workflow management endpoints.
	protectedMux.HandleFunc("POST /api/v1/workflows", workflowHandler.CreateWorkflow)
	protectedMux.HandleFunc("GET /api/v1/workflows/{id}", workflowHandler.GetWorkflow)
	protectedMux.HandleFunc("POST /api/v1/workflows/{id}/activate", workflowHandler.ActivateWorkflow)
	protectedMux.HandleFunc("GET /api/v1/workflows", workflowHandler.ListWorkflows)

	// Event ingestion and timeline endpoints.
	protectedMux.HandleFunc("POST /api/v1/events", eventHandler.IngestEvent)
	protectedMux.HandleFunc("GET /api/v1/instances/{id}/timeline", eventHandler.GetInstanceTimeline)

	// Instance endpoints.
	protectedMux.HandleFunc("GET /api/v1/instances", instanceHandler.List)
	protectedMux.HandleFunc("POST /api/v1/instances/{id}/retry", instanceHandler.Retry)

	// Execution endpoints.
	protectedMux.HandleFunc("GET /api/v1/executions", executionHandler.List)
	protectedMux.HandleFunc("GET /api/v1/executions/{id}", executionHandler.Get)
	protectedMux.HandleFunc("POST /api/v1/executions/{id}/retry", executionHandler.Retry)

	// Form capture endpoint.
	protectedMux.HandleFunc("POST /api/v1/forms/{form_id}/submit", formHandler.SubmitForm)

	// Webhook receive endpoint.
	protectedMux.HandleFunc("POST /api/v1/webhooks/{webhook_id}", webhookReceiveHandler.ReceiveWebhook)

	// Health checks.
	publicMux := http.NewServeMux()
	publicMux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})
	publicMux.HandleFunc("GET /readyz", func(w http.ResponseWriter, r *http.Request) {
		pool := dbManager.GetPool(r.Context(), datastore.DefaultPoolName)
		if pool == nil {
			http.Error(w, "database not ready", http.StatusServiceUnavailable)
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})
	publicMux.Handle("/", securityhttp.TenancyAccessMiddleware(
		handlers.RequestIDMiddleware(handlers.LimitBodySize(protectedMux)),
		tenancyAccessChecker,
	))

	// Queue workers.
	executionWorker := queues.NewExecutionWorker(engine, defRepo, registry)
	eventRouterWorker := queues.NewEventRouterWorker(eventRouter)

	svc.Init(ctx,
		frame.WithHTTPHandler(publicMux),

		// Execution dispatch publisher (schedulers publish here).
		frame.WithRegisterPublisher(
			cfg.QueueExecDispatchName,
			cfg.QueueExecDispatchURL,
		),

		// Execution worker subscriber (processes dispatched executions).
		frame.WithRegisterSubscriber(
			cfg.QueueExecWorkerName,
			cfg.QueueExecWorkerURL,
			executionWorker,
		),

		// Event ingest publisher (outbox scheduler publishes here).
		frame.WithRegisterPublisher(
			cfg.QueueEventIngestName,
			cfg.QueueEventIngestURL,
		),

		// Event router subscriber (processes ingested events).
		frame.WithRegisterSubscriber(
			cfg.QueueEventRouterName,
			cfg.QueueEventRouterURL,
			eventRouterWorker,
		),
	)

	log.Info("starting trustage orchestrator",
		"port", cfg.ServerPort,
	)

	if runErr := svc.Run(ctx, cfg.ServerPort); runErr != nil {
		log.WithError(runErr).Fatal("could not run service")
	}

	// Graceful scheduler shutdown.
	schedulerCancel()
	schedulerWg.Wait()
	log.Info("all schedulers stopped")
}

func setupConnectorRegistry(httpClient *http.Client) *connector.Registry {
	registry := connector.NewRegistry()

	allAdapters := []connector.Adapter{
		adapters.NewWebhookAdapter(httpClient),
		adapters.NewHTTPAdapter(httpClient),
		adapters.NewNotificationSendAdapter(httpClient),
		adapters.NewNotificationStatusAdapter(httpClient),
		adapters.NewPaymentInitiateAdapter(httpClient),
		adapters.NewPaymentVerifyAdapter(httpClient),
		adapters.NewDataTransformAdapter(),
		adapters.NewLogEntryAdapter(),
		adapters.NewFormValidateAdapter(),
		adapters.NewApprovalRequestAdapter(httpClient),
		adapters.NewAIChatAdapter(),
	}

	for _, a := range allAdapters {
		if regErr := registry.Register(a); regErr != nil {
			panic("failed to register adapter: " + regErr.Error())
		}
	}

	return registry
}
