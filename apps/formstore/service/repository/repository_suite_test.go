//nolint:testpackage // package-local repository tests exercise unexported helpers intentionally.
package repository

import (
	"context"
	"testing"

	"github.com/pitabwire/frame/datastore"
	datastoremanager "github.com/pitabwire/frame/datastore/manager"
	"github.com/pitabwire/frame/datastore/pool"
	"github.com/pitabwire/frame/frametests"
	"github.com/pitabwire/frame/frametests/definition"
	"github.com/pitabwire/frame/frametests/deps/testpostgres"
	"github.com/pitabwire/frame/security"
	"github.com/stretchr/testify/suite"

	"github.com/antinvestor/service-trustage/apps/formstore/service/models"
)

type RepositorySuite struct {
	frametests.FrameBaseTestSuite

	dbPool  pool.Pool
	defRepo FormDefinitionRepository
	subRepo FormSubmissionRepository
}

func TestRepositorySuite(t *testing.T) {
	suite.Run(t, new(RepositorySuite))
}

func (s *RepositorySuite) SetupSuite() {
	s.InitResourceFunc = func(_ context.Context) []definition.TestResource {
		return []definition.TestResource{testpostgres.New()}
	}
	s.FrameBaseTestSuite.SetupSuite()

	ctx := context.Background()
	dsn := s.Resources()[0].GetDS(ctx)
	p := pool.NewPool(ctx)
	s.Require().NoError(p.AddConnection(
		ctx,
		pool.WithConnection(string(dsn), false),
		pool.WithPreparedStatements(false),
	))

	db := p.DB(ctx, false)
	s.Require().NoError(db.AutoMigrate(&models.FormDefinition{}, &models.FormSubmission{}))

	s.dbPool = p
	s.defRepo = NewFormDefinitionRepository(p)
	s.subRepo = NewFormSubmissionRepository(p)
}

func (s *RepositorySuite) SetupTest() {
	ctx := context.Background()
	s.Require().NoError(s.dbPool.DB(ctx, false).Exec(
		"TRUNCATE form_definitions, form_submissions CASCADE",
	).Error)
}

func (s *RepositorySuite) TearDownSuite() {
	if s.dbPool != nil {
		s.dbPool.Close(context.Background())
	}
	s.FrameBaseTestSuite.TearDownSuite()
}

func (s *RepositorySuite) tenantCtx() context.Context {
	claims := &security.AuthenticationClaims{
		TenantID:    "test-tenant",
		PartitionID: "test-partition",
	}
	claims.Subject = "test-user"
	return claims.ClaimsToContext(context.Background())
}

func (s *RepositorySuite) TestFormDefinitionRepository_CRUDAndListing() {
	ctx := s.tenantCtx()

	defs := []*models.FormDefinition{
		{FormID: "loan-application", Name: "Loan", JSONSchema: `{"type":"object"}`, Active: true},
		{FormID: "archived-form", Name: "Archive", JSONSchema: `{}`, Active: true},
	}
	for _, def := range defs {
		s.Require().NoError(s.defRepo.Create(ctx, def))
	}
	s.Require().NoError(s.dbPool.DB(ctx, false).
		Model(&models.FormDefinition{}).
		Where("id = ?", defs[1].ID).
		UpdateColumn("active", false).Error)

	loaded, err := s.defRepo.GetByFormID(ctx, defs[0].FormID)
	s.Require().NoError(err)
	s.Equal(defs[0].ID, loaded.ID)

	cases := []struct {
		name       string
		activeOnly bool
		limit      int
		offset     int
		want       int
	}{
		{name: "all", activeOnly: false, limit: 10, offset: 0, want: 2},
		{name: "active only", activeOnly: true, limit: 10, offset: 0, want: 1},
		{name: "default limit when zero", activeOnly: false, limit: 0, offset: 0, want: 2},
		{name: "negative offset clamps", activeOnly: false, limit: 10, offset: -5, want: 2},
	}

	for _, tc := range cases {
		s.Run(tc.name, func() {
			items, listErr := s.defRepo.List(ctx, tc.activeOnly, tc.limit, tc.offset)
			s.Require().NoError(listErr)
			s.Len(items, tc.want)
		})
	}

	defs[0].Description = "updated"
	s.Require().NoError(s.defRepo.Update(ctx, defs[0]))
	updated, err := s.defRepo.GetByID(ctx, defs[0].ID)
	s.Require().NoError(err)
	s.Equal("updated", updated.Description)

	s.Require().NoError(s.defRepo.SoftDelete(ctx, defs[0]))
	_, err = s.defRepo.GetByID(ctx, defs[0].ID)
	s.Require().Error(err)
}

func (s *RepositorySuite) TestFormSubmissionRepository_CRUDIdempotencyAndListing() {
	ctx := s.tenantCtx()
	def := &models.FormDefinition{FormID: "savings", Name: "Savings", JSONSchema: `{}`, Active: true}
	s.Require().NoError(s.defRepo.Create(ctx, def))

	submissions := []*models.FormSubmission{
		{
			FormID:         def.FormID,
			SubmitterID:    "user-1",
			Status:         models.SubmissionStatusPending,
			Data:           `{"amount":100}`,
			IdempotencyKey: "idem-1",
			Metadata:       `{"source":"web"}`,
		},
		{
			FormID:      def.FormID,
			SubmitterID: "user-2",
			Status:      models.SubmissionStatusComplete,
			Data:        `{"amount":200}`,
			Metadata:    `{"source":"api"}`,
		},
	}
	for _, sub := range submissions {
		s.Require().NoError(s.subRepo.Create(ctx, sub))
	}

	found, err := s.subRepo.FindByIdempotencyKey(ctx, "idem-1")
	s.Require().NoError(err)
	s.Equal(submissions[0].ID, found.ID)

	cases := []struct {
		name   string
		limit  int
		offset int
		want   int
	}{
		{name: "all", limit: 10, offset: 0, want: 2},
		{name: "paged", limit: 1, offset: 1, want: 1},
		{name: "defaults", limit: 0, offset: -1, want: 2},
	}

	for _, tc := range cases {
		s.Run(tc.name, func() {
			items, listErr := s.subRepo.ListByFormID(ctx, def.FormID, tc.limit, tc.offset)
			s.Require().NoError(listErr)
			s.Len(items, tc.want)
		})
	}

	submissions[0].Status = models.SubmissionStatusArchived
	s.Require().NoError(s.subRepo.Update(ctx, submissions[0]))
	updated, err := s.subRepo.GetByID(ctx, submissions[0].ID)
	s.Require().NoError(err)
	s.Equal(models.SubmissionStatusArchived, updated.Status)

	s.Require().NoError(s.subRepo.SoftDelete(ctx, submissions[1]))
	items, err := s.subRepo.ListByFormID(ctx, def.FormID, 10, 0)
	s.Require().NoError(err)
	s.Len(items, 1)
	s.Equal(submissions[0].ID, items[0].ID)
}

func (s *RepositorySuite) TestMigrate_CreatesTablesAndIndexes() {
	ctx := s.tenantCtx()

	manager, err := datastoremanager.NewManager(ctx)
	s.Require().NoError(err)
	manager.AddPool(ctx, datastore.DefaultPoolName, s.dbPool)

	s.Require().NoError(Migrate(ctx, manager))

	db := s.dbPool.DB(ctx, false)
	s.True(db.Migrator().HasTable(&models.FormDefinition{}))
	s.True(db.Migrator().HasTable(&models.FormSubmission{}))

	for _, indexDef := range migrationIndexes() {
		for _, indexName := range indexDef.Names {
			s.True(db.Migrator().HasIndex(indexDef.Model, indexName), indexName)
		}
	}

	s.Equal("form_definitions", formDefinitionIndexModel{}.TableName())
	s.Equal("form_submissions", formSubmissionIndexModel{}.TableName())
}
