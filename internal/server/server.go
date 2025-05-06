package server

import (
	"encoding/json"
	"fmt"
	"github.com/twirapp/executron/internal/runtime"
	"net/http"
	"os"
)

type Server struct {
	s *http.Server
}

func New() *Server {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	http.HandleFunc("/run", runHandler)
	fmt.Println("Server starting on :" + port)

	s := &http.Server{
		Handler: http.DefaultServeMux,
		Addr:    ":" + port,
	}

	return &Server{
		s,
	}
}

func (c *Server) Run() {
	go func() {
		if err := c.s.ListenAndServe(); err != nil {
			fmt.Fprintf(os.Stderr, "Server failed: %v\n", err)
			os.Exit(1)
		}
	}()
}

func (c *Server) Stop() {
	c.s.Shutdown(nil)
}

// Request represents the JSON payload for code execution.
type Request struct {
	Language string `json:"language"` // "python" or "javascript"
	Code     string `json:"code"`     // Code to execute
}

// Response represents the execution result.
type Response struct {
	Output interface{} `json:"output"` // Changed to interface{} to handle any JSON value
	Error  string      `json:"error,omitempty"`
}

func runHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req Request
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.Language != "python" && req.Language != "javascript" {
		http.Error(w, "Unsupported language", http.StatusBadRequest)
		return
	}

	resp, err := runtime.ExecuteCode(r.Context(), req.Language, req.Code)
	if err != nil {
		http.Error(w, fmt.Sprintf("Execution failed: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}
