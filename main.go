package main

import (
    "os"
    "os/exec"
    "path/filepath"
)

func main() {
    // Get absolute path to src/main.go
    srcPath := filepath.Join(".", "src", "main.go")
    
    // Build the command
    cmd := exec.Command("go", "run", srcPath)
    cmd.Stdout = os.Stdout
    cmd.Stderr = os.Stderr
    
    // Run it
    if err := cmd.Run(); err != nil {
        os.Exit(1)
    }
}
