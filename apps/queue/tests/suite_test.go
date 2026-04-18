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

package tests_test

import (
	"context"
	"fmt"
	"net/url"
	"testing"

	"github.com/pitabwire/frame/cache"
	"github.com/pitabwire/frame/config"
	"github.com/pitabwire/frame/datastore/pool"
	"github.com/pitabwire/frame/frametests"
	"github.com/pitabwire/frame/frametests/definition"
	"github.com/pitabwire/frame/frametests/deps/testpostgres"
	"github.com/pitabwire/frame/security"
	"github.com/pitabwire/frame/security/authorizer"
	"github.com/pitabwire/util"
	"github.com/stretchr/testify/suite"

	"github.com/antinvestor/service-trustage/apps/queue/service/authz"
	"github.com/antinvestor/service-trustage/apps/queue/service/business"
	"github.com/antinvestor/service-trustage/apps/queue/service/models"
	"github.com/antinvestor/service-trustage/apps/queue/service/repository"
	"github.com/antinvestor/service-trustage/apps/queue/tests/testketo"
)

const (
	testTenantID    = "test-tenant-001"
	testPartitionID = "test-partition-001"
)

type QueueSuite struct {
	frametests.FrameBaseTestSuite

	dbPool       pool.Pool
	rawCache     cache.RawCache
	ketoReadURI  string
	ketoWriteURI string
	authz        security.Authorizer
	defRepo      repository.QueueDefinitionRepository
	itemRepo     repository.QueueItemRepository
	counterRepo  repository.QueueCounterRepository
	stats        business.QueueStatsService
	manager      business.QueueManager
}

func TestQueueSuite(t *testing.T) {
	suite.Run(t, new(QueueSuite))
}

func (s *QueueSuite) SetupSuite() {
	s.InitResourceFunc = func(_ context.Context) []definition.TestResource {
		pg := testpostgres.New()
		keto := testketo.NewWithOpts(
			definition.WithDependancies(pg),
			definition.WithEnableLogging(true),
		)
		return []definition.TestResource{pg, keto}
	}
	s.FrameBaseTestSuite.SetupSuite()

	ctx := context.Background()

	// Extract Keto URIs from the test resource.
	var ketoDep definition.DependancyConn
	for _, res := range s.Resources() {
		if res.Name() == testketo.ImageName {
			ketoDep = res
			break
		}
	}
	s.Require().NotNil(ketoDep, "keto dependency should be available")

	// Write API: default port (4467/tcp, first in port list).
	writeURL, err := url.Parse(string(ketoDep.GetDS(ctx)))
	s.Require().NoError(err)
	s.ketoWriteURI = writeURL.Host

	// Read API: port 4466/tcp (second in port list).
	readPort, err := ketoDep.PortMapping(ctx, "4466/tcp")
	s.Require().NoError(err)
	s.ketoReadURI = fmt.Sprintf("%s:%s", writeURL.Hostname(), readPort)

	// Create Keto authorizer directly (no frame.Service needed).
	// The gRPC-based adapter expects host:port without scheme.
	cfg := &config.ConfigurationDefault{
		AuthorizationServiceReadURI:  s.ketoReadURI,
		AuthorizationServiceWriteURI: s.ketoWriteURI,
	}
	s.authz = authorizer.NewKetoAdapter(cfg, nil)

	// Database setup.
	var pgDep definition.DependancyConn
	for _, res := range s.Resources() {
		if res.GetDS(ctx).IsDB() {
			pgDep = res
			break
		}
	}
	s.Require().NotNil(pgDep, "postgres dependency should be available")

	dsn := pgDep.GetDS(ctx)

	p := pool.NewPool(ctx)
	err = p.AddConnection(ctx,
		pool.WithConnection(string(dsn), false),
		pool.WithPreparedStatements(false),
	)
	s.Require().NoError(err, "connect to test database")

	db := p.DB(ctx, false)
	err = db.AutoMigrate(
		&models.QueueDefinition{},
		&models.QueueItem{},
		&models.QueueCounter{},
	)
	s.Require().NoError(err, "auto-migrate")

	s.dbPool = p
	s.rawCache = cache.NewInMemoryCache()

	s.defRepo = repository.NewQueueDefinitionRepository(p)
	s.itemRepo = repository.NewQueueItemRepository(p)
	s.counterRepo = repository.NewQueueCounterRepository(p)
	s.stats = business.NewQueueStatsService(s.itemRepo, s.counterRepo, s.rawCache, 30)
	s.manager = business.NewQueueManager(s.defRepo, s.itemRepo, s.counterRepo, s.stats)
}

func (s *QueueSuite) SetupTest() {
	// Use background context (not tenant-scoped) so TRUNCATE isn't affected by GORM tenant scoping.
	ctx := context.Background()
	db := s.dbPool.DB(ctx, false)
	db.Exec("TRUNCATE queue_definitions, queue_items, queue_counters CASCADE")
	_ = s.rawCache.Flush(ctx)
}

func (s *QueueSuite) TearDownSuite() {
	ctx := context.Background()
	if s.rawCache != nil {
		_ = s.rawCache.Close()
	}
	if s.dbPool != nil {
		s.dbPool.Close(ctx)
	}
	s.FrameBaseTestSuite.TearDownSuite()
}

func (s *QueueSuite) tenantCtx() context.Context {
	return s.WithAuthClaims(context.Background(), testTenantID, "test-profile-001")
}

// WithAuthClaims creates a context with fully populated authentication claims.
func (s *QueueSuite) WithAuthClaims(ctx context.Context, tenantID, profileID string) context.Context {
	claims := &security.AuthenticationClaims{
		TenantID:    tenantID,
		PartitionID: testPartitionID,
		AccessID:    util.IDString(),
		ContactID:   profileID,
		SessionID:   util.IDString(),
		DeviceID:    "test-device",
	}
	claims.Subject = profileID
	return claims.ClaimsToContext(ctx)
}

// SeedTenantAccess writes a tenancy_access member tuple so the profile can pass
// the TenancyAccessChecker (data access layer).
func (s *QueueSuite) SeedTenantAccess(ctx context.Context, tenantID, profileID string) {
	tenancyPath := fmt.Sprintf("%s/%s", tenantID, testPartitionID)
	err := s.authz.WriteTuple(ctx, authz.BuildAccessTuple(tenancyPath, profileID))
	s.Require().NoError(err, "failed to seed tenant access")
}

// SeedTenantRole writes functional permission tuples in the service_trustage
// namespace for the given role.
func (s *QueueSuite) SeedTenantRole(ctx context.Context, tenantID, profileID, role string) {
	tenancyPath := fmt.Sprintf("%s/%s", tenantID, testPartitionID)
	permissions := authz.RolePermissions()[role]
	tuples := make([]security.RelationTuple, 0, 1+len(permissions))

	tuples = append(tuples, security.RelationTuple{
		Object:   security.ObjectRef{Namespace: authz.NamespaceProfile, ID: tenancyPath},
		Relation: role,
		Subject:  security.SubjectRef{Namespace: authz.NamespaceProfileUser, ID: profileID},
	})
	for _, perm := range permissions {
		tuples = append(tuples, security.RelationTuple{
			Object:   security.ObjectRef{Namespace: authz.NamespaceProfile, ID: tenancyPath},
			Relation: perm,
			Subject:  security.SubjectRef{Namespace: authz.NamespaceProfileUser, ID: profileID},
		})
	}

	err := s.authz.WriteTuples(ctx, tuples)
	s.Require().NoError(err, "failed to seed tenant role")
}

// createQueue is a test helper that creates a queue definition.
func (s *QueueSuite) createQueue(name string, maxCapacity, priorityLevels int) *models.QueueDefinition {
	s.T().Helper()
	ctx := s.tenantCtx()
	def := &models.QueueDefinition{
		Name:           name,
		Active:         true,
		PriorityLevels: priorityLevels,
		MaxCapacity:    maxCapacity,
		SLAMinutes:     30,
	}
	err := s.manager.CreateQueue(ctx, def)
	s.Require().NoError(err)
	s.Require().NotEmpty(def.ID)
	return def
}

// createCounter is a test helper that creates and opens a counter.
func (s *QueueSuite) createAndOpenCounter(queueID, name, staffID string) *models.QueueCounter {
	s.T().Helper()
	ctx := s.tenantCtx()
	counter := &models.QueueCounter{
		QueueID: queueID,
		Name:    name,
	}
	err := s.manager.CreateCounter(ctx, counter)
	s.Require().NoError(err)

	err = s.manager.OpenCounter(ctx, counter.ID, staffID)
	s.Require().NoError(err)
	return counter
}
