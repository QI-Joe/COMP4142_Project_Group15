package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
)

type Data struct {
	Message string `json:"message"`
}

func main() {
	address := "localhost"
	port := "8080"

	serverAddr := fmt.Sprintf("%s:%s", address, port)

	http.HandleFunc("/", handler)

	log.Printf("Starting server on %s", serverAddr)
	err := http.ListenAndServe(serverAddr, nil)
	if err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}

func handler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var data Data
	err := json.NewDecoder(r.Body).Decode(&data)
	if err != nil {
		http.Error(w, "Invalid JSON data", http.StatusBadRequest)
		return
	}

	responseChannel := make(chan string)
	go processRequest(data, responseChannel)

	response := <-responseChannel

	w.Header().Set("Content-Type", "application/json")
	fmt.Fprint(w, response)
}

func processRequest(data Data, responseChannel chan<- string) {
	// Encode message using SHA256
	hash := sha256.Sum256([]byte(data.Message))
	encodedMessage := hex.EncodeToString(hash[:])

	response := struct {
		EncodedMessage string `json:"encoded_message"`
	}{
		EncodedMessage: encodedMessage,
	}

	responseJSON, err := json.Marshal(response)
	if err != nil {
		responseChannel <- ""
		return
	}

	responseChannel <- string(responseJSON)
}