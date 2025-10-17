# Multi-arch Dockerfile with platform selector
ARG BUILD_FROM=ghcr.io/hassio-addons/base:18.2.1

# Platform selector stage
FROM alpine:3.22 AS platform_selector

COPY mqtt-bridge-* /binaries/

# Select the correct binary based on runtime architecture detection
RUN ARCH=$(uname -m) && \
    if [ "$ARCH" = "x86_64" ]; then \
        ARCH=amd64; \
    elif [ "$ARCH" = "aarch64" ] || [ "$ARCH" = "arm64" ]; then \
        ARCH=arm64; \
    fi; \
    echo "Using mqtt-bridge-$ARCH binary"; \
    cp /binaries/mqtt-bridge-$ARCH /binaries/mqtt-bridge

# Runtime stage
FROM $BUILD_FROM

COPY rootfs /
COPY --from=platform_selector /binaries/mqtt-bridge /usr/local/bin/mqtt-bridge

RUN chmod a+x /usr/local/bin/mqtt-bridge
