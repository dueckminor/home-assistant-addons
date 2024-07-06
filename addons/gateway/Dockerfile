ARG BUILD_FROM
FROM $BUILD_FROM

COPY rootfs /

COPY gateway /usr/local/bin/gateway
COPY dist /usr/local/bin/dist/
RUN chmod a+x /usr/local/bin/gateway
