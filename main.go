package main

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"sync"
	"syscall"
)

// reapChildren calls wait4 with WNOHANG to harvest any terminated child processes (orphans)
// and prevent them from lingering as zombies in the Linux kernel process table.
func reapChildren(mainPid int, mainExitChan chan<- int, once *sync.Once) {
	for {
		var wstatus syscall.WaitStatus
		// Wait on ANY terminated child process (-1) without blocking (WNOHANG)
		pid, err := syscall.Wait4(-1, &wstatus, syscall.WNOHANG, nil)
		if err != nil || pid <= 0 {
			break
		}

		// If the process reaped is our primary supervised child, capture its exit code
		if pid == mainPid {
			code := 0
			if wstatus.Signaled() {
				code = 128 + int(wstatus.Signal())
			} else {
				code = wstatus.ExitStatus()
			}
			once.Do(func() {
				mainExitChan <- code
			})
		}
	}
}

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintf(os.Stderr, "usage: %s <command> [args...]\n", os.Args[0])
		os.Exit(1)
	}

	commandName := os.Args[1]
	commandArgs := os.Args[2:]

	cmd := exec.Command(commandName, commandArgs...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	// Start child in a new session / process group (PGID == child PID)
	// This allows goMiniInit to signal the entire process group rooted at the child.
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setsid: true,
	}

	// 1. Start primary child process (clone + setsid + execve)
	if err := cmd.Start(); err != nil {
		fmt.Fprintf(os.Stderr, "goMiniInit: failed to start %s: %v\n", commandName, err)
		os.Exit(1)
	}

	mainPid := cmd.Process.Pid
	mainExitChan := make(chan int, 1)
	var once sync.Once

	// 2. Set up signal channel to intercept signals sent to PID 1
	sigChan := make(chan os.Signal, 64)
	signal.Notify(sigChan,
		syscall.SIGTERM,
		syscall.SIGINT,
		syscall.SIGQUIT,
		syscall.SIGHUP,
		syscall.SIGCHLD, // Delivered when any child or orphan terminates
		syscall.SIGUSR1,
		syscall.SIGUSR2,
		syscall.SIGWINCH,
	)

	// 3. Asynchronous signal router and zombie reaper
	go func() {
		for sig := range sigChan {
			if sig == syscall.SIGCHLD {
				// Reap any dead child/orphan processes
				reapChildren(mainPid, mainExitChan, &once)
			} else {
				// In Linux, passing negative PID (-mainPid) broadcasts the signal
				// to every process in the child's process group.
				if sigVal, ok := sig.(syscall.Signal); ok {
					_ = syscall.Kill(-mainPid, sigVal)
				} else if cmd.Process != nil {
					_ = cmd.Process.Signal(sig)
				}
			}
		}
	}()

	// 4. Concurrently wait on primary child via cmd.Wait()
	go func() {
		err := cmd.Wait()
		code := 0
		if err != nil {
			var exitErr *exec.ExitError
			if errors.As(err, &exitErr) {
				if status, ok := exitErr.Sys().(syscall.WaitStatus); ok {
					if status.Signaled() {
						code = 128 + int(status.Signal())
					} else {
						code = status.ExitStatus()
					}
				} else {
					code = exitErr.ExitCode()
				}
			} else {
				// If cmd.Wait() errors because wait4 already reaped it, return
				return
			}
		}
		once.Do(func() {
			mainExitChan <- code
		})
	}()

	// 5. Block until the primary supervised child terminates
	exitCode := <-mainExitChan

	// Unregister signal handlers
	signal.Stop(sigChan)
	close(sigChan)

	// Final sweep to reap any remaining orphans before PID 1 exits
	reapChildren(mainPid, mainExitChan, &once)

	os.Exit(exitCode)
}
