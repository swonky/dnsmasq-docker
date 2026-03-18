variable "VERSION" {
  default = "2.91"
}

variable "COPTS" {
  default = "-DNO_TFTP -DNO_DHCP -DNO_SCRIPT -DNO_DBUS -DNO_LUA"
}

variable "CHECKSUM" {
  default = "sha256:2d26a048df452b3cfa7ba05efbbcdb19b12fe7a0388761eb5d00938624bd76c8"
}

variable "REGISTRY" {
  default = "ghcr.io/swonky"
}

variable "AUTHOR" {
  default = "swonky"
}

variable "DESCRIPTION" {
  default = join(" ", [
    "dnsmasq packaged as a minimal, production-ready container with",
    "integrated Prometheus metrics export. Provides DNS forwarding,",
    "caching, and DHCP services with structured logging suitable for",
    "monitoring pipelines."
  ])
}

group "default" {
  targets = ["dnsmasq" ]
}

target "dnsmasq" {
  context = "."
  dockerfile = "Dockerfile"

  platforms = ["linux/amd64"]

  annotations = [
    "org.opencontainers.image.author=${AUTHOR}"
    "org.opencontainers.image.description=${DESCRIPTION}"
  ]

  args = {
    VERSION  = "${VERSION}"
    COPTS    = ""
    CHECKSUM = "${CHECKSUM}"
    DESCRIPTION = "${DESCRIPTION}"
  }

  tags = [
    "${REGISTRY}/dnsmasq:${VERSION}",
    "${REGISTRY}/dnsmasq:latest"
  ]
}
# target "dnsmasq-minimal" {
#   context = "."
#   dockerfile = "Dockerfile"
#
#   args = {
#     VERSION  = "${VERSION}"
#     COPTS    = "${COPTS}"
#     CHECKSUM = "${CHECKSUM}"
#   }
#
#   tags = [
#     "${REGISTRY}/dnsmasq:${VERSION}-minimal",
#     "${REGISTRY}/dnsmasq:minimal"
#   ]
# }

