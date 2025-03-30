package main

import (
    "log"
    "os"
    "time"
)

func init() {
    log.SetOutput(os.Stdout)
    log.Println("Starting initialization...")
    
    // Wait for dependencies
    time.Sleep(5 * time.Second)
}
