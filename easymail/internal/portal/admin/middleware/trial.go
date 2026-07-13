package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// TrialMode returns a middleware that blocks write operations (POST, PUT, PATCH, DELETE)
// when trial mode is enabled. Read-only operations (GET) are allowed.
func TrialMode(enabled bool) gin.HandlerFunc {
	return func(c *gin.Context) {
		if !enabled {
			c.Next()
			return
		}
		switch c.Request.Method {
		case http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete:
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"code":    -1,
				"message": "Trial mode: this operation is not allowed",
			})
			return
		}
		c.Next()
	}
}