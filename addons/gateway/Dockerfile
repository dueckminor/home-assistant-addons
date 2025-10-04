# Multi-arch Dockerfile with platform selector
ARG BUILD_FROM=ghcr.io/hassio-addons/base:18.1.4

# Platform selector stage
FROM alpine:3.22 AS platform_selector

COPY gateway-* /binaries/

# Select the correct binary based on runtime architecture detection
RUN ARCH=$(uname -m) && \
    if [ "$ARCH" = "x86_64" ]; then \
        ARCH=amd64; \
    elif [ "$ARCH" = "aarch64" ]; then \
        ARCH=arm64; \
    fi; \
    echo "Using gateway-$ARCH binary"; \
    cp /binaries/gateway-$ARCH /binaries/gateway

# Runtime stage
FROM $BUILD_FROM

COPY rootfs /
COPY --from=platform_selector /binaries/gateway /usr/local/bin/gateway
COPY dist /usr/local/bin/dist/

RUN chmod a+x /usr/local/bin/gateway
