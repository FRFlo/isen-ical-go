package middleware

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"regexp"
	"testing"

	"github.com/gin-gonic/gin"
)

func setupTestRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	SetupMiddleware(router)
	return router
}

func TestTraceHeadersMiddleware_GeneratesHeaders(t *testing.T) {
	router := setupTestRouter()

	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/test", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}

	requestID := w.Header().Get("x-request-id")
	traceID := w.Header().Get("x-trace-id")
	traceparent := w.Header().Get("traceparent")

	if requestID == "" {
		t.Fatal("expected x-request-id header")
	}
	if traceID == "" {
		t.Fatal("expected x-trace-id header")
	}
	if traceparent == "" {
		t.Fatal("expected traceparent header")
	}

	traceparentRegex := regexp.MustCompile(`^00-[a-f0-9]{32}-[a-f0-9]{16}-01$`)
	if !traceparentRegex.MatchString(traceparent) {
		t.Fatalf("unexpected traceparent format: %q", traceparent)
	}
}

func TestTraceHeadersMiddleware_PreservesIncomingHeaders(t *testing.T) {
	router := setupTestRouter()

	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	const reqID = "req-custom"
	const traceID = "abc123"
	const traceparent = "00-11111111111111111111111111111111-2222222222222222-01"

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("x-request-id", reqID)
	req.Header.Set("x-trace-id", traceID)
	req.Header.Set("traceparent", traceparent)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}

	if got := w.Header().Get("x-request-id"); got != reqID {
		t.Fatalf("expected x-request-id %q, got %q", reqID, got)
	}
	if got := w.Header().Get("x-trace-id"); got != traceID {
		t.Fatalf("expected x-trace-id %q, got %q", traceID, got)
	}
	if got := w.Header().Get("traceparent"); got != traceparent {
		t.Fatalf("expected traceparent %q, got %q", traceparent, got)
	}
}

func TestRecoveryMiddleware_ReturnsJSONOnPanic(t *testing.T) {
	router := setupTestRouter()

	router.GET("/panic", func(c *gin.Context) {
		panic("intentional test panic")
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/panic", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected status 500, got %d", w.Code)
	}

	var response map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to decode JSON response: %v", err)
	}

	if response["error"] != "Internal Server Error" {
		t.Fatalf("unexpected error message: %q", response["error"])
	}

	if w.Header().Get("x-request-id") == "" {
		t.Fatal("expected x-request-id on panic response")
	}
}

func TestCORSMiddleware_Preflight(t *testing.T) {
	router := setupTestRouter()

	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodOptions, "/test", nil)
	req.Header.Set("Origin", "https://example.com")
	req.Header.Set("Access-Control-Request-Method", "GET")
	req.Header.Set("Access-Control-Request-Headers", "Authorization, Content-Type")
	router.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent && w.Code != http.StatusOK {
		t.Fatalf("expected status 204 or 200, got %d", w.Code)
	}

	if got := w.Header().Get("Access-Control-Allow-Origin"); got == "" {
		t.Fatal("expected Access-Control-Allow-Origin header")
	}

	if got := w.Header().Get("Access-Control-Allow-Methods"); got == "" {
		t.Fatal("expected Access-Control-Allow-Methods header")
	}
}
