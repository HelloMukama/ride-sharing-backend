package main

import (
    "net/http"
    "testing"
    "time"
)

func BenchmarkRequestRide(b *testing.B) {

    InitRedis()
    InitDB()
    initAuth()
    
    client := &http.Client{}
    req, _ := http.NewRequest("POST", "http://localhost:8080/request-ride", 
        strings.NewReader(`{"lat":0.3135,"lng":32.5811}`))
    req.Header.Set("Authorization", "Bearer TEST_TOKEN")
    req.Header.Set("Content-Type", "application/json")
    
    for i := 0; i < b.N; i++ {
        resp, err := client.Do(req)
        if err != nil {
            b.Fatal(err)
        }
        resp.Body.Close()
    }
}
