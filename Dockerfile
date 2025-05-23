FROM golang:1.23-alpine AS builder

WORKDIR /app
COPY ./ ./

RUN CGO_ENABLED=0 GOOS=linux go build -a -ldflags '-extldflags "-static"' -o sentinel

FROM scratch

LABEL org.opencontainers.image.title="Sentinel"
LABEL org.opencontainers.image.description="Docker Swarm DNS Failover Manager"
LABEL org.opencontainers.image.source="https://github.com/sguter90/sentinel"
LABEL org.opencontainers.image.vendor="Flying Lama"
LABEL org.opencontainers.image.licenses="MIT"
LABEL org.opencontainers.image.version="${VERSION}"

COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /app/sentinel /sentinel

ENTRYPOINT ["/sentinel"]