package requestprotection

import (
	"context"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

const admissionInstrumentationName = "github.com/blackfyre/wga/internal/requestprotection"

func recordAdmissionDecision(ctx context.Context, profile Profile, decision Decision, status int) {
	profileValue, profileOK := profile.telemetryValue()
	decisionValue, decisionOK := decision.telemetryValue()
	if !profileOK || !decisionOK || !admissionStatusAllowed(status) {
		return
	}
	counter, err := otel.Meter(admissionInstrumentationName).Int64Counter(
		"wga.request_protection.admissions",
		metric.WithDescription("Public-read admission decisions."),
		metric.WithUnit("{admission}"),
	)
	if err != nil {
		return
	}
	counter.Add(ctx, 1, metric.WithAttributes(
		attribute.String("wga.request_protection.profile", profileValue),
		attribute.String("wga.request_protection.decision", decisionValue),
		attribute.Int("http.response.status_code", status),
	))
}

func recordAdmissionCapacity(ctx context.Context, current int, configured int) {
	if current < 0 || configured <= 0 || current > configured {
		return
	}
	meter := otel.Meter(admissionInstrumentationName)
	currentGauge, err := meter.Int64Gauge(
		"wga.request_protection.capacity.current",
		metric.WithDescription("Current public-read work using configured capacity."),
		metric.WithUnit("{request}"),
	)
	if err == nil {
		currentGauge.Record(ctx, int64(current))
	}
	configuredGauge, err := meter.Int64Gauge(
		"wga.request_protection.capacity.configured",
		metric.WithDescription("Configured public-read concurrent-work capacity."),
		metric.WithUnit("{request}"),
	)
	if err == nil {
		configuredGauge.Record(ctx, int64(configured))
	}
}

func (profile Profile) telemetryValue() (string, bool) {
	switch profile {
	case ProfileUnclassified, ProfileExempt, ProfileSearch, ProfileFragment, ProfileDetail:
		return string(profile), true
	default:
		return "", false
	}
}

func (decision Decision) telemetryValue() (string, bool) {
	switch decision {
	case DecisionOff, DecisionBypass, DecisionAllow, DecisionHost, DecisionOriginAuth, DecisionIdentityReject, DecisionClientRate, DecisionGlobalCapacity:
		return string(decision), true
	default:
		return "", false
	}
}

func admissionStatusAllowed(status int) bool {
	switch status {
	case 0, 403, 421, 429, 503:
		return true
	default:
		return false
	}
}
