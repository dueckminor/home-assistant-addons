#!/usr/bin/env bash
PORT=${PORT:-8080}
echo "Starting service on port ${PORT}"
echo '  ---------- set ---------- '
set
echo '  ---------- python3 --version ---------- '
python3 --version
echo '  ---------- pwd ---------- '
pwd
echo '  ---------- ls -l ---------- '
ls -l
echo '  ---------- ls -l $HOME ---------- '
ls -l $HOME
echo '  ---------- ls -l /etc ---------- '
ls -l /etc
echo '  ---------- sleep 1000 ---------- '
sleep 1000
