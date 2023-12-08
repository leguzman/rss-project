package main

import (
	"encoding/json"
	"log"
	"net/http"
)

type WrappedSlice struct {
	Results interface{} `json:"results"`
	Size    int         `json:"size"`
}

func respondWithJson(w http.ResponseWriter, code int, payload interface{}) {
	data, err := json.Marshal(payload)
	if err != nil {
		log.Printf("Failed to marshal: %v", payload)
		w.WriteHeader(500)
		return
	}
	w.Header().Add("Content-Type", "application/json")
	w.WriteHeader(code)
	w.Write(data)
}

func respondWithError(w http.ResponseWriter, code int, msg string) {
	if code > 499 {
		log.Println("Responding with 5xx error:", msg)
	}
	type ErrorResponse struct {
		Error string `json:"error"`
	}
	respondWithJson(w, code, ErrorResponse{Error: msg})
}
