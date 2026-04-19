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

package models_test

import (
	"strings"
	"testing"

	"github.com/antinvestor/service-trustage/apps/default/service/models"
)

// TestEventLog_BeforeCreate_PayloadSizeGuard verifies that the GORM BeforeCreate
// hook rejects payloads that exceed MaxEventLogPayloadBytes and accepts those
// that are within the limit.
func TestEventLog_BeforeCreate_PayloadSizeGuard(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name        string
		payloadSize int
		wantErr     bool
	}{
		{
			name:        "empty payload is allowed",
			payloadSize: 0,
			wantErr:     false,
		},
		{
			name:        "small payload is allowed",
			payloadSize: 1024, // 1 KiB
			wantErr:     false,
		},
		{
			name:        "exactly at limit is allowed",
			payloadSize: models.MaxEventLogPayloadBytes,
			wantErr:     false,
		},
		{
			name:        "one byte over the limit is rejected",
			payloadSize: models.MaxEventLogPayloadBytes + 1,
			wantErr:     true,
		},
		{
			name:        "2 MiB payload is rejected",
			payloadSize: 2 * (1 << 20),
			wantErr:     true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			e := &models.EventLog{
				Payload: strings.Repeat("x", tc.payloadSize),
			}

			err := e.BeforeCreate(nil)
			if tc.wantErr && err == nil {
				t.Fatalf("BeforeCreate(%d bytes) = nil, want error", tc.payloadSize)
			}
			if !tc.wantErr && err != nil {
				t.Fatalf("BeforeCreate(%d bytes) = %v, want nil", tc.payloadSize, err)
			}
		})
	}
}
