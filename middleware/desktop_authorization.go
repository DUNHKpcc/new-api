package middleware

import (
	"errors"
	"net/http"
	"strings"

	"github.com/QuantumNous/new-api/service"
	"github.com/gin-gonic/gin"
)

const desktopAccessContextKey = "desktop_access"

func DesktopAuthorizationSecurityHeaders() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Cache-Control", "no-store")
		c.Header("Pragma", "no-cache")
		c.Header("Expires", "0")
		c.Header("Content-Security-Policy", "default-src 'self'; script-src 'self'; style-src 'self' 'unsafe-inline'; img-src 'self' data:; font-src 'self' data:; connect-src 'self'; frame-ancestors 'none'; base-uri 'self'; form-action 'self'")
		c.Header("X-Frame-Options", "DENY")
		c.Header("Referrer-Policy", "no-referrer")
		c.Next()
	}
}

func DesktopAuthorizationStartRateLimit() gin.HandlerFunc {
	return rateLimitFactory(30, 60, "DAR")
}

func DesktopAuthorizationDecisionRateLimit() gin.HandlerFunc {
	return userRateLimitFactory(20, 60, "DAD")
}

func DesktopTokenExchangeRateLimit() gin.HandlerFunc {
	return rateLimitFactory(30, 60, "DAT")
}

func DesktopTokenReadRateLimit() gin.HandlerFunc {
	return rateLimitFactory(120, 60, "DAG")
}

func DesktopTokenAuth(requiredScope string) gin.HandlerFunc {
	return func(c *gin.Context) {
		access, err := service.AuthenticateDesktopAccessToken(c.GetHeader("Authorization"), requiredScope)
		if err != nil {
			status := http.StatusUnauthorized
			code := "DESKTOP_TOKEN_INVALID"
			if errors.Is(err, service.ErrDesktopScopeDenied) {
				status = http.StatusForbidden
				code = "DESKTOP_SCOPE_DENIED"
			}
			c.AbortWithStatusJSON(status, gin.H{
				"error":             code,
				"error_description": http.StatusText(status),
			})
			return
		}
		c.Set("id", access.User.Id)
		c.Set("token_id", access.Token.Id)
		c.Set("token_key", access.Token.Key)
		c.Set(desktopAccessContextKey, access)
		c.Next()
	}
}

func GetDesktopAccess(c *gin.Context) (*service.DesktopAccess, bool) {
	value, ok := c.Get(desktopAccessContextKey)
	if !ok {
		return nil, false
	}
	access, ok := value.(*service.DesktopAccess)
	return access, ok && access != nil
}

func DesktopPageSecurityHeaders() gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.Request.URL.Path == "/desktop/authorize" || strings.HasPrefix(c.Request.URL.Path, "/desktop/authorize/") {
			c.Header("Cache-Control", "no-store")
			c.Header("Pragma", "no-cache")
			c.Header("Expires", "0")
			c.Header("Content-Security-Policy", "default-src 'self'; script-src 'self'; style-src 'self' 'unsafe-inline'; img-src 'self' data:; font-src 'self' data:; connect-src 'self'; frame-ancestors 'none'; base-uri 'self'; form-action 'self'")
			c.Header("X-Frame-Options", "DENY")
			c.Header("Referrer-Policy", "no-referrer")
		}
		c.Next()
	}
}
