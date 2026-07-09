# Convallaria — Coding Agent Harness
# Multi-stage build for minimal image size

FROM golang:1.23-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o convallaria ./cmd/convallaria/

FROM alpine:latest
RUN apk --no-cache add ca-certificates git
WORKDIR /app
COPY --from=builder /app/convallaria .
COPY --from=builder /app/web ./web

EXPOSE 8080
ENTRYPOINT ["./convallaria"]
CMD ["-port", "8080"]