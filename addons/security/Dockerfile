# Multi-arch Dockerfile with platform selector
ARG BUILD_FROM=ghcr.io/hassio-addons/base:20.0.1

# Platform selector stage
FROM alpine:3.23 AS platform_selector

COPY security-* /binaries/

# Select the correct binary based on runtime architecture detection
RUN ARCH=$(uname -m) && \
    if [ "$ARCH" = "x86_64" ]; then \
        ARCH=amd64; \
    elif [ "$ARCH" = "aarch64" ] || [ "$ARCH" = "arm64" ]; then \
        ARCH=arm64; \
    fi; \
    echo "Using security-$ARCH binary"; \
    cp /binaries/security-$ARCH /binaries/security

# Runtime stage
FROM $BUILD_FROM

COPY rootfs /
COPY --from=platform_selector /binaries/security /usr/local/bin/security

RUN chmod a+x /usr/local/bin/security
