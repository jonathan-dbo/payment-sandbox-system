FROM golang:1.24-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /out/api ./cmd/api

FROM alpine:3.20

RUN adduser -D -u 10001 appuser
WORKDIR /app

COPY --from=builder /out/api /usr/local/bin/api
COPY --from=builder /app/internal/api/api.yaml /app/internal/api/api.yaml

EXPOSE 8080

HEALTHCHECK --interval=10s --timeout=3s --retries=10 CMD wget -qO- http://localhost:8080/health >/dev/null || exit 1

USER appuser

ENTRYPOINT ["/usr/local/bin/api"]
