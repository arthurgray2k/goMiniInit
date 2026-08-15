# Using `goMiniInit` in `goMiniContainer`

This guide explains how to use `goMiniInit` as the **PID 1 process supervisor** inside your custom container engine, [`goMiniContainer`](../goMiniContainer).

---

## 🏛️ Architecture & Separation of Concerns

```text
Host Operating System
│
└── goMiniContainer (Container Runtime)
       │
       │ 1. Creates Linux namespaces (CLONE_NEWPID, CLONE_NEWNS, CLONE_NEWUTS)
       │ 2. Sets up pivot_root & Cgroups v2 resource limits
       │ 3. Hands off container entrypoint to /goMiniInit
       ▼
   [ Container PID Namespace ]
   ├── PID 1: goMiniInit (Supervises, traps signals, reaps zombies)
   │     │
   │     ├── (creates process group & forwards signals to -PGID)
   │     ▼
   └── PID 2+: Supervised Application (e.g. /bin/sh, Python, Node, Go)
```

| Component | Responsibility |
| :--- | :--- |
| **`goMiniContainer`** | **Creates the environment**: Isolates PID, Mount, UTS namespaces, applies memory/PID limits, and sets up `pivot_root`. |
| **`goMiniInit`** | **Manages the lifecycle**: Runs as PID 1, traps signals, reaps orphaned zombies, and ensures clean container exit. |

---

## 🚀 Setup & Installation (3 Steps)

### Step 1: Build the Static `gominiinit` Binary
From inside the `goMiniInit/` directory:
```bash
cd /home/mint/golang_toolshed/goMiniInit
CGO_ENABLED=0 GOOS=linux go build -o gominiinit .
```

### Step 2: Prepare `alpine-rootfs` in `goMiniContainer`
If `alpine-rootfs` is not already downloaded:
```bash
cd /home/mint/golang_toolshed/goMiniContainer
mkdir -p alpine-rootfs

# Download and extract minimal Alpine rootfs
curl -L -o alpine-minirootfs.tar.gz "https://dl-cdn.alpinelinux.org/alpine/latest-stable/releases/x86_64/alpine-minirootfs-3.21.3-x86_64.tar.gz"
tar -xzf alpine-minirootfs.tar.gz -C alpine-rootfs
rm alpine-minirootfs.tar.gz
```

### Step 3: Copy `gominiinit` into `alpine-rootfs/`
```bash
cp /home/mint/golang_toolshed/goMiniInit/gominiinit /home/mint/golang_toolshed/goMiniContainer/alpine-rootfs/goMiniInit
chmod +x /home/mint/golang_toolshed/goMiniContainer/alpine-rootfs/goMiniInit
```

---

## 🧪 Practical Examples

### Example 1: Interactive Shell as PID 1 Supervisor

Run an interactive container:
```bash
cd /home/mint/golang_toolshed/goMiniContainer
go build -o gominicontainer ./cmd/goMiniContainer

sudo ./gominicontainer run -rootfs ./alpine-rootfs /goMiniInit /bin/sh
```

Inside the container shell, inspect the process hierarchy:
```sh
/ # ps -ef
```
**Output:**
```text
PID   USER     TIME  COMMAND
    1 root      0:00 /goMiniInit /bin/sh
   11 root      0:00 /bin/sh
   15 root      0:00 ps -ef
```
* `/goMiniInit` is **PID 1** (root of the container namespace).
* `/bin/sh` is **PID 11** (supervised child in its own process group).

---

### Example 2: Automatic Zombie Reaping Demonstration

Inside the container, run a test that spawns background child processes that terminate immediately:

```sh
# 1. Ensure /dev/null exists for background jobs
mknod -m 666 /dev/null c 1 3 2>/dev/null || true

# 2. Run automated multi-worker lifecycle script
sh -c '
echo "=== Step 1: Launching 3 background workers ==="
sleep 4 &
sleep 4 &
sleep 4 &

echo "=== Step 2: Active process table ==="
ps -ef

echo "=== Step 3: Waiting for workers to exit... ==="
sleep 6

echo "=== Step 4: Final process table (Reaped by goMiniInit) ==="
ps -ef
'
```

**Observation:**
* In Step 2, the workers appear actively running.
* In Step 4, all 3 finished workers were **immediately reaped by `goMiniInit`**.
* **Zero `<defunct>` zombie processes remain!**

---

### Example 3: Exit Code Preservation

`goMiniInit` transparently passes the child application's exact exit code back to `goMiniContainer`:

```bash
sudo ./gominicontainer run -rootfs ./alpine-rootfs /goMiniInit /bin/sh -c "exit 42"
echo "Exit status: $?"
# Output: Exit status: 42
```

---

### Example 4: Resource Limits with Cgroups v2

You can combine `goMiniContainer`'s cgroup limits with `goMiniInit`'s process supervision:

```bash
# Limit container to 64MB RAM and maximum 20 PIDs
sudo ./gominicontainer run -rootfs ./alpine-rootfs -mem 64 -pids 20 /goMiniInit /bin/sh
```

---

## 💡 Troubleshooting & Notes

1. **Why does `/bin/sh` print `can't access tty; job control turned off`?**
   * In Stage 7, `goMiniInit` invokes `setsid()` to place the child into a new session and process group for group-wide signal propagation (`kill(-PGID, sig)`).
   * Creating a new session without a dedicated controlling pseudo-terminal (PTY) informs the shell that interactive job control (`Ctrl+Z`, `fg`, `bg`) is disabled. The shell remains 100% operational for commands, scripts, and background pipelines.

2. **Why do background jobs (`&`) complain about `/dev/null`?**
   * When job control is off, shells redirect background stdin from `/dev/null`.
   * Since `goMiniContainer` mounts a fresh `tmpfs` on `/dev`, create the null device with:
     ```sh
     mknod -m 666 /dev/null c 1 3
     ```
