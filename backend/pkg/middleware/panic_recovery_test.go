package middleware

import (
	"bytes"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

// A panicking handler must produce a 500 JSON response and a structured log
// entry carrying the stack trace — not a silent stderr dump.
func TestPanicRecovery_Returns500AndLogsStack(t *testing.T) {
	gin.SetMode(gin.TestMode)

	var logBuf bytes.Buffer
	prev := slog.Default()
	slog.SetDefault(slog.New(slog.NewJSONHandler(&logBuf, nil)))
	defer slog.SetDefault(prev)

	r := gin.New()
	r.Use(PanicRecovery())
	r.GET("/boom", func(c *gin.Context) { panic("kaboom") })

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/boom", nil))

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("status: want 500, got %d", w.Code)
	}
	if !strings.Contains(w.Body.String(), "internal server error") {
		t.Errorf("body missing generic error message: %s", w.Body.String())
	}
	logged := logBuf.String()
	if !strings.Contains(logged, "kaboom") {
		t.Errorf("log missing panic value: %s", logged)
	}
	if !strings.Contains(logged, "panic_recovery_test.go") && !strings.Contains(logged, "goroutine") {
		t.Errorf("log missing stack trace: %s", logged)
	}
}

// A healthy handler must pass through untouched.
func TestPanicRecovery_PassThrough(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(PanicRecovery())
	r.GET("/ok", func(c *gin.Context) { c.String(http.StatusOK, "fine") })

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/ok", nil))

	if w.Code != http.StatusOK || w.Body.String() != "fine" {
		t.Fatalf("want 200 'fine', got %d %q", w.Code, w.Body.String())
	}
}
