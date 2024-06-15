#!/usr/bin/env bash
PORT=${PORT:-8080}
echo "Starting service on port ${PORT}"
set
python3 --version
ping supervisor