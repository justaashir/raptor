package server

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"raptor/model"
	"strings"
	"time"
)

const tokenTTL = 30 * 24 * time.Hour

type contextKey string

const usernameKey contextKey = "username"

func GenerateToken(username, secret string) string {
	expiry := time.Now().Add(tokenTTL).Unix()
	payload := fmt.Sprintf("%s:%d", username, expiry)
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(payload))
	sig := hex.EncodeToString(mac.Sum(nil))
	raw := fmt.Sprintf("%s:%d:%s", username, expiry, sig)
	return base64.RawURLEncoding.EncodeToString([]byte(raw))
}

func ValidateToken(token, secret string) (string, error) {
	raw, err := base64.RawURLEncoding.DecodeString(token)
	if err != nil {
		return "", fmt.Errorf("invalid token encoding")
	}
	parts := strings.SplitN(string(raw), ":", 3)
	if len(parts) != 3 {
		return "", fmt.Errorf("invalid token format")
	}
	username := parts[0]
	expiryStr := parts[1]
	sig := parts[2]

	// Verify HMAC
	payload := fmt.Sprintf("%s:%s", username, expiryStr)
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(payload))
	expected := hex.EncodeToString(mac.Sum(nil))
	if !hmac.Equal([]byte(sig), []byte(expected)) {
		return "", fmt.Errorf("invalid token signature")
	}

	// Check expiry
	var expiry int64
	fmt.Sscanf(expiryStr, "%d", &expiry)
	if time.Now().Unix() > expiry {
		return "", fmt.Errorf("token expired")
	}

	return username, nil
}

func UsernameFromContext(ctx context.Context) string {
	if v, ok := ctx.Value(usernameKey).(string); ok {
		return v
	}
	return ""
}

func isPublicRoute(path string) bool {
	switch {
	case path == "/api/version":
		return true
	case path == "/api/auth":
		return true
	case path == "/install.sh":
		return true
	case path == "/ws":
		return true
	}
	return false
}

func (s *Server) authMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if isPublicRoute(r.URL.Path) || s.secret == "" {
			next.ServeHTTP(w, r)
			return
		}

		auth := r.Header.Get("Authorization")
		if !strings.HasPrefix(auth, "Bearer ") {
			http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
			return
		}
		token := strings.TrimPrefix(auth, "Bearer ")
		username, err := ValidateToken(token, s.secret)
		if err != nil {
			http.Error(w, fmt.Sprintf(`{"error":"%s"}`, err.Error()), http.StatusUnauthorized)
			return
		}
		ctx := context.WithValue(r.Context(), usernameKey, username)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func (s *Server) handleAuth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var input struct {
		Username string `json:"username"`
	}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	if input.Username == "" {
		http.Error(w, `{"error":"username required"}`, http.StatusBadRequest)
		return
	}

	// Check workspace membership — user must belong to at least one workspace
	isMember, _ := s.db.IsWorkspaceMember(input.Username)
	if !isMember {
		allowed := false
		// Check seed allowlist
		for _, u := range s.allowedUsers {
			if strings.EqualFold(u, input.Username) {
				allowed = true
				break
			}
		}
		// If no allowlist and no workspace members exist yet (fresh install), allow anyone
		if !allowed && len(s.allowedUsers) == 0 {
			var count int64
			s.db.conn.Model(&model.WorkspaceMember{}).Count(&count)
			if count == 0 {
				allowed = true
			}
		}
		if !allowed {
			http.Error(w, `{"error":"user not allowed"}`, http.StatusForbidden)
			return
		}
	}

	token := GenerateToken(input.Username, s.secret)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"token":    token,
		"username": input.Username,
	})
}
