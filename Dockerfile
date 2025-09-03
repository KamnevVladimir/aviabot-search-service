FROM golang:1.21-alpine AS builder
WORKDIR /app
COPY . .
# Explicitly download shared modules to ensure correct versions are used
RUN go mod download github.com/KamnevVladimir/aviabot-shared-logging@v1.0.3 && \
    go mod download github.com/KamnevVladimir/aviabot-shared-core@v1.0.1 && \
    go mod download github.com/KamnevVladimir/aviabot-shared-utils@v1.0.1 && \
    go mod download && go mod tidy
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -a -installsuffix cgo -ldflags="-w -s" -o main ./cmd/main.go

FROM alpine:latest
RUN apk --no-cache add ca-certificates && adduser -D -s /bin/sh appuser
WORKDIR /app/
COPY --from=builder /app/main /app/main
RUN chown -R appuser:appuser /app
USER appuser
EXPOSE 8084
HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 CMD wget -qO- http://localhost:8084/health >/dev/null 2>&1 || exit 1
CMD ["/app/main"]
