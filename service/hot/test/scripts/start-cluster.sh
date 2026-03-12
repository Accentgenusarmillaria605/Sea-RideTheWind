#!/bin/bash

cd "$(dirname "$0")/../../rpc"

echo "Building hot service..."
go build -o hot || exit 1

echo "Starting 3 instances..."

./hot -f etc/hot.yaml > hot1.log 2>&1 &
PID1=$!
echo "Instance 1 started (PID: $PID1)"

./hot -f etc/hot2.yaml > hot2.log 2>&1 &
PID2=$!
echo "Instance 2 started (PID: $PID2)"

./hot -f etc/hot3.yaml > hot3.log 2>&1 &
PID3=$!
echo "Instance 3 started (PID: $PID3)"

echo ""
echo "All instances started!"
echo "PIDs: $PID1, $PID2, $PID3"
echo ""
echo "To stop all instances:"
echo "  kill $PID1 $PID2 $PID3"
echo ""
echo "To view logs:"
echo "  tail -f hot1.log"
echo "  tail -f hot2.log"
echo "  tail -f hot3.log"
