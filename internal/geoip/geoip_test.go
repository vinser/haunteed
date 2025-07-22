package geoip

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// Helper to reset default client after each test
func resetDefaults() {
	Default.url = "http://ip-api.com/json/"
	Default.httpTimeout = 5 * time.Second
	Default.cacheTTL = 1 * time.Hour
	Default.cache = nil
	Default.cacheTime = time.Time{}
}

// Basic happy path + cache test
func TestGetLocationInfo_Cache(t *testing.T) {
	defer resetDefaults()

	mockResponse := `{
		"continent": "Europe",
		"country": "UK",
		"city": "London",
		"lat": 51.5,
		"lon": -0.1,
		"timezone": "Europe/London",
		"query": "1.2.3.4"
	}`

	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		w.Write([]byte(mockResponse))
	}))
	defer server.Close()

	Default.url = server.URL
	Default.cacheTTL = time.Minute

	loc1, err := GetLocationInfo()
	if err != nil {
		t.Fatal(err)
	}
	loc2, err := GetLocationInfo()
	if err != nil {
		t.Fatal(err)
	}
	if loc1 != loc2 {
		t.Errorf("Expected cached result on second call")
	}
	if callCount != 1 {
		t.Errorf("Expected one API call, got %d", callCount)
	}
}

// Cache expiry test
func TestCacheExpiry(t *testing.T) {
	defer resetDefaults()

	mock1 := `{
		"query": "1.1.1.1",
		"lat": 10,
		"lon": 20,
		"timezone": "UTC"
	}`
	mock2 := `{
		"query": "2.2.2.2",
		"lat": 11,
		"lon": 21,
		"timezone": "UTC"
	}`

	responses := []string{mock1, mock2}
	i := 0

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(responses[i]))
		i++
	}))
	defer server.Close()

	Default.url = server.URL
	Default.cacheTTL = 100 * time.Millisecond

	loc1, err := GetLocationInfo()
	if err != nil {
		t.Fatal(err)
	}
	time.Sleep(150 * time.Millisecond)
	loc2, err := GetLocationInfo()
	if err != nil {
		t.Fatal(err)
	}
	if loc1.Lat == loc2.Lat {
		t.Errorf("Expected different lat due to refreshed data")
	}
}

// HTTP error
func TestGetLocationInfo_HTTPError(t *testing.T) {
	defer resetDefaults()

	// Unreachable address
	Default.url = "http://localhost:9999"
	Default.httpTimeout = 100 * time.Millisecond

	_, err := GetLocationInfo()
	if err == nil {
		t.Fatal("expected error from unreachable server")
	}
}

// Non-200 status code
func TestGetLocationInfo_Non200(t *testing.T) {
	defer resetDefaults()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "fail", http.StatusInternalServerError)
	}))
	defer server.Close()

	Default.url = server.URL
	_, err := GetLocationInfo()
	if err == nil || err.Error() != "geoip: non-200 response from API" {
		t.Fatalf("expected non-200 error, got %v", err)
	}
}

// Invalid JSON
func TestGetLocationInfo_BadJSON(t *testing.T) {
	defer resetDefaults()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("{bad json"))
	}))
	defer server.Close()

	Default.url = server.URL
	_, err := GetLocationInfo()
	if err == nil {
		t.Fatal("expected JSON decode error")
	}
}

// Missing timezone
func TestGetLocationInfo_NoTimezone(t *testing.T) {
	defer resetDefaults()

	resp := `{"lat": 1, "lon": 2, "query": "x"}`
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(resp))
	}))
	defer server.Close()

	Default.url = server.URL
	_, err := GetLocationInfo()
	if err == nil || err.Error() != "geoip: timezone not provided" {
		t.Fatalf("expected missing timezone error, got %v", err)
	}
}

// Invalid timezone
func TestGetLocationInfo_InvalidTimezone(t *testing.T) {
	defer resetDefaults()

	resp := `{"lat": 1, "lon": 2, "timezone": "Bad/Zone", "query": "x"}`
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(resp))
	}))
	defer server.Close()

	Default.url = server.URL
	_, err := GetLocationInfo()
	if err == nil {
		t.Fatal("expected error for invalid timezone, got nil")
	}
	if !strings.Contains(err.Error(), "unknown time zone") {
		t.Fatalf("expected unknown time zone error, got: %v", err)
	}
}

// Test SetCacheTTL / SetHTTPTimeout (via effect)
func TestSetters(t *testing.T) {
	SetCacheTTL(123 * time.Second)
	if Default.cacheTTL != 123*time.Second {
		t.Error("SetCacheTTL failed")
	}
	SetHTTPTimeout(456 * time.Millisecond)
	if Default.httpTimeout != 456*time.Millisecond {
		t.Error("SetHTTPTimeout failed")
	}
}
