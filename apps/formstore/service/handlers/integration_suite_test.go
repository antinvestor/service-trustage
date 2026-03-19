package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"testing"

	"github.com/pitabwire/frame/datastore/pool"
	"github.com/pitabwire/frame/frametests"
	"github.com/pitabwire/frame/frametests/definition"
	"github.com/pitabwire/frame/frametests/deps/testpostgres"
	"github.com/pitabwire/frame/security"
	"github.com/stretchr/testify/suite"

	formauthz "github.com/antinvestor/service-trustage/apps/formstore/service/authz"
	"github.com/antinvestor/service-trustage/apps/formstore/service/business"
	"github.com/antinvestor/service-trustage/apps/formstore/service/models"
	"github.com/antinvestor/service-trustage/apps/formstore/service/repository"
)

type HandlerSuite struct {
	frametests.FrameBaseTestSuite

	dbPool pool.Pool

	defRepo repository.FormDefinitionRepository
	subRepo repository.FormSubmissionRepository
	biz     business.FormStoreBusiness
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
	s.Require().NoError(db.AutoMigrate(&models.FormDefinition{}, &models.FormSubmission{}))

	s.dbPool = p
	s.defRepo = repository.NewFormDefinitionRepository(p)
	s.subRepo = repository.NewFormSubmissionRepository(p)
	s.biz = business.NewFormStoreBusiness(s.defRepo, s.subRepo, nil)
}

func (s *HandlerSuite) SetupTest() {
	ctx := context.Background()
	s.Require().NoError(s.dbPool.DB(ctx, false).Exec(
		"TRUNCATE form_definitions, form_submissions CASCADE",
	).Error)
}

func (s *HandlerSuite) TearDownSuite() {
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

type allowAllAuthz struct{}

func (a allowAllAuthz) CanFormDefinitionManage(_ context.Context) error { return nil }
func (a allowAllAuthz) CanFormDefinitionView(_ context.Context) error   { return nil }
func (a allowAllAuthz) CanFormSubmit(_ context.Context) error           { return nil }
func (a allowAllAuthz) CanSubmissionView(_ context.Context) error       { return nil }
func (a allowAllAuthz) CanSubmissionUpdate(_ context.Context) error     { return nil }
func (a allowAllAuthz) CanSubmissionDelete(_ context.Context) error     { return nil }

var _ formauthz.Middleware = allowAllAuthz{}

func encodeBody(v any) *bytes.Reader {
	data, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	return bytes.NewReader(data)
}
