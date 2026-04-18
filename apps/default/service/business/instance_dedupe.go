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
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"

	"gorm.io/gorm"
)

const (
	deterministicInstanceIDPrefix = "wfi_"
	deterministicInstanceIDHexLen = 40
)

func deterministicEventInstanceID(
	tenantID, partitionID, workflowName string,
	workflowVersion int,
	triggerEventID string,
) string {
	key := fmt.Sprintf(
		"tenant=%s|partition=%s|workflow=%s|version=%d|event=%s",
		tenantID,
		partitionID,
		workflowName,
		workflowVersion,
		triggerEventID,
	)

	sum := sha256.Sum256([]byte(key))

	return deterministicInstanceIDPrefix + hex.EncodeToString(sum[:])[:deterministicInstanceIDHexLen]
}

func isDuplicateCreateError(err error) bool {
	return errors.Is(err, gorm.ErrDuplicatedKey) || strings.Contains(strings.ToLower(err.Error()), "duplicate")
}
