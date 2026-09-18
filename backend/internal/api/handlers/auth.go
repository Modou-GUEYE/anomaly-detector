package handlers

import (
	"anomaly-detector/internal/auth"
	"anomaly-detector/internal/email"
	"anomaly-detector/internal/store"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

const (
	invalidCredentialsMessage = "Identifiants invalides"
	invalidRequestMessage     = "Requête invalide"
	passwordResetSentMessage  = "Un email contenant votre lien de réinitialisation vient de vous être envoyé."
	passwordResetInvalidError = "Lien de réinitialisation invalide ou expiré"
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
		Role     string `json:"role,omitempty"`
	}
	if err := ctx.ShouldBindJSON(&loginReq); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": invalidRequestMessage})
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

// HandleForgotPassword déclenche l'envoi d'un lien de réinitialisation de mot de passe.
func (h *AuthHandler) HandleForgotPassword(ctx *gin.Context) {
	var req struct {
		Email string `json:"email" binding:"required,email"`
	}
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": invalidRequestMessage})
		return
	}

	user, err := h.store.GetUserByEmail(ctx, strings.TrimSpace(req.Email))
	if err != nil || user == nil || !user.Actif {
		ctx.JSON(http.StatusOK, gin.H{"message": passwordResetSentMessage})
		return
	}

	resetToken, err := auth.NewPasswordResetTokenFromEnv(user.ID, user.Email, getPasswordResetTTL())
	if err != nil {
		log.Printf("error signing password reset token for user %s: %v", user.ID, err)
		ctx.JSON(http.StatusOK, gin.H{"message": passwordResetSentMessage})
		return
	}

	resetLink := buildPasswordResetLink(ctx, resetToken)
	if err := email.NewService().SendPasswordResetEmail(user.Email, user.NomUtilisateur, resetLink); err != nil {
		log.Printf("password reset email failed for user %s: %v", user.Email, err)
	}

	ctx.JSON(http.StatusOK, gin.H{"message": passwordResetSentMessage})
}

// HandleResetPassword applique un nouveau mot de passe à partir d'un token de réinitialisation.
func (h *AuthHandler) HandleResetPassword(ctx *gin.Context) {
	var req struct {
		Token       string `json:"token" binding:"required"`
		NewPassword string `json:"newPassword" binding:"required,min=6"`
	}
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": invalidRequestMessage})
		return
	}

	claims, err := auth.ParseAndValidateToken(req.Token)
	if err != nil || !isPasswordResetToken(claims) {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": passwordResetInvalidError})
		return
	}

	userID, ok := claimString(claims, "sub")
	if !ok || userID == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": passwordResetInvalidError})
		return
	}

	user, err := h.store.GetUserByID(ctx, userID)
	if err != nil || user == nil || !user.Actif {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": passwordResetInvalidError})
		return
	}

	hashed, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Erreur lors du hachage du mot de passe"})
		return
	}

	updates := map[string]interface{}{
		"mot_de_passe_hash":         string(hashed),
		"doit_changer_mot_de_passe": false,
	}

	if _, err = h.store.UpdateUser(ctx, userID, updates); err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Impossible de mettre à jour le mot de passe"})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"message": "Mot de passe réinitialisé avec succès"})
}

func getPasswordResetTTL() time.Duration {
	if v := os.Getenv("KEY_JWT_RESET_TTL_MIN"); v != "" && v != "0" {
		if d, err := time.ParseDuration(v + "m"); err == nil {
			return d
		}
	}
	return 15 * time.Minute
}

func buildPasswordResetLink(ctx *gin.Context, token string) string {
	baseURL := strings.TrimSpace(os.Getenv("PASSWORD_RESET_URL"))
	if baseURL == "" && ctx != nil {
		if origin := strings.TrimSpace(ctx.GetHeader("Origin")); origin != "" {
			baseURL = strings.TrimRight(origin, "/") + "/forgot-password"
		} else if referer := strings.TrimSpace(ctx.GetHeader("Referer")); referer != "" {
			if u, err := url.Parse(referer); err == nil && u.Scheme != "" && u.Host != "" {
				baseURL = u.Scheme + "://" + u.Host + "/forgot-password"
			}
		}
	}
	if baseURL == "" {
		baseURL = "http://localhost:4200/forgot-password"
	}

	separator := "?"
	if strings.Contains(baseURL, "?") {
		separator = "&"
	}
	return baseURL + separator + "token=" + url.QueryEscape(token)
}

func isPasswordResetToken(claims jwt.MapClaims) bool {
	purpose, ok := claimString(claims, "purpose")
	return ok && purpose == "password_reset"
}

func claimString(claims jwt.MapClaims, key string) (string, bool) {
	v, ok := claims[key]
	if !ok {
		return "", false
	}
	value, isString := v.(string)
	if !isString {
		return "", false
	}
	return value, true
}

// checkPasswordHash vérifie si un mot de passe correspond à son hash bcrypt
func checkPasswordHash(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}
