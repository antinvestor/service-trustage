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
	"time"

	"connectrpc.com/connect"
	"github.com/antinvestor/common/v2/permissions"
	"github.com/antinvestor/common/v2/timescale"
	"github.com/pitabwire/frame/v2"
	"github.com/pitabwire/frame/v2/config"
	"github.com/pitabwire/frame/v2/datastore"
	"github.com/pitabwire/frame/v2/datastore/pool"
	"github.com/pitabwire/frame/v2/security"
	"github.com/pitabwire/frame/v2/security/authorizer"
	connectInterceptors "github.com/pitabwire/frame/v2/security/interceptors/connect"
	securityhttp "github.com/pitabwire/frame/v2/security/interceptors/httptor"
	"github.com/pitabwire/util"

	appconfig "github.com/antinvestor/service-trustage/apps/default/config"
	"github.com/antinvestor/service-trustage/apps/default/service/authz"
	"github.com/antinvestor/service-trustage/apps/default/service/business"
	appcache "github.com/antinvestor/service-trustage/apps/default/service/cache"
	"github.com/antinvestor/service-trustage/apps/default/service/handlers"
	"github.com/antinvestor/service-trustage/apps/default/service/models"
	"github.com/antinvestor/service-trustage/apps/default/service/queues"
	"github.com/antinvestor/service-trustage/apps/default/service/repository"
	"github.com/antinvestor/service-trustage/apps/default/service/schedulers"
	"github.com/antinvestor/service-trustage/apps/default/service/scheduling"
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

func main() { //nolint:funlen,gocyclo // main wires roles, queues, and progress drivers
	ctx := context.Background()

	cfg, err := config.LoadWithOIDC[appconfig.Config](ctx)
	if err != nil {
		util.Log(ctx).WithError(err).Fatal("failed to load configuration")
	}

	if cfg.Name() == "" {
		cfg.ServiceName = "trustage-api"
	}

	if valErr := cfg.ValidateRoleAndProgress(); valErr != nil {
		util.Log(ctx).WithError(valErr).Fatal("invalid role/progress configuration")
	}

	cfg.ApplyQueueOverrides()
	cfg.DatabaseMaxOpenConnections = cfg.DatabasePoolMaxConns

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
	role := cfg.ParsedRole()
	log.Info("service role and progress driver",
		"service_role", role,
		"progress_driver", cfg.ParsedProgressDriver(),
		"reconcile_in_worker", cfg.ReconcileInWorker(),
	)

	dbManager := svc.DatastoreManager()

	// Migrate-only Job: exit before queues / HTTP.
	if cfg.DoDatabaseMigrate() {
		if migrateErr := repository.Migrate(ctx, dbManager); migrateErr != nil {
			log.WithError(migrateErr).Fatal("database migration failed")
		}
		log.Info("database migration completed (migrate-only mode)")
		return
	}

	// Serving path never migrates (out-of-band Job only).
	// Belt-and-braces: still run AutoMigrate for local/dev when role=all unless explicitly disabled.
	// Production must set DO_DATABASE_MIGRATE Job; serving pods skip to avoid DDL races.
	// Design: serving roles never call Migrate. Strict compliance:
	// no migrate on serve.

	dbPool := dbManager.GetPool(ctx, datastore.DefaultPoolName)
	ensureHypertables(ctx, log, dbPool)

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

	httpClient := svc.HTTPClientManager().Client(ctx)
	registry := setupConnectorRegistry(httpClient)

	rawCache, cacheErr := appcache.SetupCache(cfg.ValkeyCacheURL, cfg.CacheRequireValkey)
	if cacheErr != nil {
		log.WithError(cacheErr).Fatal("cache setup failed")
	}

	metrics := telemetry.NewMetrics()
	schemaReg := business.NewSchemaRegistry(schemaRepo, rawCache)

	defaultTO := time.Duration(cfg.DefaultExecutionTimeoutSeconds) * time.Second
	maxTO := time.Duration(cfg.CloudRunMaxStepSeconds) * time.Second
	timeoutResolver := scheduling.NewTimeoutResolver(defRepo, defaultTO, maxTO)

	delayed, delayedErr := scheduling.NewCloudTasksDelayedPublisherFromTemplate(
		ctx, cfg.CloudTasksDelayedURLTemplate, httpClient, cfg.CloudTasksMaxHorizonHours,
	)
	if delayedErr != nil {
		log.WithError(delayedErr).Warn("delayed publisher init failed; using noop")
		delayed = scheduling.NoopDelayedPublisher{}
	}

	var notifier business.WorkNotifier = scheduling.NoopNotifier{}
	if cfg.EnableWorkNotifier {
		notifier = scheduling.NewWorkNotifier(svc.QueueManager(), &cfg, delayed)
	}

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
		business.WithTimeoutResolver(timeoutResolver),
		business.WithWorkNotifier(notifier),
		business.WithDefaultExecutionTimeout(defaultTO),
		business.WithMaxStepTimeout(maxTO),
	)
	eventRouter := business.NewEventRouter(
		triggerRepo,
		defRepo,
		scheduleRepo,
		instanceRepo,
		auditRepo,
		engine,
		metrics,
		cfg.EventRouterBindingLimit,
		execRepo,
	)
	workflowBiz := business.NewWorkflowBusiness(defRepo, scheduleRepo, schemaReg, metrics)

	sm := svc.SecurityManager()
	auth := sm.GetAuthorizer(ctx)
	tenancyAccessChecker := authorizer.NewTenancyAccessChecker(auth, authz.NamespaceTenancyAccess)
	tenancyAccessInterceptor := connectInterceptors.NewTenancyAccessInterceptor(tenancyAccessChecker)

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

	// Schedulers (constructed for progress driver + wake workers; only started when allowed).
	schedulerCtx, schedulerCancel := context.WithCancel(security.SkipTenancyChecksOnClaims(ctx))
	var schedulerWg sync.WaitGroup

	dispatchSched := schedulers.NewDispatchScheduler(execRepo, engine, svc.QueueManager(), &cfg, metrics)
	retrySched := schedulers.NewRetryScheduler(execRepo, instanceRepo, &cfg, metrics)
	timerSched := schedulers.NewTimerScheduler(timerRepo, engine, &cfg, metrics)
	signalSched := schedulers.NewSignalScheduler(signalWaitRepo, engine, &cfg)
	scopeSched := schedulers.NewScopeScheduler(scopeRepo, engine, &cfg)
	timeoutSched := schedulers.NewTimeoutScheduler(
		execRepo, instanceRepo, retryPolicyRepo, auditRepo, &cfg, metrics,
	)
	outboxSched := schedulers.NewOutboxScheduler(eventLogRepo, svc.QueueManager(), &cfg, metrics)

	// Scheduler pool for cron fire path.
	var cronSched *schedulers.CronScheduler
	var cleanupSched *schedulers.CleanupScheduler
	if cfg.SubscribesReconcile() || cfg.ShouldRunProgressLoops() {
		schedulerPool := pool.NewPool(ctx)
		dbURLs := cfg.GetDatabasePrimaryHostURL()
		if len(dbURLs) == 0 {
			//nolint:gocritic // exitAfterDefer: intentional startup failure
			log.Fatal("no database primary URL available for scheduler pool")
		}
		if poolErr := schedulerPool.AddConnection(ctx,
			pool.WithConnection(dbURLs[0], false),
			pool.WithPreparedStatements(false),
			pool.WithPreferSimpleProtocol(true),
			pool.WithMaxOpen(cfg.SchedulerPoolMaxConns),
		); poolErr != nil {
			log.WithError(poolErr).Fatal("scheduler pool init")
		}
		svc.DatastoreManager().AddPool(ctx, "scheduler", schedulerPool)
		schedulerScheduleRepo := repository.NewScheduleRepository(schedulerPool)
		cronSched = schedulers.NewCronScheduler(schedulerScheduleRepo, &cfg, metrics)
		cleanupSched = schedulers.NewCleanupScheduler(eventLogRepo, auditRepo, &cfg,
			schedulers.WithWorkflowRowRepos(execRepo, timerRepo, signalWaitRepo),
		)
	}

	multiSweep := scheduling.NewMultiSweepRunner(&cfg, map[string]scheduling.SweepFunc{
		"dispatch": dispatchSched.RunOnce,
		"retry":    retrySched.RunOnce,
		"timer":    timerSched.RunOnce,
		"signal":   signalSched.RunOnce,
		"scope":    scopeSched.RunOnce,
		"timeout":  timeoutSched.RunOnce,
		"outbox":   outboxSched.RunUntilDrained,
		"cron": func(c context.Context) int {
			if cronSched == nil {
				return 0
			}
			return cronSched.RunOnce(c)
		},
		"cleanup": func(c context.Context) int {
			if cleanupSched == nil {
				return 0
			}
			return int(cleanupSched.RunOnce(c))
		},
	})

	startBackground := func(name string, startFn func(context.Context)) {
		schedulerWg.Add(1)
		go func() {
			defer schedulerWg.Done()
			log.Debug("background starting", "name", name)
			startFn(schedulerCtx)
			log.Debug("background stopped", "name", name)
		}()
	}

	if cfg.ShouldRunLegacyTickers() {
		startBackground("dispatch", dispatchSched.Start)
		startBackground("retry", retrySched.Start)
		startBackground("timer", timerSched.Start)
		startBackground("signal", signalSched.Start)
		startBackground("scope", scopeSched.Start)
		startBackground("timeout", timeoutSched.Start)
		startBackground("outbox", outboxSched.Start)
		if cleanupSched != nil {
			startBackground("cleanup", cleanupSched.Start)
		}
		if cronSched != nil {
			startBackground("cron", cronSched.Start)
		}
	} else if cfg.ShouldRunMultiSweep() {
		startBackground("multi_sweep", multiSweep.Start)
	}

	// HTTP mux.
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
		if cfg.CacheRequireValkey && rawCache == nil {
			http.Error(w, "cache not ready", http.StatusServiceUnavailable)
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	if role.ExposesAPI(cfg.WorkerExposeAPI) {
		eventRateLimiter := handlers.NewNamedRateLimiter(rawCache, "trustage:event_ingest", cfg.EventIngestRateLimit)
		formRateLimiter := handlers.NewNamedRateLimiter(rawCache, "trustage:form_ingress", cfg.EventIngestRateLimit)
		webhookRateLimiter := handlers.NewNamedRateLimiter(rawCache, "trustage:webhook_ingress", cfg.EventIngestRateLimit)

		var outboxPub *handlers.OutboxPublisher
		if role.PublishesEventIngest() {
			outboxPub = handlers.NewOutboxPublisher(svc.QueueManager(), eventLogRepo, cfg.QueueEventIngestName)
		}

		formHandler := handlers.NewFormHandler(eventLogRepo, metrics, formRateLimiter, outboxPub)
		webhookReceiveHandler := handlers.NewWebhookReceiveHandler(eventLogRepo, metrics, webhookRateLimiter, outboxPub)
		workflowServer := handlers.NewWorkflowConnectServer(workflowBiz)
		eventServer := handlers.NewEventConnectServer(eventLogRepo, auditRepo, metrics, eventRateLimiter, outboxPub)
		runtimeServer := handlers.NewRuntimeConnectServer(
			instanceRepo, execRepo, outputRepo, auditRepo, scopeRepo, signalWaitRepo, signalMsgRepo, engine,
		)
		signalServer := handlers.NewSignalConnectServer(engine)

		workflowPath, workflowHandler := workflowv1connect.NewWorkflowServiceHandler(
			workflowServer, connect.WithInterceptors(defaultInterceptorList...),
		)
		eventPath, eventHandler := eventv1connect.NewEventServiceHandler(
			eventServer, connect.WithInterceptors(defaultInterceptorList...),
		)
		runtimePath, runtimeHandler := runtimev1connect.NewRuntimeServiceHandler(
			runtimeServer, connect.WithInterceptors(defaultInterceptorList...),
		)
		signalPath, signalHandler := signalv1connect.NewSignalServiceHandler(
			signalServer, connect.WithInterceptors(defaultInterceptorList...),
		)

		protectedMux := http.NewServeMux()
		protectedMux.HandleFunc("POST /api/v1/forms/{form_id}/submit", formHandler.SubmitForm)
		protectedMux.HandleFunc("POST /api/v1/webhooks/{webhook_id}", webhookReceiveHandler.ReceiveWebhook)

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
	}

	// Queue registration options.
	var queueOpts []frame.Option
	queueOpts = append(queueOpts,
		frame.WithPermissionRegistration(workflowSD),
		frame.WithPermissionRegistration(eventSD),
		frame.WithPermissionRegistration(runtimeSD),
		frame.WithPermissionRegistration(signalSD),
		frame.WithHTTPHandler(publicMux),
	)

	if role.PublishesEventIngest() {
		queueOpts = append(queueOpts, frame.WithRegisterPublisher(cfg.QueueEventIngestName, cfg.QueueEventIngestURL))
	}
	if role.PublishesExecDispatch() {
		queueOpts = append(queueOpts, frame.WithRegisterPublisher(cfg.QueueExecDispatchName, cfg.QueueExecDispatchURL))
	}

	// Wake publishers (same process may publish delayed wakes via Manager for mem/nats).
	if role.PublishesExecDispatch() || cfg.SubscribesReconcile() {
		for _, pair := range []struct{ name, url string }{
			{cfg.QueueSchedDispatchName, cfg.QueueSchedDispatchURL},
			{cfg.QueueSchedRetryName, cfg.QueueSchedRetryURL},
			{cfg.QueueSchedTimerName, cfg.QueueSchedTimerURL},
			{cfg.QueueSchedTimeoutName, cfg.QueueSchedTimeoutURL},
			{cfg.QueueSchedSignalName, cfg.QueueSchedSignalURL},
			{cfg.QueueSchedReconcileName, cfg.QueueSchedReconcileURL},
			{cfg.QueueSchedCronName, cfg.QueueSchedCronURL},
			{cfg.QueueSchedCleanupName, cfg.QueueSchedCleanupURL},
		} {
			queueOpts = append(queueOpts, frame.WithRegisterPublisher(pair.name, pair.url))
		}
	}

	if role.SubscribesHotPath() {
		executionWorker := queues.NewExecutionWorker(engine, defRepo, registry)
		eventRouterWorker := queues.NewEventRouterWorker(eventRouter)
		queueOpts = append(queueOpts,
			frame.WithRegisterSubscriber(cfg.QueueExecWorkerName, cfg.QueueExecWorkerURL, executionWorker),
			frame.WithRegisterSubscriber(cfg.QueueEventRouterName, cfg.QueueEventRouterURL, eventRouterWorker),
		)
	}

	if cfg.SubscribesReconcile() || role.SubscribesHotPath() {
		// Wake + reconcile subscribers (push or pull via URL scheme).
		queueOpts = append(queueOpts,
			frame.WithRegisterSubscriber(
				cfg.QueueSchedDispatchName, cfg.QueueSchedDispatchURL,
				queues.NewDispatchWakeWorker(dispatchSched, engine, execRepo, svc.QueueManager(), &cfg),
			),
			frame.WithRegisterSubscriber(
				cfg.QueueSchedRetryName, cfg.QueueSchedRetryURL,
				queues.NewRetryWakeWorker(retrySched),
			),
			frame.WithRegisterSubscriber(
				cfg.QueueSchedTimerName, cfg.QueueSchedTimerURL,
				queues.NewTimerWakeWorker(timerSched),
			),
			frame.WithRegisterSubscriber(
				cfg.QueueSchedTimeoutName, cfg.QueueSchedTimeoutURL,
				queues.NewTimeoutWakeWorker(timeoutSched),
			),
			frame.WithRegisterSubscriber(
				cfg.QueueSchedSignalName, cfg.QueueSchedSignalURL,
				queues.NewSignalWakeWorker(signalSched),
			),
		)
	}

	if cfg.SubscribesReconcile() {
		queueOpts = append(queueOpts,
			frame.WithRegisterSubscriber(
				cfg.QueueSchedReconcileName, cfg.QueueSchedReconcileURL,
				queues.NewReconcileWorker(multiSweep),
			),
			frame.WithRegisterSubscriber(
				cfg.QueueSchedCronName, cfg.QueueSchedCronURL,
				queues.NewCronWorker(func(c context.Context) int {
					if cronSched == nil {
						return 0
					}
					return cronSched.RunOnce(c)
				}),
			),
			frame.WithRegisterSubscriber(
				cfg.QueueSchedCleanupName, cfg.QueueSchedCleanupURL,
				queues.NewCleanupWorker(func(c context.Context) int {
					if cleanupSched == nil {
						return 0
					}
					return int(cleanupSched.RunOnce(c))
				}),
			),
		)
	}

	svc.Init(ctx, queueOpts...)

	log.Info("starting trustage orchestrator",
		"port", cfg.ServerPort,
		"service_role", role,
		"progress_driver", cfg.ParsedProgressDriver(),
	)

	if runErr := svc.Run(ctx, cfg.ServerPort); runErr != nil {
		log.WithError(runErr).Fatal("could not run service")
	}

	schedulerCancel()
	schedulerWg.Wait()
	log.Debug("all background workers stopped")
}

func ensureHypertables(ctx context.Context, log *util.LogEntry, dbPool pool.Pool) {
	if tsErr := timescale.Ensure(ctx, dbPool.DB(ctx, false), models.Hypertables()); tsErr != nil {
		log.WithError(tsErr).Warn("timescale hypertable setup skipped — will retry after cluster migration")
	}
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
