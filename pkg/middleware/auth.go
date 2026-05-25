package middleware

import (
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"

	"github.com/herj1025/kumquat/pkg/constant"
	"github.com/herj1025/kumquat/pkg/i18n"
	"github.com/herj1025/kumquat/pkg/response"
)

// TokenPair contains both access and refresh tokens.
type TokenPair struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

// RevokedTokenStore provides refresh token revocation checks (for rotation).
type RevokedTokenStore interface {
	Contains(jti string) (bool, error)
	Revoke(jti string, ttl time.Duration) error
}

// Authorization returns a middleware that validates JWT access tokens.
func Authorization(secret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		token := c.GetHeader(constant.AuthorizationKey)
		if token == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, response.Response[any]{
				Code:    constant.Unauthorized,
				Message: i18n.Translate(i18n.GinGetLang(c), i18n.KeyUnauthorized),
				Data:    nil,
			})
			return
		}

		claims, err := parseToken(token, secret)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, response.Response[any]{
				Code:    constant.TokenInvalid,
				Message: i18n.Translate(i18n.GinGetLang(c), i18n.KeyTokenInvalid),
				Data:    nil,
			})
			return
		}

		c.Set("claims", claims)
		c.Next()
	}
}

// GenerateAccessToken creates a short-lived JWT access token.
func GenerateAccessToken(claims jwt.MapClaims, secret string, expire time.Duration) (string, error) {
	claims["exp"] = time.Now().Add(expire).Unix()
	claims["iat"] = time.Now().Unix()
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(secret))
}

// GenerateRefreshToken creates a long-lived JWT refresh token.
func GenerateRefreshToken(claims jwt.MapClaims, secret string, expire time.Duration) (string, error) {
	claims["exp"] = time.Now().Add(expire).Unix()
	claims["iat"] = time.Now().Unix()
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(secret))
}

// GenerateTokenPair creates both access and refresh tokens.
func GenerateTokenPair(
	claims jwt.MapClaims,
	accessSecret, refreshSecret string,
	accessExpire, refreshExpire time.Duration,
) (*TokenPair, error) {
	accessToken, err := GenerateAccessToken(claims, accessSecret, accessExpire)
	if err != nil {
		return nil, fmt.Errorf("generate access token: %w", err)
	}
	refreshToken, err := GenerateRefreshToken(claims, refreshSecret, refreshExpire)
	if err != nil {
		return nil, fmt.Errorf("generate refresh token: %w", err)
	}
	return &TokenPair{AccessToken: accessToken, RefreshToken: refreshToken}, nil
}

// RefreshAccessToken validates a refresh token and returns a new token pair.
// When store is non-nil, it performs rotation: revokes the old token ID and
// rejects reuse of previously revoked tokens.
func RefreshAccessToken(
	refreshToken, accessSecret, refreshSecret string,
	accessExpire, refreshExpire time.Duration,
	store RevokedTokenStore,
) (*TokenPair, error) {
	claims, err := parseToken(refreshToken, refreshSecret)
	if err != nil {
		return nil, fmt.Errorf("invalid refresh token: %w", err)
	}

	if store != nil {
		jti, _ := claims["jti"].(string)
		if jti != "" {
			revoked, err := store.Contains(jti)
			if err != nil {
				return nil, fmt.Errorf("check revocation: %w", err)
			}
			if revoked {
				return nil, fmt.Errorf("refresh token has been revoked")
			}
			exp, _ := claims["exp"].(float64)
			ttl := time.Duration(0)
			if exp > 0 {
				ttl = time.Until(time.Unix(int64(exp), 0))
			}
			if err := store.Revoke(jti, ttl); err != nil {
				return nil, fmt.Errorf("revoke refresh token: %w", err)
			}
		}
	}

	newClaims := jwt.MapClaims{}
	for k, v := range claims {
		if k == "exp" || k == "iat" || k == "jti" {
			continue
		}
		newClaims[k] = v
	}
	newClaims["jti"] = fmt.Sprintf("%d", time.Now().UnixNano())

	return GenerateTokenPair(newClaims, accessSecret, refreshSecret, accessExpire, refreshExpire)
}

// GetClaims returns the JWT claims stored in the context.
func GetClaims(c *gin.Context) jwt.MapClaims {
	claims, ok := c.Get("claims")
	if !ok {
		return nil
	}
	return claims.(jwt.MapClaims)
}

func parseToken(tokenString, secret string) (jwt.MapClaims, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (any, error) {
		return []byte(secret), nil
	})
	if err != nil {
		return nil, err
	}
	if !token.Valid {
		return nil, jwt.ErrTokenInvalidClaims
	}
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return nil, jwt.ErrTokenInvalidClaims
	}
	return claims, nil
}

// GetClaim retrieves a typed value from JWT claims by key.
func GetClaim[T any](c *gin.Context, key string) (T, error) {
	var zero T
	claims := GetClaims(c)
	if claims == nil {
		return zero, fmt.Errorf("claims not found")
	}
	v, ok := claims[key]
	if !ok {
		return zero, fmt.Errorf("key %q not found in claims", key)
	}
	t, ok := v.(T)
	if !ok {
		return zero, fmt.Errorf("key %q type mismatch", key)
	}
	return t, nil
}

// GetUserID returns the user ID from JWT claims.
func GetUserID(c *gin.Context) string {
	id, _ := GetClaim[string](c, "userId")
	return id
}
