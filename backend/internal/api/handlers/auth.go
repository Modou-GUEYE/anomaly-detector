package handlers

import (
	"anomaly-detector/internal/api/middleware"
	"anomaly-detector/internal/store"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

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
	}
	if err := ctx.ShouldBindJSON(&loginReq); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Requête invalide"})
		return
	}
	user, err := h.store.GetUserByEmail(ctx, loginReq.Email)
	if err != nil {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "Identifiants invalides"})
		return
	}
	if !checkPasswordHash(loginReq.Password, user.MotDePasseHash) {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "Identifiants invalides"})
		return
	}
	claims := jwt.MapClaims{
		"sub":  user.ID,
		"role": user.Role,
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString(middleware.GetJwtKey())
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Impossible de générer le token"})
		return
	}
	ctx.JSON(http.StatusOK, gin.H{"token": tokenString})
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
	claims := jwt.MapClaims{
		"sub":  req.SystemID,
		"role": req.Role,
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString(middleware.GetJwtKey())
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Impossible de générer le token"})
		return
	}
	ctx.JSON(http.StatusOK, gin.H{"token": tokenString})
}

// checkPasswordHash vérifie si un mot de passe correspond à son hash bcrypt
func checkPasswordHash(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}
