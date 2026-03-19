# dnsmasq-docker

A Docker image for [dnsmasq](https://thekelleys.org.uk/dnsmasq/doc.html), built from source on Alpine Linux. 
Provides DNS forwarding, caching, and DHCP services with integrated Prometheus metrics export and structured logging suitable for monitoring pipelines.

A small Go init process manages dnsmasq as the container entrypoint.

## Usage

### docker run

```sh
docker run -d \
  --name dnsmasq \
  --user 1000:1000 \
  -p 53:53/tcp \
  -p 53:53/udp \
  -p 9153:9153 \
  -v ./dnsmasq.conf:/etc/dnsmasq.conf \
  --restart unless-stopped \
  ghcr.io/swonky/dnsmasq:latest
```

Add `--cap-add NET_ADMIN` if using dnsmasq as a DHCP server.

### Docker Compose

```yaml
services:
  dnsmasq:
    image: ghcr.io/swonky/dnsmasq:latest
    user: 1000:1000
    volumes:
      - "./dnsmasq.conf:/etc/dnsmasq.conf"
    ports:
      - "53:53/tcp"   # DNS
      - "53:53/udp"   # DNS
      - "9153:9153"   # Prometheus metrics
    # cap_add: [ NET_ADMIN ]  # required for DHCP
    restart: unless-stopped
```

```sh
docker compose up -d
```

Prometheus metrics are exposed on port `9153` at `/metrics`. Add it to your Prometheus scrape config:

```yaml
scrape_configs:
  - job_name: dnsmasq
    static_configs:
      - targets: ['localhost:9153']
```

## Environment variables

Any `DNSMASQ_`-prefixed environment variable is translated into a dnsmasq CLI flag by the init process: the prefix is stripped, underscores are replaced with hyphens, and the result is lowercased and prepended with `--`.

```
DNSMASQ_NO_RESOLV=true      →  --no-resolv
DNSMASQ_LISTEN_ADDRESS=::1  →  --listen-address=::1
```

True values (`true`, `yes`, `on`) produce a bare flag. False values (`false`, `no`, `off`) are omitted entirely. All other values are passed as `--flag=value`.

The following variables control the init process itself and are not passed to dnsmasq:

| Variable | Default | Description |
|---|---|---|
| `DNSMASQ_INIT_EXPORTER_LISTEN` | `:9153` | Address the Prometheus exporter listens on |
| `DNSMASQ_INIT_EXPORTER_DNSMASQ_ADDR` | `localhost:53` | dnsmasq address the exporter queries |
| `DNSMASQ_INIT_LEASES_PATH` | `/var/lib/misc/dnsmasq.leases` | Path to the DHCP leases file |

See the [dnsmasq man page](https://thekelleys.org.uk/dnsmasq/docs/dnsmasq-man.html) for all supported flags.

## Attribution

- [dnsmasq](https://thekelleys.org.uk/dnsmasq/doc.html) by Simon Kelley — DNS/DHCP server, licensed GPL-2.0
- Prometheus metrics via [google/dnsmasq_exporter](https://github.com/google/dnsmasq_exporter)
- Base image: [Alpine Linux](https://alpinelinux.org/)
