FROM golang:1.15 AS builder
WORKDIR /tmp/compile
COPY . .
RUN CGO_ENABLED=0 go build -v -ldflags="-s -w" -o /usr/bin/cdc-pubsub .

# This is triggered from docker-compose.test.yml
FROM builder AS test
RUN go fmt ./...
RUN go vet ./...
RUN go run golang.org/x/lint/golint -set_exit_status ./...
RUN go run honnef.co/go/tools/cmd/staticcheck -checks all ./...

FROM scratch
VOLUME /data
WORKDIR /data
ENTRYPOINT ["/usr/bin/cdc-pubsub"]
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /usr/bin/cdc-pubsub /usr/bin/
