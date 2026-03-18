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

RUN make \
  COPTS="$COPTS" \
  LDFLAGS="-s"

# ----

FROM golang:1.24-alpine AS build-init

WORKDIR /build

COPY init/* ./

RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o dnsmasq-init .

# ----

FROM alpine:3.20 AS runtime

ARG VERSION

ARG AUTHOR
ARG DESCRIPTION

LABEL org.opencontainers.image.title="dnsmasq"
LABEL org.opencontainers.image.url="https://github.com/swonky/dnsmasq-docker"
LABEL org.opencontainers.image.source="https://github.com/swonky/dnsmasq-docker"
LABEL org.opencontainers.image.documentation="https://github.com/swonky/dnsmasq-docker/blob/main/README.md"
LABEL org.opencontainers.image.licenses="GPL-2.0"
LABEL org.opencontainers.image.vendor="${AUTHOR}"
LABEL org.opencontainers.image.author="${AUTHOR}"
LABEL org.opencontainers.image.description="${DESCRIPTION}"

COPY --from=build /build/dnsmasq-${VERSION}/src/dnsmasq /usr/sbin/dnsmasq
COPY --from=build /build/dnsmasq-${VERSION}/dnsmasq.conf.example /etc/dnsmasq.conf
COPY --from=build-init /build/dnsmasq-init /dnsmasq-init

EXPOSE 53/tcp
EXPOSE 53/udp
EXPOSE 9153

ENV DNSMASQ_KEEP_IN_FOREGROUND=TRUE \
  DNSMASQ_LOG_FACILITY="-" \
  DNSMASQ_CONF_FILE="/etc/dnsmasq.conf" \
  DNSMASQ_DOMAIN_NEEDED=TRUE \
  DNSMASQ_BOGUS_PRIV=TRUE \
  DNSMASQ_STOP_DNS_REBIND=TRUE \
  DNSMASQ_NO_DAEMON=TRUE

STOPSIGNAL SIGTERM

ENTRYPOINT ["/dnsmasq-init"]

HEALTHCHECK --interval=30s --timeout=3s --start-period=10s \
  CMD pidof dnsmasq || exit 1

