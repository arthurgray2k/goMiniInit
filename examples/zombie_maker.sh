#!/bin/sh
echo "=== Zombie Generator Started ==="
echo "Main process PID: $$"

# Spawn an intermediate process that spawns grandchildren and dies
sh -c '
for i in 1 2 3 4 5; do
    (sleep 1; exit 0) &
done
echo "Intermediate parent exiting... 5 background workers are now orphaned to PID 1!"
'

echo "Waiting 3 seconds for orphaned workers to exit and become zombies..."
sleep 3

echo "Process table inside container:"
ps -ef

echo "Sleeping to allow external inspection..."
sleep 60
