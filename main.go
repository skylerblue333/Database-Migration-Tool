// Database-Migration-Tool: versioned migration registry API.
package main

import (
	"encoding/json"
	"errors"
	"net/http"
	"sort"
	"sync"
	"time"
)

type Migration struct {
	Version int    `json:"version"`
	Name    string `json:"name"`
	SQL     string `json:"sql"`
}

var (
	mu      sync.RWMutex
	applied []Migration
)

func validateMigration(m Migration) error {
	if m.Version <= 0 {
		return errors.New("version must be positive")
	}
	if m.Name == "" {
		return errors.New("name is required")
	}
	if m.SQL == "" {
		return errors.New("sql is required")
	}
	return nil
}

func handleProcess(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var m Migration
	if err := json.NewDecoder(r.Body).Decode(&m); err != nil {
		http.Error(w, "invalid migration payload", http.StatusBadRequest)
		return
	}
	if err := validateMigration(m); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	mu.Lock()
	defer mu.Unlock()
	for _, a := range applied {
		if a.Version == m.Version {
			writeJSON(w, http.StatusOK, map[string]interface{}{"status": "already_applied", "version": m.Version})
			return
		}
	}
	applied = append(applied, m)
	sort.Slice(applied, func(i, j int) bool { return applied[i].Version < applied[j].Version })
	writeJSON(w, http.StatusCreated, map[string]interface{}{"status": "registered", "version": m.Version, "total": len(applied)})
}

func handleList(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	mu.RLock()
	defer mu.RUnlock()
	writeJSON(w, http.StatusOK, applied)
}

func handleHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"status": "healthy", "service": "Database-Migration-Tool", "timestamp": time.Now().UTC().Format(time.RFC3339),
	})
}

func writeJSON(w http.ResponseWriter, status int, value interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func main() {
	mux := http.NewServeMux()
	mux.HandleFunc("/health", handleHealth)
	mux.HandleFunc("/api/v1/migrations", handleProcess)
	mux.HandleFunc("/api/v1/migrations/list", handleList)
	_ = http.ListenAndServe(":8080", mux)
}
