FROM golang:1.23-alpine

WORKDIR /app

COPY go.mod ./
RUN go mod download

COPY *.go ./

ENV TEST_INWX_USER=""
ENV TEST_INWX_PASSWORD=""
ENV TEST_INWX_RECORD_ID=""
ENV TEST_IP="1.2.3.4"
ENV LOG_LEVEL="DEBUG"

CMD ["go", "test", "-v", "./..."]