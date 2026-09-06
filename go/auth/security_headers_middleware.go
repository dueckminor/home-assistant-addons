package auth

import "github.com/gin-gonic/gin"

func authSecurityHeadersMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		headers := c.Writer.Header()

		headers.Set("X-Content-Type-Options", "nosniff")
		headers.Set("X-Frame-Options", "DENY")
		headers.Set("Referrer-Policy", "no-referrer")
		headers.Set("Permissions-Policy", "camera=(), microphone=(), geolocation=()")
		headers.Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")

		c.Next()
	}
}
