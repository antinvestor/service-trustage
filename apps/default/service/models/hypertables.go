// Copyright 2023-2026 Ant Investor Ltd.
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

package models

import (
	"time"

	"github.com/antinvestor/common/timescale"
)

const (
	eventLogChunkInterval = 24 * time.Hour
	eventLogCompressAfter = 3 * 24 * time.Hour
	eventLogRetainFor     = 30 * 24 * time.Hour
)

// Hypertables returns the TimescaleDB configuration for this app's
// append-only tables. Applied idempotently by timescale.Ensure at
// service startup.
func Hypertables() []timescale.Hypertable {
	return []timescale.Hypertable{
		{
			Table:         "event_log",
			TimeColumn:    "created_at",
			ChunkInterval: eventLogChunkInterval,
			SegmentBy:     []string{"partition_id", "event_type"},
			CompressAfter: eventLogCompressAfter,
			RetainFor:     eventLogRetainFor,
		},
	}
}
