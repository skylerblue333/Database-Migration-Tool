// Database-Migration-Tool: bounded migration registry and validation service.
package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"regexp"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"time"
)

const (
	maxMigrationBody = 256 << 10
	maxRequestBody   = 2 << 20
)

var migrationNamePattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]{0,127}$`)

type Migration struct {
	Version int    `json:"version"`
	Name    string `json:"name"`
	SQL     string `json:"sql"`
}

type MigrationRecord struct {
	Version   int    `json:"version"`
	Name      string `json:"name"`
	SHA256    string `json:"sha256"`
	SizeBytes int    `json:"size_bytes"`
}

type MigrationStore struct {
	mu      sync.RWMutex
	records map[int]MigrationRecord
}

func NewMigrationStore() *MigrationStore {
	return &MigrationStore{records: make(map[int]MigrationRecord)}
}

func validateMigration(m Migration) error {
	if m.Version <= 0 {
		return errors.New("version must be positive")
	}
	if !migrationNamePattern.MatchString(m.Name) {
		return errors.New("name must be 1-128 safe characters")
	}
	if strings.TrimSpace(m.SQL) == "" {
		return errors.New("sql is required")
	}
	if len(m.SQL) > maxMigrationBody {
		return errors.New("sql exceeds 256 KiB limit")
	}
	return nil
}

func migrationRecord(m Migration) MigrationRecord {
	digest := sha256.Sum256([]byte(m.SQL))
	return MigrationRecord{Version: m.Version, Name: m.Name, SHA256: hex.EncodeToString(digest[:]), SizeBytes: len(m.SQL)}
}

func (s *MigrationStore) Register(m Migration) (MigrationRecord, bool, error) {
	if err := validateMigration(m); err != nil {
		return MigrationRecord{}, false, err
	}
	record := migrationRecord(m)
	s.mu.Lock()
	defer s.mu.Unlock()
	if existing, ok := s.records[m.Version]; ok {
		if existing.Name == record.Name && existing.SHA256 == record.SHA256 {
			return existing, false, nil
		}
		return MigrationRecord{}, false, errors.New("version already registered with different content")
	}
	s.records[m.Version] = record
	return record, true, nil
}

func (s *MigrationStore) List() []MigrationRecord {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make([]MigrationRecord, 0, len(s.records))
	for _, record := range s.records {
		result = append(result, record)
	}
	sort.Slice(result, func(i, j int) bool { return result[i].Version < result[j].Version })
	return result
}

func (s *MigrationStore) Count() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.records)
}

type API struct {
	store    *MigrationStore
	requests atomic.Uint64
	rejected atomic.Uint64
}

func NewAPI(store *MigrationStore) *API { return &API{store: store} }

func (a *API) routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", a.handleHealth)
	mux.HandleFunc("GET /ready", a.handleReady)
	mux.HandleFunc("GET /metrics", a.handleMetrics)
	mux.HandleFunc("POST /api/v1/migrations", a.handleRegister)
	mux.HandleFunc("GET /api/v1/migrations", a.handleList)
	return requestSecurityHeaders(a.countRequests(mux))
}

func (a *API) countRequests(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		a.requests.Add(1)
		next.ServeHTTP(w, r)
	})
}

func requestSecurityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Cache-Control", "no-store")
		next.ServeHTTP(w, r)
	})
}

func (a *API) handleRegister(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, maxRequestBody)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	var m Migration
	if err := decoder.Decode(&m); err != nil {
		a.rejected.Add(1)
		writeError(w, http.StatusBadRequest, "invalid migration payload")
		return
	}
	var trailing any
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		a.rejected.Add(1)
		writeError(w, http.StatusBadRequest, "migration payload must contain exactly one JSON value")
		return
	}
	record, created, err := a.store.Register(m)
	if err != nil {
		a.rejected.Add(1)
		status := http.StatusBadRequest
		if strings.Contains(err.Error(), "already registered") {
			status = http.StatusConflict
		}
		writeError(w, status, err.Error())
		return
	}
	status := http.StatusOK
	state := "already_registered"
	if created {
		status = http.StatusCreated
		state = "registered"
	}
	writeJSON(w, status, map[string]any{"status": state, "migration": record})
}

func (a *API) handleList(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, a.store.List())
}

func (a *API) handleHealth(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "healthy", "service": "sky-migration-registry"})
}

func (a *API) handleReady(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ready"})
}

func (a *API) handleMetrics(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"requests_total": a.requests.Load(), "rejected_total": a.rejected.Load(), "migrations_registered": a.store.Count()})
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(value); err != nil {
		slog.Error("encode response", "error", err)
	}
}

func main() {
	api := NewAPI(NewMigrationStore())
	server := &http.Server{Addr: ":8080", Handler: api.routes(), ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 10 * time.Second, WriteTimeout: 10 * time.Second, IdleTimeout: 30 * time.Second}
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		slog.Info("migration registry listening", "address", server.Addr)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Error("server failed", "error", err)
			os.Exit(1)
		}
	}()
	<-stop
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := server.Shutdown(ctx); err != nil {
		slog.Error("graceful shutdown failed", "error", err)
	}
}
