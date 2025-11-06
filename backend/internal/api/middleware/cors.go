package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// CorsMiddleware configure les headers CORS pour permettre les requêtes cross-origin
// Autorise toutes les méthodes HTTP nécessaires et gère les requêtes OPTIONS preflight
func CorsMiddleware() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		ctx.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		ctx.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		ctx.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
		ctx.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if ctx.Request.Method == "OPTIONS" {
			ctx.AbortWithStatus(http.StatusNoContent)
			return
		}

		ctx.Next()
	}
}
