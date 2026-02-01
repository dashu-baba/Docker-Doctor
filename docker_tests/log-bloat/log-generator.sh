#!/bin/bash

# Generate large logs to simulate log bloat
# This will create logs larger than 100MB to trigger the rule

echo "Starting log generator for log bloat test"

# Generate a lot of log output
for i in {1..100000}; do
    echo "This is a log line number $i with some filler text to make it longer and consume more space in the log files. $(date) - Random data: $RANDOM $RANDOM $RANDOM $RANDOM"
done

echo "Log generation complete"

# Sleep to keep container running
sleep infinity