package main

import (
    "os"
    "os/exec"
    "path/filepath"
    "testing"
)

func TestDetectsRunningExecutableReplacedByPackageUpdate(t *testing.T) {
    sleep, err := exec.LookPath("sleep")
    if err != nil { t.Skip("sleep unavailable") }
    image, err := os.ReadFile(sleep); if err != nil { t.Fatal(err) }
    executable := filepath.Join(t.TempDir(), "vesktop")
    if err := os.WriteFile(executable, image, 0700); err != nil { t.Fatal(err) }
    cmd := exec.Command(executable, "10")
    if err := cmd.Start(); err != nil { t.Fatal(err) }
    defer func() { _ = cmd.Process.Kill(); _ = cmd.Wait() }()
    if err := os.Remove(executable); err != nil { t.Fatal(err) }
    if got := vesktopProcesses()[cmd.Process.Pid]; got != executable {
        t.Fatalf("running Vesktop lost after executable replacement: got %q, want %q", got, executable)
    }
}
