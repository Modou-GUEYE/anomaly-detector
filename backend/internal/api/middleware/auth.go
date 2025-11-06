package middleware

import (
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

// jwtKey est la clé secrète pour signer les tokens JWT
var jwtKey = []byte("ma_super_cle_secrete_qui_doit_etre_longue_et_complexe")

// JwtAuthMiddleware authentifie les requêtes via un token JWT passé dans le header Authorization.
// Il vérifie la présence et la validité du token, extrait les claims utiles (userID, role)
// et les injecte dans le contexte Gin pour les middlewares/handlers suivants.
// Si le token est absent, mal formé ou invalide, la requête est stoppée avec un code 401.
func JwtAuthMiddleware() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		// Récupère le header Authorization (doit être de la forme 'Bearer <token>')
		authHeader := ctx.GetHeader("Authorization")
		if authHeader == "" {
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Header d'autorisation manquant"})
			return
		}
		headerParts := strings.Split(authHeader, " ")
		if len(headerParts) != 2 || headerParts[0] != "Bearer" {
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Header d'autorisation mal formaté"})
			return
		}
		tokenString := headerParts[1]
		claims := jwt.MapClaims{}
		// Vérifie et parse le token JWT avec la clé secrète
		token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, errors.New("méthode de signature inattendue")
			}
			return jwtKey, nil
		})
		if err != nil || !token.Valid {
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Token invalide"})
			return
		}
		// Extraction des claims utiles et injection dans le contexte Gin
		if sub, ok := claims["sub"].(string); ok {
			ctx.Set("userID", sub)
		}
		if role, ok := claims["role"].(string); ok {
			ctx.Set("role", role)
		}
		ctx.Next()
	}
}

// RoleMiddleware restreint l'accès à certaines routes selon le rôle de l'utilisateur.
// Vérifie que le claim 'role' du token JWT correspond à l'un des rôles autorisés.
func RoleMiddleware(allowedRoles ...string) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		role, ok := ctx.Get("role")
		if !ok {
			ctx.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "Rôle non trouvé dans le token"})
			return
		}
		roleStr, ok := role.(string)
		if !ok {
			ctx.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "Rôle du token invalide"})
			return
		}
		// Vérifie si le rôle de l'utilisateur fait partie des rôles autorisés
		roleStrLower := strings.ToLower(roleStr)
		for _, allowed := range allowedRoles {
			if roleStrLower == strings.ToLower(allowed) {
				ctx.Next()
				return
			}
		}
		ctx.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "Accès refusé pour ce rôle"})
	}
}

// CheckRole est un middleware qui vérifie si l'utilisateur a un rôle spécifique.
// À utiliser pour les contrôles de rôle supplémentaires au niveau des routes.
func CheckRole(requiredRole string) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		role, ok := ctx.Get("role")
		if !ok {
			ctx.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "Rôle non trouvé"})
			return
		}
		roleStr, ok := role.(string)
		if !ok || !strings.EqualFold(roleStr, requiredRole) {
			ctx.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "Rôle insuffisant"})
			return
		}
		ctx.Next()
	}
}

// GetJwtKey retourne la clé JWT pour les autres packages qui en ont besoin
func GetJwtKey() []byte {
	return jwtKey
}
