FROM alpine:3.20

RUN apk add --no-cache bash

WORKDIR /app

# Always unhealthy
RUN printf '%s\n' \
'#!/usr/bin/env sh' \
'echo "healthcheck failing intentionally"' \
'exit 1' \
> /app/healthcheck.sh && chmod +x /app/healthcheck.sh

CMD ["sh", "-c", "echo 'container running; healthcheck will fail'; sleep 1000000"]

