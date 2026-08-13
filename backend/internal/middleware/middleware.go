package middleware

import (
	"net/http"
	"slices"

	"github.com/gin-gonic/gin"
	"github.com/homegauge/homegauge/backend/internal/auth"
	"github.com/homegauge/homegauge/backend/internal/platform/httpx"
)

func Authenticate(svc *auth.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		sid, err := c.Cookie(auth.SessionCookie)
		if err != nil || sid == "" {
			httpx.Unauthorized(c, "Please sign in.")
			return
		}
		user, err := svc.SessionUser(c.Request.Context(), sid)
		if err != nil {
			httpx.Unauthorized(c, "Please sign in.")
			return
		}
		c.Set("auth_user", *user)
		c.Set("session_id", sid)
		c.Next()
	}
}

func OptionalAuth(svc *auth.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		sid, err := c.Cookie(auth.SessionCookie)
		if err == nil && sid != "" {
			if user, err := svc.SessionUser(c.Request.Context(), sid); err == nil {
				c.Set("auth_user", *user)
				c.Set("session_id", sid)
			}
		}
		c.Next()
	}
}

func RequireRoles(roles ...auth.Role) gin.HandlerFunc {
	return func(c *gin.Context) {
		raw, ok := c.Get("auth_user")
		if !ok {
			httpx.Unauthorized(c, "Please sign in.")
			return
		}
		user := raw.(auth.SessionUser)
		if !slices.Contains(roles, user.Role) {
			httpx.Forbidden(c, "You do not have permission to do that.")
			return
		}
		c.Next()
	}
}

func CORS(origins []string) gin.HandlerFunc {
	allowed := map[string]struct{}{}
	for _, o := range origins {
		allowed[o] = struct{}{}
	}
	return func(c *gin.Context) {
		origin := c.GetHeader("Origin")
		if _, ok := allowed[origin]; ok {
			c.Header("Access-Control-Allow-Origin", origin)
			c.Header("Access-Control-Allow-Credentials", "true")
			c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization")
			c.Header("Access-Control-Allow-Methods", "GET,POST,PUT,PATCH,DELETE,OPTIONS")
		}
		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	}
}
