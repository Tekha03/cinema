package httpapi

import (
	"net/http/httptest"
	"strings"
	"testing"
)

// These endpoints and invalid requests must work without a database.
func TestHTTPBoundary(t *testing.T) {
	handler := New(nil)
	cases := []struct {
		name        string
		method      string
		path        string
		contentType string
		body        string
		status      int
	}{
		{"frontend", "GET", "/", "", "", 200},
		{"javascript", "GET", "/app.js", "", "", 200},
		{"stylesheet", "GET", "/styles.css", "", "", 200},
		{"health", "GET", "/healthz", "", "", 200},
		{"invalid id", "GET", "/api/movies/abc", "", "", 400},
		{"zero id", "GET", "/api/movies/0", "", "", 400},
		{"invalid limit", "GET", "/api/movies?limit=0", "", "", 400},
		{"invalid offset", "GET", "/api/movies?offset=-1", "", "", 400},
		{"content type", "POST", "/api/movies", "text/plain", `{}`, 415},
		{"broken JSON", "POST", "/api/movies", "application/json", `{`, 400},
		{"unknown field", "POST", "/api/movies", "application/json", `{"unknown":1}`, 400},
		{"two objects", "POST", "/api/movies", "application/json", `{} {}`, 400},
		{"null", "POST", "/api/movies", "application/json", `null`, 400},
		{"oversized body", "POST", "/api/movies", "application/json", `{"title":"` + strings.Repeat("a", 65<<10) + `"}`, 413},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			r := httptest.NewRequest(tc.method, tc.path, strings.NewReader(tc.body))
			r.Header.Set("Content-Type", tc.contentType)

			w := httptest.NewRecorder()
			handler.ServeHTTP(w, r)

			if w.Code != tc.status {
				t.Fatalf("status = %d, want %d; body: %s", w.Code, tc.status, w.Body.String())
			}
		})
	}
}
