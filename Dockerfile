ARG BUILD_FROM=ghcr.io/home-assistant/base:latest

# ── Stage 1: build the Go binary ─────────────────────────────────────────────
FROM golang:1.26-alpine AS builder
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /ha-trmnld ./cmd/ha-trmnld

# ── Stage 2: minimal HA base image ───────────────────────────────────────────
FROM $BUILD_FROM
COPY --from=builder /ha-trmnld /ha-trmnld
COPY run.sh /run.sh
RUN chmod a+x /run.sh
CMD ["/run.sh"]
