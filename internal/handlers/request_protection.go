package handlers

import (
	"fmt"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/blackfyre/wga/internal/config"
	"github.com/blackfyre/wga/internal/logging"
	"github.com/blackfyre/wga/internal/requestfailure"
	"github.com/blackfyre/wga/internal/requestprotection"
	"github.com/blackfyre/wga/internal/requesttrust"
	"github.com/pocketbase/pocketbase/core"
)

func registerProtectedReadMiddleware(app core.App, environment config.Environment, publicURL string, authenticateOrigin requesttrust.OriginAuthenticator, resolveIdentity requesttrust.Resolver, policy *requestprotection.Policy) error {
	if authenticateOrigin == nil || resolveIdentity == nil || policy == nil {
		return fmt.Errorf("protected-read middleware requires origin authenticator, identity resolver, and admission policy")
	}

	canonicalHost, err := requestprotection.NewCanonicalHost(publicURL)
	if err != nil {
		return err
	}

	events := newProtectionEventSampler(protectionEventInterval, time.Now)
	app.OnServe().BindFunc(func(se *core.ServeEvent) error {
		se.Router.BindFunc(func(e *core.RequestEvent) error {
			return protectPublicRead(app, environment, canonicalHost, authenticateOrigin, resolveIdentity, policy, events, e)
		})
		return se.Next()
	})

	return nil
}

func protectPublicRead(app core.App, environment config.Environment, canonicalHost requestprotection.CanonicalHost, authenticateOrigin requesttrust.OriginAuthenticator, resolveIdentity requesttrust.Resolver, policy *requestprotection.Policy, events *protectionEventSampler, e *core.RequestEvent) error {
	profile := requestprotection.Classify(e.Request.Method, e.Request.URL.EscapedPath())
	if !profile.Protected() {
		return e.Next()
	}

	if environment != config.EnvironmentDevelopment && environment != config.EnvironmentTest {
		if !canonicalHost.Allows(profile, e.Request.Host) {
			logProtectionDecision(app, e, events, policy.IngressDecision(profile, requestprotection.DecisionHost, http.StatusMisdirectedRequest))
			return plainProtectionResponse(e, http.StatusMisdirectedRequest, 0)
		}

		if !authenticateOrigin(e.Request) {
			logProtectionDecision(app, e, events, policy.IngressDecision(profile, requestprotection.DecisionOriginAuth, http.StatusForbidden))
			return plainProtectionResponse(e, http.StatusForbidden, 0)
		}
	}
	identity, resolved := resolveIdentity(e.Request)
	if !resolved {
		logProtectionDecision(app, e, events, policy.IngressDecision(profile, requestprotection.DecisionIdentityReject, http.StatusForbidden))
		return plainProtectionResponse(e, http.StatusForbidden, 0)
	}

	admission := policy.Admit(e.Request.Context(), profile, identity, true)
	defer admission.Release()
	if admission.WouldReject() || !admission.Allowed() {
		logProtectionDecision(app, e, events, admission)
	}
	if !admission.Allowed() {
		return plainProtectionResponse(e, admission.Status(), admission.RetryAfter())
	}

	logger := logging.RequestLogger(app, e)
	e.Request = e.Request.WithContext(requestprotection.WithCancellationObserver(
		e.Request.Context(),
		profile,
		func(event requestprotection.CancellationEvent) {
			const name = "request_protection.cancelled"
			suppressed, emit := events.admit(name, event.Profile)
			if !emit {
				return
			}
			logger.Info("Protected request cancelled",
				"event", name,
				"profile", string(event.Profile),
				"stage", event.Stage,
				"suppressed", suppressed,
			)
		},
	))

	return e.Next()
}

func plainProtectionResponse(e *core.RequestEvent, status int, retryAfter time.Duration) error {
	// Rejections are per-client outcomes and never shareable.
	e.Response.Header().Set("Cache-Control", "private, no-store")
	if retryAfter > 0 {
		seconds := (retryAfter + time.Second - 1) / time.Second
		e.Response.Header().Set("Retry-After", strconv.FormatInt(int64(seconds), 10))
	}
	err := e.String(status, http.StatusText(status)+"\n")
	if err == nil {
		requestfailure.MarkExpectedResponse(e)
		e.Set(protectionRejectionKey, true)
	}
	return err
}

func logProtectionDecision(app core.App, e *core.RequestEvent, events *protectionEventSampler, admission requestprotection.Admission) {
	event := "request_protection.decision"
	switch admission.Decision() {
	case requestprotection.DecisionHost:
		event = "request_protection.host_rejected"
	case requestprotection.DecisionOriginAuth:
		event = "request_protection.origin_authentication_rejected"
	case requestprotection.DecisionIdentityReject:
		event = "request_protection.identity_rejected"
	case requestprotection.DecisionClientRate:
		event = "request_protection.client_rate_rejected"
	case requestprotection.DecisionGlobalCapacity:
		event = "request_protection.capacity_rejected"
	}
	if admission.WouldReject() {
		event = "request_protection.admission_observed"
	}

	suppressed, emit := events.admit(event, admission.Profile())
	if !emit {
		return
	}

	fields := append([]any{"event", event}, admission.Fields()...)
	fields = append(fields, "suppressed", suppressed)
	logger := logging.RequestLogger(app, e)
	if admission.WouldReject() {
		logger.Info("Public request admission observed", fields...)
		return
	}
	logger.Warn("Public request rejected", fields...)
}

// protectionEventInterval is the minimum spacing between persisted events of
// one type and route profile. Admission metrics still count every decision.
const protectionEventInterval = time.Minute

type protectionEventKey struct {
	event   string
	profile requestprotection.Profile
}

type protectionEventWindow struct {
	emittedAt  time.Time
	suppressed int
}

// protectionEventSampler bounds persisted protection events so that a flood of
// cheap rejections does not become one database write per request. Keys are the
// fixed event names and route profiles, so its state stays bounded.
type protectionEventSampler struct {
	mu       sync.Mutex
	interval time.Duration
	now      func() time.Time
	windows  map[protectionEventKey]*protectionEventWindow
}

func newProtectionEventSampler(interval time.Duration, now func() time.Time) *protectionEventSampler {
	return &protectionEventSampler{
		interval: interval,
		now:      now,
		windows:  make(map[protectionEventKey]*protectionEventWindow),
	}
}

// admit reports whether an event should be emitted and, if so, how many events
// of the same type and profile were suppressed since the previous emission.
func (s *protectionEventSampler) admit(event string, profile requestprotection.Profile) (int, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := s.now()
	key := protectionEventKey{event: event, profile: profile}
	window := s.windows[key]
	if window == nil {
		s.windows[key] = &protectionEventWindow{emittedAt: now}
		return 0, true
	}
	if now.Sub(window.emittedAt) < s.interval {
		window.suppressed++
		return 0, false
	}

	suppressed := window.suppressed
	window.emittedAt = now
	window.suppressed = 0
	return suppressed, true
}
