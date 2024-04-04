ARG GOLANG_VERSION=1.22.2
FROM oven/bun:alpine as builder

RUN wget https://go.dev/dl/go${GOLANG_VERSION}.linux-amd64.tar.gz && \
    rm -rf /usr/local/go && tar -C /usr/local -xzf go${GOLANG_VERSION}.linux-amd64.tar.gz && \
    rm go${GOLANG_VERSION}.linux-amd64.tar.gz
ENV PATH="${PATH}:/usr/local/go/bin"

WORKDIR /usr/src/app
COPY go.mod go.sum ./
RUN go mod download && go mod verify
COPY . .
RUN go install github.com/a-h/templ/cmd/templ@latest
RUN templ generate
# RUN curl -fsSL https://bun.sh/install | bash
RUN bun install
RUN bun run build
RUN go build -v -o /run-app .


FROM alpine:latest

COPY --from=builder /run-app /usr/local/bin/
CMD ["run-app", "serve", "--http", "0.0.0.0:8090"]
