FROM alpine:3.13
RUN apk add --no-cache bash python3 py3-pip
COPY run.sh /run.sh
RUN chmod a+x /run.sh
ENV PORT 10080
CMD [ "/run.sh" ]
