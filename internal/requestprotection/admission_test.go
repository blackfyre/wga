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
