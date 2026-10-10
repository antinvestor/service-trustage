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

//nolint:testpackage // package-local worker tests exercise unexported queue internals intentionally.
package queues

import (
	"context"
	"encoding/json"
	"sync"

	"github.com/antinvestor/service-trustage/apps/default/service/models"
	"github.com/antinvestor/service-trustage/connector"
	"github.com/antinvestor/service-trustage/connector/adapters"
)

// replyAdapter records every input it receives and replies with the
// input's "reply" object, so a test controls each step's output.
type replyAdapter struct {
	mu     *sync.Mutex
	inputs map[string][]map[string]any
}

func newReplyAdapter() replyAdapter {
	return replyAdapter{mu: &sync.Mutex{}, inputs: map[string][]map[string]any{}}
}

func (a replyAdapter) Type() string                  { return "script.reply" }
func (a replyAdapter) DisplayName() string           { return "Scripted reply" }
func (a replyAdapter) InputSchema() json.RawMessage  { return json.RawMessage(`{"type":"object"}`) }
func (a replyAdapter) ConfigSchema() json.RawMessage { return json.RawMessage(`{"type":"object"}`) }
func (a replyAdapter) OutputSchema() json.RawMessage { return json.RawMessage(`{"type":"object"}`) }
func (a replyAdapter) Validate(_ *connector.ExecuteRequest) error {
	return nil
}

func (a replyAdapter) Execute(
	_ context.Context,
	req *connector.ExecuteRequest,
) (*connector.ExecuteResponse, *connector.ExecutionError) {
	a.mu.Lock()
	defer a.mu.Unlock()

	tag, _ := req.Input["tag"].(string)
	a.inputs[tag] = append(a.inputs[tag], req.Input)

	reply, _ := req.Input["reply"].(map[string]any)
	if reply == nil {
		reply = map[string]any{}
	}

	return &connector.ExecuteResponse{Output: reply}, nil
}

func (a replyAdapter) input(tag string) map[string]any {
	a.mu.Lock()
	defer a.mu.Unlock()

	list := a.inputs[tag]
	if len(list) == 0 {
		return nil
	}

	return list[len(list)-1]
}

func (s *WorkerSuite) varsRegistry(reply replyAdapter) *connector.Registry {
	registry := connector.NewRegistry()
	s.Require().NoError(registry.Register(adapters.NewLogEntryAdapter()))
	s.Require().NoError(registry.Register(reply))

	return registry
}

// runPending dispatches and executes the instance's pending executions one
// at a time until none is pending. Every step uses a fresh engine and
// worker, so anything a later step reads must come from the database.
func (s *WorkerSuite) runPending(
	ctx context.Context,
	instanceID string,
	registry *connector.Registry,
) []string {
	var states []string

	for range 50 {
		exec, err := s.execRepo.GetLatestByInstance(ctx, instanceID)
		s.Require().NoError(err)
		if exec.Status != models.ExecStatusPending {
			return states
		}

		engine := s.stateEngine()
		cmd, err := engine.Dispatch(ctx, exec)
		s.Require().NoError(err)

		payload, err := json.Marshal(cmd)
		s.Require().NoError(err)
		s.Require().NoError(NewExecutionWorker(engine, s.defRepo, registry).Handle(ctx, nil, payload))

		states = append(states, exec.State)
	}

	s.FailNow("instance did not settle", "instance %s", instanceID)

	return states
}

func (s *WorkerSuite) startInstance(ctx context.Context, dslBlob, input string) *models.WorkflowInstance {
	def, err := s.workflowBusiness().CreateWorkflow(ctx, []byte(dslBlob))
	s.Require().NoError(err)

	spec := struct {
		Steps []struct {
			ID string `json:"id"`
		} `json:"steps"`
	}{}
	s.Require().NoError(json.Unmarshal([]byte(dslBlob), &spec))

	instance := &models.WorkflowInstance{
		WorkflowName:    def.Name,
		WorkflowVersion: def.WorkflowVersion,
		CurrentState:    spec.Steps[0].ID,
		Status:          models.InstanceStatusRunning,
		Revision:        1,
	}
	s.Require().NoError(s.instanceRepo.Create(ctx, instance))

	_, err = s.stateEngine().CreateInitialExecution(ctx, instance, json.RawMessage(input))
	s.Require().NoError(err)

	return instance
}

const varsFlowDSL = `{
  "version": "1.2",
  "name": "vars-flow",
  "steps": [
    {"id": "a", "type": "call", "call": {"action": "script.reply",
      "input": {"tag": "a", "reply": {"x": {"y": "deep"}}}, "output_var": "a"}},
    {"id": "b", "type": "call", "call": {"action": "log.entry",
      "input": {"level": "info", "message": "unrelated step for {{ payload.room }}"}}},
    {"id": "c", "type": "call", "call": {"action": "script.reply",
      "input": {"tag": "c", "seen": "{{ vars.a.x.y }}", "room": "{{ payload.room }}",
                "instance": "{{ metadata.instance_id }}", "reply": {"ok": true}},
      "output_var": "c"},
      "on_success": [
        {"condition": "has(vars.a) && vars.a.x.y == 'deep' && vars.c.ok", "target": "wait"},
        {"condition": "", "target": "d_no"}
      ]},
    {"id": "d_no", "type": "call", "call": {"action": "log.entry",
      "input": {"level": "warn", "message": "vars missing"}}, "on_success": "done"},
    {"id": "wait", "type": "signal_wait", "signal_wait": {"signal_name": "go", "output_var": "sig"}},
    {"id": "e", "type": "call", "call": {"action": "script.reply",
      "input": {"tag": "e", "answer": "{{ vars.sig.answer }}", "still": "{{ vars.a.x.y }}"}}},
    {"id": "check", "type": "if", "if": {
      "expr": "has(vars.a) && vars.sig.answer == 'yes' && payload.room != ''",
      "then": [{"id": "ok", "type": "call", "call": {"action": "log.entry", "input": {"level": "info", "message": "ok"}}}],
      "else": [{"id": "bad", "type": "call", "call": {"action": "log.entry", "input": {"level": "warn", "message": "bad"}}}]
    }},
    {"id": "done", "type": "call", "call": {"action": "log.entry", "input": {"level": "info", "message": "done"}}}
  ]
}`

func (s *WorkerSuite) TestInstanceVars_OutputVarsReachEveryLaterStep() {
	ctx := s.tenantCtx()
	reply := newReplyAdapter()
	registry := s.varsRegistry(reply)

	// The payload value looks like a template: it must reach step c
	// verbatim, which only holds if each template is resolved exactly once.
	instance := s.startInstance(ctx, varsFlowDSL, `{"room":"{{ metadata.instance_id }}"}`)

	states := s.runPending(ctx, instance.ID, registry)
	s.Equal([]string{"a", "b", "c", "wait"}, states, "c's transition must see vars.a and vars.c")

	c := reply.input("c")
	s.Require().NotNil(c)
	s.Equal("deep", c["seen"], "vars.a is available two steps later")
	s.Equal("{{ metadata.instance_id }}", c["room"], "payload is the instance input and resolved once")
	s.Equal(instance.ID, c["instance"])

	delivered, err := s.stateEngine().SendSignal(ctx, instance.ID, "go", json.RawMessage(`{"answer":"yes"}`))
	s.Require().NoError(err)
	s.True(delivered)

	states = s.runPending(ctx, instance.ID, registry)
	s.Equal([]string{"e", "check", "ok", "done"}, states, "the if step must see vars.sig and vars.a")

	e := reply.input("e")
	s.Require().NotNil(e)
	s.Equal("yes", e["answer"], "a signal_wait output_var is available to later steps")
	s.Equal("deep", e["still"])

	stored, err := s.instanceRepo.GetByID(ctx, instance.ID)
	s.Require().NoError(err)
	s.Equal(models.InstanceStatusCompleted, stored.Status)
	s.JSONEq(`{"a":{"x":{"y":"deep"}},"c":{"ok":true},"sig":{"answer":"yes"}}`, stored.Vars)
}

const varsParallelDSL = `{
  "version": "1.2",
  "name": "vars-parallel",
  "steps": [
    {"id": "race", "type": "parallel", "parallel": {"wait_all": false, "steps": [
      {"id": "w1", "type": "signal_wait", "signal_wait": {"signal_name": "s1", "output_var": "got1"}},
      {"id": "w2", "type": "signal_wait", "signal_wait": {"signal_name": "s2", "output_var": "got2"}}
    ]}},
    {"id": "route", "type": "call", "call": {"action": "script.reply",
      "input": {"tag": "route", "v": "{{ vars.got2.v }}", "before": "{{ vars.pre.n }}"}},
      "on_success": [
        {"condition": "has(vars.got2) && vars.got2.v == 2.0 && !has(vars.got1) && vars.pre.n == 1.0", "target": "yes"},
        {"condition": "", "target": "no"}
      ]},
    {"id": "no", "type": "call", "call": {"action": "log.entry", "input": {"level": "warn", "message": "no"}}, "on_success": "end"},
    {"id": "yes", "type": "call", "call": {"action": "log.entry", "input": {"level": "info", "message": "yes"}}},
    {"id": "end", "type": "call", "call": {"action": "log.entry", "input": {"level": "info", "message": "end"}}}
  ]
}`

func (s *WorkerSuite) TestInstanceVars_BranchOutputVarsMergeIntoParent() {
	ctx := s.tenantCtx()
	reply := newReplyAdapter()
	registry := s.varsRegistry(reply)

	instance := s.startInstance(ctx, varsParallelDSL, `{"room":"r-1"}`)

	// Seed a var set before the scope: children inherit it, the parent keeps it.
	s.Require().NoError(s.dbPool.DB(ctx, false).Exec(
		`UPDATE workflow_instances SET vars = '{"pre":{"n":1}}' WHERE id = ?`, instance.ID,
	).Error)

	s.Equal([]string{"race"}, s.runPending(ctx, instance.ID, registry))

	raceExec, err := s.execRepo.GetLatestByInstance(ctx, instance.ID)
	s.Require().NoError(err)
	children, err := s.instanceRepo.ListByParentExecutionID(ctx, raceExec.ID)
	s.Require().NoError(err)
	s.Require().Len(children, 2)
	for _, child := range children {
		s.JSONEq(`{"pre":{"n":1}}`, child.Vars, "branches start with the parent's vars")
		s.runPending(ctx, child.ID, registry)
	}

	delivered, err := s.stateEngine().SendSignal(ctx, children[1].ID, "s2", json.RawMessage(`{"v":2}`))
	s.Require().NoError(err)
	s.True(delivered)

	scope, err := s.scopeRepo.GetByParentExecutionID(ctx, raceExec.ID)
	s.Require().NoError(err)
	s.Require().NoError(s.stateEngine().ReconcileBranchScope(ctx, scope.ID))

	s.Equal([]string{"route", "yes", "end"}, s.runPending(ctx, instance.ID, registry))

	route := reply.input("route")
	s.Require().NotNil(route)
	s.Equal("2", route["v"])
	s.Equal("1", route["before"])
}
