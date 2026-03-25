package middleware

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

const secretToken = "super-secret-admin-token"

func performAdminAuthRequest(router *gin.Engine, authHeader string) *httptest.ResponseRecorder {
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/admin", nil)
	if authHeader != "" {
		req.Header.Set("Authorization", authHeader)
	}
	router.ServeHTTP(w, req)
	return w
}

func setupAdminAuthRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(AdminAuthMiddleware(secretToken))
	router.GET("/admin", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})
	return router
}

func TestAdminAuthMiddleware_MissingHeader(t *testing.T) {
	router := setupAdminAuthRouter()
	w := performAdminAuthRequest(router, "")

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected status 401, got %d", w.Code)
	}
	if got := w.Header().Get("WWW-Authenticate"); got != "Bearer" {
		t.Errorf("expected WWW-Authenticate header \"Bearer\", got %q", got)
	}
	var resp map[string]string
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp["error"] != "Authorization header required" {
		t.Errorf("expected error \"Authorization header required\", got %q", resp["error"])
	}
}

func TestAdminAuthMiddleware_MalformedScheme(t *testing.T) {
	router := setupAdminAuthRouter()
	w := performAdminAuthRequest(router, "Basic dXNlcjpwYXNz")

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected status 401, got %d", w.Code)
	}
	var resp map[string]string
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp["error"] != "Invalid authorization format" {
		t.Errorf("expected error \"Invalid authorization format\", got %q", resp["error"])
	}
}

func TestAdminAuthMiddleware_EmptyToken(t *testing.T) {
	router := setupAdminAuthRouter()
	w := performAdminAuthRequest(router, "Bearer ")

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected status 401, got %d", w.Code)
	}
	var resp map[string]string
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp["error"] != "Invalid token" {
		t.Errorf("expected error \"Invalid token\", got %q", resp["error"])
	}
}

func TestAdminAuthMiddleware_WrongToken(t *testing.T) {
	router := setupAdminAuthRouter()
	w := performAdminAuthRequest(router, "Bearer wrong-token")

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected status 401, got %d", w.Code)
	}
	var resp map[string]string
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp["error"] != "Invalid token" {
		t.Errorf("expected error \"Invalid token\", got %q", resp["error"])
	}
}

func TestAdminAuthMiddleware_Success(t *testing.T) {
	router := setupAdminAuthRouter()
	w := performAdminAuthRequest(router, "Bearer "+secretToken)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}
}

func TestAdminAuthMiddleware_CaseInsensitivity(t *testing.T) {
	router := setupAdminAuthRouter()

	t.Run("lowercase", func(t *testing.T) {
		w := performAdminAuthRequest(router, "bearer "+secretToken)
		if w.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d", w.Code)
		}
	})

	t.Run("uppercase", func(t *testing.T) {
		w := performAdminAuthRequest(router, "BEARER "+secretToken)
		if w.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d", w.Code)
		}
	})
}
