package middleware

import (
	"net/http"
	"strings"

	"anomaly-detector/internal/auth"
	"anomaly-detector/internal/store"

	"github.com/gin-gonic/gin"
)

// Clés de contexte utilisées localement dans ce package middleware.
const (
	CtxUserID = "userID"
	CtxRoles  = "roles"
)

// JwtAuthMiddleware vérifie l'Authorization header (Bearer <token>), valide
// le JWT et injecte les claims utiles (`userID`, `roles`) dans le
// contexte Gin. En cas d'erreur la requête est arrêtée avec 401.
func JwtAuthMiddleware(s *store.Store) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		// Récupère l'en-tête Authorization (doit être de la forme 'Bearer <token>')
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

		// Analyse et valide le token en utilisant la clé RSA locale (RS256)
		claims, err := auth.ParseAndValidateToken(tokenString)
		if err != nil {
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Token invalide"})
			return
		}

		// Extraction des claims utiles et injection dans le contexte Gin
		if sub, ok := claims["sub"].(string); ok {
			ctx.Set(CtxUserID, sub)
		}

		// Normaliser les rôles : préférer un tableau `roles`, convertir
		// un claim unique `role` en slice de longueur 1 si nécessaire.
		var roles []string
		if rolesRaw, ok := claims["roles"].([]interface{}); ok {
			roles = make([]string, 0, len(rolesRaw))
			for _, r := range rolesRaw {
				if rs, ok := r.(string); ok {
					roles = append(roles, strings.ToLower(rs))
				}
			}
		} else if roleStr, ok := claims["role"].(string); ok {
			roles = []string{strings.ToLower(roleStr)}
		}
		if len(roles) > 0 {
			ctx.Set(CtxRoles, roles)
		}

		// Vérifier que l'utilisateur existe et est actif (si le store est fourni)
		if sub, ok := claims["sub"].(string); ok {
			if s != nil {
				if usr, err := s.GetUserByIDCached(ctx, sub); err == nil {
					if !usr.Actif {
						ctx.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Identifiants invalides"})
						return
					}
				}
			}
		}
		ctx.Next()
	}
}

// RoleMiddleware restreint l'accès à certaines routes selon le rôle de l'utilisateur.
// Vérifie que le claim 'role' du token JWT correspond à l'un des rôles autorisés.
func RoleMiddleware(allowedRoles ...string) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		// Récupère les rôles normalisés depuis le contexte
		rolesVal, ok := ctx.Get(CtxRoles)
		if !ok {
			ctx.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "Rôle non trouvé dans le token"})
			return
		}
		rolesSlice, ok := rolesVal.([]string)
		if !ok {
			ctx.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "Rôle du token invalide"})
			return
		}
		// build a set for fast lookup
		allowedSet := map[string]struct{}{}
		for _, a := range allowedRoles {
			allowedSet[strings.ToLower(a)] = struct{}{}
		}
		for _, r := range rolesSlice {
			if _, found := allowedSet[strings.ToLower(r)]; found {
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
		rolesVal, ok := ctx.Get(CtxRoles)
		if !ok {
			ctx.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "Rôle non trouvé"})
			return
		}
		rolesSlice, ok := rolesVal.([]string)
		if !ok {
			ctx.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "Rôle du token invalide"})
			return
		}
		// Vérifie que requiredRole est présent dans la slice
		for _, r := range rolesSlice {
			if strings.EqualFold(r, requiredRole) {
				ctx.Next()
				return
			}
		}
		ctx.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "Rôle insuffisant"})
		ctx.Next()
	}
}
