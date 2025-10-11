package main

import (
	"encoding/json"
	"log"
	"net/http"
)

type ApiResponse struct {
	Success bool        `json:"success"`
	Message string      `json:"message,omitempty"`
	Data    interface{} `json:"data,omitempty"`
}

type ApiError struct {
	Code    string      `json:"code"`
	Message string      `json:"message"`
	Details interface{} `json:"details,omitempty"`
}

type ErrorResponse struct {
	Success bool     `json:"success"`
	Error   ApiError `json:"error"`
}

func respondWithError(w http.ResponseWriter, code int, apiErr ApiError) {
	if code >= 500 {
		log.Printf("Responding with 5XX error: %s - %s", apiErr.Code, apiErr.Message)
	}

	response := ErrorResponse{
		Success: false,
		Error:   apiErr,
	}

	respondWithJSON(w, code, response)
}

func respondWithJSON(w http.ResponseWriter, code int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")

	data, err := json.Marshal(payload)
	if err != nil {
		log.Printf("Error marshalling JSON: %s", err)
		w.WriteHeader(http.StatusInternalServerError)
		fallbackError := ErrorResponse{
			Success: false,
			Error: ApiError{
				Code:    "INTERNAL_ERROR",
				Message: "Failed to generate response",
			},
		}
		json.NewEncoder(w).Encode(fallbackError)
		return
	}

	w.WriteHeader(code)
	w.Write(data)
}
