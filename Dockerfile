ARG GO_VERSION=1
FROM golang:${GO_VERSION}-alpine as builder

WORKDIR /usr/src/app
COPY go.mod go.sum ./
RUN go mod download && go mod verify
COPY . .
RUN go install github.com/a-h/templ/cmd/templ@latest
RUN templ generate
RUN apk add --update nodejs npm
RUN npm ci
RUN npm run build
RUN go build -v -o /run-app .


FROM alpine:latest

COPY --from=builder /run-app /usr/local/bin/
CMD ["run-app", "serve", "--http", "0.0.0.0:8090"]
