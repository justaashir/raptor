package server

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"net/http"
	"raptor/model"
	"strings"
	"time"

	"github.com/labstack/echo/v4"
)

const tokenTTL = 30 * 24 * time.Hour

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

	payload := fmt.Sprintf("%s:%s", username, expiryStr)
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(payload))
	expected := hex.EncodeToString(mac.Sum(nil))
	if !hmac.Equal([]byte(sig), []byte(expected)) {
		return "", fmt.Errorf("invalid token signature")
	}

	var expiry int64
	fmt.Sscanf(expiryStr, "%d", &expiry)
	if time.Now().Unix() > expiry {
		return "", fmt.Errorf("token expired")
	}

	return username, nil
}

func (s *Server) authMiddleware(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		if s.secret == "" {
			return next(c)
		}

		auth := c.Request().Header.Get("Authorization")
		if !strings.HasPrefix(auth, "Bearer ") {
			return c.JSON(http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		}
		token := strings.TrimPrefix(auth, "Bearer ")
		user, err := ValidateToken(token, s.secret)
		if err != nil {
			return c.JSON(http.StatusUnauthorized, map[string]string{"error": err.Error()})
		}
		c.Set("username", user)
		return next(c)
	}
}

func (s *Server) handleAuth(c echo.Context) error {
	var input struct {
		Username string `json:"username"`
	}
	if err := c.Bind(&input); err != nil {
		return c.String(http.StatusBadRequest, "bad request")
	}
	if input.Username == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "username required"})
	}

	isMember, _ := s.db.IsWorkspaceMember(input.Username)
	if !isMember {
		allowed := false
		for _, u := range s.allowedUsers {
			if strings.EqualFold(u, input.Username) {
				allowed = true
				break
			}
		}
		if !allowed && len(s.allowedUsers) == 0 {
			var count int64
			s.db.conn.Model(&model.WorkspaceMember{}).Count(&count)
			if count == 0 {
				allowed = true
			}
		}
		if !allowed {
			return c.JSON(http.StatusForbidden, map[string]string{"error": "user not allowed"})
		}
	}

	token := GenerateToken(input.Username, s.secret)
	return c.JSON(http.StatusOK, map[string]string{
		"token":    token,
		"username": input.Username,
	})
}
