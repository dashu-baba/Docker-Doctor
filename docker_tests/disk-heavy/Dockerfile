FROM alpine:3.20

RUN apk add --no-cache bash coreutils util-linux

WORKDIR /work
COPY worker.sh /work/worker.sh
RUN chmod +x /work/worker.sh

CMD ["/work/worker.sh"]
