package requestprotection

import (
	"context"
	"crypto/sha256"
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/blackfyre/wga/internal/config"
	"go.opentelemetry.io/otel"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
)

func TestPolicyOffBypassesState(t *testing.T) {
	policy := testPolicy(t, config.ProtectionModeOff, 1, 1)

	for range 2 {
		admission := policy.Admit(context.Background(), ProfileSearch, "198.51.100.7", true)
		if !admission.Allowed() || admission.Decision() != DecisionOff || admission.WouldReject() {
			t.Fatalf("off admission = allowed %t, decision %q, would reject %t", admission.Allowed(), admission.Decision(), admission.WouldReject())
		}
	}
	if len(policy.rates.clients) != 0 || policy.capacity.InUse() != 0 {
		t.Fatalf("off mode retained rate/capacity state: clients=%d capacity=%d", len(policy.rates.clients), policy.capacity.InUse())
	}
}

func TestPolicyObserveRecordsWouldBeRateRejection(t *testing.T) {
	policy := testPolicy(t, config.ProtectionModeObserve, 1, 1)

	first := policy.Admit(context.Background(), ProfileSearch, "198.51.100.7", true)
	if !first.Allowed() || first.Decision() != DecisionAllow || first.WouldReject() {
		t.Fatalf("first observe admission = allowed %t, decision %q, would reject %t", first.Allowed(), first.Decision(), first.WouldReject())
	}
	if policy.capacity.InUse() != 0 {
		t.Fatalf("observe mode retained %d capacity slots", policy.capacity.InUse())
	}
	defer first.Release()

	second := policy.Admit(context.Background(), ProfileSearch, "198.51.100.7", true)
	if !second.Allowed() || second.Decision() != DecisionClientRate || !second.WouldReject() || second.Status() != http.StatusTooManyRequests {
		t.Fatalf("second observe admission = allowed %t, decision %q, would reject %t, status %d", second.Allowed(), second.Decision(), second.WouldReject(), second.Status())
	}
	if len(policy.rates.clients) != 1 || policy.capacity.InUse() != 0 {
		t.Fatalf("observe mode state = clients %d, capacity %d; want 1, 0", len(policy.rates.clients), policy.capacity.InUse())
	}
}

func TestPolicyObserveRecordsWouldBeCapacityRejection(t *testing.T) {
	policy := testPolicy(t, config.ProtectionModeObserve, 2, 1)
	first := policy.Admit(context.Background(), ProfileDetail, "198.51.100.7", true)

	admission := policy.Admit(context.Background(), ProfileDetail, "198.51.100.8", true)
	if !admission.Allowed() || admission.Decision() != DecisionGlobalCapacity || !admission.WouldReject() || admission.Status() != http.StatusServiceUnavailable {
		t.Fatalf("observe capacity admission = allowed %t, decision %q, would reject %t, status %d", admission.Allowed(), admission.Decision(), admission.WouldReject(), admission.Status())
	}
	if policy.capacity.InUse() != 0 {
		t.Fatalf("observe requests retained scarce capacity: in use %d; want 0", policy.capacity.InUse())
	}
	first.Release()
	admission.Release()
	admission.Release()
	if policy.observed.Load() != 0 {
		t.Fatalf("released observe requests retained %d active observations", policy.observed.Load())
	}
}

func TestPolicyEnforceRejectsAndHoldsSuccessfulCapacity(t *testing.T) {
	t.Run("identity", func(t *testing.T) {
		policy := testPolicy(t, config.ProtectionModeEnforce, 2, 1)
		admission := policy.Admit(context.Background(), ProfileSearch, "visitor-supplied", false)
		assertEnforcedRejection(t, admission, DecisionIdentityReject, http.StatusForbidden)
	})

	t.Run("client rate", func(t *testing.T) {
		policy := testPolicy(t, config.ProtectionModeEnforce, 1, 1)
		first := policy.Admit(context.Background(), ProfileSearch, "198.51.100.7", true)
		first.Release()
		admission := policy.Admit(context.Background(), ProfileSearch, "198.51.100.7", true)
		assertEnforcedRejection(t, admission, DecisionClientRate, http.StatusTooManyRequests)
	})

	t.Run("global capacity", func(t *testing.T) {
		policy := testPolicy(t, config.ProtectionModeEnforce, 2, 1)
		first := policy.Admit(context.Background(), ProfileDetail, "198.51.100.7", true)
		defer first.Release()
		admission := policy.Admit(context.Background(), ProfileDetail, "198.51.100.8", true)
		assertEnforcedRejection(t, admission, DecisionGlobalCapacity, http.StatusServiceUnavailable)
	})

	t.Run("allow and release", func(t *testing.T) {
		policy := testPolicy(t, config.ProtectionModeEnforce, 2, 1)
		admission := policy.Admit(context.Background(), ProfileFragment, "198.51.100.7", true)
		if !admission.Allowed() || admission.Decision() != DecisionAllow || policy.capacity.InUse() != 1 {
			t.Fatalf("enforced allow = allowed %t, decision %q, capacity %d", admission.Allowed(), admission.Decision(), policy.capacity.InUse())
		}
		admission.Release()
		admission.Release()
		if policy.capacity.InUse() != 0 {
			t.Fatalf("released admission retained %d slots", policy.capacity.InUse())
		}
	})
}

func TestPolicyBypassesUnprotectedProfiles(t *testing.T) {
	policy := testPolicy(t, config.ProtectionModeEnforce, 1, 1)
	for _, profile := range []Profile{ProfileExempt, ProfileUnclassified} {
		admission := policy.Admit(nil, profile, "", false)
		if !admission.Allowed() || admission.Decision() != DecisionBypass {
			t.Fatalf("profile %q was not bypassed", profile)
		}
	}
	if len(policy.rates.clients) != 0 || policy.capacity.InUse() != 0 {
		t.Fatal("bypassed profiles retained admission state")
	}
}

func TestAdmissionFieldsAreStableAndPrivate(t *testing.T) {
	const identity = "198.51.100.7"
	policy := testPolicy(t, config.ProtectionModeObserve, 1, 2)
	policy.Admit(context.Background(), ProfileDetail, identity, true)
	admission := policy.Admit(context.Background(), ProfileDetail, identity, true)

	fields := admission.Fields()
	want := []any{
		"protection_mode", "observe",
		"profile", "detail",
		"decision", "client_rate",
		"status", http.StatusTooManyRequests,
		"configured_limit", 1,
		"capacity_in_use", 0,
		"capacity_limit", 2,
	}
	if fmt.Sprint(fields) != fmt.Sprint(want) {
		t.Fatalf("fields = %v; want %v", fields, want)
	}
	formatted := fmt.Sprint(fields)
	for _, sensitive := range []string{identity, "CF-Connecting-IP", "X-Forwarded-For", "X-WGA-Edge-Secret", "record-slug"} {
		if strings.Contains(formatted, sensitive) {
			t.Fatalf("admission fields exposed %q: %s", sensitive, formatted)
		}
	}
	if admission.RetryAfter() != 5*time.Second {
		t.Fatalf("retry interval = %s; want 5s", admission.RetryAfter())
	}
}

func TestAdmissionMetricsAreBoundedAndTrackCapacityTransitions(t *testing.T) {
	reader := sdkmetric.NewManualReader()
	provider := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader))
	previousProvider := otel.GetMeterProvider()
	otel.SetMeterProvider(provider)
	t.Cleanup(func() {
		otel.SetMeterProvider(previousProvider)
		if err := provider.Shutdown(context.Background()); err != nil {
			t.Errorf("shutdown metric provider: %v", err)
		}
	})

	ctx := context.Background()
	enforced := testPolicy(t, config.ProtectionModeEnforce, 1, 1)
	allowed := enforced.Admit(ctx, ProfileDetail, "sensitive-client-a", true)
	capacityRejected := enforced.Admit(ctx, ProfileDetail, "sensitive-client-b", true)
	assertEnforcedRejection(t, capacityRejected, DecisionGlobalCapacity, http.StatusServiceUnavailable)
	allowed.Release()
	rateRejected := enforced.Admit(ctx, ProfileDetail, "sensitive-client-b", true)
	assertEnforcedRejection(t, rateRejected, DecisionClientRate, http.StatusTooManyRequests)

	observed := testPolicy(t, config.ProtectionModeObserve, 2, 1)
	observedFirst := observed.Admit(ctx, ProfileDetail, "sensitive-client-c", true)
	observedSecond := observed.Admit(ctx, ProfileDetail, "sensitive-client-d", true)
	if observedSecond.Decision() != DecisionGlobalCapacity || !observedSecond.WouldReject() {
		t.Fatalf("observe-mode capacity decision = %q, would reject %t", observedSecond.Decision(), observedSecond.WouldReject())
	}
	observedFirst.Release()
	observedSecond.Release()

	var metrics metricdata.ResourceMetrics
	if err := reader.Collect(ctx, &metrics); err != nil {
		t.Fatalf("collect admission metrics: %v", err)
	}
	decisions := admissionMetricCounts(t, metrics)
	for key, want := range map[string]int64{
		"allow/0":             2,
		"global_capacity/503": 2,
		"client_rate/429":     1,
	} {
		if got := decisions[key]; got != want {
			t.Errorf("admission metric %q = %d, want %d", key, got, want)
		}
	}
	assertAdmissionGauge(t, metrics, "wga.request_protection.capacity.current", 0)
	assertAdmissionGauge(t, metrics, "wga.request_protection.capacity.configured", 1)
}

func admissionMetricCounts(t *testing.T, metrics metricdata.ResourceMetrics) map[string]int64 {
	t.Helper()
	counts := make(map[string]int64)
	for _, scope := range metrics.ScopeMetrics {
		for _, candidate := range scope.Metrics {
			if candidate.Name != "wga.request_protection.admissions" {
				continue
			}
			sum, ok := candidate.Data.(metricdata.Sum[int64])
			if !ok {
				t.Fatalf("admission metric data type = %T", candidate.Data)
			}
			for _, point := range sum.DataPoints {
				profile, decision, status := "", "", int64(-1)
				for _, attr := range point.Attributes.ToSlice() {
					switch string(attr.Key) {
					case "wga.request_protection.profile":
						profile = attr.Value.AsString()
					case "wga.request_protection.decision":
						decision = attr.Value.AsString()
					case "http.response.status_code":
						status = attr.Value.AsInt64()
					default:
						t.Errorf("admission metric has unexpected attribute %q", attr.Key)
					}
				}
				if profile != "detail" {
					t.Errorf("admission metric profile = %q, want detail", profile)
				}
				counts[fmt.Sprintf("%s/%d", decision, status)] += point.Value
			}
		}
	}
	formatted := fmt.Sprint(counts)
	for _, forbidden := range []string{"sensitive-client", "record-slug", "/artists/"} {
		if strings.Contains(formatted, forbidden) {
			t.Errorf("admission metrics expose %q: %s", forbidden, formatted)
		}
	}
	return counts
}

func assertAdmissionGauge(t *testing.T, metrics metricdata.ResourceMetrics, name string, want int64) {
	t.Helper()
	for _, scope := range metrics.ScopeMetrics {
		for _, candidate := range scope.Metrics {
			if candidate.Name != name {
				continue
			}
			gauge, ok := candidate.Data.(metricdata.Gauge[int64])
			if !ok {
				t.Fatalf("%s data type = %T", name, candidate.Data)
			}
			if len(gauge.DataPoints) != 1 {
				t.Fatalf("%s points = %d, want 1 bounded profile", name, len(gauge.DataPoints))
			}
			if got := gauge.DataPoints[0].Value; got != want {
				t.Errorf("%s = %d, want %d", name, got, want)
			}
			return
		}
	}
	t.Errorf("metric %q is absent", name)
}

func TestNewPolicyRejectsInvalidSettings(t *testing.T) {
	settings := config.PublicRequestProtection{
		Mode:                      config.ProtectionMode("invalid"),
		MaxConcurrentReads:        1,
		SearchRequestsPerMinute:   1,
		FragmentRequestsPerMinute: 1,
		DetailRequestsPerMinute:   1,
		LimiterEntryCapacity:      1,
		RetryAfter:                time.Second,
	}
	if _, err := NewPolicy(settings); err == nil {
		t.Fatal("invalid protection mode succeeded; want error")
	}
}

func assertEnforcedRejection(t *testing.T, admission Admission, decision Decision, status int) {
	t.Helper()
	if admission.Allowed() || admission.Decision() != decision || admission.Status() != status || admission.WouldReject() {
		t.Fatalf("admission = allowed %t, decision %q, status %d, would reject %t", admission.Allowed(), admission.Decision(), admission.Status(), admission.WouldReject())
	}
}

func testPolicy(t *testing.T, mode config.ProtectionMode, rateLimit int, capacity int) *Policy {
	t.Helper()
	limits := RateLimits{Search: rateLimit, Fragment: rateLimit, Detail: rateLimit, EntryCapacity: 8}
	key := [sha256.Size]byte{1}
	rates := newRateLimiter(limits, key, time.Now)
	gate, err := NewCapacityGate(capacity)
	if err != nil {
		t.Fatal(err)
	}
	return newPolicy(mode, rates, gate, 5*time.Second)
}
