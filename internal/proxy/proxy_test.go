package proxy

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
)

func TestControlPlanePassthrough(t *testing.T) {
	var receivedPath string

	runtime := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedPath = r.URL.Path
		w.WriteHeader(http.StatusOK)
	}))
	defer runtime.Close()

	targetURL, _ := url.Parse(runtime.URL)
	handler := ProxyHandler(targetURL)

	req := httptest.NewRequest("GET", "/functions/list", nil)
	req.Host = "localhost"

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if receivedPath != "/functions/list" {
		t.Fatalf("expected /functions/list, got %s", receivedPath)
	}
}

func TestInvokeRewrite(t *testing.T) {
	var receivedPath string

	runtime := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedPath = r.URL.Path
		w.WriteHeader(http.StatusOK)
	}))
	defer runtime.Close()

	targetURL, _ := url.Parse(runtime.URL)
	handler := ProxyHandler(targetURL)

	req := httptest.NewRequest("POST", "/", nil)
	req.Host = "hello.localhost"

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	expected := "/functions/hello/invoke"

	if receivedPath != expected {
		t.Fatalf("expected %s, got %s", expected, receivedPath)
	}
}

func TestGETToPOSTConversion(t *testing.T) {
	var method string
	var body []byte

	runtime := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		method = r.Method
		body, _ = io.ReadAll(r.Body)
		w.WriteHeader(http.StatusOK)
	}))
	defer runtime.Close()

	targetURL, _ := url.Parse(runtime.URL)
	handler := ProxyHandler(targetURL)

	req := httptest.NewRequest("GET", "/?a=10&b=test", nil)
	req.Host = "calc.localhost"

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if method != http.MethodPost {
		t.Fatalf("expected POST, got %s", method)
	}

	var data map[string]interface{}
	err := json.Unmarshal(body, &data)
	if err != nil {
		t.Fatalf("invalid JSON body: %v", err)
	}

	// Query params are treated as strings, not implicitly converted to numbers
	if data["a"] != "10" || data["b"] != "test" {
		t.Fatalf("unexpected body: %v", data)
	}
}

func TestMissingFunction(t *testing.T) {
	targetURL, _ := url.Parse("http://example.com")
	handler := ProxyHandler(targetURL)

	req := httptest.NewRequest("GET", "/", nil)
	req.Host = "localhost"

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rr.Code)
	}
}

func TestHeadersForwarded(t *testing.T) {
	var headerVal string

	runtime := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		headerVal = r.Header.Get("X-Test")
		w.WriteHeader(http.StatusOK)
	}))
	defer runtime.Close()

	targetURL, _ := url.Parse(runtime.URL)
	handler := ProxyHandler(targetURL)

	req := httptest.NewRequest("POST", "/", nil)
	req.Host = "test.localhost"
	req.Header.Set("X-Test", "value")

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if headerVal != "value" {
		t.Fatalf("expected header to be forwarded, got %s", headerVal)
	}
}

func TestExtractFunctionName(t *testing.T) {
	tests := []struct {
		host     string
		expected string
	}{
		{"hello.localhost", "hello"},
		{"test.127.0.0.1.nip.io", "test"},
		{"api.example.com", "api"},
		{"localhost", ""},
		{"hello.localhost:8080", "hello"},
	}

	for _, tt := range tests {
		result := extractFunctionName(tt.host)
		if result != tt.expected {
			t.Errorf("host=%s expected=%s got=%s", tt.host, tt.expected, result)
		}
	}
}
