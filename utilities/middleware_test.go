package utilities

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

const middlewareTestEncryptKey = "0123456789abcdef0123456789abcdef"

func configureMiddlewareTest(t *testing.T) {
	t.Helper()

	t.Setenv("DB_PORT2", "5432")
	t.Setenv("DB_HOST2", "localhost")
	t.Setenv("DB_USER2", "test")
	t.Setenv("DB_PASS2", "test")
	t.Setenv("DB_NAME2", "test")
	t.Setenv("REDIS_PORT", "6379")
	t.Setenv("REDIS_HOST", "localhost")
	t.Setenv("APP_PORT", "8080")
	t.Setenv("APP_HOST", "localhost")
	t.Setenv("SESSION_TIME", "3600")
	t.Setenv("REFRESH_TIME", "300")
	t.Setenv("ENCRYPT_KEY", middlewareTestEncryptKey)
}

func createMiddlewareTestToken(t *testing.T, claims map[string]interface{}) string {
	t.Helper()

	token, err := JWEEncryptAES(claims, []byte(middlewareTestEncryptKey))
	if err != nil {
		t.Fatalf("failed to create encrypted middleware test token: %v", err)
	}
	return token
}

func sendMiddlewareTestRequest(handler http.Handler, token string) *httptest.ResponseRecorder {
	request := httptest.NewRequest(http.MethodPost, "/api/auth/logout", nil)
	if token != "" {
		request.Header.Set("Authorization", "Bearer "+token)
	}
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	return response
}

func TestLogoutAuthMiddlewareAllowsExpiredToken(t *testing.T) {
	configureMiddlewareTest(t)
	expiredToken := createMiddlewareTestToken(t, map[string]interface{}{
		"Expiry":   time.Now().Add(-time.Minute).Unix(),
		"id":       "test-user-id",
		"username": "johndoe",
		"roleName": "Student",
	})

	called := false
	handler := LogoutAuthMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		if got := GetUsername(r.Context()); got != "johndoe" {
			t.Errorf("expected username in request context to be johndoe, got %q", got)
		}
		w.WriteHeader(http.StatusNoContent)
	}))

	response := sendMiddlewareTestRequest(handler, expiredToken)
	if response.Code != http.StatusNoContent {
		t.Fatalf("expected expired token to reach logout handler with status %d, got %d: %s", http.StatusNoContent, response.Code, response.Body.String())
	}
	if !called {
		t.Fatal("expected logout handler to be called for an expired token")
	}
	if got := response.Header().Get("xrfs-token"); got != "" {
		t.Errorf("expected expired token not to be returned in xrfs-token header, got %q", got)
	}
}

func TestAuthMiddlewareRejectsExpiredToken(t *testing.T) {
	configureMiddlewareTest(t)
	expiredToken := createMiddlewareTestToken(t, map[string]interface{}{
		"Expiry":   time.Now().Add(-time.Minute).Unix(),
		"id":       "test-user-id",
		"username": "johndoe",
	})

	called := false
	handler := AuthMiddleware(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		called = true
		w.WriteHeader(http.StatusNoContent)
	}))
	response := sendMiddlewareTestRequest(handler, expiredToken)

	if response.Code != http.StatusUnauthorized {
		t.Errorf("expected expired token to be rejected with status %d, got %d", http.StatusUnauthorized, response.Code)
	}
	if called {
		t.Error("expected regular authenticated handler not to be called for an expired token")
	}
}

func TestLogoutAuthMiddlewareRejectsInvalidTokens(t *testing.T) {
	configureMiddlewareTest(t)

	tests := []struct {
		name  string
		token string
	}{
		{name: "missing authorization token"},
		{name: "malformed encrypted token", token: "not-a-jwe"},
		{
			name: "missing expiry claim",
			token: createMiddlewareTestToken(t, map[string]interface{}{
				"id": "test-user-id", "username": "johndoe",
			}),
		},
		{
			name: "invalid expiry claim",
			token: createMiddlewareTestToken(t, map[string]interface{}{
				"Expiry": "not-a-timestamp", "id": "test-user-id", "username": "johndoe",
			}),
		},
		{
			name: "missing user id",
			token: createMiddlewareTestToken(t, map[string]interface{}{
				"Expiry": time.Now().Add(-time.Minute).Unix(), "username": "johndoe",
			}),
		},
		{
			name: "missing username",
			token: createMiddlewareTestToken(t, map[string]interface{}{
				"Expiry": time.Now().Add(-time.Minute).Unix(), "id": "test-user-id",
			}),
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			called := false
			handler := LogoutAuthMiddleware(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				called = true
				w.WriteHeader(http.StatusNoContent)
			}))
			response := sendMiddlewareTestRequest(handler, tc.token)

			if response.Code != http.StatusUnauthorized {
				t.Errorf("expected invalid token to be rejected with status %d, got %d", http.StatusUnauthorized, response.Code)
			}
			if called {
				t.Error("expected logout handler not to be called for an invalid token")
			}
		})
	}
}
