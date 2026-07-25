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

package handlers_test

import (
	"context"
	"testing"

	"github.com/pitabwire/frame/v2/data"

	"github.com/antinvestor/service-trustage/apps/default/service/handlers"
	"github.com/antinvestor/service-trustage/apps/default/service/models"
)

func TestOutboxPublisher_NilSafe(t *testing.T) {
	t.Parallel()
	var p *handlers.OutboxPublisher
	p.PublishAfterCreate(context.Background(), &models.EventLog{})

	p = handlers.NewOutboxPublisher(nil, nil, "event-ingest")
	ev := &models.EventLog{Payload: `{}`}
	ev.BaseModel = data.BaseModel{ID: "evt_test"}
	p.PublishAfterCreate(context.Background(), ev)
}
