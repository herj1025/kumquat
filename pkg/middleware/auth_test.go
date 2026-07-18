package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

func TestAuthorizationAcceptsBearerToken(t *testing.T) {
	gin.SetMode(gin.TestMode)

	token, err := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"userId": "u1",
		"exp":    time.Now().Add(time.Minute).Unix(),
		"iat":    time.Now().Unix(),
	}).SignedString([]byte("secret"))
	if err != nil {
		t.Fatalf("SignedString returned error: %v", err)
	}

	r := gin.New()
	r.GET("/protected", Authorization("secret"), func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, w.Code)
	}
}

func TestAuthorizationAcceptsRawToken(t *testing.T) {
	gin.SetMode(gin.TestMode)

	token, err := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"userId": "u1",
		"exp":    time.Now().Add(time.Minute).Unix(),
		"iat":    time.Now().Unix(),
	}).SignedString([]byte("secret"))
	if err != nil {
		t.Fatalf("SignedString returned error: %v", err)
	}

	r := gin.New()
	r.GET("/protected", Authorization("secret"), func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", token)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, w.Code)
	}
}
