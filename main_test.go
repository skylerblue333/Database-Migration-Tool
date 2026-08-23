package main

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHealth(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rr := httptest.NewRecorder()
	handleHealth(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
}

func TestProcessRegistersAndDeduplicates(t *testing.T) {
	mu.Lock()
	applied = nil
	mu.Unlock()

	body := bytes.NewBufferString(`{"version":1,"name":"init","sql":"CREATE TABLE users(id INT);"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/migrations", body)
	rr := httptest.NewRecorder()
	handleProcess(rr, req)
	if rr.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d", rr.Code)
	}

	body = bytes.NewBufferString(`{"version":1,"name":"init","sql":"CREATE TABLE users(id INT);"}`)
	req = httptest.NewRequest(http.MethodPost, "/api/v1/migrations", body)
	rr = httptest.NewRecorder()
	handleProcess(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected duplicate to return 200, got %d", rr.Code)
	}
}

func TestProcessRejectsInvalidMigration(t *testing.T) {
	body := bytes.NewBufferString(`{"version":0,"name":"","sql":""}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/migrations", body)
	rr := httptest.NewRecorder()
	handleProcess(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rr.Code)
	}
}
