FROM golang:1.21-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .

# Run tests with coverage - mandatory for deploy
RUN go test ./... -cover -coverprofile=coverage.out

# Enforce coverage for critical packages (infrastructure, interfaces)
RUN go tool cover -func=coverage.out | grep -E '(internal/infrastructure/aviasales|internal/interfaces/http)' | \
    awk 'BEGIN{ok=1} {split($3,a,"%"); if(a[1]+0 < 90.0){print "Coverage too low for " $1 ": "$3; ok=0}} END{exit ok?0:1}'

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -a -installsuffix cgo -ldflags="-w -s" -o main ./cmd/main.go

FROM alpine:latest
RUN apk --no-cache add ca-certificates wget && adduser -D -s /bin/sh appuser
WORKDIR /app/
COPY --from=builder /app/main /app/main
RUN chown -R appuser:appuser /app
USER appuser
EXPOSE 8084
HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 CMD wget -qO- http://localhost:8084/health >/dev/null 2>&1 || exit 1
CMD ["/app/main"]
