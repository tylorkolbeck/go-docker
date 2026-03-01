package server

import (
	"fmt"
	"net/http"
)

func NewServer() *http.Server {
	mux := http.NewServeMux()

	server := &http.Server{
		Addr:    ":3003",
		Handler: mux,
	}

	// mux.HandleFunc("GET /healthz", healthCheckHandler)

	fmt.Println("Server listening on :3003")
	return server
}

// func healthCheckHandler(w http.ResponseWriter, r *http.Request) {
// 	w.WriteHeader(http.StatusOK)
// 	w.Write([]byte("ok"))
// }
