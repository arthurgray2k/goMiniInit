#!/bin/sh
echo "=== Multiprocess App Started ==="
echo "Parent PID: $$"

# Spawn 2 background workers in the same process group
sleep 100 &
PID1=$!
sleep 100 &
PID2=$!

echo "Spawned background worker PIDs: $PID1 and $PID2"
echo "Process tree before signal:"
ps -ef

# Wait for children
wait
