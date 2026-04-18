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
	"strings"

	"buf.build/gen/go/antinvestor/files/connectrpc/go/files/v1/filesv1connect"
	"github.com/pitabwire/frame"
	frameclient "github.com/pitabwire/frame/client"
	"github.com/pitabwire/frame/config"
	"github.com/pitabwire/frame/datastore"
	"github.com/pitabwire/frame/security/authorizer"
	securityhttp "github.com/pitabwire/frame/security/interceptors/httptor"
	"github.com/pitabwire/util"

	appconfig "github.com/antinvestor/service-trustage/apps/formstore/config"
	"github.com/antinvestor/service-trustage/apps/formstore/service/authz"
	"github.com/antinvestor/service-trustage/apps/formstore/service/business"
	appcache "github.com/antinvestor/service-trustage/apps/formstore/service/cache"
	"github.com/antinvestor/service-trustage/apps/formstore/service/handlers"
	"github.com/antinvestor/service-trustage/apps/formstore/service/repository"
)

func setupRoutes(
	defHandler *handlers.FormDefinitionHandler,
	subHandler *handlers.FormSubmissionHandler,
	dbManager datastore.Manager,
) (*http.ServeMux, *http.ServeMux) {
	publicMux := http.NewServeMux()
	protectedMux := http.NewServeMux()

	// Form definition endpoints.
	protectedMux.HandleFunc("POST /api/v1/form-definitions", defHandler.Create)
	protectedMux.HandleFunc("GET /api/v1/form-definitions", defHandler.List)
	protectedMux.HandleFunc("GET /api/v1/form-definitions/{id}", defHandler.Get)
	protectedMux.HandleFunc("PUT /api/v1/form-definitions/{id}", defHandler.Update)
	protectedMux.HandleFunc("DELETE /api/v1/form-definitions/{id}", defHandler.Delete)

	// Form submission endpoints.
	protectedMux.HandleFunc("POST /api/v1/forms/{form_id}/submissions", subHandler.Submit)
	protectedMux.HandleFunc("GET /api/v1/forms/{form_id}/submissions", subHandler.ListByForm)
	protectedMux.HandleFunc("GET /api/v1/submissions/{id}", subHandler.Get)
	protectedMux.HandleFunc("PUT /api/v1/submissions/{id}", subHandler.Update)
	protectedMux.HandleFunc("DELETE /api/v1/submissions/{id}", subHandler.Delete)

	// Health checks.
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

	return publicMux, protectedMux
}

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
		fileServiceURL := cfg.FileServiceURL
		if mtlsClient := fileServiceHTTPClient(ctx, cfg); mtlsClient != nil {
			httpClient = mtlsClient
			fileServiceURL = fileServiceEndpoint(cfg)
		}
		filesClient := filesv1connect.NewFilesServiceClient(httpClient, fileServiceURL)
		uploadFn := business.NewFileUploadFunc(filesClient)
		uploader = business.NewFileUploader(uploadFn)
		log.Info("file uploader enabled", "file_service_url", fileServiceURL)
	}

	// Business layer.
	formBiz := business.NewFormStoreBusiness(defRepo, subRepo, uploader)

	// Authorisation.
	sm := svc.SecurityManager()
	auth := sm.GetAuthorizer(ctx)
	tenancyAccessChecker := authorizer.NewTenancyAccessChecker(auth, authz.NamespaceTenancyAccess)

	// Rate limiter for submission operations.
	var submitLimiter *handlers.RateLimiter
	if cfg.SubmissionRateLimit > 0 {
		submitLimiter = handlers.NewRateLimiter(rawCache, cfg.SubmissionRateLimit)
	}

	// HTTP handlers.
	defHandler := handlers.NewFormDefinitionHandler(formBiz)
	subHandler := handlers.NewFormSubmissionHandler(formBiz, submitLimiter)

	publicMux, protectedMux := setupRoutes(defHandler, subHandler, dbManager)
	publicMux.Handle("/", securityhttp.TenancyAccessMiddleware(
		handlers.RequestIDMiddleware(handlers.LimitBodySize(protectedMux, cfg.MaxSubmissionSize)),
		tenancyAccessChecker,
	))

	svc.Init(ctx,
		frame.WithHTTPHandler(publicMux),
	)

	log.Info("starting formstore service",
		"port", cfg.ServerPort,
	)

	if runErr := svc.Run(ctx, cfg.ServerPort); runErr != nil {
		log.WithError(runErr).Fatal("could not run service")
	}
}

func fileServiceEndpoint(cfg appconfig.Config) string {
	if cfg.GetTrustedDomain() == "" || strings.TrimSpace(cfg.FileServiceWorkloadAPITargetPath) == "" {
		return cfg.FileServiceURL
	}

	switch {
	case strings.HasPrefix(cfg.FileServiceURL, "https://"):
		return cfg.FileServiceURL
	case strings.HasPrefix(cfg.FileServiceURL, "http://"):
		return "https://" + strings.TrimPrefix(cfg.FileServiceURL, "http://")
	case strings.Contains(cfg.FileServiceURL, "://"):
		return cfg.FileServiceURL
	default:
		return "https://" + cfg.FileServiceURL
	}
}

func fileServiceHTTPClient(ctx context.Context, cfg appconfig.Config) *http.Client {
	if cfg.GetTrustedDomain() == "" || strings.TrimSpace(cfg.FileServiceWorkloadAPITargetPath) == "" {
		return nil
	}

	return frameclient.NewHTTPClient(
		ctx,
		frameclient.WithHTTPWorkloadAPITargetPath(cfg.FileServiceWorkloadAPITargetPath),
	)
}
