package tests_test

import (
	"context"
	"testing"

	"github.com/pitabwire/frame/cache"
	"github.com/pitabwire/frame/datastore/pool"
	"github.com/pitabwire/frame/frametests"
	"github.com/pitabwire/frame/frametests/definition"
	"github.com/pitabwire/frame/frametests/deps/testpostgres"
	"github.com/pitabwire/frame/security"
	"github.com/stretchr/testify/suite"

	"github.com/antinvestor/service-trustage/apps/formstore/service/business"
	"github.com/antinvestor/service-trustage/apps/formstore/service/models"
	"github.com/antinvestor/service-trustage/apps/formstore/service/repository"
)

const (
	testTenantID    = "test-tenant-001"
	testPartitionID = "test-partition-001"
)

type FormStoreSuite struct {
	frametests.FrameBaseTestSuite

	dbPool   pool.Pool
	rawCache cache.RawCache
	defRepo  repository.FormDefinitionRepository
	subRepo  repository.FormSubmissionRepository
	biz      business.FormStoreBusiness
}

func TestFormStoreSuite(t *testing.T) {
	suite.Run(t, new(FormStoreSuite))
}

func (s *FormStoreSuite) SetupSuite() {
	s.InitResourceFunc = func(_ context.Context) []definition.TestResource {
		return []definition.TestResource{
			testpostgres.New(),
		}
	}
	s.FrameBaseTestSuite.SetupSuite()

	ctx := context.Background()

	dsn := s.Resources()[0].GetDS(ctx)

	p := pool.NewPool(ctx)
	err := p.AddConnection(ctx,
		pool.WithConnection(string(dsn), false),
		pool.WithPreparedStatements(false),
	)
	s.Require().NoError(err, "connect to test database")

	db := p.DB(ctx, false)
	err = db.AutoMigrate(
		&models.FormDefinition{},
		&models.FormSubmission{},
	)
	s.Require().NoError(err, "auto-migrate")

	s.dbPool = p
	s.rawCache = cache.NewInMemoryCache()

	s.defRepo = repository.NewFormDefinitionRepository(p)
	s.subRepo = repository.NewFormSubmissionRepository(p)
	// No file uploader in tests — nil is safe.
	s.biz = business.NewFormStoreBusiness(s.defRepo, s.subRepo, nil)
}

func (s *FormStoreSuite) SetupTest() {
	// Use background context (not tenant-scoped) so TRUNCATE isn't affected by GORM tenant scoping.
	ctx := context.Background()
	db := s.dbPool.DB(ctx, false)
	db.Exec("TRUNCATE form_definitions, form_submissions CASCADE")
	_ = s.rawCache.Flush(ctx)
}

func (s *FormStoreSuite) TearDownSuite() {
	ctx := context.Background()
	if s.rawCache != nil {
		_ = s.rawCache.Close()
	}
	if s.dbPool != nil {
		s.dbPool.Close(ctx)
	}
	s.FrameBaseTestSuite.TearDownSuite()
}

func (s *FormStoreSuite) tenantCtx() context.Context {
	claims := &security.AuthenticationClaims{
		TenantID:    testTenantID,
		PartitionID: testPartitionID,
	}
	return claims.ClaimsToContext(context.Background())
}
