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

	"github.com/antinvestor/service-trustage/apps/formstore/service/authz"
	"github.com/antinvestor/service-trustage/apps/formstore/service/business"
	"github.com/antinvestor/service-trustage/apps/formstore/service/models"
	"github.com/antinvestor/service-trustage/apps/formstore/service/repository"
	"github.com/antinvestor/service-trustage/apps/formstore/tests/testketo"
)

const (
	testTenantID    = "test-tenant-001"
	testPartitionID = "test-partition-001"
)

type FormStoreSuite struct {
	frametests.FrameBaseTestSuite

	dbPool       pool.Pool
	rawCache     cache.RawCache
	ketoReadURI  string
	ketoWriteURI string
	authz        security.Authorizer
	defRepo      repository.FormDefinitionRepository
	subRepo      repository.FormSubmissionRepository
	biz          business.FormStoreBusiness
}

func TestFormStoreSuite(t *testing.T) {
	suite.Run(t, new(FormStoreSuite))
}

func (s *FormStoreSuite) SetupSuite() {
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
	return s.WithAuthClaims(context.Background(), testTenantID, "test-profile-001")
}

// WithAuthClaims creates a context with fully populated authentication claims.
func (s *FormStoreSuite) WithAuthClaims(ctx context.Context, tenantID, profileID string) context.Context {
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
func (s *FormStoreSuite) SeedTenantAccess(ctx context.Context, tenantID, profileID string) {
	tenancyPath := fmt.Sprintf("%s/%s", tenantID, testPartitionID)
	err := s.authz.WriteTuple(ctx, authz.BuildAccessTuple(tenancyPath, profileID))
	s.Require().NoError(err, "failed to seed tenant access")
}

// SeedTenantRole writes functional permission tuples in the service_trustage
// namespace for the given role.
func (s *FormStoreSuite) SeedTenantRole(ctx context.Context, tenantID, profileID, role string) {
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
