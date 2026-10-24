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

package repository

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/pitabwire/frame/v3/data"
)

// WorkflowRoutingTarget identifies the workflow version whose trigger
// bindings or signal routes are being reconciled.
type WorkflowRoutingTarget struct {
	WorkflowName    string
	WorkflowVersion int
	TenantID        string
	PartitionID     string
}

const routingRowIDHexLen = 40

// routingRowID is the deterministic id of the index-th routing row of kind
// for target, so re-asserting a version's routing upserts the same rows.
func routingRowID(prefix, kind string, target WorkflowRoutingTarget, index int) string {
	key := fmt.Sprintf(
		"kind=%s|tenant=%s|partition=%s|workflow=%s|version=%d|index=%d",
		kind, target.TenantID, target.PartitionID, target.WorkflowName, target.WorkflowVersion, index,
	)
	sum := sha256.Sum256([]byte(key))

	return prefix + hex.EncodeToString(sum[:])[:routingRowIDHexLen]
}

// stampDeterministicRow fills the BaseModel fields for a row whose id is
// caller-supplied (not an xid), so BeforeCreate does not try to derive the
// creation time from the id.
func stampDeterministicRow(model *data.BaseModel, id string, target WorkflowRoutingTarget, now time.Time) {
	model.ID = id
	model.TenantID = target.TenantID
	model.PartitionID = target.PartitionID
	model.CreatedAt = now
	model.ModifiedAt = now
	model.Version = 1
}
