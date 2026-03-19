package events

// NATS subject patterns.
const (
	SubjectExecPrefix  = "wf.exec."
	SubjectEventPrefix = "wf.events."
)

// Audit event type constants.
const (
	EventInstanceCreated   = "instance.created"
	EventInstanceCompleted = "instance.completed"
	EventInstanceFailed    = "instance.failed"
	EventInstanceCancelled = "instance.cancelled"

	EventStateDispatched        = "state.dispatched"
	EventStateRunning           = "state.running"
	EventStateWaiting           = "state.waiting"
	EventStateCompleted         = "state.completed"
	EventStateFailed            = "state.failed"
	EventStateRetried           = "state.retried"
	EventStateTimedOut          = "state.timed_out"
	EventStateContractViolation = "state.contract_violation"

	EventTransitionCommitted = "transition.committed"
	EventTransitionRejected  = "transition.rejected"

	EventSignalReceived   = "signal.received"
	EventMappingEvaluated = "mapping.evaluated"

	EventWorkflowCreated   = "workflow.created"
	EventWorkflowActivated = "workflow.activated"
	EventWorkflowArchived  = "workflow.archived"

	EventTriggerMatched = "trigger.matched"
	EventTriggerDeduped = "trigger.deduplicated"
)

// ExecutionMessage is the minimal message sent via NATS for execution dispatch.
type ExecutionMessage struct {
	ExecutionID string `json:"execution_id"`
}

// IngestedEventMessage is the message published to the event ingestion stream.
type IngestedEventMessage struct {
	EventID     string         `json:"event_id"`
	TenantID    string         `json:"tenant_id"`
	PartitionID string         `json:"partition_id"`
	EventType   string         `json:"event_type"`
	Source      string         `json:"source"`
	Payload     map[string]any `json:"payload"`
}
