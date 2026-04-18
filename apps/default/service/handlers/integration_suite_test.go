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
	"context"
	"encoding/json"
	"strings"
	"testing"

	"connectrpc.com/connect"
	"github.com/pitabwire/frame/cache"
	"github.com/pitabwire/frame/datastore/pool"
	"github.com/pitabwire/frame/frametests"
	"github.com/pitabwire/frame/frametests/definition"
	"github.com/pitabwire/frame/frametests/deps/testpostgres"
	"github.com/pitabwire/frame/security"
	"github.com/stretchr/testify/suite"
	"google.golang.org/protobuf/types/known/structpb"

	"github.com/antinvestor/service-trustage/apps/default/service/business"
	"github.com/antinvestor/service-trustage/apps/default/service/models"
	"github.com/antinvestor/service-trustage/apps/default/service/repository"
	"github.com/antinvestor/service-trustage/pkg/telemetry"
)

const (
	handlerTestTenantID    = "test-tenant-001"
	handlerTestPartitionID = "test-partition-001"
)

var handlerTruncateTables = []string{ //nolint:gochecknoglobals // shared test fixture
	"event_log",
	"workflow_audit_events",
	"workflow_state_outputs",
	"workflow_state_executions",
	"workflow_signal_messages",
	"workflow_signal_waits",
	"workflow_scope_runs",
	"workflow_timers",
	"workflow_state_schemas",
	"workflow_state_mappings",
	"workflow_instances",
	"workflow_definitions",
	"workflow_retry_policies",
	"trigger_bindings",
	"schedule_definitions",
	"connector_configs",
	"connector_credentials",
}

type HandlerSuite struct {
	frametests.FrameBaseTestSuite

	dbPool pool.Pool
	cache  cache.RawCache

	metrics *telemetry.Metrics

	eventRepo      repository.EventLogRepository
	auditRepo      repository.AuditEventRepository
	defRepo        repository.WorkflowDefinitionRepository
	schemaRepo     repository.SchemaRegistryRepository
	instanceRepo   repository.WorkflowInstanceRepository
	execRepo       repository.WorkflowExecutionRepository
	outputRepo     repository.WorkflowOutputRepository
	triggerRepo    repository.TriggerBindingRepository
	retryRepo      repository.RetryPolicyRepository
	scheduleRepo   repository.ScheduleRepository
	timerRepo      repository.WorkflowTimerRepository
	scopeRepo      repository.WorkflowScopeRunRepository
	signalWaitRepo repository.WorkflowSignalWaitRepository
	signalMsgRepo  repository.WorkflowSignalMessageRepository
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
		&models.EventLog{},
		&models.WorkflowAuditEvent{},
		&models.WorkflowStateOutput{},
		&models.WorkflowStateExecution{},
		&models.WorkflowScopeRun{},
		&models.WorkflowSignalWait{},
		&models.WorkflowSignalMessage{},
		&models.WorkflowTimer{},
		&models.WorkflowStateSchema{},
		&models.WorkflowStateMapping{},
		&models.WorkflowInstance{},
		&models.WorkflowDefinition{},
		&models.WorkflowRetryPolicy{},
		&models.TriggerBinding{},
		&models.ScheduleDefinition{},
		&models.ConnectorConfig{},
		&models.ConnectorCredential{},
	))
	s.Require().NoError(db.Exec(
		`CREATE UNIQUE INDEX IF NOT EXISTS uniq_workflow_state_schema
		 ON workflow_state_schemas (tenant_id, workflow_name, workflow_version, state, schema_type)`,
	).Error)
	s.Require().NoError(db.Exec(
		`CREATE UNIQUE INDEX IF NOT EXISTS uniq_workflow_instance_trigger_dedupe
		 ON workflow_instances (tenant_id, partition_id, workflow_name, workflow_version, trigger_event_id)
		 WHERE trigger_event_id IS NOT NULL AND trigger_event_id <> '' AND deleted_at IS NULL`,
	).Error)

	s.dbPool = p
	s.cache = cache.NewInMemoryCache()
	s.metrics = telemetry.NewMetrics()

	s.eventRepo = repository.NewEventLogRepository(p)
	s.auditRepo = repository.NewAuditEventRepository(p)
	s.defRepo = repository.NewWorkflowDefinitionRepository(p)
	s.schemaRepo = repository.NewSchemaRegistryRepository(p)
	s.instanceRepo = repository.NewWorkflowInstanceRepository(p)
	s.execRepo = repository.NewWorkflowExecutionRepository(p)
	s.outputRepo = repository.NewWorkflowOutputRepository(p)
	s.triggerRepo = repository.NewTriggerBindingRepository(p)
	s.retryRepo = repository.NewRetryPolicyRepository(p)
	s.scheduleRepo = repository.NewScheduleRepository(p)
	s.timerRepo = repository.NewWorkflowTimerRepository(p)
	s.scopeRepo = repository.NewWorkflowScopeRunRepository(p)
	s.signalWaitRepo = repository.NewWorkflowSignalWaitRepository(p)
	s.signalMsgRepo = repository.NewWorkflowSignalMessageRepository(p)
}

func (s *HandlerSuite) SetupTest() {
	ctx := context.Background()
	s.Require().NoError(s.dbPool.DB(ctx, false).Exec(
		"TRUNCATE " + strings.Join(handlerTruncateTables, ", ") + " CASCADE",
	).Error)
	s.Require().NoError(s.cache.Flush(ctx))
}

func (s *HandlerSuite) TearDownSuite() {
	ctx := context.Background()
	if s.cache != nil {
		_ = s.cache.Close()
	}
	if s.dbPool != nil {
		s.dbPool.Close(ctx)
	}
	s.FrameBaseTestSuite.TearDownSuite()
}

func (s *HandlerSuite) tenantCtx() context.Context {
	claims := &security.AuthenticationClaims{
		TenantID:    handlerTestTenantID,
		PartitionID: handlerTestPartitionID,
	}
	claims.Subject = "test-user-001"
	return claims.ClaimsToContext(context.Background())
}

func (s *HandlerSuite) schemaRegistry() business.SchemaRegistry {
	return business.NewSchemaRegistry(s.schemaRepo, s.cache)
}

func (s *HandlerSuite) workflowBusiness() business.WorkflowBusiness {
	return business.NewWorkflowBusiness(s.defRepo, s.scheduleRepo, s.schemaRegistry())
}

func (s *HandlerSuite) stateEngine() business.StateEngine {
	return business.NewStateEngine(
		s.instanceRepo,
		s.execRepo,
		repository.NewWorkflowRuntimeRepository(s.dbPool),
		s.timerRepo,
		s.scopeRepo,
		s.signalWaitRepo,
		s.signalMsgRepo,
		s.outputRepo,
		s.auditRepo,
		s.defRepo,
		s.retryRepo,
		s.schemaRegistry(),
		s.metrics,
		s.cache,
	)
}

func (s *HandlerSuite) createWorkflow(ctx context.Context, dslBlob string) *models.WorkflowDefinition {
	def, err := s.workflowBusiness().CreateWorkflow(ctx, []byte(dslBlob))
	s.Require().NoError(err)
	s.Require().NotEmpty(def.ID)
	return def
}

func (s *HandlerSuite) sampleDSL() string {
	return `{
  "version": "1.0",
  "name": "sample-workflow",
  "steps": [
    {
      "id": "log_step",
      "type": "call",
      "call": {
        "action": "log.entry",
        "input": {
          "level": "info",
          "message": "hello"
        }
      }
    }
  ]
}`
}

func mustStructFromJSON(raw string) *structpb.Struct {
	var payload map[string]any
	if err := json.Unmarshal([]byte(raw), &payload); err != nil {
		panic(err)
	}

	value, err := structpb.NewStruct(payload)
	if err != nil {
		panic(err)
	}

	return value
}

func mustStructFromMap(payload map[string]any) *structpb.Struct {
	value, err := structpb.NewStruct(payload)
	if err != nil {
		panic(err)
	}

	return value
}

func connectReq[T any](msg *T) *connect.Request[T] {
	return connect.NewRequest(msg)
}
