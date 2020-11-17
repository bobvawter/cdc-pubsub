FROM golang:1.15 AS builder

WORKDIR /tmp/compile
COPY . .

RUN ./build.sh

FROM ubuntu:20.04
COPY --from=builder /cdc-pubsub /usr/bin/
VOLUME /data
WORKDIR /data
RUN apt-get update && apt-get install -y ca-certificates \
    && rm -rf /var/lib/apt/lists/*
ENTRYPOINT ["/usr/bin/cdc-pubsub"]