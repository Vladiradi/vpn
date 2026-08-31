package main

import (
	"log"
	"net/http"

	"github.com/vpn-project/backend/internal/config"
)

func main() {
	cfg := config.Load()

	port := cfg.Port
	if port == "" {
		port = "8081"
	}

	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok","component":"agent"}`))
	})

	log.Printf("VPN node-agent starting on :%s", port)
	log.Printf("Connecting to panel: %s", cfg.PanelURL)

	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatalf("agent failed: %v", err)
	}
}
