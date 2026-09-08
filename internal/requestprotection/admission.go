package requestprotection

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	"github.com/blackfyre/wga/internal/config"
)

// Decision is the bounded, non-sensitive outcome of an admission check.
type Decision string

// Admission decisions.
const (
	DecisionOff            Decision = "off"
	DecisionBypass         Decision = "bypass"
	DecisionAllow          Decision = "allow"
	DecisionHost           Decision = "host"
	DecisionOriginAuth     Decision = "origin_authentication"
	DecisionIdentityReject Decision = "identity_reject"
	DecisionClientRate     Decision = "client_rate"
	DecisionGlobalCapacity Decision = "global_capacity"
)

// Policy composes per-client rate accounting and global capacity for protected
// public reads.
type Policy struct {
	mode       config.ProtectionMode
	rates      *RateLimiter
	capacity   *CapacityGate
	retryAfter time.Duration
	observed   atomic.Int64
}

// Admission contains only bounded fields safe for structured telemetry. A
// successful enforced admission privately owns its capacity lease.
type Admission struct {
	mode         config.ProtectionMode
	profile      Profile
	decision     Decision
	status       int
	limit        int
	capacityUsed int
	capacityMax  int
	enforced     bool
	lease        *CapacityLease
	release      func()
	retryAfter   time.Duration
}

// NewPolicy creates a public-read admission policy from validated application
// settings.
func NewPolicy(settings config.PublicRequestProtection) (*Policy, error) {
	switch settings.Mode {
	case config.ProtectionModeOff, config.ProtectionModeObserve, config.ProtectionModeEnforce:
	default:
		return nil, fmt.Errorf("unsupported public-request protection mode %q", settings.Mode)
	}

	rates, err := NewRateLimiter(RateLimits{
		Search:        settings.SearchRequestsPerMinute,
		Fragment:      settings.FragmentRequestsPerMinute,
		Detail:        settings.DetailRequestsPerMinute,
		EntryCapacity: settings.LimiterEntryCapacity,
	})
	if err != nil {
		return nil, err
	}
	capacity, err := NewCapacityGate(settings.MaxConcurrentReads)
	if err != nil {
		return nil, err
	}
	if settings.RetryAfter <= 0 {
		return nil, fmt.Errorf("public-request retry interval must be positive")
	}

	return newPolicy(settings.Mode, rates, capacity, settings.RetryAfter), nil
}

func newPolicy(mode config.ProtectionMode, rates *RateLimiter, capacity *CapacityGate, retryAfter time.Duration) *Policy {
	return &Policy{mode: mode, rates: rates, capacity: capacity, retryAfter: retryAfter}
}

// Admit evaluates one route profile. Observe mode charges the same bounded rate
// state and reports the decision enforcement would make, but always allows the
// request and returns any capacity slot before returning.
func (p *Policy) Admit(ctx context.Context, profile Profile, identity string, resolved bool) Admission {
	if p == nil || p.mode == config.ProtectionModeOff {
		return newAdmission(config.ProtectionModeOff, profile, DecisionOff, 0, 0, nil, false, 0)
	}
	if !profile.Protected() {
		return newAdmission(p.mode, profile, DecisionBypass, 0, 0, p.capacity, false, p.retryAfter)
	}

	limit := p.rates.limit(profile)
	if !resolved || identity == "" {
		return p.result(profile, DecisionIdentityReject, http.StatusForbidden, limit, nil)
	}
	if !p.rates.AllowResolved(identity, true, profile) {
		return p.result(profile, DecisionClientRate, http.StatusTooManyRequests, limit, nil)
	}
	if p.mode == config.ProtectionModeObserve {
		return p.observeCapacity(profile, limit)
	}

	lease, acquired := p.capacity.TryAcquire(ctx)
	if !acquired {
		return p.result(profile, DecisionGlobalCapacity, http.StatusServiceUnavailable, limit, nil)
	}
	return newAdmission(p.mode, profile, DecisionAllow, 0, limit, p.capacity, false, p.retryAfter).withLease(lease)
}

// IngressDecision creates privacy-safe structured fields for a terminal host or
// origin-authentication rejection without touching admission state.
func (p *Policy) IngressDecision(profile Profile, decision Decision, status int) Admission {
	if p == nil {
		return newAdmission(config.ProtectionModeOff, profile, decision, status, 0, nil, true, 0)
	}
	return newAdmission(p.mode, profile, decision, status, p.rates.limit(profile), p.capacity, true, p.retryAfter)
}

func (p *Policy) observeCapacity(profile Profile, limit int) Admission {
	used := int(p.observed.Add(1))
	capacityMax := p.capacity.Capacity()
	decision := DecisionAllow
	status := 0
	if used > capacityMax {
		decision = DecisionGlobalCapacity
		status = http.StatusServiceUnavailable
		used = capacityMax
	}

	once := &sync.Once{}
	return Admission{
		mode:         p.mode,
		profile:      profile,
		decision:     decision,
		status:       status,
		limit:        limit,
		capacityUsed: used,
		capacityMax:  capacityMax,
		retryAfter:   p.retryAfter,
		release: func() {
			once.Do(func() {
				p.observed.Add(-1)
			})
		},
	}
}

func (p *Policy) result(profile Profile, decision Decision, status int, limit int, lease *CapacityLease) Admission {
	enforced := p.mode == config.ProtectionModeEnforce
	return newAdmission(p.mode, profile, decision, status, limit, p.capacity, enforced, p.retryAfter).withLease(lease)
}

func newAdmission(mode config.ProtectionMode, profile Profile, decision Decision, status int, limit int, capacity *CapacityGate, enforced bool, retryAfter time.Duration) Admission {
	result := Admission{
		mode:       mode,
		profile:    profile,
		decision:   decision,
		status:     status,
		limit:      limit,
		enforced:   enforced,
		retryAfter: retryAfter,
	}
	if capacity != nil {
		result.capacityUsed = capacity.InUse()
		result.capacityMax = capacity.Capacity()
	}
	return result
}

func (a Admission) withLease(lease *CapacityLease) Admission {
	a.lease = lease
	return a
}

// Allowed reports whether the caller may continue processing.
func (a Admission) Allowed() bool {
	return !a.enforced
}

// WouldReject reports whether observe mode allowed a request that enforcement
// would have rejected.
func (a Admission) WouldReject() bool {
	return !a.enforced && a.status != 0
}

// Decision returns the admission outcome.
func (a Admission) Decision() Decision {
	return a.decision
}

// Status returns the rejection status, or zero when enforcement would allow the
// request.
func (a Admission) Status() int {
	return a.status
}

// RetryAfter returns the configured retry interval for admission failures.
func (a Admission) RetryAfter() time.Duration {
	return a.retryAfter
}

// Fields returns stable, bounded key-value pairs safe for structured logging.
func (a Admission) Fields() []any {
	return []any{
		"protection_mode", string(a.mode),
		"profile", string(a.profile),
		"decision", string(a.decision),
		"status", a.status,
		"configured_limit", a.limit,
		"capacity_in_use", a.capacityUsed,
		"capacity_limit", a.capacityMax,
	}
}

// Release returns any enforced capacity lease exactly once.
func (a Admission) Release() {
	a.lease.Release()
	if a.release != nil {
		a.release()
	}
}
