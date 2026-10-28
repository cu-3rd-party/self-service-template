package main

import (
	"context"
	"errors"
	"github.com/gin-gonic/gin"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

type pingResult struct{ err error }

func (p pingResult) PingContext(context.Context) error { return p.err }
func TestReadinessDependsOnDatabase(t *testing.T) {
	gin.SetMode(gin.TestMode)
	for _, tc := range []struct {
		name string
		db   databasePinger
		want int
	}{
		{"connected", pingResult{}, 200}, {"unavailable", pingResult{errors.New("offline")}, 503}, {"not configured", nil, 503},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r := httptest.NewRecorder()
			newRouter(tc.db).ServeHTTP(r, httptest.NewRequest("GET", "/api/ready", nil))
			if r.Code != tc.want {
				t.Fatalf("got %d, want %d", r.Code, tc.want)
			}
		})
	}
}
func TestHealthAndMetricsWithoutDatabase(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := newRouter(nil)
	for _, path := range []string{"/api/health", "/metrics"} {
		r := httptest.NewRecorder()
		router.ServeHTTP(r, httptest.NewRequest("GET", path, nil))
		if r.Code != http.StatusOK {
			t.Fatal(r.Code)
		}
		if path == "/metrics" && (!strings.Contains(r.Body.String(), "service_http_requests_total") || !strings.Contains(r.Header().Get("Content-Type"), "text/plain")) {
			t.Fatal("missing Prometheus exposition")
		}
	}
}
