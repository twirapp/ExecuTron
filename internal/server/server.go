package server

import (
	"encoding/json"
	"fmt"
	"github.com/twirapp/executron/internal/runtime"
	"log/slog"
	"net/http"
	"os"
)

type Server struct {
	httpServer *http.Server
	executor   *runtime.Executor
}

func New() *Server {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	mux := http.NewServeMux()

	s := Server{
		httpServer: &http.Server{
			Handler: mux,
			Addr:    "0.0.0.0:" + port,
		},
		executor: runtime.New(),
	}

	mux.HandleFunc("/run", s.runHandler)
	slog.Info("Server starting on :" + port)

	return &s
}

func (c *Server) Run() {
	go func() {
		if err := c.httpServer.ListenAndServe(); err != nil {
			fmt.Fprintf(os.Stderr, "Server failed: %v\n", err)
			os.Exit(1)
		}
	}()
}

func (c *Server) Stop() {
	c.httpServer.Shutdown(nil)
}

// Request represents the JSON payload for code execution.
type Request struct {
	Language string `json:"language"` // "python" or "javascript"
	Code     string `json:"code"`     // Code to execute
}

func (c *Server) runHandler(w http.ResponseWriter, r *http.Request) {
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

	resp, err := c.executor.ExecuteCode(r.Context(), req.Language, req.Code)
	if err != nil {
		http.Error(w, fmt.Sprintf("Execution failed: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}
