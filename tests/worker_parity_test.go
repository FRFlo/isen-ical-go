package tests

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestWorkerParity_GenerateTokenAcceptsUsernameField(t *testing.T) {
	ts := setupTestServer(t)
	defer ts.Cleanup()

	requestBody := map[string]string{
		"username": "test@student.junia.com",
		"password": "testpass",
	}

	jsonBody, _ := json.Marshal(requestBody)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/api/generate-token", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	ts.router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected %d, got %d. body=%s", http.StatusOK, w.Code, w.Body.String())
	}
}

func TestWorkerParity_MissingSpecialRoutesReturn204(t *testing.T) {
	ts := setupTestServer(t)
	defer ts.Cleanup()

	paths := []string{"/favicon.ico", "/.well-known/appspecific/com.chrome.devtools.json"}
	for _, path := range paths {
		t.Run(path, func(t *testing.T) {
			w := httptest.NewRecorder()
			req, _ := http.NewRequest(http.MethodGet, path, nil)
			ts.router.ServeHTTP(w, req)

			if w.Code != http.StatusNoContent {
				t.Fatalf("expected %d for %s, got %d", http.StatusNoContent, path, w.Code)
			}
		})
	}
}

func TestWorkerParity_BasicAuthFailuresAre401WithWorkerRealmAndTextBody(t *testing.T) {
	ts := setupTestServer(t)
	defer ts.Cleanup()

	t.Run("missing_header", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodGet, "/", nil)
		ts.router.ServeHTTP(w, req)

		if w.Code != http.StatusUnauthorized {
			t.Fatalf("expected %d, got %d", http.StatusUnauthorized, w.Code)
		}

		if got := w.Header().Get("WWW-Authenticate"); got != `Basic realm="Identifiants Aurion"` {
			t.Fatalf("unexpected WWW-Authenticate header: %q", got)
		}

		if strings.TrimSpace(w.Body.String()) != "Authorization required" {
			t.Fatalf("unexpected body: %q", w.Body.String())
		}
	})

	t.Run("invalid_format", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodGet, "/", nil)
		req.Header.Set("Authorization", "Bearer abc")
		ts.router.ServeHTTP(w, req)

		if w.Code != http.StatusUnauthorized {
			t.Fatalf("expected %d, got %d", http.StatusUnauthorized, w.Code)
		}

		if got := w.Header().Get("WWW-Authenticate"); got != `Basic realm="Identifiants Aurion"` {
			t.Fatalf("unexpected WWW-Authenticate header: %q", got)
		}

		if strings.TrimSpace(w.Body.String()) != "Invalid authorization format" {
			t.Fatalf("unexpected body: %q", w.Body.String())
		}
	})
}

func TestWorkerParity_CalendarResponseHeadersAndICSShape(t *testing.T) {
	ts := setupTestServer(t)
	defer ts.Cleanup()

	credentials := base64.StdEncoding.EncodeToString([]byte("test@student.junia.com:testpass"))
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Basic "+credentials)
	ts.router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected %d, got %d. body=%s", http.StatusOK, w.Code, w.Body.String())
	}

	if got := w.Header().Get("Content-Disposition"); got != `attachment; filename="isen-ical.ics"` {
		t.Fatalf("unexpected Content-Disposition: %q", got)
	}

	body := w.Body.String()
	if !strings.Contains(body, "X-WR-CALNAME:Aurion") {
		t.Fatalf("missing X-WR-CALNAME in ICS")
	}

	if strings.Contains(body, "BEGIN:VTIMEZONE") {
		t.Fatalf("unexpected VTIMEZONE block in ICS")
	}
}

func TestWorkerParity_ResponseIncludesTracingHeaders(t *testing.T) {
	ts := setupTestServer(t)
	defer ts.Cleanup()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/health", nil)
	ts.router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected %d, got %d", http.StatusOK, w.Code)
	}

	for _, header := range []string{"x-request-id", "x-trace-id", "traceparent"} {
		if w.Header().Get(header) == "" {
			t.Fatalf("missing response header %s", header)
		}
	}
}

func TestWorkerParity_TrackEndpointAcceptsFrontendEvents(t *testing.T) {
	ts := setupTestServer(t)
	defer ts.Cleanup()

	body := map[string]any{
		"event":      "frontend_homepage_viewed",
		"distinctId": "user:test@student.junia.com",
		"properties": map[string]any{"source": "test"},
	}

	payload, _ := json.Marshal(body)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/api/track", bytes.NewBuffer(payload))
	req.Header.Set("Content-Type", "application/json")
	ts.router.ServeHTTP(w, req)

	if w.Code != http.StatusAccepted {
		t.Fatalf("expected %d, got %d. body=%s", http.StatusAccepted, w.Code, w.Body.String())
	}
}
