#!/usr/bin/env bash
cat > a
for e in $(seq 1 3); do
    touch a.$e
done

