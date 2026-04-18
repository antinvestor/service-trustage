// Copyright 2023-2026 Ant Investor Ltd
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"context"
	"net/http"
	"sync"

	"connectrpc.com/connect"
	"github.com/antinvestor/common/permissions"
	"github.com/pitabwire/frame"
	"github.com/pitabwire/frame/config"
	"github.com/pitabwire/frame/datastore"
	"github.com/pitabwire/frame/datastore/pool"
	"github.com/pitabwire/frame/security"
	"github.com/pitabwire/frame/security/authorizer"
	connectInterceptors "github.com/pitabwire/frame/security/interceptors/connect"
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
	eventv1 "github.com/antinvestor/service-trustage/gen/go/event/v1"
	"github.com/antinvestor/service-trustage/gen/go/event/v1/eventv1connect"
	runtimev1 "github.com/antinvestor/service-trustage/gen/go/runtime/v1"
	"github.com/antinvestor/service-trustage/gen/go/runtime/v1/runtimev1connect"
	signalv1 "github.com/antinvestor/service-trustage/gen/go/signal/v1"
	"github.com/antinvestor/service-trustage/gen/go/signal/v1/signalv1connect"
	workflowv1 "github.com/antinvestor/service-trustage/gen/go/workflow/v1"
	"github.com/antinvestor/service-trustage/gen/go/workflow/v1/workflowv1connect"
	"github.com/antinvestor/service-trustage/pkg/telemetry"
	eventv1spec "github.com/antinvestor/service-trustage/proto/event/v1"
	runtimev1spec "github.com/antinvestor/service-trustage/proto/runtime/v1"
	signalv1spec "github.com/antinvestor/service-trustage/proto/signal/v1"
	workflowv1spec "github.com/antinvestor/service-trustage/proto/workflow/v1"
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
		log.Debug("database migration completed")
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
	runtimeRepo := repository.NewWorkflowRuntimeRepository(dbPool)
	timerRepo := repository.NewWorkflowTimerRepository(dbPool)
	scopeRepo := repository.NewWorkflowScopeRunRepository(dbPool)
	signalWaitRepo := repository.NewWorkflowSignalWaitRepository(dbPool)
	signalMsgRepo := repository.NewWorkflowSignalMessageRepository(dbPool)
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
		runtimeRepo,
		timerRepo,
		scopeRepo,
		signalWaitRepo,
		signalMsgRepo,
		outputRepo,
		auditRepo,
		defRepo,
		retryPolicyRepo,
		schemaReg,
		metrics,
		rawCache,
	)
	eventRouter := business.NewEventRouter(triggerRepo, defRepo, instanceRepo, auditRepo, engine, metrics)
	workflowBiz := business.NewWorkflowBusiness(defRepo, scheduleRepo, schemaReg)

	sm := svc.SecurityManager()
	auth := sm.GetAuthorizer(ctx)
	tenancyAccessChecker := authorizer.NewTenancyAccessChecker(auth, authz.NamespaceTenancyAccess)
	tenancyAccessInterceptor := connectInterceptors.NewTenancyAccessInterceptor(tenancyAccessChecker)

	// Layer 2: FunctionAccessInterceptor enforces per-RPC permissions automatically.
	workflowSD := workflowv1.File_v1_workflow_proto.Services().ByName("WorkflowService")
	eventSD := eventv1.File_v1_event_proto.Services().ByName("EventService")
	runtimeSD := runtimev1.File_v1_runtime_proto.Services().ByName("RuntimeService")
	signalSD := signalv1.File_v1_signal_proto.Services().ByName("SignalService")
	procMap := permissions.BuildProcedureMap(workflowSD)
	for k, v := range permissions.BuildProcedureMap(eventSD) {
		procMap[k] = v
	}
	for k, v := range permissions.BuildProcedureMap(runtimeSD) {
		procMap[k] = v
	}
	for k, v := range permissions.BuildProcedureMap(signalSD) {
		procMap[k] = v
	}
	svcPerms := permissions.ForService(workflowSD)
	functionChecker := authorizer.NewFunctionChecker(auth, svcPerms.Namespace)
	functionAccessInterceptor := connectInterceptors.NewFunctionAccessInterceptor(functionChecker, procMap)

	defaultInterceptorList, err := connectInterceptors.DefaultList(
		ctx,
		sm.GetAuthenticator(ctx),
		tenancyAccessInterceptor,
		functionAccessInterceptor,
	)
	if err != nil {
		log.WithError(err).Fatal("failed to create connect interceptors")
	}

	// Schedulers (background goroutines with coordinated shutdown).
	// Schedulers process all tenants, so skip tenancy checks on BaseRepository queries.
	schedulerCtx, schedulerCancel := context.WithCancel(security.SkipTenancyChecksOnClaims(ctx))

	var schedulerWg sync.WaitGroup

	dispatchSched := schedulers.NewDispatchScheduler(execRepo, engine, svc.QueueManager(), &cfg, metrics)
	retrySched := schedulers.NewRetryScheduler(execRepo, instanceRepo, &cfg, metrics)
	timerSched := schedulers.NewTimerScheduler(timerRepo, engine, &cfg, metrics)
	signalSched := schedulers.NewSignalScheduler(signalWaitRepo, engine, &cfg)
	scopeSched := schedulers.NewScopeScheduler(scopeRepo, engine, &cfg)
	timeoutSched := schedulers.NewTimeoutScheduler(execRepo, instanceRepo, retryPolicyRepo, auditRepo, &cfg, metrics)
	outboxSched := schedulers.NewOutboxScheduler(eventLogRepo, svc.QueueManager(), &cfg, metrics)

	startScheduler := func(name string, startFn func(context.Context)) {
		schedulerWg.Add(1)

		go func() {
			defer schedulerWg.Done()
			log.Debug("scheduler starting", "scheduler", name)
			startFn(schedulerCtx)
			log.Debug("scheduler stopped", "scheduler", name)
		}()
	}

	cleanupSched := schedulers.NewCleanupScheduler(eventLogRepo, auditRepo, &cfg)
	cronSched := schedulers.NewCronScheduler(scheduleRepo, &cfg)

	startScheduler("dispatch", dispatchSched.Start)
	startScheduler("retry", retrySched.Start)
	startScheduler("timer", timerSched.Start)
	startScheduler("signal", signalSched.Start)
	startScheduler("scope", scopeSched.Start)
	startScheduler("timeout", timeoutSched.Start)
	startScheduler("outbox", outboxSched.Start)
	startScheduler("cleanup", cleanupSched.Start)
	startScheduler("cron", cronSched.Start)

	// HTTP handlers.
	eventRateLimiter := handlers.NewNamedRateLimiter(rawCache, "trustage:event_ingest", cfg.EventIngestRateLimit)
	formRateLimiter := handlers.NewNamedRateLimiter(rawCache, "trustage:form_ingress", cfg.EventIngestRateLimit)
	webhookRateLimiter := handlers.NewNamedRateLimiter(rawCache, "trustage:webhook_ingress", cfg.EventIngestRateLimit)

	formHandler := handlers.NewFormHandler(eventLogRepo, metrics, formRateLimiter)
	webhookReceiveHandler := handlers.NewWebhookReceiveHandler(
		eventLogRepo,
		metrics,
		webhookRateLimiter,
	)

	workflowServer := handlers.NewWorkflowConnectServer(workflowBiz)
	eventServer := handlers.NewEventConnectServer(eventLogRepo, auditRepo, metrics, eventRateLimiter)
	runtimeServer := handlers.NewRuntimeConnectServer(
		instanceRepo,
		execRepo,
		outputRepo,
		auditRepo,
		scopeRepo,
		signalWaitRepo,
		signalMsgRepo,
		engine,
	)
	signalServer := handlers.NewSignalConnectServer(engine)

	workflowPath, workflowHandler := workflowv1connect.NewWorkflowServiceHandler(
		workflowServer,
		connect.WithInterceptors(defaultInterceptorList...),
	)
	eventPath, eventHandler := eventv1connect.NewEventServiceHandler(
		eventServer,
		connect.WithInterceptors(defaultInterceptorList...),
	)
	runtimePath, runtimeHandler := runtimev1connect.NewRuntimeServiceHandler(
		runtimeServer,
		connect.WithInterceptors(defaultInterceptorList...),
	)
	signalPath, signalHandler := signalv1connect.NewSignalServiceHandler(
		signalServer,
		connect.WithInterceptors(defaultInterceptorList...),
	)

	protectedMux := http.NewServeMux()

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
	publicMux.Handle(workflowPath, workflowHandler)
	publicMux.Handle(eventPath, eventHandler)
	publicMux.Handle(runtimePath, runtimeHandler)
	publicMux.Handle(signalPath, signalHandler)
	publicMux.Handle("/openapi/workflow.yaml", handlers.EmbeddedSpecHandler(workflowv1spec.APISpecFile))
	publicMux.Handle("/openapi/event.yaml", handlers.EmbeddedSpecHandler(eventv1spec.APISpecFile))
	publicMux.Handle("/openapi/runtime.yaml", handlers.EmbeddedSpecHandler(runtimev1spec.APISpecFile))
	publicMux.Handle("/openapi/signal.yaml", handlers.EmbeddedSpecHandler(signalv1spec.APISpecFile))
	publicMux.Handle("/", securityhttp.TenancyAccessMiddleware(
		handlers.RequestIDMiddleware(handlers.LimitBodySize(protectedMux)),
		tenancyAccessChecker,
	))

	// Queue workers.
	executionWorker := queues.NewExecutionWorker(engine, defRepo, registry)
	eventRouterWorker := queues.NewEventRouterWorker(eventRouter)

	svc.Init(ctx,
		frame.WithHTTPHandler(publicMux),

		// Permission namespace registration for all proto services.
		frame.WithPermissionRegistration(workflowSD),
		frame.WithPermissionRegistration(eventSD),
		frame.WithPermissionRegistration(runtimeSD),
		frame.WithPermissionRegistration(signalSD),

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
	log.Debug("all schedulers stopped")
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
