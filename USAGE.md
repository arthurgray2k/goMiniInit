# goMiniInit Usage Guide

This guide provides practical examples of using and testing `goMiniInit` both locally on Linux and inside container runtimes (Docker / Podman).

---

## 1. Command Syntax

```bash
gominiinit <command> [arguments...]
```

`goMiniInit` expects the target executable as the first argument, followed by any arguments to be passed to that program.

---

## 2. Direct Execution (Host / Local Testing)

You can run `gominiinit` directly in your terminal to launch and supervise any process:

```bash
# Run a basic command
./gominiinit echo "Hello from goMiniInit"

# Launch an interactive shell
./gominiinit /bin/bash

# Run a long-running command
./gominiinit sleep 100
```

### Exit Code Propagation
`goMiniInit` returns the exact exit status of the child process:

```bash
./gominiinit sh -c 'exit 42'
echo $?
# Output: 42
```

---

## 3. Running as PID 1 Inside Containers

### Containerfile / Dockerfile Example

```dockerfile
FROM alpine:latest

# Copy statically compiled goMiniInit binary
COPY gominiinit /goMiniInit

# Set goMiniInit as the container entrypoint
ENTRYPOINT ["/goMiniInit"]

# Default command to run
CMD ["sleep", "1000"]
```

### Building and Running

```bash
# 1. Build static binary for Linux
CGO_ENABLED=0 GOOS=linux go build -o gominiinit .

# 2. Build container image
podman build -t gominiinit-test .

# 3. Run container in detached mode
podman run -d --name test-init gominiinit-test

# 4. Inspect container process tree
podman exec test-init ps -ef
```

You will observe:
```text
PID   USER     TIME  COMMAND
    1 root      0:00 /goMiniInit sleep 1000
    7 root      0:00 sleep 1000
```

---

## 4. Testing Signal Handling

### Testing Graceful Shutdown (`SIGTERM`)

When running inside Docker/Podman, `docker stop` sends `SIGTERM` to PID 1:

```bash
# Stop container gracefully
time podman stop test-init
```

When signal forwarding is enabled, `goMiniInit` intercepts `SIGTERM` and immediately propagates it to the child process, allowing the container to terminate cleanly without hitting the 10-second timeout.

### Testing Interactive Interrupt (`SIGINT` / Ctrl+C)

```bash
# Run foreground container
podman run --rm -it gominiinit-test sh -c 'trap "echo Caught SIGINT; exit 0" INT; sleep 100'

# Pressing Ctrl+C sends SIGINT to goMiniInit, which delivers it to the shell script.
```

---

## 5. Cleaning Up

```bash
# Remove test container
podman rm -f test-init 2>/dev/null || true

# Clean compiled binaries
make clean
# Or manually:
rm -f gominiinit
```

