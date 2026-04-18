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
		frame.WithDatastore(),
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

	// Authorisation.
	sm := svc.SecurityManager()
	auth := sm.GetAuthorizer(ctx)
	tenancyAccessChecker := authorizer.NewTenancyAccessChecker(auth, authz.NamespaceTenancyAccess)

	// Rate limiter for enqueue operations.
	var enqueueLimiter *handlers.RateLimiter
	if cfg.EnqueueRateLimit > 0 {
		enqueueLimiter = handlers.NewRateLimiter(rawCache, cfg.EnqueueRateLimit)
	}

	// HTTP handlers.
	defHandler := handlers.NewQueueDefinitionHandler(mgr)
	itemHandler := handlers.NewQueueItemHandler(mgr, enqueueLimiter)
	counterHandler := handlers.NewQueueCounterHandler(mgr)
	statsHandler := handlers.NewQueueStatsHandler(stats)

	protectedMux := http.NewServeMux()

	// Queue definition endpoints.
	protectedMux.HandleFunc("POST /api/v1/queues", defHandler.Create)
	protectedMux.HandleFunc("GET /api/v1/queues", defHandler.List)
	protectedMux.HandleFunc("GET /api/v1/queues/{id}", defHandler.Get)
	protectedMux.HandleFunc("PUT /api/v1/queues/{id}", defHandler.Update)
	protectedMux.HandleFunc("DELETE /api/v1/queues/{id}", defHandler.Delete)

	// Queue item endpoints.
	protectedMux.HandleFunc("POST /api/v1/queues/{queue_id}/items", itemHandler.Enqueue)
	protectedMux.HandleFunc("GET /api/v1/queues/{queue_id}/items", itemHandler.ListWaiting)
	protectedMux.HandleFunc("GET /api/v1/items/{id}", itemHandler.Get)
	protectedMux.HandleFunc("GET /api/v1/items/{id}/position", itemHandler.GetPosition)
	protectedMux.HandleFunc("POST /api/v1/items/{id}/cancel", itemHandler.Cancel)
	protectedMux.HandleFunc("POST /api/v1/items/{id}/no-show", itemHandler.NoShow)
	protectedMux.HandleFunc("POST /api/v1/items/{id}/requeue", itemHandler.Requeue)
	protectedMux.HandleFunc("POST /api/v1/items/{id}/transfer", itemHandler.Transfer)

	// Counter endpoints.
	protectedMux.HandleFunc("POST /api/v1/queues/{queue_id}/counters", counterHandler.Create)
	protectedMux.HandleFunc("GET /api/v1/queues/{queue_id}/counters", counterHandler.List)
	protectedMux.HandleFunc("POST /api/v1/counters/{id}/open", counterHandler.Open)
	protectedMux.HandleFunc("POST /api/v1/counters/{id}/close", counterHandler.Close)
	protectedMux.HandleFunc("POST /api/v1/counters/{id}/pause", counterHandler.Pause)
	protectedMux.HandleFunc("POST /api/v1/counters/{id}/call-next", counterHandler.CallNext)
	protectedMux.HandleFunc("POST /api/v1/counters/{id}/begin-service", counterHandler.BeginService)
	protectedMux.HandleFunc("POST /api/v1/counters/{id}/complete-service", counterHandler.CompleteService)

	// Stats endpoint.
	protectedMux.HandleFunc("GET /api/v1/queues/{queue_id}/stats", statsHandler.GetStats)

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

	svc.Init(ctx,
		frame.WithHTTPHandler(publicMux),
	)

	log.Info("starting queue service",
		"port", cfg.ServerPort,
	)

	if runErr := svc.Run(ctx, cfg.ServerPort); runErr != nil {
		log.WithError(runErr).Fatal("could not run service")
	}
}
