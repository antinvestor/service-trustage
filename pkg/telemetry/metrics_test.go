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

//nolint:testpackage // package-local tests exercise unexported telemetry helpers intentionally.
package telemetry

import (
	"context"
	"errors"
	"testing"
)

func TestNewMetricsAndSpanHelpers(t *testing.T) {
	t.Parallel()

	metrics := NewMetrics()
	if metrics == nil || metrics.ExecutionsTotal == nil || metrics.ExecutionLatency == nil {
		t.Fatalf("NewMetrics() = %+v", metrics)
	}

	ctx, span := StartSpan(context.Background(), TracerEngine, SpanCommit)
	if ctx == nil || span == nil {
		t.Fatalf("StartSpan() = %v %v", ctx, span)
	}

	EndSpan(span, errors.New("boom"))
}
