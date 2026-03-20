package server

import (
	"fmt"
	"net/http"
	"raptor/model"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/labstack/echo/v4"
)

const tokenTTL = 30 * 24 * time.Hour

func GenerateToken(username, secret string) (string, error) {
	claims := jwt.RegisteredClaims{
		Subject:   username,
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(tokenTTL)),
		IssuedAt:  jwt.NewNumericDate(time.Now()),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(secret))
}

func ValidateToken(tokenStr, secret string) (string, error) {
	token, err := jwt.ParseWithClaims(tokenStr, &jwt.RegisteredClaims{}, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return []byte(secret), nil
	})
	if err != nil || !token.Valid {
		return "", fmt.Errorf("invalid token")
	}
	sub, err := token.Claims.GetSubject()
	if err != nil || sub == "" {
		return "", fmt.Errorf("invalid token: missing subject")
	}
	return sub, nil
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
			return c.JSON(http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
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

	token, err := GenerateToken(input.Username, s.secret)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to generate token"})
	}
	return c.JSON(http.StatusOK, map[string]string{
		"token":    token,
		"username": input.Username,
	})
}
