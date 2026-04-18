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

//nolint:testpackage // package-local tests exercise unexported business helpers intentionally.
package business

import (
	"context"
	"encoding/json"
	"time"

	"github.com/pitabwire/frame/security"

	"github.com/antinvestor/service-trustage/apps/default/service/models"
	"github.com/antinvestor/service-trustage/pkg/events"
)

func (s *BusinessSuite) TestWorkflowBusiness_CreateActivateAndSchemaRegistration() {
	ctx := s.tenantCtx()

	def, err := s.workflowBusiness().CreateWorkflow(ctx, []byte(`{
  "version": "1.0",
  "name": "schema-workflow",
  "steps": [
    {
      "id": "seq",
      "type": "sequence",
      "sequence": {
        "steps": [
          {
            "id": "wait",
            "type": "delay",
            "delay": { "duration": "1m" }
          },
          {
            "id": "check",
            "type": "if",
            "if": {
              "expr": "payload.amount > 10",
              "then": [
                {"id": "high", "type": "call", "call": {"action": "log.entry", "input": {}}}
              ],
              "else": [
                {"id": "low", "type": "call", "call": {"action": "log.entry", "input": {}}}
              ]
            }
          }
        ]
      }
    }
  ]
}`))
	s.Require().NoError(err)
	s.Equal(models.WorkflowStatusDraft, def.Status)

	s.Require().NoError(s.workflowBusiness().ActivateWorkflow(ctx, def.ID))
	active, err := s.workflowBusiness().GetWorkflow(ctx, def.ID)
	s.Require().NoError(err)
	s.Equal(models.WorkflowStatusActive, active.Status)

	listed, err := s.workflowBusiness().ListWorkflows(ctx, def.Name, 10)
	s.Require().NoError(err)
	s.Len(listed, 1)

	cases := []struct {
		state      string
		schemaType models.SchemaType
		contains   string
	}{
		{state: "wait", schemaType: models.SchemaTypeOutput, contains: `"type": "object"`},
		{state: "check", schemaType: models.SchemaTypeOutput, contains: `"branch"`},
		{state: "seq", schemaType: models.SchemaTypeInput, contains: `"type": "object"`},
	}

	for _, tc := range cases {
		schema, lookupErr := s.schemaRepo.Lookup(
			context.Background(),
			def.Name,
			def.WorkflowVersion,
			tc.state,
			tc.schemaType,
		)
		s.Require().NoError(lookupErr)
		s.Contains(string(schema.SchemaBlob), tc.contains)
	}
}

func (s *BusinessSuite) TestCreateWorkflow_MaterialisesSchedules() {
	ctx := s.tenantCtx()

	dslBlob := []byte(`{
		"version": "v1",
		"name": "w-sched",
		"steps": [{"id": "s", "type": "delay", "delay": {"duration": "1s"}}],
		"schedules": [
			{"name": "nightly", "cron_expr": "0 2 * * *"},
			{"name": "hourly",  "cron_expr": "0 * * * *"}
		]
	}`)

	def, err := s.workflowBusiness().CreateWorkflow(ctx, dslBlob)
	s.Require().NoError(err)

	// Workflow in DRAFT.
	s.Equal(models.WorkflowStatusDraft, def.Status)

	// Schedules materialised, active=false, next_fire_at=nil.
	out, listErr := s.scheduleRepo.ListByWorkflow(ctx, def.Name, def.WorkflowVersion)
	s.Require().NoError(listErr)
	s.Len(out, 2)
	for _, sch := range out {
		s.False(sch.Active, "new schedule should be inactive")
		s.Nil(sch.NextFireAt, "new schedule should have no next_fire_at")
		s.Equal(def.Name, sch.WorkflowName)
		s.Equal(def.WorkflowVersion, sch.WorkflowVersion)
	}
}

func (s *BusinessSuite) TestCreateWorkflow_InvalidScheduleCronRejected() {
	ctx := s.tenantCtx()

	dslBlob := []byte(`{
		"version": "v1",
		"name": "w-bad-sched",
		"steps": [{"id": "s", "type": "delay", "delay": {"duration": "1s"}}],
		"schedules": [{"name": "bad", "cron_expr": "not-a-cron"}]
	}`)

	_, err := s.workflowBusiness().CreateWorkflow(ctx, dslBlob)
	s.Require().Error(err)
	s.Contains(err.Error(), "cron")
}

func (s *BusinessSuite) TestWorkflowBusiness_RejectsUnsupportedStepType() {
	ctx := s.tenantCtx()

	_, err := s.workflowBusiness().CreateWorkflow(ctx, []byte(`{
  "version": "1.0",
  "name": "bad-workflow",
  "steps": [
    {
      "id": "unknown",
      "type": "unsupported"
    }
  ]
}`))
	s.Require().Error(err)
	s.ErrorIs(err, ErrDSLValidationFailed)
}

func (s *BusinessSuite) TestEventRouter_RouteEventCreatesAndDeduplicatesInstance() {
	ctx := s.tenantCtx()
	def := s.createWorkflow(ctx, s.sampleDSL())
	s.Require().NoError(s.defRepo.Update(ctx, &models.WorkflowDefinition{
		BaseModel:       def.BaseModel,
		Name:            def.Name,
		WorkflowVersion: def.WorkflowVersion,
		Status:          models.WorkflowStatusActive,
		DSLBlob:         def.DSLBlob,
		TimeoutSeconds:  def.TimeoutSeconds,
	}))

	binding := &models.TriggerBinding{
		EventType:       "payment.requested",
		WorkflowName:    def.Name,
		WorkflowVersion: def.WorkflowVersion,
		InputMapping:    "{}",
		Active:          true,
	}
	s.Require().NoError(s.triggerRepo.Create(ctx, binding))

	router := s.eventRouter()
	event := &events.IngestedEventMessage{
		EventID:     "evt-1",
		TenantID:    "test-tenant-001",
		PartitionID: "test-partition-001",
		EventType:   "payment.requested",
		Payload: map[string]any{
			"amount": 150,
		},
	}

	created, err := router.RouteEvent(ctx, event)
	s.Require().NoError(err)
	s.Equal(1, created)

	instance, err := s.instanceRepo.FindByTriggerEvent(ctx, def.Name, def.WorkflowVersion, event.EventID)
	s.Require().NoError(err)
	s.Equal("log_step", instance.CurrentState)

	exec, err := s.execRepo.GetLatestByInstance(ctx, instance.ID)
	s.Require().NoError(err)
	s.Equal(models.ExecStatusPending, exec.Status)

	created, err = router.RouteEvent(ctx, event)
	s.Require().NoError(err)
	s.Equal(0, created)

	audits, err := s.auditRepo.ListByInstance(ctx, instance.ID)
	s.Require().NoError(err)
	s.NotEmpty(audits)
}

func (s *BusinessSuite) TestActivateWorkflow_ActivatesSchedulesAndDeactivatesPrevious() {
	ctx := s.tenantCtx()

	// v1 with two schedules.
	v1DSL := []byte(`{
		"version": "v1",
		"name": "w-activate",
		"steps": [{"id": "s", "type": "delay", "delay": {"duration": "1s"}}],
		"schedules": [
			{"name": "a", "cron_expr": "*/5 * * * *"},
			{"name": "b", "cron_expr": "0 * * * *"}
		]
	}`)
	biz := s.workflowBusiness()
	v1, err := biz.CreateWorkflow(ctx, v1DSL)
	s.Require().NoError(err)
	s.Require().NoError(biz.ActivateWorkflow(ctx, v1.ID))

	// Both v1 schedules must now be active with next_fire_at set.
	v1Scheds, err := s.scheduleRepo.ListByWorkflow(ctx, v1.Name, v1.WorkflowVersion)
	s.Require().NoError(err)
	s.Len(v1Scheds, 2)
	for _, sch := range v1Scheds {
		s.True(sch.Active, "schedule %s must be active after workflow activation", sch.Name)
		s.NotNil(sch.NextFireAt)
		s.True(sch.NextFireAt.After(time.Now()), "next_fire_at must be in the future")
	}

	// v2 of the same workflow — manually created with workflow_version=2 since
	// CreateWorkflow always uses version=1 and the unique constraint prevents duplicates.
	v2Def := &models.WorkflowDefinition{
		Name:            v1.Name,
		WorkflowVersion: 2,
		Status:          models.WorkflowStatusDraft,
		DSLBlob:         string(v1DSL),
	}
	v2Def.CopyPartitionInfo(&v1.BaseModel)
	s.Require().NoError(s.defRepo.Create(ctx, v2Def))

	v2Sched := &models.ScheduleDefinition{
		Name:            "only",
		CronExpr:        "*/10 * * * *",
		WorkflowName:    v2Def.Name,
		WorkflowVersion: v2Def.WorkflowVersion,
		InputPayload:    "{}",
		Active:          false,
	}
	v2Sched.CopyPartitionInfo(&v2Def.BaseModel)
	s.Require().NoError(s.scheduleRepo.Create(ctx, v2Sched))

	s.NotEqual(v1.WorkflowVersion, v2Def.WorkflowVersion, "v2 must have a different workflow_version")

	s.Require().NoError(biz.ActivateWorkflow(ctx, v2Def.ID))

	// v2 schedule now active.
	v2Scheds, err := s.scheduleRepo.ListByWorkflow(ctx, v2Def.Name, v2Def.WorkflowVersion)
	s.Require().NoError(err)
	s.Len(v2Scheds, 1)
	s.True(v2Scheds[0].Active)
	s.NotNil(v2Scheds[0].NextFireAt)

	// v1 schedules must be deactivated.
	v1After, err := s.scheduleRepo.ListByWorkflow(ctx, v1.Name, v1.WorkflowVersion)
	s.Require().NoError(err)
	s.Len(v1After, 2)
	for _, sch := range v1After {
		s.False(sch.Active, "v1 schedule %s must be deactivated after v2 activation", sch.Name)
	}
}

func (s *BusinessSuite) TestSchemaRegistry_RegisterValidateAndCache() {
	ctx := s.tenantCtx()

	schemaBlob := json.RawMessage(`{"type":"object","required":["amount"],"properties":{"amount":{"type":"number"}}}`)
	hash, err := s.schemaRegistry().RegisterSchema(ctx, "payments", 1, "step_a", models.SchemaTypeInput, schemaBlob)
	s.Require().NoError(err)
	s.NotEmpty(hash)

	gotHash, err := s.schemaRegistry().ValidateInput(ctx, "payments", 1, "step_a", json.RawMessage(`{"amount":100}`))
	s.Require().NoError(err)
	s.Equal(hash, gotHash)

	err = s.schemaRegistry().ValidateOutput(ctx, "payments", 1, "step_a", json.RawMessage(`{"bad":true}`))
	s.Require().ErrorIs(err, ErrSchemaNotFound)

	_, err = s.schemaRegistry().ValidateInput(ctx, "payments", 1, "step_a", json.RawMessage(`{"amount":"bad"}`))
	s.Require().Error(err)
	s.ErrorIs(err, ErrInputContractViolation)
}

func (s *BusinessSuite) TestActivateWorkflow_DoesNotAffectOtherTenants() {
	tenantA := s.tenantCtx() // existing helper: returns ctx with "test-tenant-001"

	// Create a workflow in tenant A with a schedule, activate it.
	dslA := []byte(`{
		"version": "v1",
		"name": "w-iso",
		"steps": [{"id": "s", "type": "delay", "delay": {"duration": "1s"}}],
		"schedules": [{"name": "t", "cron_expr": "*/5 * * * *"}]
	}`)
	biz := s.workflowBusiness()
	wA, err := biz.CreateWorkflow(tenantA, dslA)
	s.Require().NoError(err)
	s.Require().NoError(biz.ActivateWorkflow(tenantA, wA.ID))

	// Craft a tenant-B ctx.
	tenantBClaims := &security.AuthenticationClaims{TenantID: "tenant-B", PartitionID: "partition-B"}
	tenantBClaims.Subject = "user-B"
	tenantB := tenantBClaims.ClaimsToContext(context.Background())

	// Create + activate another workflow with SAME name in tenant B.
	wB, err := biz.CreateWorkflow(tenantB, dslA)
	s.Require().NoError(err)
	s.Require().NoError(biz.ActivateWorkflow(tenantB, wB.ID))

	// Tenant A's schedule must remain active — tenant B's activation should not have flipped it off.
	aScheds, err := s.scheduleRepo.ListByWorkflow(tenantA, wA.Name, wA.WorkflowVersion)
	s.Require().NoError(err)
	s.Len(aScheds, 1)
	s.True(aScheds[0].Active, "tenant A's schedule must remain active after tenant B activates a same-name workflow")
	s.NotNil(aScheds[0].NextFireAt)
}

func (s *BusinessSuite) TestGetWorkflowWithSchedules_ReturnsMaterialised() {
	ctx := s.tenantCtx()
	dslBlob := []byte(`{
		"version": "v1",
		"name": "w-get",
		"steps": [{"id": "s", "type": "delay", "delay": {"duration": "1s"}}],
		"schedules": [{"name": "x", "cron_expr": "*/5 * * * *"}]
	}`)
	def, err := s.workflowBusiness().CreateWorkflow(ctx, dslBlob)
	s.Require().NoError(err)

	got, scheds, err := s.workflowBusiness().GetWorkflowWithSchedules(ctx, def.ID)
	s.Require().NoError(err)
	s.Equal(def.ID, got.ID)
	s.Len(scheds, 1)
	s.Equal("x", scheds[0].Name)
}

func (s *BusinessSuite) TestCreateWorkflow_SchedulesAtomicRollback() {
	ctx := s.tenantCtx()

	// Two schedules with the same name — second hits idx_sd_workflow_unique.
	blob := []byte(`{
		"version":"v1","name":"w-dup",
		"steps":[{"id":"s","type":"delay","delay":{"duration":"1s"}}],
		"schedules":[
			{"name":"dup","cron_expr":"*/5 * * * *"},
			{"name":"dup","cron_expr":"0 * * * *"}
		]
	}`)

	_, err := s.workflowBusiness().CreateWorkflow(ctx, blob)
	// v1.1: CreateBatch is atomic — duplicate fails the whole batch.
	s.Require().Error(err, "duplicate schedule names must fail CreateBatch")

	// With CreateWorkflow atomicity via two ordered tx: workflow row was created,
	// schedule batch failed. Retry must be blocked by idx_wd_name_version.
	_, retryErr := s.workflowBusiness().CreateWorkflow(ctx, blob)
	s.Require().Error(retryErr, "retry must be blocked by workflow unique index")
}

func (s *BusinessSuite) TestArchiveWorkflow_DeactivatesSchedulesThenArchives() {
	ctx := s.tenantCtx()

	blob := []byte(`{
		"version":"v1","name":"w-arch",
		"steps":[{"id":"s","type":"delay","delay":{"duration":"1s"}}],
		"schedules":[{"name":"h","cron_expr":"0 * * * *"}]
	}`)
	biz := s.workflowBusiness()
	v1, err := biz.CreateWorkflow(ctx, blob)
	s.Require().NoError(err)
	s.Require().NoError(biz.ActivateWorkflow(ctx, v1.ID))

	s.Require().NoError(biz.ArchiveWorkflow(ctx, v1.ID))

	scheds, err := s.scheduleRepo.ListByWorkflow(ctx, v1.Name, v1.WorkflowVersion)
	s.Require().NoError(err)
	s.Len(scheds, 1)
	s.False(scheds[0].Active)
	s.Nil(scheds[0].NextFireAt)

	got, err := biz.GetWorkflow(ctx, v1.ID)
	s.Require().NoError(err)
	s.Equal(models.WorkflowStatusArchived, got.Status)
}

func (s *BusinessSuite) TestListByWorkflow_TenantIsolated() {
	ctxA := s.tenantCtx()
	blob := []byte(`{
		"version":"v1","name":"w-iso-list",
		"steps":[{"id":"s","type":"delay","delay":{"duration":"1s"}}],
		"schedules":[{"name":"x","cron_expr":"*/5 * * * *"}]
	}`)
	_, err := s.workflowBusiness().CreateWorkflow(ctxA, blob)
	s.Require().NoError(err)

	claimsB := &security.AuthenticationClaims{TenantID: "tenant-B", PartitionID: "partition-B"}
	claimsB.Subject = "user-B"
	ctxB := claimsB.ClaimsToContext(context.Background())

	out, err := s.scheduleRepo.ListByWorkflow(ctxB, "w-iso-list", 1)
	s.Require().NoError(err)
	s.Empty(out, "cross-tenant read must return empty")
}
