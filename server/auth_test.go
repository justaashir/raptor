package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

func TestGenerateAndValidateToken(t *testing.T) {
	secret := "test-secret-key"
	token, err := GenerateToken("alice", secret)
	if err != nil {
		t.Fatalf("unexpected error generating token: %v", err)
	}
	username, err := ValidateToken(token, secret)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if username != "alice" {
		t.Fatalf("expected alice, got %s", username)
	}
}

func TestValidateToken_BadSecret(t *testing.T) {
	token, _ := GenerateToken("alice", "secret1")
	_, err := ValidateToken(token, "secret2")
	if err == nil {
		t.Fatal("expected error for bad secret")
	}
}

func TestValidateToken_Expired(t *testing.T) {
	secret := "test-secret"
	token := makeExpiredToken("alice", secret)
	_, err := ValidateToken(token, secret)
	if err == nil {
		t.Fatal("expected error for expired token")
	}
}

func TestValidateToken_InvalidFormat(t *testing.T) {
	_, err := ValidateToken("not-a-jwt-token", "secret")
	if err == nil {
		t.Fatal("expected error for invalid token")
	}
}

func TestValidateToken_WrongSigningMethod(t *testing.T) {
	// Create a token with "none" signing method
	token := jwt.NewWithClaims(jwt.SigningMethodNone, jwt.RegisteredClaims{
		Subject:   "alice",
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
	})
	tokenStr, _ := token.SignedString(jwt.UnsafeAllowNoneSignatureType)
	_, err := ValidateToken(tokenStr, "secret")
	if err == nil {
		t.Fatal("expected error for none signing method")
	}
}

func TestAuthMiddleware_PublicRoutes(t *testing.T) {
	srv := newTestServerWithAuth(t, "secret", []string{"alice"})
	// GET routes
	for _, path := range []string{"/api/version", "/install.sh"} {
		req := httptest.NewRequest("GET", path, nil)
		w := httptest.NewRecorder()
		srv.ServeHTTP(w, req)
		if w.Code == http.StatusUnauthorized {
			t.Fatalf("expected public route %s to not require auth, got 401", path)
		}
	}
	// POST-only route
	req := httptest.NewRequest("POST", "/api/auth", strings.NewReader(`{"username":"alice"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)
	if w.Code == http.StatusUnauthorized {
		t.Fatalf("expected public route /api/auth to not require auth, got 401")
	}
}

func TestAuthMiddleware_RequiresToken(t *testing.T) {
	srv := newTestServerWithAuth(t, "secret", []string{"alice"})

	req := httptest.NewRequest("GET", "/api/workspaces/", nil)
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 without token, got %d", w.Code)
	}
}

func TestAuthMiddleware_ValidToken(t *testing.T) {
	srv := newTestServerWithAuth(t, "secret", []string{"alice"})
	token, _ := GenerateToken("alice", "secret")

	req := httptest.NewRequest("GET", "/api/workspaces/", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 with valid token, got %d: %s", w.Code, w.Body.String())
	}
}

func TestHandleAuth_AllowedUser(t *testing.T) {
	srv := newTestServerWithAuth(t, "secret", []string{"alice", "bob"})

	body := `{"username":"alice"}`
	req := httptest.NewRequest("POST", "/api/auth", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]string
	json.NewDecoder(w.Body).Decode(&resp)
	if resp["token"] == "" {
		t.Fatal("expected token in response")
	}
	if resp["username"] != "alice" {
		t.Fatalf("expected username alice, got %s", resp["username"])
	}

	// Validate the returned token works
	username, err := ValidateToken(resp["token"], "secret")
	if err != nil {
		t.Fatalf("returned token should be valid: %v", err)
	}
	if username != "alice" {
		t.Fatalf("expected alice, got %s", username)
	}
}

func TestHandleAuth_DisallowedUser(t *testing.T) {
	srv := newTestServerWithAuth(t, "secret", []string{"alice"})

	body := `{"username":"eve"}`
	req := httptest.NewRequest("POST", "/api/auth", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d: %s", w.Code, w.Body.String())
	}
}

func TestHandleAuth_EmptyUsername(t *testing.T) {
	srv := newTestServerWithAuth(t, "secret", []string{"alice"})

	body := `{"username":""}`
	req := httptest.NewRequest("POST", "/api/auth", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

// helpers

func newTestServerWithAuth(t *testing.T, secret string, users []string) *Server {
	t.Helper()
	db, err := NewDB(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	hub := NewHub()
	go hub.Run()
	return NewServer(db, hub, WithSecret(secret), WithAllowedUsers(users))
}

func makeExpiredToken(username, secret string) string {
	claims := jwt.RegisteredClaims{
		Subject:   username,
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(-time.Hour)),
		IssuedAt:  jwt.NewNumericDate(time.Now().Add(-2 * time.Hour)),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	str, _ := token.SignedString([]byte(secret))
	return str
}
