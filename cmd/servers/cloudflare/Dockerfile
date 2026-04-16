FROM golang:1.25-alpine AS builder

RUN apk add --no-cache git ca-certificates

WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /bin/cloudflare-mcp ./cmd/servers/cloudflare/

FROM alpine:3.21

RUN apk add --no-cache ca-certificates
COPY --from=builder /bin/cloudflare-mcp /usr/local/bin/cloudflare-mcp

ENV CF_TRANSPORT=http
ENV CF_ADDR=:8080

EXPOSE 8080

HEALTHCHECK --interval=30s --timeout=3s --start-period=5s \
    CMD wget -qO- http://localhost:8080/health || exit 1

ENTRYPOINT ["cloudflare-mcp"]
