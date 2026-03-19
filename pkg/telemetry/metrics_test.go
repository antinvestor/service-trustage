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
