package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"testing"
)

var testBinaryPath string

func TestMain(m *testing.M) {
	tempDir, err := os.MkdirTemp("", "gominiinit-test-*")
	if err != nil {
		panic(err)
	}
	defer os.RemoveAll(tempDir)

	testBinaryPath = filepath.Join(tempDir, "gominiinit")
	buildCmd := exec.Command("go", "build", "-o", testBinaryPath, ".")
	if err := buildCmd.Run(); err != nil {
		panic(err)
	}

	os.Exit(m.Run())
}

func TestGoMiniInit_Success(t *testing.T) {
	cmd := exec.Command(testBinaryPath, "echo", "hello world")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("expected success, got err: %v, output: %s", err, string(output))
	}
	if string(output) != "hello world\n" {
		t.Fatalf("expected 'hello world\\n', got '%s'", string(output))
	}
}

func TestGoMiniInit_ExitCode(t *testing.T) {
	cmd := exec.Command(testBinaryPath, "sh", "-c", "exit 42")
	err := cmd.Run()
	if err == nil {
		t.Fatalf("expected error with exit code 42, got nil")
	}

	exitErr, ok := err.(*exec.ExitError)
	if !ok {
		t.Fatalf("expected *exec.ExitError, got %T", err)
	}

	if exitErr.ExitCode() != 42 {
		t.Fatalf("expected exit code 42, got %d", exitErr.ExitCode())
	}
}

func TestGoMiniInit_ReapFunction(t *testing.T) {
	var wstatus syscall.WaitStatus
	pid, err := syscall.Wait4(-1, &wstatus, syscall.WNOHANG, nil)
	if pid > 0 {
		t.Logf("Reaped leftover background PID: %d", pid)
	}
	if err != nil && err != syscall.ECHILD {
		t.Fatalf("unexpected error from Wait4: %v", err)
	}
}
