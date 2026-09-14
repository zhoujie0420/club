package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHealth(t *testing.T) {
	app := &application{allowedOrigins: parseOrigins("https://example.com")}
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	app.routes().ServeHTTP(recorder, request)
	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", recorder.Code)
	}
}

func TestReadyWithoutDatabase(t *testing.T) {
	app := &application{allowedOrigins: map[string]struct{}{}}
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/readyz", nil)
	app.routes().ServeHTTP(recorder, request)
	if recorder.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d", recorder.Code)
	}
}

func TestCORSAllowsIdempotencyKey(t *testing.T) {
	app := &application{allowedOrigins: parseOrigins("https://example.com")}
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodOptions, "/api/v1/orders", nil)
	request.Header.Set("Origin", "https://example.com")
	request.Header.Set("Access-Control-Request-Headers", "authorization,content-type,idempotency-key")
	app.routes().ServeHTTP(recorder, request)
	if recorder.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", recorder.Code)
	}
	allow := strings.ToLower(recorder.Header().Get("Access-Control-Allow-Headers"))
	if !strings.Contains(allow, "idempotency-key") {
		t.Fatalf("missing Idempotency-Key: %q", recorder.Header().Get("Access-Control-Allow-Headers"))
	}
}
