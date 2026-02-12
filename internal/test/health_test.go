package test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"faas-engine-go/internal/api"
)

func TestGreetHandler_Success(t *testing.T) {
	req := httptest.NewRequest(
		http.MethodGet,
		"/greet?name=Nischit",
		nil,
	)

	rr := httptest.NewRecorder()

	api.GreetHandler(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rr.Code)
	}

	expected := `{"message":"Hello, Nischit!"}`
	body := strings.TrimSpace(rr.Body.String())

	if body != expected {
		t.Fatalf("expected body %s, got %s", expected, body)
	}
}

func TestGreetHandler_MissingName(t *testing.T) {
	req := httptest.NewRequest(
		http.MethodGet,
		"/greet",
		nil,
	)

	rr := httptest.NewRecorder()

	api.GreetHandler(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", rr.Code)
	}

	expected := `{"message":"Missing 'name' query parameter"}`
	body := strings.TrimSpace(rr.Body.String())

	if body != expected {
		t.Fatalf("expected body %s, got %s", expected, body)
	}
}
