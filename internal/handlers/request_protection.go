package handlers

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/blackfyre/wga/internal/logging"
	"github.com/blackfyre/wga/internal/requestprotection"
	"github.com/blackfyre/wga/internal/requesttrust"
	"github.com/pocketbase/pocketbase/core"
)

func registerProtectedReadMiddleware(app core.App, publicURL string, resolveIdentity requesttrust.Resolver, policy *requestprotection.Policy) error {
	if resolveIdentity == nil || policy == nil {
		return fmt.Errorf("protected-read middleware requires identity resolver and admission policy")
	}

	canonicalHost, err := requestprotection.NewCanonicalHost(publicURL)
	if err != nil {
		return err
	}

	app.OnServe().BindFunc(func(se *core.ServeEvent) error {
		se.Router.BindFunc(func(e *core.RequestEvent) error {
			return protectPublicRead(app, canonicalHost, resolveIdentity, policy, e)
		})
		return se.Next()
	})

	return nil
}

func protectPublicRead(app core.App, canonicalHost requestprotection.CanonicalHost, resolveIdentity requesttrust.Resolver, policy *requestprotection.Policy, e *core.RequestEvent) error {
	profile := requestprotection.Classify(e.Request.Method, e.Request.URL.Path)
	if !profile.Protected() {
		return e.Next()
	}

	if !canonicalHost.Allows(profile, e.Request.Host) {
		logIngressRejection(app, e, profile, "host", http.StatusMisdirectedRequest)
		return plainProtectionResponse(e, http.StatusMisdirectedRequest, 0)
	}

	identity, resolved := resolveIdentity(e.Request)
	if !resolved {
		logIngressRejection(app, e, profile, "origin_authentication", http.StatusForbidden)
		return plainProtectionResponse(e, http.StatusForbidden, 0)
	}

	admission := policy.Admit(e.Request.Context(), profile, identity, true)
	defer admission.Release()
	if admission.WouldReject() || !admission.Allowed() {
		logAdmissionDecision(app, e, admission)
	}
	if !admission.Allowed() {
		return plainProtectionResponse(e, admission.Status(), admission.RetryAfter())
	}

	return e.Next()
}

func plainProtectionResponse(e *core.RequestEvent, status int, retryAfter time.Duration) error {
	if retryAfter > 0 {
		seconds := (retryAfter + time.Second - 1) / time.Second
		e.Response.Header().Set("Retry-After", strconv.FormatInt(int64(seconds), 10))
	}
	return e.String(status, http.StatusText(status)+"\n")
}

func logIngressRejection(app core.App, e *core.RequestEvent, profile requestprotection.Profile, decision string, status int) {
	logging.RequestLogger(app, e).Warn("Public request ingress rejected",
		"event", "request_protection.ingress_rejected",
		"profile", string(profile),
		"decision", decision,
		"status", status,
	)
}

func logAdmissionDecision(app core.App, e *core.RequestEvent, admission requestprotection.Admission) {
	fields := append([]any{"event", "request_protection.admission_decided"}, admission.Fields()...)
	logging.RequestLogger(app, e).Warn("Public request admission decided", fields...)
}
