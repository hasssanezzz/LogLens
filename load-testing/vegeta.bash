#!/bin/bash

# params: port, rate, duration
echo "POST http://127.0.0.1:$1" | vegeta attack \
    -rate=$2 \
    -duration=$3 \
    -header "Content-Type: application/json" \
    -header "KV-environment: production" \
    -header "KV-level: info" \
    -body body.txt | vegeta report
