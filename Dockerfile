ARG BUILD_FROM=ghcr.io/home-assistant/base:latest
FROM ${BUILD_FROM}

ARG TRMNL_VERSION="v0.3.0"
ARG TARGETARCH

# Download the correct trmnld binary for the target architecture
RUN set -eu; \
    case "${TARGETARCH}" in \
      "amd64")  ARCH="linux-amd64" ;; \
      "arm64")  ARCH="linux-arm64" ;; \
      *)        echo "Unsupported architecture: ${TARGETARCH}" && exit 1 ;; \
    esac; \
    URL="https://github.com/gesellix/go-trmnl/releases/download/${TRMNL_VERSION}/trmnld-${TRMNL_VERSION}-${ARCH}"; \
    echo "Downloading ${URL}"; \
    wget -q -O /usr/local/bin/trmnld "${URL}"; \
    chmod +x /usr/local/bin/trmnld

COPY run.sh /run.sh
RUN chmod +x /run.sh

CMD ["/run.sh"]
