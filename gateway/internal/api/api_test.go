package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// Эти запросы отклоняются до обращения к хранилищу, поэтому тесту не нужна база.
func TestErrorResponses(t *testing.T) {
	cases := []struct {
		name, method, path, body string
		status                   int
	}{
		{"broken json", http.MethodPost, "/api/transactions", `{"amount":`, http.StatusBadRequest},
		{"negative amount", http.MethodPost, "/api/transactions", `{"amount":-1,"category":"еда"}`, http.StatusBadRequest},
		{"empty category", http.MethodPost, "/api/transactions", `{"amount":1,"category":""}`, http.StatusBadRequest},
		{"bad date", http.MethodPost, "/api/transactions", `{"amount":1,"category":"еда","date":"10.09.2025"}`, http.StatusBadRequest},
		{"zero limit", http.MethodPost, "/api/budgets", `{"category":"еда","limit":0}`, http.StatusBadRequest},
		{"report without dates", http.MethodGet, "/api/reports/summary", "", http.StatusBadRequest},
		{"report bad range", http.MethodGet, "/api/reports/summary?from=2026-02-01&to=2026-01-01", "", http.StatusBadRequest},
		{"wrong method", http.MethodDelete, "/api/budgets", "", http.StatusMethodNotAllowed},
		{"unknown path", http.MethodGet, "/api/unknown", "", http.StatusNotFound},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			NewRouter().ServeHTTP(rec, httptest.NewRequest(c.method, c.path, strings.NewReader(c.body)))

			if rec.Code != c.status {
				t.Fatalf("want status %d, got %d: %s", c.status, rec.Code, rec.Body)
			}
			if got := rec.Header().Get("Content-Type"); got != "application/json; charset=utf-8" {
				t.Fatalf("wrong Content-Type: %q", got)
			}
			var body map[string]string
			if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil || body["error"] == "" {
				t.Fatalf("want {\"error\":...}, got %s", rec.Body)
			}
		})
	}
}
