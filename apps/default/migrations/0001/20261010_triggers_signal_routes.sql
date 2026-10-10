-- Copyright 2023-2026 Ant Investor Ltd
--
-- Licensed under the Apache License, Version 2.0 (the "License").

-- DSL triggers[] and signals[] (v0.5.0).
--
-- GORM AutoMigrate in repository/migrate.go is the source of truth; this file
-- mirrors it and is idempotent so it is safe in either order.
--
-- trigger_bindings.key_expr: the trigger's CEL key expression. Its value over
--   the trigger payload is stored on the instance as correlation_key.
-- workflow_instances.correlation_key: matched by key signal routes.
-- signal_routes: routes ingested events to running instances as signals.

ALTER TABLE trigger_bindings ADD COLUMN IF NOT EXISTS key_expr TEXT;

CREATE INDEX IF NOT EXISTS idx_tb_workflow
    ON trigger_bindings (tenant_id, workflow_name, workflow_version)
    WHERE deleted_at IS NULL;

ALTER TABLE workflow_instances ADD COLUMN IF NOT EXISTS correlation_key TEXT;

CREATE INDEX IF NOT EXISTS idx_wi_correlation
    ON workflow_instances (tenant_id, workflow_name, workflow_version, correlation_key, status)
    WHERE correlation_key <> '' AND deleted_at IS NULL;

CREATE TABLE IF NOT EXISTS signal_routes (
    id               VARCHAR(50) PRIMARY KEY,
    tenant_id        VARCHAR(50),
    partition_id     VARCHAR(50),
    access_id        VARCHAR(50),
    created_by       VARCHAR(50),
    modified_by      VARCHAR(50),
    version          BIGINT DEFAULT 0,
    event_type       TEXT NOT NULL,
    workflow_name    TEXT NOT NULL,
    workflow_version BIGINT NOT NULL,
    signal_name      TEXT NOT NULL,
    filter           TEXT,
    key_expr         TEXT,
    instance_expr    TEXT,
    active           BOOLEAN NOT NULL DEFAULT TRUE,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    modified_at      TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at       TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_sr_event
    ON signal_routes (tenant_id, event_type)
    WHERE active = true AND deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_sr_workflow
    ON signal_routes (tenant_id, workflow_name, workflow_version)
    WHERE deleted_at IS NULL;
