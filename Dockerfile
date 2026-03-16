# syntax=docker/dockerfile:1

FROM alpine:3.20 AS build

RUN apk add --no-cache \
  build-base \ 
  linux-headers

ARG VERSION
ARG COPTS
ARG CHECKSUM

WORKDIR /build

ADD --link --unpack --checksum=${CHECKSUM} \
  https://thekelleys.org.uk/dnsmasq/dnsmasq-${VERSION}.tar.gz \
  /build/

WORKDIR /build/dnsmasq-${VERSION}

RUN make COPTS="$COPTS" \
  LDFLAGS="-s"

# ----

FROM golang:1.23-alpine AS build-init

WORKDIR /build

COPY go.mod ./
COPY main.go .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o dnsmasq-init .

# ----

FROM alpine:3.20 AS runtime

ARG VERSION

LABEL org.opencontainers.image.title="dnsmasq"
LABEL org.opencontainers.image.description="minimal dnsmasq container"
LABEL org.opencontainers.image.source="https://thekelleys.org.uk/dnsmasq/"

COPY --from=build /build/dnsmasq-${VERSION}/src/dnsmasq /usr/sbin/dnsmasq
COPY --from=build /build/dnsmasq-${VERSION}/dnsmasq.conf.example /etc/dnsmasq.conf
COPY --from=build-init /build/dnsmasq-init /dnsmasq-init

EXPOSE 53/tcp
EXPOSE 53/udp

ENV DNSMASQ_KEEP_IN_FOREGROUND=TRUE
ENV DNSMASQ_LOG_FACILITY="-"
ENV DNSMASQ_CONF_FILE="/etc/dnsmasq.conf"
ENV DNSMASQ_DOMAIN_NEEDED=TRUE
ENV DNSMASQ_BOGUS_PRIV=TRUE
ENV DNSMASQ_STOP_DNS_REBIND=TRUE

STOPSIGNAL SIGTERM

ENTRYPOINT ["/dnsmasq-init"]

HEALTHCHECK --interval=30s --timeout=3s --start-period=10s \
  CMD pidof dnsmasq || exit 1

