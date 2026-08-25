package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestRegisterAndList(t *testing.T) {
	api := NewAPI(NewMigrationStore())

	body := bytes.NewBufferString(`{"version":2,"name":"users","sql":"CREATE TABLE users(id INT);"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/migrations", body)
	rr := httptest.NewRecorder()
	api.routes().ServeHTTP(rr, req)
	if rr.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rr.Code, rr.Body.String())
	}

	var response struct {
		Migration MigrationRecord `json:"migration"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	if response.Migration.SHA256 == "" || response.Migration.SizeBytes == 0 {
		t.Fatalf("expected digest metadata, got %+v", response.Migration)
	}

	req = httptest.NewRequest(http.MethodGet, "/api/v1/migrations", nil)
	rr = httptest.NewRecorder()
	api.routes().ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	if strings.Contains(rr.Body.String(), "CREATE TABLE") {
		t.Fatal("list endpoint must not expose SQL bodies")
	}
}

func TestMethodAwareRoutes(t *testing.T) {
	api := NewAPI(NewMigrationStore())
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rr := httptest.NewRecorder()
	api.routes().ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("GET /health expected 200, got %d", rr.Code)
	}

	req = httptest.NewRequest(http.MethodPost, "/health", nil)
	rr = httptest.NewRecorder()
	api.routes().ServeHTTP(rr, req)
	if rr.Code != http.StatusMethodNotAllowed {
		t.Fatalf("POST /health expected 405, got %d", rr.Code)
	}
}

func TestRejectsTrailingJSON(t *testing.T) {
	api := NewAPI(NewMigrationStore())
	payload := `{"version":1,"name":"init","sql":"SELECT 1"} {"extra":true}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/migrations", strings.NewReader(payload))
	rr := httptest.NewRecorder()
	api.routes().ServeHTTP(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected trailing JSON to return 400, got %d", rr.Code)
	}
	if api.store.Count() != 0 {
		t.Fatalf("trailing JSON must not mutate store; count=%d", api.store.Count())
	}
}

func TestDuplicateIsIdempotentAndConflictIsRejected(t *testing.T) {
	api := NewAPI(NewMigrationStore())
	payload := `{"version":1,"name":"init","sql":"CREATE TABLE users(id INT);"}`

	for i, expected := range []int{http.StatusCreated, http.StatusOK} {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/migrations", strings.NewReader(payload))
		rr := httptest.NewRecorder()
		api.routes().ServeHTTP(rr, req)
		if rr.Code != expected {
			t.Fatalf("request %d expected %d, got %d", i+1, expected, rr.Code)
		}
	}

	conflict := `{"version":1,"name":"init","sql":"DROP TABLE users;"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/migrations", strings.NewReader(conflict))
	rr := httptest.NewRecorder()
	api.routes().ServeHTTP(rr, req)
	if rr.Code != http.StatusConflict {
		t.Fatalf("expected 409, got %d", rr.Code)
	}
}

func TestValidationAndUnknownFields(t *testing.T) {
	api := NewAPI(NewMigrationStore())
	cases := []string{
		`{"version":0,"name":"bad name","sql":""}`,
		`{"version":1,"name":"ok","sql":"SELECT 1","unexpected":true}`,
	}

	for _, payload := range cases {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/migrations", strings.NewReader(payload))
		rr := httptest.NewRecorder()
		api.routes().ServeHTTP(rr, req)
		if rr.Code != http.StatusBadRequest {
			t.Fatalf("expected 400 for %s, got %d", payload, rr.Code)
		}
	}
}

func TestOperationalEndpoints(t *testing.T) {
	api := NewAPI(NewMigrationStore())
	for _, path := range []string{"/health", "/ready", "/metrics"} {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		rr := httptest.NewRecorder()
		api.routes().ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			t.Fatalf("%s expected 200, got %d", path, rr.Code)
		}
		if rr.Header().Get("X-Content-Type-Options") != "nosniff" {
			t.Fatalf("%s missing security header", path)
		}
	}
}
