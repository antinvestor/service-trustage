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

package business

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/pitabwire/util"

	"github.com/antinvestor/service-trustage/apps/default/service/models"
	"github.com/antinvestor/service-trustage/apps/default/service/repository"
	"github.com/antinvestor/service-trustage/dsl"
	"github.com/antinvestor/service-trustage/pkg/cryptoutil"
)

const signalDeliveryLeaseTTL = 30 * time.Second

func (e *stateEngine) StartSignalWait(
	ctx context.Context,
	cmd *ExecutionCommand,
	step *dsl.StepSpec,
) error {
	if step == nil || step.SignalWait == nil {
		return errors.New("signal_wait step is missing signal_wait configuration")
	}

	exec, err := e.execRepo.GetByID(ctx, cmd.ExecutionID)
	if err != nil {
		return fmt.Errorf("%w: %w", ErrExecutionNotFound, err)
	}

	instance, err := e.instanceRepo.GetByID(ctx, exec.InstanceID)
	if err != nil {
		return fmt.Errorf("load instance: %w", err)
	}

	var timeoutAt *time.Time
	if step.SignalWait.Timeout.Duration > 0 {
		value := time.Now().Add(step.SignalWait.Timeout.Duration)
		timeoutAt = &value
	}

	tokenHash := cryptoutilHash(cmd.ExecutionToken)
	if txErr := e.runtimeRepo.StartSignalWait(ctx, &repository.StartSignalWaitRequest{
		Execution:  exec,
		Instance:   instance,
		TokenHash:  tokenHash,
		SignalName: step.SignalWait.SignalName,
		OutputVar:  step.SignalWait.OutputVar,
		TimeoutAt:  timeoutAt,
		AuditTrace: exec.TraceID,
	}); txErr != nil {
		if errors.Is(txErr, repository.ErrInvalidExecutionToken) {
			return fmt.Errorf("%w: %w", ErrInvalidToken, txErr)
		}
		if errors.Is(txErr, repository.ErrStaleMutation) {
			return ErrStaleExecution
		}
		return txErr
	}

	_, deliverErr := e.tryDeliverSignal(ctx, exec.InstanceID, step.SignalWait.SignalName)
	return deliverErr
}

func (e *stateEngine) SendSignal(
	ctx context.Context,
	instanceID string,
	signalName string,
	payload json.RawMessage,
) (bool, error) {
	if instanceID == "" {
		return false, errors.New("instance_id is required")
	}
	if signalName == "" {
		return false, errors.New("signal_name is required")
	}

	if payload == nil {
		payload = json.RawMessage(`{}`)
	}

	message := &models.WorkflowSignalMessage{
		TargetInstanceID: instanceID,
		SignalName:       signalName,
		Payload:          string(payload),
		Status:           "pending",
	}
	if err := e.signalMsgRepo.Create(ctx, message); err != nil {
		return false, fmt.Errorf("store signal message: %w", err)
	}

	return e.tryDeliverSignal(ctx, instanceID, signalName)
}

func (e *stateEngine) tryDeliverSignal(
	ctx context.Context,
	instanceID string,
	signalName string,
) (bool, error) {
	owner := fmt.Sprintf("signal-delivery:%d", time.Now().UnixNano())
	claim, txErr := e.runtimeRepo.ClaimSignalDelivery(ctx, &repository.ClaimSignalDeliveryRequest{
		InstanceID: instanceID,
		SignalName: signalName,
		Owner:      owner,
		LeaseUntil: time.Now().Add(signalDeliveryLeaseTTL),
	})
	if txErr != nil {
		return false, txErr
	}

	if claim == nil {
		return false, nil
	}
	message := claim.Message
	wait := claim.Wait
	if wait == nil {
		_ = e.signalMsgRepo.ReleaseClaim(ctx, message.ID, owner)
		return false, nil
	}

	output, err := buildSignalOutputPayload(wait.OutputVar, json.RawMessage(message.Payload))
	if err != nil {
		return true, err
	}

	resumeErr := e.ResumeWaitingExecution(ctx, wait.ExecutionID, output)
	if resumeErr != nil && !errors.Is(resumeErr, ErrStaleExecution) {
		util.Log(ctx).WithError(resumeErr).Error("signal delivery could not resume waiting execution",
			"instance_id", instanceID,
			"signal_name", signalName,
			"execution_id", wait.ExecutionID,
		)
		return true, resumeErr
	}

	return true, nil
}

func (e *stateEngine) FailWaitingExecution(
	ctx context.Context,
	executionID string,
	status models.ExecutionStatus,
	failure *CommitError,
) error {
	exec, err := e.execRepo.GetByID(ctx, executionID)
	if err != nil {
		return fmt.Errorf("%w: %w", ErrExecutionNotFound, err)
	}
	if exec.Status != models.ExecStatusWaiting {
		return ErrStaleExecution
	}

	instance, err := e.instanceRepo.GetByID(ctx, exec.InstanceID)
	if err != nil {
		return fmt.Errorf("load instance: %w", err)
	}

	failErr := e.runtimeRepo.FailExecution(ctx, &repository.FailExecutionRequest{
		Execution:      exec,
		Instance:       instance,
		ExpectedStatus: models.ExecStatusWaiting,
		Status:         status,
		ErrorClass:     failure.Class,
		ErrorMessage:   failure.Message,
		AuditTrace:     exec.TraceID,
	})
	if errors.Is(failErr, repository.ErrStaleMutation) {
		return ErrStaleExecution
	}

	return failErr
}

func buildSignalOutputPayload(outputVar string, payload json.RawMessage) (json.RawMessage, error) {
	if payload == nil {
		payload = json.RawMessage(`{}`)
	}
	if outputVar == "" {
		return payload, nil
	}

	var value any
	if err := json.Unmarshal(payload, &value); err != nil {
		return nil, fmt.Errorf("unmarshal signal payload: %w", err)
	}

	output, err := json.Marshal(map[string]any{
		outputVar: value,
	})
	if err != nil {
		return nil, fmt.Errorf("marshal signal output payload: %w", err)
	}

	return output, nil
}

func cryptoutilHash(token string) string {
	return cryptoutil.HashToken(token)
}
