ARG BUILD_FROM=ghcr.io/hassio-addons/base:16.3.6
FROM $BUILD_FROM

COPY rootfs /

COPY mqtt-bridge /usr/local/bin/mqtt-bridge
RUN chmod a+x /usr/local/bin/mqtt-bridge
