FROM golang:1.22-bullseye AS builder
WORKDIR /src
COPY . .
RUN set -e; \
    go mod tidy; \
    go build -o ./bin/syslog -trimpath -ldflags '-s -w' .

FROM debian:11
RUN set -e; \
    apt-get update; \
    apt-get upgrade -y curl ca-certificates xz-utils
COPY --from=builder /src/bin/syslog /usr/bin/syslog
WORKDIR /data
ENTRYPOINT ["/usr/bin/syslog"]