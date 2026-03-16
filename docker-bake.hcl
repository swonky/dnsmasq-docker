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

group "default" {
  targets = ["dnsmasq", "dnsmasq-minimal"]
}

target "dnsmasq" {
  context = "."
  dockerfile = "Dockerfile"

  args = {
    VERSION  = "${VERSION}"
    COPTS    = ""
    CHECKSUM = "${CHECKSUM}"
  }

  tags = [
    "${REGISTRY}/dnsmasq:${VERSION}",
    "${REGISTRY}/dnsmasq:latest"
  ]
}


target "dnsmasq-minimal" {
  context = "."
  dockerfile = "Dockerfile"

  args = {
    VERSION  = "${VERSION}"
    COPTS    = "${COPTS}"
    CHECKSUM = "${CHECKSUM}"
  }

  tags = [
    "${REGISTRY}/dnsmasq:${VERSION}-minimal",
    "${REGISTRY}/dnsmasq:minimal"
  ]
}

