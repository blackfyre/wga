package logging

import (
	"context"
	"log/slog"
	"reflect"

	"github.com/google/uuid"
	"github.com/pocketbase/pocketbase/core"
)

const (
	RequestIDHeader = "X-Request-ID"
	Redacted        = "[REDACTED]"

	requestIDKey = "request_id"
)

type requestIDContextKey struct{}

func RegisterRequestIDMiddleware(app core.App) {
	app.OnServe().BindFunc(func(se *core.ServeEvent) error {
		se.Router.BindFunc(func(e *core.RequestEvent) error {
			AttachRequestID(e)

			return e.Next()
		})

		return se.Next()
	})
}

// AttachRequestID assigns a server-generated identifier without trusting public request headers.
func AttachRequestID(e *core.RequestEvent) string {
	requestID := uuid.NewString()
	SetRequestID(e, requestID)

	return requestID
}

func SetRequestID(e *core.RequestEvent, requestID string) {
	e.Set(requestIDKey, requestID)
	e.Request = e.Request.WithContext(WithRequestID(e.Request.Context(), requestID))
	e.Response.Header().Set(RequestIDHeader, requestID)
}

func WithRequestID(ctx context.Context, requestID string) context.Context {
	return context.WithValue(ctx, requestIDContextKey{}, requestID)
}

func RequestID(e *core.RequestEvent) string {
	if requestID, ok := e.Get(requestIDKey).(string); ok {
		return requestID
	}

	return RequestIDFromContext(e.Request.Context())
}

func RequestIDFromContext(ctx context.Context) string {
	requestID, _ := ctx.Value(requestIDContextKey{}).(string)

	return requestID
}

func RequestLogger(app core.App, e *core.RequestEvent) *slog.Logger {
	return app.Logger().With("request_id", RequestID(e))
}

func ContextLogger(app core.App, ctx context.Context) *slog.Logger {
	return app.Logger().With("request_id", RequestIDFromContext(ctx))
}

func NewRunID() string {
	return uuid.NewString()
}

func RunLogger(app core.App, runID string) *slog.Logger {
	return app.Logger().With("run_id", runID)
}

// Redact intentionally discards all input to prevent sensitive data from reaching logs.
func Redact(_ any) string {
	return Redacted
}

func ErrorType(err error) string {
	if err == nil {
		return ""
	}

	return reflect.TypeOf(err).String()
}
