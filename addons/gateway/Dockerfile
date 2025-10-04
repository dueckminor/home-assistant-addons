# Simple Dockerfile using pre-compiled binaries
ARG BUILD_FROM=ghcr.io/hassio-addons/base:16.3.6
FROM $BUILD_FROM

COPY rootfs /
COPY gateway /usr/local/bin/gateway
COPY dist /usr/local/bin/dist/

RUN chmod a+x /usr/local/bin/gateway
