FROM python:3.12-slim

WORKDIR /app

# Simple memory hog that will get OOM-killed under a low memory limit.
RUN printf '%s\n' \
'import os, time' \
'mb = int(os.getenv("ALLOC_MB", "512"))' \
'print(f"oom_hog: allocating ~{mb} MB then holding...")' \
'b = bytearray(mb * 1024 * 1024)' \
'for i in range(0, len(b), 1024 * 1024):' \
'    b[i] = 1' \
'print("oom_hog: allocated; sleeping")' \
'time.sleep(10**9)' \
> /app/oom_hog.py

CMD ["python", "/app/oom_hog.py"]

