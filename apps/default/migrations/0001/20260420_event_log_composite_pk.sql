-- Copyright 2023-2026 Ant Investor Ltd
--
-- Licensed under the Apache License, Version 2.0 (the "License").

-- event_log is promoted to a TimescaleDB hypertable (outbox pattern, high
-- insert volume, retention + compression both matter). TimescaleDB requires
-- the time-partition column to participate in every UNIQUE/PRIMARY
-- constraint, so replace the BaseModel-default PK (id) with a composite
-- (id, created_at).

ALTER TABLE event_log DROP CONSTRAINT IF EXISTS event_log_pkey;
ALTER TABLE event_log ADD PRIMARY KEY (id, created_at);
