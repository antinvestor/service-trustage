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

//nolint:testpackage // package-local integration suite wires unexported handler dependencies intentionally.
package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"testing"

	"github.com/pitabwire/frame/cache"
	"github.com/pitabwire/frame/datastore/pool"
	"github.com/pitabwire/frame/frametests"
	"github.com/pitabwire/frame/frametests/definition"
	"github.com/pitabwire/frame/frametests/deps/testpostgres"
	"github.com/pitabwire/frame/security"
	"github.com/stretchr/testify/suite"

	"github.com/antinvestor/service-trustage/apps/queue/service/business"
	"github.com/antinvestor/service-trustage/apps/queue/service/models"
	"github.com/antinvestor/service-trustage/apps/queue/service/repository"
)

type HandlerSuite struct {
	frametests.FrameBaseTestSuite

	dbPool   pool.Pool
	rawCache cache.RawCache

	defRepo     repository.QueueDefinitionRepository
	itemRepo    repository.QueueItemRepository
	counterRepo repository.QueueCounterRepository
	stats       business.QueueStatsService
	manager     business.QueueManager
}

func TestHandlerSuite(t *testing.T) {
	suite.Run(t, new(HandlerSuite))
}

func (s *HandlerSuite) SetupSuite() {
	s.InitResourceFunc = func(_ context.Context) []definition.TestResource {
		return []definition.TestResource{testpostgres.New()}
	}
	s.FrameBaseTestSuite.SetupSuite()

	ctx := context.Background()
	dsn := s.Resources()[0].GetDS(ctx)
	p := pool.NewPool(ctx)
	s.Require().NoError(p.AddConnection(ctx,
		pool.WithConnection(string(dsn), false),
		pool.WithPreparedStatements(false),
	))

	db := p.DB(ctx, false)
	s.Require().NoError(db.AutoMigrate(
		&models.QueueDefinition{},
		&models.QueueItem{},
		&models.QueueCounter{},
	))

	s.dbPool = p
	s.rawCache = cache.NewInMemoryCache()
	s.defRepo = repository.NewQueueDefinitionRepository(p)
	s.itemRepo = repository.NewQueueItemRepository(p)
	s.counterRepo = repository.NewQueueCounterRepository(p)
	s.stats = business.NewQueueStatsService(s.itemRepo, s.counterRepo, s.rawCache, 30)
	s.manager = business.NewQueueManager(s.defRepo, s.itemRepo, s.counterRepo, s.stats)
}

func (s *HandlerSuite) SetupTest() {
	ctx := context.Background()
	s.Require().NoError(s.dbPool.DB(ctx, false).Exec(
		"TRUNCATE queue_definitions, queue_items, queue_counters CASCADE",
	).Error)
	s.Require().NoError(s.rawCache.Flush(ctx))
}

func (s *HandlerSuite) TearDownSuite() {
	if s.rawCache != nil {
		_ = s.rawCache.Close()
	}
	if s.dbPool != nil {
		s.dbPool.Close(context.Background())
	}
	s.FrameBaseTestSuite.TearDownSuite()
}

func (s *HandlerSuite) tenantCtx() context.Context {
	claims := &security.AuthenticationClaims{TenantID: "test-tenant", PartitionID: "test-partition"}
	claims.Subject = "test-user"
	return claims.ClaimsToContext(context.Background())
}

func encodeBody(v any) *bytes.Reader {
	data, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	return bytes.NewReader(data)
}
