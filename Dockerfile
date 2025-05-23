FROM golang:1.23-alpine AS builder

WORKDIR /app
COPY ./ ./

RUN CGO_ENABLED=0 GOOS=linux go build -a -ldflags '-extldflags "-static"' -o sentinel

FROM scratch

COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /app/sentinel /sentinel

ENTRYPOINT ["/sentinel"]