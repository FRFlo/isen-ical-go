package middleware

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
)

func setupTestRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	SetupMiddleware(router)
	return router
}

func TestLoggerMiddleware(t *testing.T) {
	var buf bytes.Buffer
	logger := zerolog.New(&buf)
	router := setupTestRouter()

	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	logOutput := buf.String()
	assert.Contains(t, logOutput, "method")
	assert.Contains(t, logOutput, "GET")
	assert.Contains(t, logOutput, "path")
	assert.Contains(t, logOutput, "/test")
	assert.Contains(t, logOutput, "status")
	assert.Contains(t, logOutput, "duration")
	assert.Contains(t, logOutput, "client_ip")
	assert.Contains(t, logOutput, "request_id")
}

func TestLoggerMiddlewareSkipsHealth(t *testing.T) {
	var buf bytes.Buffer
	logger := zerolog.New(&buf)
	router := setupTestRouter()

	router.GET("/health", HealthCheck)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/health", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	logOutput := buf.String()
	assert.Empty(t, logOutput)
}

func TestRecoveryMiddleware(t *testing.T) {
	var buf bytes.Buffer
	logger := zerolog.New(&buf)
	router := setupTestRouter()

	router.GET("/panic", func(c *gin.Context) {
		panic("intentional test panic")
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/panic", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "Internal Server Error", response["error"])
	assert.Equal(t, "An unexpected error occurred", response["message"])
	assert.NotEmpty(t, response["request_id"])

	logOutput := buf.String()
	assert.Contains(t, logOutput, "Panic recovered")
	assert.Contains(t, logOutput, "intentional test panic")
	assert.Contains(t, logOutput, "stack")
}

func TestRequestIDMiddleware(t *testing.T) {
	var buf bytes.Buffer
	logger := zerolog.New(&buf)
	router := setupTestRouter()

	router.GET("/test", func(c *gin.Context) {
		requestID := GetRequestID(c)
		c.JSON(http.StatusOK, gin.H{"request_id": requestID})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	responseID := w.Header().Get(RequestIDHeader)
	assert.NotEmpty(t, responseID)

	// Check response body contains the same ID
	var response map[string]string
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, responseID, response["request_id"])
}

func TestRequestIDMiddlewarePreservesExistingID(t *testing.T) {
	var buf bytes.Buffer
	logger := zerolog.New(&buf)
	router := setupTestRouter()

	router.GET("/test", func(c *gin.Context) {
		requestID := GetRequestID(c)
		c.JSON(http.StatusOK, gin.H{"request_id": requestID})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	existingID := "custom-request-id-123"
	req.Header.Set(RequestIDHeader, existingID)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	// Check that existing ID was preserved
	responseID := w.Header().Get(RequestIDHeader)
	assert.Equal(t, existingID, responseID)

	var response map[string]string
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, existingID, response["request_id"])
}

func TestGetRequestIDNotFound(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	router.GET("/test", func(c *gin.Context) {
		requestID := GetRequestID(c)
		c.JSON(http.StatusOK, gin.H{"request_id": requestID})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	router.ServeHTTP(w, req)

	var response map[string]string
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Empty(t, response["request_id"])
}

func TestErrorResponse(t *testing.T) {
	var buf bytes.Buffer
	logger := zerolog.New(&buf)
	router := setupTestRouter()

	router.GET("/error", func(c *gin.Context) {
		ErrorResponse(c, http.StatusBadRequest, "Invalid input provided")
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/error", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "Bad Request", response["error"])
	assert.Equal(t, "Invalid input provided", response["message"])
	assert.NotEmpty(t, response["request_id"])
}

func TestHealthCheck(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.GET("/health", HealthCheck)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/health", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]string
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "healthy", response["status"])
	assert.Equal(t, "isen-ical-go", response["service"])
}

func TestLoggerMiddlewareWithQueryParams(t *testing.T) {
	var buf bytes.Buffer
	logger := zerolog.New(&buf)
	router := setupTestRouter()

	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test?foo=bar&baz=qux", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	// Check that query params are logged
	logOutput := buf.String()
	assert.Contains(t, logOutput, "query")
	assert.Contains(t, logOutput, "foo=bar")
}

func TestLoggerMiddlewareWithErrors(t *testing.T) {
	var buf bytes.Buffer
	logger := zerolog.New(&buf)
	router := setupTestRouter()

	router.GET("/error", func(c *gin.Context) {
		c.Error(assert.AnError)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed"})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/error", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)

	// Check that error was logged
	logOutput := buf.String()
	assert.Contains(t, logOutput, "Request completed with errors")
}
