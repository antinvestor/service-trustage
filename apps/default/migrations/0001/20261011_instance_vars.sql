-- Copyright 2023-2026 Ant Investor Ltd
--
-- Licensed under the Apache License, Version 2.0 (the "License").

-- Instance vars (v0.5.1).
--
-- GORM AutoMigrate in repository/migrate.go is the source of truth; this file
-- mirrors it and is idempotent so it is safe in either order.
--
-- workflow_instances.vars: every output_var value an instance has set,
--   {"<output_var>": output}, read by templates and expressions as vars.*.
-- workflow_scope_runs.wait_all: the column default was TRUE, which GORM
--   applied whenever a parallel step declared wait_all: false (the Go zero
--   value), so first-branch-wins parallels waited for every branch.

ALTER TABLE workflow_instances ADD COLUMN IF NOT EXISTS vars JSONB DEFAULT '{}';

ALTER TABLE workflow_scope_runs ALTER COLUMN wait_all SET DEFAULT FALSE;
