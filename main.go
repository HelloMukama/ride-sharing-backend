package main

import (
    "os"
    "os/exec"
    "path/filepath"
)

func main() {
    cmd := exec.Command("go", "run", filepath.Join("src", "*.go"))
    cmd.Stdout = os.Stdout
    cmd.Stderr = os.Stderr
    cmd.Stdin = os.Stdin
    cmd.Dir = "." // Run from project root
    cmd.Run()
}
