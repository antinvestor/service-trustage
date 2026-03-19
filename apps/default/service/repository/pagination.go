package repository

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"gorm.io/gorm"
)

type listCursor struct {
	CreatedAt time.Time `json:"created_at"`
	ID        string    `json:"id"`
}

func normalizeListLimit(limit int) int {
	if limit <= 0 {
		return 50
	}
	if limit > maxListLimit {
		return maxListLimit
	}

	return limit
}

func decodeListCursor(raw string) (*listCursor, error) {
	if strings.TrimSpace(raw) == "" {
		return nil, nil
	}

	blob, err := base64.RawURLEncoding.DecodeString(raw)
	if err != nil {
		return nil, fmt.Errorf("decode cursor: %w", err)
	}

	var cursor listCursor
	if err = json.Unmarshal(blob, &cursor); err != nil {
		return nil, fmt.Errorf("unmarshal cursor: %w", err)
	}
	if cursor.ID == "" || cursor.CreatedAt.IsZero() {
		return nil, errors.New("invalid cursor")
	}

	return &cursor, nil
}

func encodeListCursor(createdAt time.Time, id string) string {
	if id == "" || createdAt.IsZero() {
		return ""
	}

	blob, err := json.Marshal(&listCursor{
		CreatedAt: createdAt.UTC(),
		ID:        id,
	})
	if err != nil {
		return ""
	}

	return base64.RawURLEncoding.EncodeToString(blob)
}

func applyDescendingCreatedAtCursor(query *gorm.DB, raw string) (*gorm.DB, error) {
	cursor, err := decodeListCursor(raw)
	if err != nil {
		return nil, err
	}
	if cursor == nil {
		return query, nil
	}

	return query.Where(
		"(created_at < ?) OR (created_at = ? AND id < ?)",
		cursor.CreatedAt,
		cursor.CreatedAt,
		cursor.ID,
	), nil
}
