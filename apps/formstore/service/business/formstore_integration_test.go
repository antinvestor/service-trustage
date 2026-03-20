//nolint:testpackage // package-local integration tests use unexported business fixtures intentionally.
package business

import (
	"context"
	"encoding/base64"
	"testing"

	"github.com/pitabwire/frame/datastore/pool"
	"github.com/pitabwire/frame/frametests"
	"github.com/pitabwire/frame/frametests/definition"
	"github.com/pitabwire/frame/frametests/deps/testpostgres"
	"github.com/pitabwire/frame/security"
	"github.com/stretchr/testify/suite"

	"github.com/antinvestor/service-trustage/apps/formstore/service/models"
	"github.com/antinvestor/service-trustage/apps/formstore/service/repository"
)

type BusinessSuite struct {
	frametests.FrameBaseTestSuite

	dbPool pool.Pool

	defRepo repository.FormDefinitionRepository
	subRepo repository.FormSubmissionRepository
	biz     FormStoreBusiness
}

func TestBusinessSuite(t *testing.T) {
	suite.Run(t, new(BusinessSuite))
}

func (s *BusinessSuite) SetupSuite() {
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
	s.biz = NewFormStoreBusiness(s.defRepo, s.subRepo, nil)
}

func (s *BusinessSuite) SetupTest() {
	ctx := context.Background()
	s.Require().NoError(s.dbPool.DB(ctx, false).Exec(
		"TRUNCATE form_definitions, form_submissions CASCADE",
	).Error)
}

func (s *BusinessSuite) TearDownSuite() {
	if s.dbPool != nil {
		s.dbPool.Close(context.Background())
	}
	s.FrameBaseTestSuite.TearDownSuite()
}

func (s *BusinessSuite) tenantCtx() context.Context {
	claims := &security.AuthenticationClaims{TenantID: "test-tenant", PartitionID: "test-partition"}
	claims.Subject = "test-user"
	return claims.ClaimsToContext(context.Background())
}

func (s *BusinessSuite) TestDefinitionLifecycleAndValidation() {
	ctx := s.tenantCtx()

	def := &models.FormDefinition{
		FormID:      "loan-application",
		Name:        "Loan Application",
		Description: "Loan form",
		JSONSchema:  `{"type":"object","properties":{"amount":{"type":"number"}}}`,
		Active:      true,
	}
	s.Require().NoError(s.biz.CreateDefinition(ctx, def))

	got, err := s.biz.GetDefinition(ctx, def.ID)
	s.Require().NoError(err)
	s.Equal(def.FormID, got.FormID)

	gotByFormID, err := s.biz.GetDefinitionByFormID(ctx, def.FormID)
	s.Require().NoError(err)
	s.Equal(def.ID, gotByFormID.ID)

	listed, err := s.biz.ListDefinitions(ctx, true, 10, 0)
	s.Require().NoError(err)
	s.Len(listed, 1)

	def.Description = "Updated description"
	s.Require().NoError(s.biz.UpdateDefinition(ctx, def))

	updated, err := s.biz.GetDefinition(ctx, def.ID)
	s.Require().NoError(err)
	s.Equal("Updated description", updated.Description)

	s.Require().NoError(s.biz.DeleteDefinition(ctx, def.ID))
	_, err = s.biz.GetDefinition(ctx, def.ID)
	s.Require().Error(err)

	invalidDef := &models.FormDefinition{
		FormID:     "bad-form",
		Name:       "Bad Form",
		JSONSchema: `{"type":`,
	}
	s.Require().Error(s.biz.CreateDefinition(ctx, invalidDef))
}

func (s *BusinessSuite) TestSubmissionLifecycleValidationAndIdempotency() {
	ctx := s.tenantCtx()

	def := &models.FormDefinition{
		FormID:     "savings-form",
		Name:       "Savings",
		JSONSchema: `{"type":"object","required":["amount"],"properties":{"amount":{"type":"number"}}}`,
		Active:     true,
	}
	s.Require().NoError(s.defRepo.Create(ctx, def))

	sub := &models.FormSubmission{
		FormID:         def.FormID,
		SubmitterID:    "user-1",
		Status:         models.SubmissionStatusPending,
		Data:           `{"amount":100}`,
		IdempotencyKey: "idem-1",
		Metadata:       `{"source":"web"}`,
	}
	s.Require().NoError(s.biz.CreateSubmission(ctx, sub))

	duplicate := &models.FormSubmission{
		FormID:         def.FormID,
		SubmitterID:    "user-1",
		Status:         models.SubmissionStatusPending,
		Data:           `{"amount":100}`,
		IdempotencyKey: "idem-1",
	}
	s.Require().NoError(s.biz.CreateSubmission(ctx, duplicate))
	s.Equal(sub.ID, duplicate.ID)

	got, err := s.biz.GetSubmission(ctx, sub.ID)
	s.Require().NoError(err)
	s.Equal(sub.ID, got.ID)

	listed, err := s.biz.ListSubmissions(ctx, def.FormID, 10, 0)
	s.Require().NoError(err)
	s.Len(listed, 1)

	sub.Status = models.SubmissionStatusComplete
	sub.Data = `{"amount":150}`
	s.Require().NoError(s.biz.UpdateSubmission(ctx, sub))

	updated, err := s.biz.GetSubmission(ctx, sub.ID)
	s.Require().NoError(err)
	s.Equal(models.SubmissionStatusComplete, updated.Status)

	s.Require().NoError(s.biz.DeleteSubmission(ctx, sub.ID))
	_, err = s.biz.GetSubmission(ctx, sub.ID)
	s.Require().Error(err)

	invalid := &models.FormSubmission{
		FormID:      def.FormID,
		SubmitterID: "user-2",
		Status:      models.SubmissionStatusPending,
		Data:        `{"amount":"bad"}`,
	}
	s.Require().Error(s.biz.CreateSubmission(ctx, invalid))
}

func (s *BusinessSuite) TestSubmissionUpdateDeleteAndUploaderPaths() {
	ctx := s.tenantCtx()
	def := &models.FormDefinition{
		FormID:     "upload-form",
		Name:       "Upload",
		JSONSchema: `{"type":"object","properties":{"attachment":{"type":"object"}}}`,
		Active:     true,
	}
	s.Require().NoError(s.defRepo.Create(ctx, def))

	uploadBiz := NewFormStoreBusiness(s.defRepo, s.subRepo, NewFileUploader(
		func(filename, contentType string, data []byte) (string, error) {
			return "mxc://files/" + filename + "/" + contentType + "/" + string(data), nil
		},
	))

	fileData := base64.StdEncoding.EncodeToString([]byte("hello"))
	sub := &models.FormSubmission{
		FormID:      def.FormID,
		SubmitterID: "user-1",
		Status:      models.SubmissionStatusPending,
		Data:        `{"attachment":{"_type":"file","filename":"hello.txt","content_type":"text/plain","data":"` + fileData + `"}}`,
	}
	s.Require().NoError(uploadBiz.CreateSubmission(ctx, sub))
	s.Equal(1, sub.FileCount)
	s.Contains(sub.Data, `"file_ref"`)

	sub.Status = models.SubmissionStatusComplete
	sub.Data = `{"attachment":"data:text/plain;base64,` + fileData + `"}`
	s.Require().NoError(uploadBiz.UpdateSubmission(ctx, sub))
	updated, err := uploadBiz.GetSubmission(ctx, sub.ID)
	s.Require().NoError(err)
	s.Equal(models.SubmissionStatusComplete, updated.Status)
	s.Equal(1, updated.FileCount)
	s.Contains(updated.Data, `"mxc_uri"`)

	invalidStatus := *updated
	invalidStatus.Status = models.FormSubmissionStatus("bad-status")
	s.Require().Error(uploadBiz.UpdateSubmission(ctx, &invalidStatus))

	s.Require().NoError(uploadBiz.DeleteSubmission(ctx, updated.ID))
	_, err = uploadBiz.GetSubmission(ctx, updated.ID)
	s.Require().Error(err)

	s.Require().Error(uploadBiz.DeleteSubmission(ctx, "missing-submission"))

	def.JSONSchema = `{"type":`
	s.Require().Error(s.biz.UpdateDefinition(ctx, def))
	s.Require().Error(s.biz.DeleteDefinition(ctx, "missing-definition"))
}
