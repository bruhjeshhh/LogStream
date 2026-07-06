package health

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestApp_Returns200(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/healthz", nil)
	rec := httptest.NewRecorder()

	App(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusOK)
	}
}

func TestApp_ReturnsOKBody(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/healthz", nil)
	rec := httptest.NewRecorder()

	App(rec, req)

	if rec.Body.String() != "OK" {
		t.Errorf("body = %q, want %q", rec.Body.String(), "OK")
	}
}

func TestApp_ContentType(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/healthz", nil)
	rec := httptest.NewRecorder()

	App(rec, req)

	expected := "text/plain; charset=utf-8"
	if ct := rec.Header().Get("Content-Type"); ct != expected {
		t.Errorf("Content-Type = %q, want %q", ct, expected)
	}
}

func TestApp_AnyMethod(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/api/healthz", nil)
	rec := httptest.NewRecorder()

	App(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusOK)
	}
}

func TestApp_ResponseSize(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/healthz", nil)
	rec := httptest.NewRecorder()

	App(rec, req)

	if len(rec.Body.Bytes()) != 2 {
		t.Errorf("response size = %d, want 2", len(rec.Body.Bytes()))
	}
}
