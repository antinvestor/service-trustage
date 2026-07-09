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
	"time"

	"github.com/pitabwire/frame/v2/security"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
)

func TestSpanHelpers(t *testing.T) {
	ctx, span := StartSpan(context.Background(), TracerEngine, SpanCommit)
	if ctx == nil || span == nil {
		t.Fatalf("StartSpan() = %v %v", ctx, span)
	}

	EndSpan(span, errors.New("boom"))
}

// installManualReader swaps the global meter provider for one backed by a
// ManualReader so tests can collect exactly what instruments recorded.
// Not parallel-safe: it mutates global state, restored via cleanup.
func installManualReader(t *testing.T) *sdkmetric.ManualReader {
	t.Helper()
	reader := sdkmetric.NewManualReader()
	provider := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader))
	prev := otel.GetMeterProvider()
	otel.SetMeterProvider(provider)
	t.Cleanup(func() { otel.SetMeterProvider(prev) })
	return reader
}

func claimsContext(tenantID, partitionID string) context.Context {
	claims := &security.AuthenticationClaims{TenantID: tenantID, PartitionID: partitionID}
	claims.Subject = "user-" + tenantID
	return claims.ClaimsToContext(context.Background())
}

func collectAttrSets(t *testing.T, reader *sdkmetric.ManualReader, metricName string) []attribute.Set {
	t.Helper()
	var rm metricdata.ResourceMetrics
	require.NoError(t, reader.Collect(context.Background(), &rm))

	var out []attribute.Set
	for _, sm := range rm.ScopeMetrics {
		for _, m := range sm.Metrics {
			if m.Name != metricName {
				continue
			}
			switch d := m.Data.(type) {
			case metricdata.Sum[int64]:
				for _, dp := range d.DataPoints {
					out = append(out, dp.Attributes)
				}
			case metricdata.Histogram[float64]:
				for _, dp := range d.DataPoints {
					out = append(out, dp.Attributes)
				}
			}
		}
	}
	return out
}

func attrValue(t *testing.T, set attribute.Set, key string) string {
	t.Helper()
	v, ok := set.Value(attribute.Key(key))
	require.True(t, ok, "attribute %q missing from %v", key, set.ToSlice())
	return v.AsString()
}

// Engine instruments must carry tenant_id/partition_id transparently from the
// claims context — no call-site opt-in — while claim-less (system) contexts
// record without tenant attributes.
func TestEngineCounterTenantScopedTransparently(t *testing.T) {
	reader := installManualReader(t)
	metrics := NewMetrics()

	metrics.ExecutionsTotal.Add(claimsContext("tnt_1", "prt_1"), 1)
	metrics.ExecutionsTotal.Add(context.Background(), 1) // system path

	sets := collectAttrSets(t, reader, "engine.executions.total")
	require.Len(t, sets, 2, "tenant and system measurements must be distinct series")

	var tenantSet, systemSet *attribute.Set
	for i := range sets {
		if _, ok := sets[i].Value("tenant_id"); ok {
			tenantSet = &sets[i]
		} else {
			systemSet = &sets[i]
		}
	}
	require.NotNil(t, tenantSet, "claims context must yield tenant_id transparently")
	require.NotNil(t, systemSet, "claim-less context must omit tenant attributes")
	require.Equal(t, "tnt_1", attrValue(t, *tenantSet, "tenant_id"))
	require.Equal(t, "prt_1", attrValue(t, *tenantSet, "partition_id"))
}

// Cron sweeps run on a system context and iterate tenants from the fired
// batch; the explicit per-tenant attribute is the only tenant source and must
// be preserved by the wrapper.
func TestRecordSchedulerCronSweepKeepsExplicitTenants(t *testing.T) {
	reader := installManualReader(t)
	metrics := NewMetrics()

	fired := map[string]int{"tnt_a": 2, "tnt_b": 3}
	metrics.RecordSchedulerCronSweep(context.Background(), fired, 10*time.Millisecond, true)

	sets := collectAttrSets(t, reader, "scheduler_cron_fired_total")
	require.Len(t, sets, 2)

	seen := map[string]bool{}
	for _, set := range sets {
		require.Equal(t, "ok", attrValue(t, set, "result"))
		seen[attrValue(t, set, "tenant_id")] = true
	}
	require.True(t, seen["tnt_a"] && seen["tnt_b"], "per-tenant attribution must survive: %v", seen)
}

// Lifecycle counters take tenant from the request context via the wrapper.
func TestRecordWorkflowLifecycleTenantFromContext(t *testing.T) {
	reader := installManualReader(t)
	metrics := NewMetrics()

	metrics.RecordWorkflowLifecycle(claimsContext("tnt_wf", "prt_wf"), "create", true)

	sets := collectAttrSets(t, reader, "workflow_lifecycle_total")
	require.Len(t, sets, 1)
	require.Equal(t, "create", attrValue(t, sets[0], "op"))
	require.Equal(t, "ok", attrValue(t, sets[0], "result"))
	require.Equal(t, "tnt_wf", attrValue(t, sets[0], "tenant_id"))
}
