# Tasks

## 1. Log persistence

- [x] 1.1 Skip PocketBase request activity logs for requests that complete without an error and below 500 or as expected admission rejections, decided after the chain so protection short-circuits are covered. Verify with `go test ./internal/handlers -run FailureOnlyActivityLog`.
- [x] 1.2 Add a settings migration that sets persisted log retention to 7 days. Verify with `go test ./internal/migrations -run LogPersistence`.

## 2. Protection events

- [x] 2.1 Emit protection decision and cancellation events through a per-event, per-profile interval sampler that reports suppressed counts. Verify with `go test ./internal/handlers -run 'Protect|Cancellation'`.

## 3. Verification

- [x] 3.1 Run `go vet ./...`, `go test ./... -cover`, and `golangci-lint run`.
