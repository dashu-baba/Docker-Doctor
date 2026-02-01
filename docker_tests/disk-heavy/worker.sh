#!/usr/bin/env bash
set -euo pipefail

DIR="${DIR:-/data}"
FILE_MB="${FILE_MB:-2048}"     # size of each file
LOOPS="${LOOPS:-999999}"
echo "IO worker: dir=$DIR file=${FILE_MB}MB loops=$LOOPS"

mkdir -p "$DIR"
i=0
while [ "$i" -lt "$LOOPS" ]; do
  name="$DIR/blob_$i.bin"
  echo "write $name"
  dd if=/dev/urandom of="$name" bs=1M count="$FILE_MB" conv=fsync status=progress
  echo "read/verify $name"
  sha256sum "$name" >/dev/null
  rm -f "$name"
  i=$((i+1))
done
