#!/bin/bash

cd "$(dirname "$0")/../../rpc"

echo "Building hot service..."
go build -o hot || exit 1

echo "Starting single instance..."
./hot -f etc/hot.yaml

echo "Service stopped."
