# Tasks

## 1. Per-profile capacity

- [x] 1.1 Replace the global admission gate with one gate per protected route profile, including observe-mode accounting and profile-attributed capacity gauges. Verify with `go test ./internal/requestprotection ./internal/handlers`.
- [x] 1.2 Document the per-profile meaning of `WGA_PUBLIC_READ_MAX_CONCURRENT`.

## 2. Verification

- [x] 2.1 Run `go vet ./...`, `go test ./... -cover`, and `golangci-lint run`.
