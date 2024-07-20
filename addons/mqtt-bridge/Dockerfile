ARG BUILD_FROM
FROM $BUILD_FROM

COPY rootfs /

COPY mqtt-bridge /usr/local/bin/mqtt-bridge
RUN chmod a+x /usr/local/bin/mqtt-bridge
