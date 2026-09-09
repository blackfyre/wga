package handlers

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/blackfyre/wga/internal/logging"
	"github.com/blackfyre/wga/internal/requestfailure"
	"github.com/blackfyre/wga/internal/requestprotection"
	"github.com/blackfyre/wga/internal/requesttrust"
	"github.com/pocketbase/pocketbase/core"
)

func registerProtectedReadMiddleware(app core.App, publicURL string, authenticateOrigin requesttrust.OriginAuthenticator, resolveIdentity requesttrust.Resolver, policy *requestprotection.Policy) error {
	if authenticateOrigin == nil || resolveIdentity == nil || policy == nil {
		return fmt.Errorf("protected-read middleware requires origin authenticator, identity resolver, and admission policy")
	}

	canonicalHost, err := requestprotection.NewCanonicalHost(publicURL)
	if err != nil {
		return err
	}

	app.OnServe().BindFunc(func(se *core.ServeEvent) error {
		se.Router.BindFunc(func(e *core.RequestEvent) error {
			return protectPublicRead(app, canonicalHost, authenticateOrigin, resolveIdentity, policy, e)
		})
		return se.Next()
	})

	return nil
}

func protectPublicRead(app core.App, canonicalHost requestprotection.CanonicalHost, authenticateOrigin requesttrust.OriginAuthenticator, resolveIdentity requesttrust.Resolver, policy *requestprotection.Policy, e *core.RequestEvent) error {
	profile := requestprotection.Classify(e.Request.Method, e.Request.URL.EscapedPath())
	if !profile.Protected() {
		return e.Next()
	}

	if !canonicalHost.Allows(profile, e.Request.Host) {
		logProtectionDecision(app, e, policy.IngressDecision(profile, requestprotection.DecisionHost, http.StatusMisdirectedRequest))
		return plainProtectionResponse(e, http.StatusMisdirectedRequest, 0)
	}

	if !authenticateOrigin(e.Request) {
		logProtectionDecision(app, e, policy.IngressDecision(profile, requestprotection.DecisionOriginAuth, http.StatusForbidden))
		return plainProtectionResponse(e, http.StatusForbidden, 0)
	}
	identity, resolved := resolveIdentity(e.Request)
	if !resolved {
		logProtectionDecision(app, e, policy.IngressDecision(profile, requestprotection.DecisionIdentityReject, http.StatusForbidden))
		return plainProtectionResponse(e, http.StatusForbidden, 0)
	}

	admission := policy.Admit(e.Request.Context(), profile, identity, true)
	defer admission.Release()
	if admission.WouldReject() || !admission.Allowed() {
		logProtectionDecision(app, e, admission)
	}
	if !admission.Allowed() {
		return plainProtectionResponse(e, admission.Status(), admission.RetryAfter())
	}

	logger := logging.RequestLogger(app, e)
	e.Request = e.Request.WithContext(requestprotection.WithCancellationObserver(
		e.Request.Context(),
		profile,
		func(event requestprotection.CancellationEvent) {
			logger.Info("Protected request cancelled",
				"event", "request_protection.cancelled",
				"profile", string(event.Profile),
				"stage", event.Stage,
			)
		},
	))

	return e.Next()
}

func plainProtectionResponse(e *core.RequestEvent, status int, retryAfter time.Duration) error {
	if retryAfter > 0 {
		seconds := (retryAfter + time.Second - 1) / time.Second
		e.Response.Header().Set("Retry-After", strconv.FormatInt(int64(seconds), 10))
	}
	err := e.String(status, http.StatusText(status)+"\n")
	if err == nil {
		requestfailure.MarkExpectedResponse(e)
	}
	return err
}

func logProtectionDecision(app core.App, e *core.RequestEvent, admission requestprotection.Admission) {
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

	fields := append([]any{"event", event}, admission.Fields()...)
	logger := logging.RequestLogger(app, e)
	if admission.WouldReject() {
		logger.Info("Public request admission observed", fields...)
		return
	}
	logger.Warn("Public request rejected", fields...)
}
