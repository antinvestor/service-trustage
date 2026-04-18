--- Copyright 2023-2026 Ant Investor Ltd
---
--- Licensed under the Apache License, Version 2.0 (the "License");
--- you may not use this file except in compliance with the License.
--- You may obtain a copy of the License at
---
---      http://www.apache.org/licenses/LICENSE-2.0
---
--- Unless required by applicable law or agreed to in writing, software
--- distributed under the License is distributed on an "AS IS" BASIS,
--- WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
--- See the License for the specific language governing permissions and
--- limitations under the License.

-- Reference SQL for Phase 1 schema.
-- Actual migration handled by GORM AutoMigrate + manual index creation in migrate.go.

-- workflow_definitions: versioned workflow templates
CREATE TABLE IF NOT EXISTS workflow_definitions (
    id              VARCHAR(50) PRIMARY KEY,
    tenant_id       VARCHAR(50) NOT NULL,
    partition_id    VARCHAR(50) NOT NULL,
    name            VARCHAR(255) NOT NULL,
    workflow_version INT NOT NULL DEFAULT 1,
    status          VARCHAR(30) NOT NULL DEFAULT 'draft',
    dsl_blob        JSONB NOT NULL,
    input_schema_hash VARCHAR(64),
    timeout_seconds BIGINT DEFAULT 0,
    created_at      TIMESTAMPTZ DEFAULT NOW(),
    modified_at     TIMESTAMPTZ DEFAULT NOW(),
    deleted_at      TIMESTAMPTZ
);

-- workflow_instances: running copies of workflow definitions
CREATE TABLE IF NOT EXISTS workflow_instances (
    id              VARCHAR(50) PRIMARY KEY,
    tenant_id       VARCHAR(50) NOT NULL,
    partition_id    VARCHAR(50) NOT NULL,
    workflow_name   VARCHAR(255) NOT NULL,
    workflow_version INT NOT NULL,
    current_state   VARCHAR(255) NOT NULL,
    status          VARCHAR(30) NOT NULL DEFAULT 'running',
    revision        BIGINT NOT NULL DEFAULT 1,
    trigger_event_id VARCHAR(50),
    metadata        JSONB DEFAULT '{}',
    started_at      TIMESTAMPTZ,
    finished_at     TIMESTAMPTZ,
    created_at      TIMESTAMPTZ DEFAULT NOW(),
    modified_at     TIMESTAMPTZ DEFAULT NOW(),
    deleted_at      TIMESTAMPTZ
);

-- workflow_state_executions: each attempt to execute a state
CREATE TABLE IF NOT EXISTS workflow_state_executions (
    id              VARCHAR(50) PRIMARY KEY,
    tenant_id       VARCHAR(50) NOT NULL,
    partition_id    VARCHAR(50) NOT NULL,
    access_id       VARCHAR(50),
    instance_id     VARCHAR(50) NOT NULL REFERENCES workflow_instances(id),
    state           VARCHAR(255) NOT NULL,
    state_version   INT NOT NULL DEFAULT 1,
    attempt         INT NOT NULL DEFAULT 1,
    status          VARCHAR(30) NOT NULL DEFAULT 'pending',
    execution_token VARCHAR(64) NOT NULL,
    input_schema_hash VARCHAR(64) NOT NULL,
    input_payload   JSONB DEFAULT '{}'::jsonb,
    output_schema_hash VARCHAR(64),
    error_class     VARCHAR(30),
    error_message   TEXT,
    next_retry_at   TIMESTAMPTZ,
    trace_id        VARCHAR(64),
    started_at      TIMESTAMPTZ,
    finished_at     TIMESTAMPTZ,
    created_at      TIMESTAMPTZ DEFAULT NOW(),
    modified_at     TIMESTAMPTZ DEFAULT NOW(),
    version         BIGINT NOT NULL DEFAULT 0,
    deleted_at      TIMESTAMPTZ
);

-- workflow_state_schemas: immutable JSON Schema documents
CREATE TABLE IF NOT EXISTS workflow_state_schemas (
    id              VARCHAR(50) PRIMARY KEY,
    tenant_id       VARCHAR(50) NOT NULL,
    partition_id    VARCHAR(50) NOT NULL,
    access_id       VARCHAR(50),
    workflow_name   VARCHAR(255) NOT NULL,
    workflow_version INT NOT NULL,
    state           VARCHAR(255) NOT NULL,
    schema_type     VARCHAR(10) NOT NULL,
    schema_hash     VARCHAR(64) NOT NULL,
    schema_blob     JSONB NOT NULL,
    created_at      TIMESTAMPTZ DEFAULT NOW(),
    modified_at     TIMESTAMPTZ DEFAULT NOW(),
    version         BIGINT NOT NULL DEFAULT 0,
    deleted_at      TIMESTAMPTZ
);

CREATE UNIQUE INDEX IF NOT EXISTS uniq_workflow_state_schema
    ON workflow_state_schemas (tenant_id, workflow_name, workflow_version, state, schema_type);

-- workflow_state_mappings: data mapping expressions between states
CREATE TABLE IF NOT EXISTS workflow_state_mappings (
    id              VARCHAR(50) PRIMARY KEY,
    tenant_id       VARCHAR(50) NOT NULL,
    partition_id    VARCHAR(50) NOT NULL,
    workflow_name   VARCHAR(255) NOT NULL,
    workflow_version INT NOT NULL,
    from_state      VARCHAR(255) NOT NULL,
    to_state        VARCHAR(255) NOT NULL,
    mapping_expr    JSONB NOT NULL,
    created_at      TIMESTAMPTZ DEFAULT NOW(),
    modified_at     TIMESTAMPTZ DEFAULT NOW(),
    deleted_at      TIMESTAMPTZ
);

-- workflow_state_outputs: validated output of each state execution
CREATE TABLE IF NOT EXISTS workflow_state_outputs (
    id              VARCHAR(50) PRIMARY KEY,
    tenant_id       VARCHAR(50) NOT NULL,
    partition_id    VARCHAR(50) NOT NULL,
    access_id       VARCHAR(50),
    execution_id    VARCHAR(50) NOT NULL,
    instance_id     VARCHAR(50) NOT NULL,
    state           VARCHAR(255) NOT NULL,
    schema_hash     VARCHAR(64) NOT NULL,
    payload         JSONB NOT NULL,
    created_at      TIMESTAMPTZ DEFAULT NOW(),
    modified_at     TIMESTAMPTZ DEFAULT NOW(),
    version         BIGINT NOT NULL DEFAULT 0,
    deleted_at      TIMESTAMPTZ
);

-- workflow_retry_policies: retry configuration per state
CREATE TABLE IF NOT EXISTS workflow_retry_policies (
    id              VARCHAR(50) PRIMARY KEY,
    tenant_id       VARCHAR(50) NOT NULL,
    partition_id    VARCHAR(50) NOT NULL,
    workflow_name   VARCHAR(255) NOT NULL,
    workflow_version INT NOT NULL,
    state           VARCHAR(255) NOT NULL,
    max_attempts    INT NOT NULL DEFAULT 3,
    backoff_strategy VARCHAR(20) NOT NULL DEFAULT 'exponential',
    initial_delay_ms BIGINT NOT NULL DEFAULT 1000,
    max_delay_ms    BIGINT NOT NULL DEFAULT 300000,
    retry_on        TEXT[] NOT NULL DEFAULT ARRAY['retryable', 'external_dependency'],
    created_at      TIMESTAMPTZ DEFAULT NOW(),
    modified_at     TIMESTAMPTZ DEFAULT NOW(),
    deleted_at      TIMESTAMPTZ
);

-- workflow_audit_events: append-only audit trail
CREATE TABLE IF NOT EXISTS workflow_audit_events (
    id              VARCHAR(50) PRIMARY KEY,
    tenant_id       VARCHAR(50) NOT NULL,
    partition_id    VARCHAR(50) NOT NULL,
    access_id       VARCHAR(50),
    instance_id     VARCHAR(50) NOT NULL,
    execution_id    VARCHAR(50),
    event_type      VARCHAR(50) NOT NULL,
    state           VARCHAR(255),
    from_state      VARCHAR(255),
    to_state        VARCHAR(255),
    payload         JSONB DEFAULT '{}',
    trace_id        VARCHAR(64),
    created_at      TIMESTAMPTZ DEFAULT NOW(),
    modified_at     TIMESTAMPTZ DEFAULT NOW(),
    version         BIGINT NOT NULL DEFAULT 0,
    deleted_at      TIMESTAMPTZ
);

-- event_log: outbox pattern for event publishing
CREATE TABLE IF NOT EXISTS event_log (
    id              VARCHAR(50) PRIMARY KEY,
    tenant_id       VARCHAR(50) NOT NULL,
    partition_id    VARCHAR(50) NOT NULL,
    access_id       VARCHAR(50),
    event_type      VARCHAR(100) NOT NULL,
    source          VARCHAR(255),
    payload         JSONB NOT NULL,
    published       BOOLEAN NOT NULL DEFAULT FALSE,
    published_at    TIMESTAMPTZ,
    created_at      TIMESTAMPTZ DEFAULT NOW(),
    modified_at     TIMESTAMPTZ DEFAULT NOW(),
    version         BIGINT NOT NULL DEFAULT 0,
    deleted_at      TIMESTAMPTZ
);

-- trigger_bindings: maps event types to workflow instantiation
CREATE TABLE IF NOT EXISTS trigger_bindings (
    id              VARCHAR(50) PRIMARY KEY,
    tenant_id       VARCHAR(50) NOT NULL,
    partition_id    VARCHAR(50) NOT NULL,
    event_type      VARCHAR(100) NOT NULL,
    event_filter    TEXT,
    workflow_name   VARCHAR(255) NOT NULL,
    workflow_version INT NOT NULL,
    input_mapping   JSONB DEFAULT '{}'::jsonb,
    active          BOOLEAN NOT NULL DEFAULT TRUE,
    created_at      TIMESTAMPTZ DEFAULT NOW(),
    modified_at     TIMESTAMPTZ DEFAULT NOW(),
    deleted_at      TIMESTAMPTZ
);

-- connector_configs: connector adapter configuration
CREATE TABLE IF NOT EXISTS connector_configs (
    id              VARCHAR(50) PRIMARY KEY,
    tenant_id       VARCHAR(50) NOT NULL,
    partition_id    VARCHAR(50) NOT NULL,
    connector_type  VARCHAR(100) NOT NULL,
    name            VARCHAR(255) NOT NULL,
    config          JSONB DEFAULT '{}',
    active          BOOLEAN NOT NULL DEFAULT TRUE,
    created_at      TIMESTAMPTZ DEFAULT NOW(),
    modified_at     TIMESTAMPTZ DEFAULT NOW(),
    deleted_at      TIMESTAMPTZ
);

-- connector_credentials: encrypted credentials
CREATE TABLE IF NOT EXISTS connector_credentials (
    id              VARCHAR(50) PRIMARY KEY,
    tenant_id       VARCHAR(50) NOT NULL,
    partition_id    VARCHAR(50) NOT NULL,
    connector_type  VARCHAR(100) NOT NULL,
    credential_blob TEXT NOT NULL,
    key_version     INT NOT NULL DEFAULT 1,
    created_at      TIMESTAMPTZ DEFAULT NOW(),
    modified_at     TIMESTAMPTZ DEFAULT NOW(),
    deleted_at      TIMESTAMPTZ
);
