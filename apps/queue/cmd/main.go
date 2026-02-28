package main

import (
	"context"
	"net/http"

	"github.com/pitabwire/frame"
	"github.com/pitabwire/frame/config"
	"github.com/pitabwire/frame/datastore"
	"github.com/pitabwire/frame/security/authorizer"
	securityhttp "github.com/pitabwire/frame/security/interceptors/httptor"
	"github.com/pitabwire/util"

	appconfig "github.com/antinvestor/service-trustage/apps/queue/config"
	"github.com/antinvestor/service-trustage/apps/queue/service/authz"
	"github.com/antinvestor/service-trustage/apps/queue/service/business"
	appcache "github.com/antinvestor/service-trustage/apps/queue/service/cache"
	"github.com/antinvestor/service-trustage/apps/queue/service/handlers"
	"github.com/antinvestor/service-trustage/apps/queue/service/repository"
)

//nolint:funlen // wiring and dependency setup is intentionally verbose
func main() {
	ctx := context.Background()

	cfg, err := config.LoadWithOIDC[appconfig.Config](ctx)
	if err != nil {
		util.Log(ctx).WithError(err).Fatal("failed to load configuration")
	}

	if cfg.Name() == "" {
		cfg.ServiceName = "queue-api"
	}

	ctx, svc := frame.NewServiceWithContext(
		ctx,
		frame.WithName(cfg.Name()),
		frame.WithConfig(&cfg),
		frame.WithRegisterServerOauth2Client(),
		frame.WithDatastore(),
	)
	defer svc.Stop(ctx)

	log := svc.Log(ctx)

	// Database setup.
	dbManager := svc.DatastoreManager()

	if migrateErr := repository.Migrate(ctx, dbManager); migrateErr != nil {
		log.WithError(migrateErr).Fatal("database migration failed")
	}

	dbPool := dbManager.GetPool(ctx, datastore.DefaultPoolName)

	// Repositories.
	defRepo := repository.NewQueueDefinitionRepository(dbPool)
	itemRepo := repository.NewQueueItemRepository(dbPool)
	counterRepo := repository.NewQueueCounterRepository(dbPool)

	// Cache setup (Valkey with in-memory fallback).
	rawCache, cacheErr := appcache.SetupCache(cfg.ValkeyCacheURL)
	if cacheErr != nil {
		log.WithError(cacheErr).Warn("cache setup failed, using in-memory fallback")
		rawCache, _ = appcache.SetupCache("")
	}

	// Business layer.
	stats := business.NewQueueStatsService(itemRepo, counterRepo, rawCache, cfg.StatsCacheTTLSeconds)
	mgr := business.NewQueueManager(defRepo, itemRepo, counterRepo, stats)

	// Authorisation middleware.
	sm := svc.SecurityManager()
	auth := sm.GetAuthorizer(ctx)
	authzMiddleware := authz.NewMiddleware(auth)
	tenancyAccessChecker := authorizer.NewTenancyAccessChecker(auth, authz.NamespaceTenancyAccess)

	// Rate limiter for enqueue operations.
	var enqueueLimiter *handlers.RateLimiter
	if cfg.EnqueueRateLimit > 0 {
		enqueueLimiter = handlers.NewRateLimiter(rawCache, cfg.EnqueueRateLimit)
	}

	// HTTP handlers.
	defHandler := handlers.NewQueueDefinitionHandler(mgr, authzMiddleware)
	itemHandler := handlers.NewQueueItemHandler(mgr, authzMiddleware, enqueueLimiter)
	counterHandler := handlers.NewQueueCounterHandler(mgr, authzMiddleware)
	statsHandler := handlers.NewQueueStatsHandler(stats, authzMiddleware)

	mux := http.NewServeMux()

	// Queue definition endpoints.
	mux.HandleFunc("POST /api/v1/queues", defHandler.Create)
	mux.HandleFunc("GET /api/v1/queues", defHandler.List)
	mux.HandleFunc("GET /api/v1/queues/{id}", defHandler.Get)
	mux.HandleFunc("PUT /api/v1/queues/{id}", defHandler.Update)
	mux.HandleFunc("DELETE /api/v1/queues/{id}", defHandler.Delete)

	// Queue item endpoints.
	mux.HandleFunc("POST /api/v1/queues/{queue_id}/items", itemHandler.Enqueue)
	mux.HandleFunc("GET /api/v1/queues/{queue_id}/items", itemHandler.ListWaiting)
	mux.HandleFunc("GET /api/v1/items/{id}", itemHandler.Get)
	mux.HandleFunc("GET /api/v1/items/{id}/position", itemHandler.GetPosition)
	mux.HandleFunc("POST /api/v1/items/{id}/cancel", itemHandler.Cancel)
	mux.HandleFunc("POST /api/v1/items/{id}/no-show", itemHandler.NoShow)
	mux.HandleFunc("POST /api/v1/items/{id}/requeue", itemHandler.Requeue)
	mux.HandleFunc("POST /api/v1/items/{id}/transfer", itemHandler.Transfer)

	// Counter endpoints.
	mux.HandleFunc("POST /api/v1/queues/{queue_id}/counters", counterHandler.Create)
	mux.HandleFunc("GET /api/v1/queues/{queue_id}/counters", counterHandler.List)
	mux.HandleFunc("POST /api/v1/counters/{id}/open", counterHandler.Open)
	mux.HandleFunc("POST /api/v1/counters/{id}/close", counterHandler.Close)
	mux.HandleFunc("POST /api/v1/counters/{id}/pause", counterHandler.Pause)
	mux.HandleFunc("POST /api/v1/counters/{id}/call-next", counterHandler.CallNext)
	mux.HandleFunc("POST /api/v1/counters/{id}/begin-service", counterHandler.BeginService)
	mux.HandleFunc("POST /api/v1/counters/{id}/complete-service", counterHandler.CompleteService)

	// Stats endpoint.
	mux.HandleFunc("GET /api/v1/queues/{queue_id}/stats", statsHandler.GetStats)

	// Health checks.
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})
	mux.HandleFunc("GET /readyz", func(w http.ResponseWriter, r *http.Request) {
		pool := dbManager.GetPool(r.Context(), datastore.DefaultPoolName)
		if pool == nil {
			http.Error(w, "database not ready", http.StatusServiceUnavailable)
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	svc.Init(ctx,
		frame.WithHTTPHandler(securityhttp.TenancyAccessMiddleware(
			handlers.RequestIDMiddleware(handlers.LimitBodySize(mux)),
			tenancyAccessChecker,
		)),
	)

	log.Info("starting queue service",
		"port", cfg.ServerPort,
	)

	if runErr := svc.Run(ctx, cfg.ServerPort); runErr != nil {
		log.WithError(runErr).Fatal("could not run service")
	}
}
