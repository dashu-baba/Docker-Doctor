FROM python:3.12-slim

WORKDIR /app
RUN pip install --no-cache-dir --disable-pip-version-check numpy

# CPU-heavy loop: matrix multiply + hashing
RUN printf '%s\n' \
'import os, time, hashlib' \
'import numpy as np' \
'' \
'threads = int(os.getenv("THREADS","2"))' \
'size = int(os.getenv("MATRIX","1200"))' \
'print(f"CPU worker: threads={threads} matrix={size}")' \
'' \
'# numpy uses BLAS threads; limit if you want predictable CPU usage' \
'os.environ["OMP_NUM_THREADS"]=str(threads)' \
'os.environ["OPENBLAS_NUM_THREADS"]=str(threads)' \
'' \
'while True:' \
'    a = np.random.rand(size, size)' \
'    b = np.random.rand(size, size)' \
'    c = a @ b' \
'    h = hashlib.sha256(c.tobytes()).hexdigest()' \
'    print("tick", h[:12])' \
'    time.sleep(0.1)' \
> /app/worker.py

CMD ["python", "/app/worker.py"]
