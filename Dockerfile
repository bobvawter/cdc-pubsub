FROM golang:1.15 AS builder
WORKDIR /tmp/compile
COPY . .
RUN ./build.sh

FROM scratch
VOLUME /data
WORKDIR /data
ENTRYPOINT ["/usr/bin/cdc-pubsub"]
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /usr/bin/cdc-pubsub /usr/bin/
