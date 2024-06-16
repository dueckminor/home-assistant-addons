ARG BUILD_FROM
FROM $BUILD_FROM

# RUN apk add --no-cache bash python3 py3-pip

COPY rootfs /

COPY gateway /usr/local/bin/gateway
RUN chmod a+x /usr/local/bin/gateway
