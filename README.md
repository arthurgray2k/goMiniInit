# goMiniInit: A Minimal Container Init in Go

`goMiniInit` is an educational, minimalist process supervisor (init process) written in Go, designed to run as **PID 1** inside Linux container environments (such as Docker, Podman, and custom container runtimes like [`goMiniContainer`](../goMiniContainer)).

---

## 📖 Why Does a Container Need an Init Process?

When a container starts, a new **Linux PID Namespace** (`CLONE_NEWPID`) is created. The first process inside this namespace is assigned **PID 1**.

In Linux, PID 1 has two unique kernel-level behaviors compared to ordinary processes:

```text
Host Operating System (Root PID Namespace)
├── PID 1 (systemd)
└── PID 48291 (container entrypoint)
        │
        ▼ (Virtual PID inside container namespace)
    PID 1 (goMiniInit)
        │
        ├── (spawns & supervises)
        ▼
    PID 7 (target application)
```

### 1. The Signal Immunity Shield
* **Normal processes (`PID > 1`)**: If a process receives `SIGTERM` or `SIGINT` without a custom handler, the Linux kernel applies default termination actions.
* **PID 1**: The Linux kernel **ignores** all terminating signals unless PID 1 has explicitly registered a custom signal handler. If an application running as PID 1 does not install signal handlers, commands like `docker stop` hang for 10 seconds before being forcibly killed with `SIGKILL`.

### 2. Zombie Process Reaping
* In Linux, when any parent process dies before its child, the child becomes an **orphan**.
* The Linux kernel automatically re-parents all orphans to **PID 1**.
* When an orphan terminates, it remains in the process table as a **zombie (`<defunct>`)** until its parent (PID 1) calls `wait()` / `waitpid()` to acknowledge its exit code. If PID 1 does not reap zombies, the kernel PID table slowly leaks.

---

## 🎯 Project Architecture & Responsibilities

`goMiniInit` solves these problems by acting as a lightweight shim between the container runtime and the application:

```text
Docker / Podman / goMiniContainer
               │
               │ (creates PID namespace)
               ▼
          goMiniInit (PID 1)
          ├── 1. Spawns child process (os/exec -> clone + execve)
          ├── 2. Traps & forwards signals (os/signal -> SIGTERM, SIGINT, etc.)
          ├── 3. Reaps orphaned zombie processes (syscall.Wait4 / SIGCHLD)
          └── 4. Propagates exit code to the runtime
               │
               ▼
          Target Application (e.g. Node, Python, Go, Shell)
```

---

## 🚦 Incremental Stages

- [x] **Stage 0: Observe PID 1 & Signal Shield**
- [x] **Stage 1: Minimal Process Launcher** (`os/exec`, command parsing)
- [x] **Stage 2: Run as PID 1 in Container** (`Dockerfile`, PID namespace verification)
- [x] **Stage 3: Signal Trapping & Forwarding** (`os/signal`, `SIGTERM`, graceful shutdown)
- [x] **Stage 4: Exit Status Propagation** (`ProcessState.ExitCode()`, POSIX `128 + signal`)
- [x] **Stage 5: Orphan & Zombie Process Mechanics** (Observing `<defunct>` states)
- [x] **Stage 6: Non-blocking Subreaping** (`syscall.Wait4`, `WNOHANG`, `SIGCHLD`)
- [x] **Stage 7: Process Groups & Sessions** (`setsid`, negative PID signaling)
- [ ] **Stage 8: Comparison with `dumb-init`** (Architecture and production trade-offs)
- [ ] **Stage 9: Comparison with `go-init`** (Lifecycle hooks pre/main/post)
- [x] **Stage 10: Integration with `goMiniContainer`**

---

## 🚀 Integration with `goMiniContainer`

`goMiniInit` was built to be the process supervisor for [`goMiniContainer`](../goMiniContainer).

### The Architecture:
```text
Host Operating System
│
└── goMiniContainer (Container Engine)
       │
       │ 1. Creates Namespaces (CLONE_NEWPID, CLONE_NEWNS, CLONE_NEWUTS)
       │ 2. Sets up pivot_root & Cgroups
       │ 3. Hands off namespace to /goMiniInit
       ▼
   [ Container PID Namespace ]
   ├── PID 1: goMiniInit (Supervises, traps signals, reaps zombies)
   │     │
   │     ├── (spawns & manages PGID)
   │     ▼
   └── PID 2+: Target Application & Child Workers
```

### How to use `goMiniInit` inside `goMiniContainer`:
1. Copy the statically built `gominiinit` binary into the container's root filesystem (e.g., at `/goMiniInit` or `/sbin/init`):
   ```bash
   cp gominiinit /path/to/rootfs/goMiniInit
   ```
2. Launch your container command wrapped by `goMiniInit`:
   ```bash
   ./gominicontainer run -rootfs /path/to/rootfs /goMiniInit <application> [args...]
   ```
This immediately equips `goMiniContainer` with automatic zombie reaping, clean SIGTERM/SIGINT signal forwarding, and process group lifecycle control!

📖 For complete examples, workflows, and cgroups testing, see the **[goMiniContainer Integration Guide](GOMINICONTAINER_INTEGRATION.md)**.

---

## 🛠️ Build and Installation

Requires **Go 1.26+**. You can use either `make` shortcuts or standard Go/container CLI commands:

### Option A: Using `make` shortcuts
```bash
make build         # Compile local binary
make docker-build  # Compile static Linux binary and build container image
make docker-run    # Run the container image
make test          # Run tests
make clean         # Remove compiled binary
```

### Option B: Using direct CLI commands (No `make` required)
```bash
# Compile local binary
go build -o gominiinit .

# Compile static Linux binary and build container image
CGO_ENABLED=0 GOOS=linux go build -o gominiinit .
podman build -t gominiinit-test .   # or: docker build -t gominiinit-test .

# Run the container
podman run --rm -it gominiinit-test # or: docker run --rm -it gominiinit-test

# Run tests
go test -v ./...

# Clean up
rm -f gominiinit
```


---

## 🔗 Related Projects & Acknowledgments

This project draws inspiration from and references the following implementations:

* **[dumb-init (Yelp)](https://github.com/Yelp/dumb-init)** — A minimal, highly optimized C-based init system for containers that solves PID 1 signal forwarding and orphan reaping.
* **[go-init (Adam Kaplan)](https://github.com/adambkaplan/go-init)** — A container init process written in Go with support for multi-phase execution (pre-init, main, and post-init lifecycle hooks).

