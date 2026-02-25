package main

import (
	"context"
	"net/http"

	"buf.build/gen/go/antinvestor/files/connectrpc/go/files/v1/filesv1connect"
	"github.com/pitabwire/frame"
	"github.com/pitabwire/frame/config"
	"github.com/pitabwire/frame/datastore"
	"github.com/pitabwire/util"

	appconfig "github.com/antinvestor/service-trustage/apps/formstore/config"
	"github.com/antinvestor/service-trustage/apps/formstore/service/business"
	appcache "github.com/antinvestor/service-trustage/apps/formstore/service/cache"
	"github.com/antinvestor/service-trustage/apps/formstore/service/handlers"
	"github.com/antinvestor/service-trustage/apps/formstore/service/repository"
)

func main() {
	ctx := context.Background()

	cfg, err := config.LoadWithOIDC[appconfig.Config](ctx)
	if err != nil {
		util.Log(ctx).WithError(err).Fatal("failed to load configuration")
	}

	if cfg.Name() == "" {
		cfg.ServiceName = "formstore-api"
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
	defRepo := repository.NewFormDefinitionRepository(dbPool)
	subRepo := repository.NewFormSubmissionRepository(dbPool)

	// Cache setup (Valkey with in-memory fallback).
	rawCache, cacheErr := appcache.SetupCache(cfg.ValkeyCacheURL)
	if cacheErr != nil {
		log.WithError(cacheErr).Warn("cache setup failed, using in-memory fallback")
		rawCache, _ = appcache.SetupCache("")
	}

	// File uploader — enabled when FileServiceURL is configured.
	var uploader *business.FileUploader
	if cfg.FileServiceURL != "" {
		httpClient := svc.HTTPClientManager().Client(ctx)
		filesClient := filesv1connect.NewFilesServiceClient(httpClient, cfg.FileServiceURL)
		uploadFn := business.NewFileUploadFunc(filesClient)
		uploader = business.NewFileUploader(uploadFn)
		log.Info("file uploader enabled", "file_service_url", cfg.FileServiceURL)
	}

	// Business layer.
	formBiz := business.NewFormStoreBusiness(defRepo, subRepo, uploader)

	// Rate limiter for submission operations.
	var submitLimiter *handlers.RateLimiter
	if cfg.SubmissionRateLimit > 0 {
		submitLimiter = handlers.NewRateLimiter(rawCache, cfg.SubmissionRateLimit)
	}

	// HTTP handlers.
	defHandler := handlers.NewFormDefinitionHandler(formBiz)
	subHandler := handlers.NewFormSubmissionHandler(formBiz, submitLimiter)

	mux := http.NewServeMux()

	// Form definition endpoints.
	mux.HandleFunc("POST /api/v1/form-definitions", defHandler.Create)
	mux.HandleFunc("GET /api/v1/form-definitions", defHandler.List)
	mux.HandleFunc("GET /api/v1/form-definitions/{id}", defHandler.Get)
	mux.HandleFunc("PUT /api/v1/form-definitions/{id}", defHandler.Update)
	mux.HandleFunc("DELETE /api/v1/form-definitions/{id}", defHandler.Delete)

	// Form submission endpoints.
	mux.HandleFunc("POST /api/v1/forms/{form_id}/submissions", subHandler.Submit)
	mux.HandleFunc("GET /api/v1/forms/{form_id}/submissions", subHandler.ListByForm)
	mux.HandleFunc("GET /api/v1/submissions/{id}", subHandler.Get)
	mux.HandleFunc("PUT /api/v1/submissions/{id}", subHandler.Update)
	mux.HandleFunc("DELETE /api/v1/submissions/{id}", subHandler.Delete)

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
		frame.WithHTTPHandler(handlers.RequestIDMiddleware(handlers.LimitBodySize(mux, cfg.MaxSubmissionSize))),
	)

	log.Info("starting formstore service",
		"port", cfg.ServerPort,
	)

	if runErr := svc.Run(ctx, cfg.ServerPort); runErr != nil {
		log.WithError(runErr).Fatal("could not run service")
	}
}
