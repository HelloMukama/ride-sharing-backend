package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/gorilla/mux"
	"github.com/joho/godotenv"
)

func homeHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Ride-Sharing Service Backend is Running!"))
}

func main() {
	// Load environment variables from the base directory's .env file
	err := godotenv.Load(".env") 
	if err != nil {
		log.Fatalf("Error loading .env file: %v", err)
	}

	// Get port from environment variable
	port := os.Getenv("PORT")
	
	r := mux.NewRouter()
	r.HandleFunc("/", homeHandler).Methods("GET")

	fmt.Printf("Server running on port %s...\n", port)
	log.Fatal(http.ListenAndServe(":"+port, r))
}
