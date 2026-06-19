// Database-Migration-Tool: Tracks and applies versioned database migrations
package main

import (
	"encoding/json"
	"log"
	"net/http"
	"time"
)

type Migration struct {
	Version int    `json:"version"`
	Name    string `json:"name"`
	SQL     string `json:"sql"`
}

var applied []Migration

func handleProcess(w http.ResponseWriter, r *http.Request) {
	var m Migration
	if err := json.NewDecoder(r.Body).Decode(&m); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	for _, a := range applied {
		if a.Version == m.Version {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]string{"status": "already_applied"})
			return
		}
	}
	applied = append(applied, m)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"status": "applied", "version": m.Version, "total": len(applied)})
}


func handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":    "healthy",
		"service":   "Database-Migration-Tool",
		"timestamp": time.Now().Unix(),
	})
}

func main() {
	mux := http.NewServeMux()
	mux.HandleFunc("/health", handleHealth)
	mux.HandleFunc("/api/v1/process", handleProcess)
	log.Printf("Database-Migration-Tool running on :8080")
	log.Fatal(http.ListenAndServe(":8080", mux))
}
