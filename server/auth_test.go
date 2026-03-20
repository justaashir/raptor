package server

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestGenerateAndValidateToken(t *testing.T) {
	secret := "test-secret-key"
	token := GenerateToken("alice", secret)
	username, err := ValidateToken(token, secret)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if username != "alice" {
		t.Fatalf("expected alice, got %s", username)
	}
}

func TestValidateToken_BadHMAC(t *testing.T) {
	token := GenerateToken("alice", "secret1")
	_, err := ValidateToken(token, "secret2")
	if err == nil {
		t.Fatal("expected error for bad HMAC")
	}
	if !strings.Contains(err.Error(), "signature") {
		t.Fatalf("expected signature error, got: %v", err)
	}
}

func TestValidateToken_Expired(t *testing.T) {
	secret := "test-secret"
	token := makeExpiredToken("alice", secret)
	_, err := ValidateToken(token, secret)
	if err == nil {
		t.Fatal("expected error for expired token")
	}
	if !strings.Contains(err.Error(), "expired") {
		t.Fatalf("expected expired error, got: %v", err)
	}
}

func TestValidateToken_InvalidEncoding(t *testing.T) {
	_, err := ValidateToken("not-valid-base64!!!", "secret")
	if err == nil {
		t.Fatal("expected error for invalid encoding")
	}
}

func TestValidateToken_InvalidFormat(t *testing.T) {
	_, err := ValidateToken("aGVsbG8", "secret") // base64 of "hello" — no colons
	if err == nil {
		t.Fatal("expected error for invalid format")
	}
}

func TestAuthMiddleware_PublicRoutes(t *testing.T) {
	srv := newTestServerWithAuth(t, "secret", []string{"alice"})
	for _, path := range []string{"/api/version", "/api/auth", "/install.sh", "/releases/linux/amd64"} {
		req := httptest.NewRequest("GET", path, nil)
		w := httptest.NewRecorder()
		srv.ServeHTTP(w, req)
		if w.Code == http.StatusUnauthorized {
			t.Fatalf("expected public route %s to not require auth, got 401", path)
		}
	}
}

func TestAuthMiddleware_RequiresToken(t *testing.T) {
	srv := newTestServerWithAuth(t, "secret", []string{"alice"})

	req := httptest.NewRequest("GET", "/api/tickets", nil)
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 without token, got %d", w.Code)
	}
}

func TestAuthMiddleware_ValidToken(t *testing.T) {
	srv := newTestServerWithAuth(t, "secret", []string{"alice"})
	token := GenerateToken("alice", "secret")

	req := httptest.NewRequest("GET", "/api/tickets", nil)
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
	expiry := time.Now().Add(-time.Hour).Unix()
	payload := fmt.Sprintf("%s:%d", username, expiry)
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(payload))
	sig := hex.EncodeToString(mac.Sum(nil))
	raw := fmt.Sprintf("%s:%d:%s", username, expiry, sig)
	return base64.RawURLEncoding.EncodeToString([]byte(raw))
}
