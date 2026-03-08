package middleware

import (
	"net/http"
	"os"
	"strings"
	"sync"

	"github.com/gin-gonic/gin"
)

var (
	corsOnce       sync.Once
	allowedOrigins map[string]struct{}
)

func loadAllowedOrigins() {
	corsOnce.Do(func() {
		allowedOrigins = make(map[string]struct{})
		raw := strings.TrimSpace(os.Getenv("CORS_ALLOWED_ORIGINS"))
		if raw == "" {
			raw = "http://localhost:4200,http://127.0.0.1:4200"
		}
		for _, origin := range strings.Split(raw, ",") {
			o := strings.TrimSpace(origin)
			if o != "" {
				allowedOrigins[o] = struct{}{}
			}
		}
	})
}

func isOriginAllowed(origin string) bool {
	loadAllowedOrigins()
	_, ok := allowedOrigins[origin]
	return ok
}

// CorsMiddleware configure les headers CORS pour permettre les requêtes cross-origin
// Autorise toutes les méthodes HTTP nécessaires et gère les requêtes OPTIONS preflight
func CorsMiddleware() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		origin := strings.TrimSpace(ctx.GetHeader("Origin"))
		if origin != "" {
			if isOriginAllowed(origin) {
				ctx.Writer.Header().Set("Access-Control-Allow-Origin", origin)
				ctx.Writer.Header().Set("Vary", "Origin")
				ctx.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
			} else if ctx.Request.Method == "OPTIONS" {
				ctx.AbortWithStatus(http.StatusForbidden)
				return
			}
		}
		ctx.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
		ctx.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if ctx.Request.Method == "OPTIONS" {
			ctx.AbortWithStatus(http.StatusNoContent)
			return
		}

		ctx.Next()
	}
}
