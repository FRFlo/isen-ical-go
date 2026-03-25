package tests

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/FRFlo/isen-ical-go/internal/config"
	"github.com/FRFlo/isen-ical-go/internal/handlers"
	"github.com/FRFlo/isen-ical-go/internal/middleware"
	"github.com/FRFlo/isen-ical-go/internal/models"
	"github.com/FRFlo/isen-ical-go/internal/storage"
	"github.com/alicebob/miniredis/v2"
	"github.com/gin-gonic/gin"
)

// mockAurionServer creates a mock Aurion server for testing
type mockAurionServer struct {
	server *httptest.Server
}

func newMockAurionServer() *mockAurionServer {
	mux := http.NewServeMux()

	// Login endpoint
	mux.HandleFunc("/login", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		err := r.ParseForm()
		if err != nil {
			http.Error(w, "Bad request", http.StatusBadRequest)
			return
		}

		username := r.FormValue("username")
		password := r.FormValue("password")

		// Mock credentials validation
		if username == "test@student.junia.com" && password == "testpass" {
			w.Header().Set("Location", "/faces/MainMenuPage.xhtml")
			w.WriteHeader(http.StatusFound)
			return
		}

		http.Error(w, "Unauthorized", http.StatusUnauthorized)
	})

	// Home page - returns ViewState and idInit
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		html := `<!DOCTYPE html>
<html>
<head><title>Test</title></head>
<body>
<input type="hidden" name="javax.faces.ViewState" id="j_id1:javax.faces.ViewState:0" value="test-view-state-123" autocomplete="off" />
<input name="form:idInit" value="test-id-init-456" />
</body>
</html>`
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(html))
	})

	// Main menu page - returns sidebar menu ID
	mux.HandleFunc("/faces/MainMenuPage.xhtml", func(w http.ResponseWriter, r *http.Request) {
		html := `<!DOCTYPE html>
<html>
<body>
<div onclick="PrimeFaces.addSubmitParam('form',{'form:sidebar':'form:sidebar','form:sidebar_menuid':'12345'}"><span class="ui-menuitem-icon ui-icon fa fa-calendar-alt"></span><span class="ui-menuitem-text">Mon Planning</span></div>
</body>
</html>`
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(html))
	})

	mux.HandleFunc("/faces/Planning.xhtml", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			w.Header().Set("Content-Type", "application/xml")
			w.WriteHeader(http.StatusOK)

			events := []models.AurionEvent{
				{
					ID:        "1",
					Title:     "Math Class",
					Start:     time.Now().Format("2006-01-02T15:04:05"),
					End:       time.Now().Add(2 * time.Hour).Format("2006-01-02T15:04:05"),
					AllDay:    false,
					Editable:  false,
					ClassName: "course",
				},
			}

			eventsJSON, _ := json.Marshal(events)
			response := `<partial-response><changes><update id="form:j_idt100"><![CDATA[` + string(eventsJSON) + `x]]></update></changes></partial-response>`
			w.Write([]byte(response))
			return
		}

		html := `<!DOCTYPE html>
<html>
<body>
<script>PrimeFaces.cw("Schedule","schedule",{id:"form:j_idt100"</script>
<input type="hidden" name="javax.faces.ViewState" id="j_id1:javax.faces.ViewState:0" value="test-view-state-789" autocomplete="off" />
</body>
</html>`
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(html))
	})

	server := httptest.NewServer(mux)

	return &mockAurionServer{
		server: server,
	}
}

func (m *mockAurionServer) URL() string {
	return m.server.URL
}

func (m *mockAurionServer) Close() {
	m.server.Close()
}

// setupTestServer creates a test server with all dependencies
type testServer struct {
	router       *gin.Engine
	valkeyClient *storage.ValkeyClient
	mockAurion   *mockAurionServer
	miniRedis    *miniredis.Miniredis
	handlers     *handlers.Handlers
	config       *config.Config
}

func setupTestServer(t *testing.T) *testServer {
	gin.SetMode(gin.TestMode)

	// Create mock Aurion server
	mockAurion := newMockAurionServer()

	valkeyURL := os.Getenv("VALKEY_URL")
	var miniRedis *miniredis.Miniredis
	if valkeyURL == "" {
		mr, err := miniredis.Run()
		if err != nil {
			mockAurion.Close()
			t.Skipf("embedded redis not available: %v", err)
		}
		miniRedis = mr
		valkeyURL = "redis://" + mr.Addr()
	}

	// Create test configuration
	cfg := &config.Config{
		AurionBaseURL:    mockAurion.URL(),
		ValkeyURL:        valkeyURL,
		Port:             8080,
		MaxTokensPerUser: 3,
		SessionTTL:       3600,
		CacheTTL:         3600,
	}

	// Try to connect to Valkey, skip if not available
	valkeyClient, err := storage.NewValkeyClient(cfg.ValkeyURL)
	if err != nil {
		if miniRedis != nil {
			miniRedis.Close()
		}
		mockAurion.Close()
		t.Skipf("Valkey not available: %v", err)
	}

	router := gin.New()
	middleware.SetupMiddleware(router)

	// Create handlers
	h := handlers.New(cfg, valkeyClient)
	h.RegisterRoutes(router)

	return &testServer{
		router:       router,
		valkeyClient: valkeyClient,
		mockAurion:   mockAurion,
		miniRedis:    miniRedis,
		handlers:     h,
		config:       cfg,
	}
}

func (ts *testServer) Cleanup() {
	if ts.valkeyClient != nil {
		ts.valkeyClient.Close()
	}
	if ts.miniRedis != nil {
		ts.miniRedis.Close()
	}
	if ts.mockAurion != nil {
		ts.mockAurion.Close()
	}
}

// TestHealthEndpoint tests the health check endpoint
func TestHealthEndpoint(t *testing.T) {
	ts := setupTestServer(t)
	defer ts.Cleanup()

	// Test /api/health (New)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/api/health", nil)
	ts.router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d for /api/health, got %d", http.StatusOK, w.Code)
	}

	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if response["status"] != "healthy" {
		t.Errorf("Expected status 'healthy', got %v", response["status"])
	}

	// Test /health (Old, should be 404)
	w = httptest.NewRecorder()
	req, _ = http.NewRequest(http.MethodGet, "/health", nil)
	ts.router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("Expected status %d for /health, got %d", http.StatusNotFound, w.Code)
	}
}

// TestHomeEndpoint tests the home page endpoint
func TestHomeEndpoint(t *testing.T) {
	ts := setupTestServer(t)
	defer ts.Cleanup()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Accept", "text/html")
	ts.router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}

	contentType := w.Header().Get("Content-Type")
	if !strings.Contains(contentType, "text/html") {
		t.Errorf("Expected Content-Type to contain 'text/html', got %s", contentType)
	}

	body := w.Body.String()
	if !strings.Contains(body, "Calendrier JUNIA") {
		t.Error("Expected response to contain 'Calendrier JUNIA'")
	}
}

// TestPrivacyEndpoint tests the privacy page endpoint
func TestPrivacyEndpoint(t *testing.T) {
	ts := setupTestServer(t)
	defer ts.Cleanup()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/privacy", nil)
	ts.router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}

	contentType := w.Header().Get("Content-Type")
	if !strings.Contains(contentType, "text/html") {
		t.Errorf("Expected Content-Type to contain 'text/html', got %s", contentType)
	}

	body := w.Body.String()
	if !strings.Contains(body, "Confidentialité") {
		t.Error("Expected response to contain 'Confidentialité'")
	}
}

// TestGenerateTokenEndpoint tests the token generation endpoint
func TestGenerateTokenEndpoint(t *testing.T) {
	ts := setupTestServer(t)
	defer ts.Cleanup()

	requestBody := map[string]string{
		"email":    "test@student.junia.com",
		"password": "testpass",
	}

	jsonBody, _ := json.Marshal(requestBody)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/api/generate-token", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	ts.router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d. Body: %s", http.StatusOK, w.Code, w.Body.String())
	}

	var response handlers.GenerateTokenResponse
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if response.Token == "" {
		t.Error("Expected token to be non-empty")
	}

	if response.EncryptionKey == "" {
		t.Error("Expected encryptionKey to be non-empty")
	}

	if response.URL == "" {
		t.Error("Expected URL to be non-empty")
	}

	if !strings.Contains(response.URL, "/calendar/") {
		t.Error("Expected URL to contain '/calendar/'")
	}
}

// TestGenerateTokenEndpoint_InvalidCredentials tests token generation with invalid credentials
func TestGenerateTokenEndpoint_InvalidCredentials(t *testing.T) {
	ts := setupTestServer(t)
	defer ts.Cleanup()

	requestBody := map[string]string{
		"email":    "wrong@student.junia.com",
		"password": "wrongpass",
	}

	jsonBody, _ := json.Marshal(requestBody)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/api/generate-token", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	ts.router.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("Expected status %d, got %d", http.StatusForbidden, w.Code)
	}
}

// TestGenerateTokenEndpoint_InvalidRequest tests token generation with invalid request
func TestGenerateTokenEndpoint_InvalidRequest(t *testing.T) {
	ts := setupTestServer(t)
	defer ts.Cleanup()

	// Missing password
	requestBody := map[string]string{
		"email": "test@student.junia.com",
	}

	jsonBody, _ := json.Marshal(requestBody)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/api/generate-token", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	ts.router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

// TestCalendarEndpoint tests the calendar endpoint with a valid token
func TestCalendarEndpoint(t *testing.T) {
	ts := setupTestServer(t)
	defer ts.Cleanup()

	// First, generate a token
	requestBody := map[string]string{
		"email":    "test@student.junia.com",
		"password": "testpass",
	}

	jsonBody, _ := json.Marshal(requestBody)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/api/generate-token", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	ts.router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Failed to generate token: %d - %s", w.Code, w.Body.String())
	}

	var tokenResponse handlers.GenerateTokenResponse
	if err := json.Unmarshal(w.Body.Bytes(), &tokenResponse); err != nil {
		t.Fatalf("Failed to parse token response: %v", err)
	}

	// Now test the calendar endpoint
	w = httptest.NewRecorder()
	calendarURL := "/calendar/" + tokenResponse.Token + "?key=" + tokenResponse.EncryptionKey
	req, _ = http.NewRequest(http.MethodGet, calendarURL, nil)
	ts.router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d. Body: %s", http.StatusOK, w.Code, w.Body.String())
	}

	contentType := w.Header().Get("Content-Type")
	if !strings.Contains(contentType, "text/calendar") {
		t.Errorf("Expected Content-Type to contain 'text/calendar', got %s", contentType)
	}

	body := w.Body.String()
	if !strings.Contains(body, "BEGIN:VCALENDAR") {
		t.Error("Expected response to contain 'BEGIN:VCALENDAR'")
	}

	if !strings.Contains(body, "END:VCALENDAR") {
		t.Error("Expected response to contain 'END:VCALENDAR'")
	}
}

// TestCalendarEndpoint_InvalidToken tests the calendar endpoint with an invalid token
func TestCalendarEndpoint_InvalidToken(t *testing.T) {
	ts := setupTestServer(t)
	defer ts.Cleanup()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/calendar/invalid-token?key=testkey", nil)
	ts.router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("Expected status %d, got %d", http.StatusNotFound, w.Code)
	}
}

// TestCalendarEndpoint_MissingKey tests the calendar endpoint with missing key
func TestCalendarEndpoint_MissingKey(t *testing.T) {
	ts := setupTestServer(t)
	defer ts.Cleanup()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/calendar/some-token", nil)
	ts.router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

// TestCalendarEndpoint_InvalidKeyFormat tests the calendar endpoint with invalid key format
func TestCalendarEndpoint_InvalidKeyFormat(t *testing.T) {
	ts := setupTestServer(t)
	defer ts.Cleanup()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/calendar/some-token?key=invalid-key-format!!!", nil)
	ts.router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

// TestHomeWithBasicAuth tests the home endpoint with Basic Auth (returns calendar)
func TestHomeWithBasicAuth(t *testing.T) {
	ts := setupTestServer(t)
	defer ts.Cleanup()

	// Create Basic Auth header
	credentials := base64.StdEncoding.EncodeToString([]byte("test@student.junia.com:testpass"))

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Basic "+credentials)
	ts.router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d. Body: %s", http.StatusOK, w.Code, w.Body.String())
	}

	contentType := w.Header().Get("Content-Type")
	if !strings.Contains(contentType, "text/calendar") {
		t.Errorf("Expected Content-Type to contain 'text/calendar', got %s", contentType)
	}
}

// TestHomeWithBasicAuth_InvalidCredentials tests Basic Auth with invalid credentials
func TestHomeWithBasicAuth_InvalidCredentials(t *testing.T) {
	ts := setupTestServer(t)
	defer ts.Cleanup()

	// Create Basic Auth header with wrong credentials
	credentials := base64.StdEncoding.EncodeToString([]byte("wrong@student.junia.com:wrongpass"))

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Basic "+credentials)
	ts.router.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("Expected status %d, got %d", http.StatusForbidden, w.Code)
	}
}

// TestHomeWithBasicAuth_MissingHeader tests Basic Auth with missing header
func TestHomeWithBasicAuth_MissingHeader(t *testing.T) {
	ts := setupTestServer(t)
	defer ts.Cleanup()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/", nil)
	// No Authorization header
	ts.router.ServeHTTP(w, req)

	// Without Accept: text/html and without Authorization, it should return 401
	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected status %d, got %d", http.StatusUnauthorized, w.Code)
	}

	// Should include WWW-Authenticate header
	if w.Header().Get("WWW-Authenticate") == "" {
		t.Error("Expected WWW-Authenticate header to be set")
	}
}

// TestEndToEndWorkflow tests the complete workflow: generate token -> access calendar
func TestEndToEndWorkflow(t *testing.T) {
	ts := setupTestServer(t)
	defer ts.Cleanup()

	email := "test@student.junia.com"
	password := "testpass"

	// Step 1: Generate token
	requestBody := map[string]string{
		"email":    email,
		"password": password,
	}

	jsonBody, _ := json.Marshal(requestBody)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/api/generate-token", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	ts.router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Step 1 failed: Expected status %d, got %d", http.StatusOK, w.Code)
	}

	var tokenResponse handlers.GenerateTokenResponse
	if err := json.Unmarshal(w.Body.Bytes(), &tokenResponse); err != nil {
		t.Fatalf("Failed to parse token response: %v", err)
	}

	t.Logf("Generated token: %s", tokenResponse.Token)
	t.Logf("Calendar URL: %s", tokenResponse.URL)

	// Step 2: Access calendar with token
	w = httptest.NewRecorder()
	calendarURL := "/calendar/" + tokenResponse.Token + "?key=" + tokenResponse.EncryptionKey
	req, _ = http.NewRequest(http.MethodGet, calendarURL, nil)
	ts.router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Step 2 failed: Expected status %d, got %d. Body: %s", http.StatusOK, w.Code, w.Body.String())
	}

	calendarBody := w.Body.String()
	if !strings.Contains(calendarBody, "BEGIN:VCALENDAR") {
		t.Error("Calendar response doesn't contain BEGIN:VCALENDAR")
	}

	// Step 3: Access calendar again (should use cache)
	w = httptest.NewRecorder()
	req, _ = http.NewRequest(http.MethodGet, calendarURL, nil)
	ts.router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Step 3 failed: Expected status %d, got %d", http.StatusOK, w.Code)
	}

	// Step 4: Verify health endpoint still works
	w = httptest.NewRecorder()
	req, _ = http.NewRequest(http.MethodGet, "/api/health", nil)
	ts.router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Step 4 failed: Expected status %d, got %d", http.StatusOK, w.Code)
	}

	t.Log("End-to-end workflow completed successfully")
}

// TestConcurrentRequests tests handling of concurrent requests
func TestConcurrentRequests(t *testing.T) {
	ts := setupTestServer(t)
	defer ts.Cleanup()

	// Generate a token first
	requestBody := map[string]string{
		"email":    "test@student.junia.com",
		"password": "testpass",
	}

	jsonBody, _ := json.Marshal(requestBody)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/api/generate-token", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	ts.router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Failed to generate token: %d", w.Code)
	}

	var tokenResponse handlers.GenerateTokenResponse
	json.Unmarshal(w.Body.Bytes(), &tokenResponse)

	// Make concurrent requests to different endpoints
	done := make(chan bool, 4)

	// Request 1: Health
	go func() {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodGet, "/api/health", nil)
		ts.router.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Errorf("Health request failed: %d", w.Code)
		}
		done <- true
	}()

	// Request 2: Home
	go func() {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodGet, "/", nil)
		req.Header.Set("Accept", "text/html")
		ts.router.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Errorf("Home request failed: %d", w.Code)
		}
		done <- true
	}()

	// Request 3: Privacy
	go func() {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodGet, "/privacy", nil)
		ts.router.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Errorf("Privacy request failed: %d", w.Code)
		}
		done <- true
	}()

	// Request 4: Calendar
	go func() {
		w := httptest.NewRecorder()
		calendarURL := "/calendar/" + tokenResponse.Token + "?key=" + tokenResponse.EncryptionKey
		req, _ := http.NewRequest(http.MethodGet, calendarURL, nil)
		ts.router.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Errorf("Calendar request failed: %d", w.Code)
		}
		done <- true
	}()

	// Wait for all requests
	for i := 0; i < 4; i++ {
		<-done
	}
}
