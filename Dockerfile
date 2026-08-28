ARG GO_VERSION=1.26.5
FROM oven/bun:alpine AS bun-builder

RUN apk add git
WORKDIR /app/src
COPY . .
RUN bun install
RUN bun run build
RUN git rev-parse HEAD > /tmp/wga-release

FROM golang:${GO_VERSION}-alpine AS go-builder

RUN echo "Building with Go version ${GO_VERSION}"

WORKDIR /app/src
COPY --from=bun-builder /app/src /app/src
RUN go mod download && go mod verify
RUN go tool templ generate
RUN go mod tidy
RUN go build -v -ldflags "-X github.com/blackfyre/wga/internal/buildinfo.Version=$(git rev-parse HEAD)" -o /tmp/app ./cmd/wga


FROM alpine:latest

COPY --from=go-builder /tmp/app /usr/local/bin/
COPY --from=bun-builder /app/src/node_modules/@sentry/cli-linux-x64/bin/sentry-cli /usr/local/bin/sentry-cli
COPY --from=bun-builder /app/src/dist/browser-assets/js /usr/local/share/wga/sourcemaps
COPY --from=bun-builder /tmp/wga-release /usr/local/share/wga/release
COPY resources/scripts/start-wga.sh /usr/local/bin/start-wga
RUN chmod +x /usr/local/bin/start-wga

EXPOSE 8090
ENTRYPOINT ["/usr/local/bin/start-wga"]
CMD ["app", "serve", "--http", "0.0.0.0:8090"]
