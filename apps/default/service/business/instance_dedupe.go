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
