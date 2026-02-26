package tests

import (
	"context"
	"time"

	"github.com/pitabwire/frame/datastore"
	framemanager "github.com/pitabwire/frame/datastore/manager"
	"github.com/pitabwire/frame/security"
	"github.com/pitabwire/util"

	"github.com/antinvestor/service-trustage/apps/default/service/models"
	"github.com/antinvestor/service-trustage/apps/default/service/repository"
)

func (s *DefaultServiceSuite) TestWorkflowDefinitionRepository_ListActiveByName() {
	ctx := s.tenantCtx()

	def1 := &models.WorkflowDefinition{Name: "wf", WorkflowVersion: 1, Status: models.WorkflowStatusActive, DSLBlob: "{}"}
	def2 := &models.WorkflowDefinition{Name: "wf", WorkflowVersion: 2, Status: models.WorkflowStatusActive, DSLBlob: "{}"}
	def3 := &models.WorkflowDefinition{Name: "wf", WorkflowVersion: 3, Status: models.WorkflowStatusDraft, DSLBlob: "{}"}
	def4 := &models.WorkflowDefinition{Name: "other", WorkflowVersion: 1, Status: models.WorkflowStatusActive, DSLBlob: "{}"}

	s.Require().NoError(s.defRepo.Create(ctx, def1))
	s.Require().NoError(s.defRepo.Create(ctx, def2))
	s.Require().NoError(s.defRepo.Create(ctx, def3))
	s.Require().NoError(s.defRepo.Create(ctx, def4))

	list, err := s.defRepo.ListActiveByName(ctx, "wf", 10)
	s.Require().NoError(err)
	s.Len(list, 2)
	s.Equal(2, list[0].WorkflowVersion)
	s.Equal(1, list[1].WorkflowVersion)
}

func (s *DefaultServiceSuite) TestWorkflowDefinitionRepository_GetByNameAndVersion() {
	ctx := s.tenantCtx()
	def := &models.WorkflowDefinition{Name: "wf", WorkflowVersion: 1, Status: models.WorkflowStatusActive, DSLBlob: "{}"}
	s.Require().NoError(s.defRepo.Create(ctx, def))

	found, err := s.defRepo.GetByNameAndVersion(ctx, "wf", 1)
	s.Require().NoError(err)
	s.Equal(def.ID, found.ID)
}

func (s *DefaultServiceSuite) TestWorkflowInstanceRepository_CASTransition() {
	ctx := s.tenantCtx()
	inst := &models.WorkflowInstance{WorkflowName: "wf", WorkflowVersion: 1, CurrentState: "a", Status: models.InstanceStatusRunning, Revision: 1}
	s.Require().NoError(s.instanceRepo.Create(ctx, inst))

	err := s.instanceRepo.CASTransition(ctx, inst.ID, "a", 1, "b")
	s.Require().NoError(err)

	updated, err := s.instanceRepo.GetByID(ctx, inst.ID)
	s.Require().NoError(err)
	s.Equal("b", updated.CurrentState)

	err = s.instanceRepo.CASTransition(ctx, inst.ID, "a", 1, "c")
	s.Require().Error(err)
}

func (s *DefaultServiceSuite) TestWorkflowInstanceRepository_UpdateStatus() {
	ctx := s.tenantCtx()
	inst := &models.WorkflowInstance{WorkflowName: "wf", WorkflowVersion: 1, CurrentState: "a", Status: models.InstanceStatusRunning, Revision: 1}
	inst.ID = util.IDString()
	inst.TenantID = testTenantID
	inst.PartitionID = testPartitionID
	s.Require().NoError(s.instanceRepo.Create(ctx, inst))
	s.NotEmpty(inst.ID)

	unscoped := security.SkipTenancyChecksOnClaims(ctx)
	err := s.instanceRepo.UpdateStatus(unscoped, inst.ID, models.InstanceStatusCompleted)
	s.Require().NoError(err)

	var count int64
	s.Require().NoError(s.execRepo.Pool().DB(ctx, false).Raw(
		"SELECT COUNT(1) FROM workflow_instances WHERE id = ?",
		inst.ID,
	).Scan(&count).Error)
	s.Equal(int64(1), count)
}

func (s *DefaultServiceSuite) TestWorkflowOutputRepository_Getters() {
	ctx := s.tenantCtx()
	out := &models.WorkflowStateOutput{ExecutionID: "exec-1", InstanceID: "inst-1", State: "step", SchemaHash: "hash", Payload: "{}"}
	s.Require().NoError(s.outputRepo.Store(ctx, out))

	byExec, err := s.outputRepo.GetByExecution(ctx, "exec-1")
	s.Require().NoError(err)
	s.Equal(out.ID, byExec.ID)

	byInst, err := s.outputRepo.GetByInstanceAndState(ctx, "inst-1", "step")
	s.Require().NoError(err)
	s.Equal(out.ID, byInst.ID)
}

func (s *DefaultServiceSuite) TestSchemaRegistryRepository_LookupByHash() {
	ctx := s.tenantCtx()
	schema := &models.WorkflowStateSchema{
		WorkflowName:    "wf",
		WorkflowVersion: 1,
		State:           "step",
		SchemaType:      models.SchemaTypeInput,
		SchemaHash:      "hash-1",
		SchemaBlob:      []byte(`{"type":"object"}`),
	}
	s.Require().NoError(s.schemaRepo.Store(ctx, schema))

	found, err := s.schemaRepo.LookupByHash(ctx, "hash-1")
	s.Require().NoError(err)
	s.Equal(schema.ID, found.ID)
}

func (s *DefaultServiceSuite) TestSchemaRegistryRepository_StoreDuplicate() {
	ctx := s.tenantCtx()
	schema := &models.WorkflowStateSchema{
		WorkflowName:    "wf",
		WorkflowVersion: 1,
		State:           "step",
		SchemaType:      models.SchemaTypeInput,
		SchemaHash:      "hash-dup",
		SchemaBlob:      []byte(`{"type":"object"}`),
	}
	s.Require().NoError(s.schemaRepo.Store(ctx, schema))

	dup := &models.WorkflowStateSchema{
		WorkflowName:    "wf",
		WorkflowVersion: 1,
		State:           "step",
		SchemaType:      models.SchemaTypeInput,
		SchemaHash:      "hash-dup",
		SchemaBlob:      []byte(`{"type":"object"}`),
	}
	s.Require().NoError(s.schemaRepo.Store(ctx, dup))
}

func (s *DefaultServiceSuite) TestRepository_Migrate() {
	ctx := context.Background()
	mgr, err := framemanager.NewManager(ctx)
	s.Require().NoError(err)
	mgr.AddPool(ctx, datastore.DefaultPoolName, s.dbPool)

	err = repository.Migrate(ctx, mgr)
	s.Require().NoError(err)

	var count int64
	db := s.dbPool.DB(ctx, false)
	s.Require().NoError(db.Raw(
		"SELECT COUNT(1) FROM pg_indexes WHERE tablename = 'workflow_definitions' AND indexname = 'idx_wd_name_version'",
	).Scan(&count).Error)
	s.Equal(int64(1), count)
}

func (s *DefaultServiceSuite) TestEventLogRepository_DeletePublishedBefore() {
	ctx := s.tenantCtx()
	event := &models.EventLog{
		EventType: "user.created",
		Source:    "api",
		Payload:   `{"id":"1"}`,
		Published: true,
	}
	s.Require().NoError(s.eventRepo.Create(ctx, event))

	oldTime := time.Now().Add(-2 * time.Hour)
	s.execRepo.Pool().DB(ctx, false).Exec(
		"UPDATE event_log SET published_at = ? WHERE id = ?",
		oldTime, event.ID,
	)

	deleted, err := s.eventRepo.DeletePublishedBefore(security.SkipTenancyChecksOnClaims(ctx), time.Now().Add(-time.Hour), 10)
	s.Require().NoError(err)
	s.Equal(int64(1), deleted)
}
