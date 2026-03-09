package handlers

import (
	"anomaly-detector/internal/auth"
	"anomaly-detector/internal/store"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
)

const invalidCredentialsMessage = "Identifiants invalides"

// AuthHandler gère les opérations d'authentification
type AuthHandler struct {
	store *store.Store
}

// NewAuthHandler crée une nouvelle instance du handler d'authentification
func NewAuthHandler(store *store.Store) *AuthHandler {
	return &AuthHandler{store: store}
}

// @Summary      Connecter un utilisateur
// @Description  Authentifie un utilisateur avec email et mot de passe et renvoie un token JWT.
// @Tags         Authentification
// @Accept       json
// @Produce      json
// @Param        credentials  body      object{email=string,password=string}  true  "Identifiants de l'utilisateur"
// @Success      200          {object}  map[string]string "Renvoie un token JWT"
// @Failure      401          {object}  map[string]string
// @Router       /auth/login [post]
func (h *AuthHandler) HandleLogin(ctx *gin.Context) {
	var loginReq struct {
		Email    string `json:"email"`
		Password string `json:"password"`
		Role     string `json:"role,omitempty"`
	}
	if err := ctx.ShouldBindJSON(&loginReq); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Requête invalide"})
		return
	}
	user, err := h.store.GetUserByEmail(ctx, loginReq.Email)
	if err != nil {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": invalidCredentialsMessage})
		return
	}
	if !checkPasswordHash(loginReq.Password, user.MotDePasseHash) {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": invalidCredentialsMessage})
		return
	}

	// Si un rôle est spécifié dans la requête, vérifier qu'il correspond
	// au rôle de l'utilisateur dans la base.
	if loginReq.Role != "" && loginReq.Role != user.Role {
		ctx.JSON(http.StatusForbidden, gin.H{"error": "Rôle invalide"})
		return
	}

	// Si l'utilisateur est désactivé, ne pas révéler cette information:
	// renvoyer la même réponse que pour des identifiants invalides.
	if !user.Actif {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": invalidCredentialsMessage})
		return
	}
	// Génération du token JWT signé
	accessTTL := time.Duration(0)
	// Détermine la durée de vie via `KEY_JWT_ACCESS_TTL_MIN`.
	if v := os.Getenv("KEY_JWT_ACCESS_TTL_MIN"); v != "" && v != "0" {
		if d, err := time.ParseDuration(v + "m"); err == nil {
			accessTTL = d
		} else {
			log.Printf("invalid KEY_JWT_ACCESS_TTL_MIN: %v", err)
		}
	}

	roles := []string{user.Role}
	accessToken, err := auth.NewSignedTokenFromEnv(user.ID, roles, accessTTL, user.DoitChangerMotDePasse)
	if err != nil {
		log.Printf("error signing access token for user %s: %v", user.ID, err)
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "could not sign token"})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"access_token": accessToken, "expires_in": int(accessTTL.Seconds())})
}

// HandleGetSystemToken génère un token JWT pour un système externe (partenaire)
// Cette fonction est neutre et générique, utilisable par n'importe quel système.
func (h *AuthHandler) HandleGetSystemToken(ctx *gin.Context) {
	var req struct {
		SystemID string `json:"system_id" binding:"required"`
		Role     string `json:"role" binding:"required"`
	}
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Paramètres manquants: system_id et role requis"})
		return
	}

	// Vérification en base du system_id ET du rôle
	allowed, err := h.store.IsSystemIDAllowed(ctx, req.SystemID, req.Role)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Erreur interne lors de la vérification du system_id"})
		return
	}
	if !allowed {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "system_id ou rôle non autorisé"})
		return
	}

	accessTTL := time.Duration(0)
	// Détermine la durée de vie via `KEY_JWT_ACCESS_TTL_MIN` uniquement.
	if v := os.Getenv("KEY_JWT_ACCESS_TTL_MIN"); v != "" && v != "0" {
		if d, err := time.ParseDuration(v + "m"); err == nil {
			accessTTL = d
		} else {
			log.Printf("invalid KEY_JWT_ACCESS_TTL_MIN: %v", err)
		}
	}
	roles := []string{req.Role}
	token, err := auth.NewSignedTokenFromEnv(req.SystemID, roles, accessTTL, false)
	if err != nil {
		log.Printf("error signing system token for system %s: %v", req.SystemID, err)
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "could not sign token"})
		return
	}
	ctx.JSON(http.StatusOK, gin.H{"access_token": token, "expires_in": int(accessTTL.Seconds())})
}

// checkPasswordHash vérifie si un mot de passe correspond à son hash bcrypt
func checkPasswordHash(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}
