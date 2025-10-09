package api

import (
	"anomaly-detector/internal/core"
	"anomaly-detector/internal/models"
	"anomaly-detector/internal/store"
	"errors"
	"net/http"
	"strings"
	"time"

	_ "anomaly-detector/docs"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	"golang.org/x/crypto/bcrypt"
)

// Server sert les requêtes HTTP pour notre service.
type Server struct {
	store  *store.Store
	router *gin.Engine
}

// NewServer crée une nouvelle instance de serveur.
func NewServer(store *store.Store) *Server {
	server := &Server{store: store}
	router := gin.Default()

	// Route pour la documentation Swagger UI
	router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	// --- Routes Publiques ---
	apiV1 := router.Group("/api/v1")
	{
		apiV1.POST("/users", server.handleCreateUser)
		apiV1.POST("/login", server.handleLogin)
	}

	// --- Routes Protégées (pour tous les utilisateurs connectés) ---
	authRoutes := apiV1.Group("/")
	authRoutes.Use(server.authMiddleware())
	{
		authRoutes.GET("/profile", server.handleGetProfile)
		authRoutes.POST("/transactions/analyze", server.handleAnalyzeTransaction)
		authRoutes.GET("/alertes", server.handleGetAlertes)
		authRoutes.PUT("/alertes/:id/status", server.handleUpdateAlerteStatus)
	}

	// --- Routes Admin (protégées par auth ET admin middlewares) ---
	adminRoutes := apiV1.Group("/")
	adminRoutes.Use(server.authMiddleware(), server.adminMiddleware())
	{
		// Blacklist
		adminRoutes.POST("/blacklist", server.handleCreateBlacklistEntry)
		adminRoutes.GET("/blacklist", server.handleGetBlacklistEntries)
		adminRoutes.PUT("/blacklist/:id", server.handleUpdateBlacklistEntry)
		adminRoutes.DELETE("/blacklist/:id", server.handleDeleteBlacklistEntry)

		// Profils Clients
		adminRoutes.POST("/profils", server.handleCreateProfilClient)
		adminRoutes.GET("/profils", server.handleGetProfilsClient)
		adminRoutes.PUT("/profils/:id", server.handleUpdateProfilClient)
		adminRoutes.DELETE("/profils/:id", server.handleDeleteProfilClient)

		// Règles
		adminRoutes.POST("/regles", server.handleCreateRegleControle)
		adminRoutes.GET("/profils/:id/regles", server.handleGetReglesByProfil)
		adminRoutes.PUT("/regles/:id", server.handleUpdateRegleControle)
		adminRoutes.DELETE("/regles/:id", server.handleDeleteRegleControle)

		// Clients
		adminRoutes.POST("/clients", server.handleCreateClient)
	}

	server.router = router
	return server
}

// Start lance le serveur HTTP sur une adresse spécifique.
func (server *Server) Start(address string) error {
	return server.router.Run(address)
}

// hashPassword hache un mot de passe en utilisant l'algorithme bcrypt.
func HashPassword(password string) (string, error) {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), 10)
	if err != nil {
		return "", err
	}
	return string(hashedPassword), nil
}

// checkPasswordHash compare un mot de passe en clair avec son hash bcrypt.
func checkPasswordHash(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

var jwtKey = []byte("ma_super_cle_secrete_qui_doit_etre_longue_et_complexe")

// @Summary      Créer un nouvel utilisateur
// @Description  Crée un nouvel utilisateur (admin ou analyste).
// @Tags         Authentification
// @Accept       json
// @Produce      json
// @Param        utilisateur  body      models.Utilisateur  true  "Le mot de passe doit être dans le champ 'password'"
// @Success      201          {object}  map[string]string
// @Failure      400          {object}  map[string]string
// @Router       /users [post]
func (server *Server) handleCreateUser(ctx *gin.Context) {
	var req models.Utilisateur
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "JSON invalide: " + err.Error()})
		return
	}
	hashedPassword, err := HashPassword(req.MotDePasseHash)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Impossible de sécuriser le mot de passe"})
		return
	}
	user := &models.Utilisateur{
		NomUtilisateur: req.NomUtilisateur,
		Email:          req.Email,
		MotDePasseHash: hashedPassword,
		Role:           req.Role,
		Actif:          true,
	}
	userID, err := server.store.CreateUser(ctx, user)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Impossible de créer l'utilisateur: " + err.Error()})
		return
	}
	ctx.JSON(http.StatusCreated, gin.H{"message": "Utilisateur créé avec succès", "id": userID})
}

// @Summary      Connecter un utilisateur
// @Description  Authentifie un utilisateur avec email et mot de passe et renvoie un token JWT.
// @Tags         Authentification
// @Accept       json
// @Produce      json
// @Param        credentials  body      object{email=string,password=string}  true  "Identifiants de l'utilisateur"
// @Success      200          {object}  map[string]string "Renvoie un token JWT"
// @Failure      401          {object}  map[string]string
// @Router       /login [post]
func (server *Server) handleLogin(ctx *gin.Context) {
	var loginReq struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if err := ctx.ShouldBindJSON(&loginReq); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Requête invalide"})
		return
	}
	user, err := server.store.GetUserByEmail(ctx, loginReq.Email)
	if err != nil {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "Identifiants invalides"})
		return
	}
	if !checkPasswordHash(loginReq.Password, user.MotDePasseHash) {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "Identifiants invalides"})
		return
	}
	expirationTime := time.Now().Add(24 * time.Hour)
	claims := &jwt.RegisteredClaims{
		Subject:   user.ID,
		ExpiresAt: jwt.NewNumericDate(expirationTime),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString(jwtKey)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Impossible de générer le token"})
		return
	}
	ctx.JSON(http.StatusOK, gin.H{"token": tokenString})
}

// @Summary      Obtenir le profil de l'utilisateur connecté
// @Description  Renvoie les informations de l'utilisateur actuellement authentifié via le token JWT.
// @Tags         Utilisateurs
// @Produce      json
// @Success      200  {object}  models.Utilisateur
// @Failure      401  {object}  map[string]string
// @Security     BearerAuth
// @Router       /profile [get]
func (server *Server) handleGetProfile(ctx *gin.Context) {
	userID, exists := ctx.Get("userID")
	if !exists {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "ID utilisateur non trouvé"})
		return
	}
	user, err := server.store.GetUserByID(ctx, userID.(string))
	if err != nil {
		ctx.JSON(http.StatusNotFound, gin.H{"error": "Utilisateur non trouvé"})
		return
	}
	ctx.JSON(http.StatusOK, user)
}

// @Summary      Analyser une transaction (bidirectionnel)
// @Description  Soumet une transaction pour analyse par le MDA, en appliquant les règles à la fois pour le client source et le client destinataire.
// @Tags         Moteur de Détection
// @Accept       json
// @Produce      json
// @Param        transaction  body      models.Transaction  true  "Détails de la transaction à analyser"
// @Success      200          {object}  map[string]string "Aucune anomalie détectée"
// @Success      201          {object}  map[string]interface{} "Anomalie(s) détectée(s) et alerte(s) créée(s)"
// @Failure      403          {object}  map[string]string "Transaction refusée (client interdit)"
// @Security     BearerAuth
// @Router       /transactions/analyze [post]
// dans internal/api/server.go

func (server *Server) handleAnalyzeTransaction(ctx *gin.Context) {
	var tx models.Transaction
	if err := ctx.ShouldBindJSON(&tx); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Format de transaction invalide: " + err.Error()})
		return
	}

	// 1. On enregistre d'abord la transaction pour avoir un ID
	createdTx, err := server.store.CreateTransaction(ctx, &tx)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Impossible d'enregistrer la transaction: " + err.Error()})
		return
	}

	// 2. On appelle le moteur de règles avec la transaction enregistrée
	rulesEngine := core.NewRulesEngine(server.store)
	alertes, err := rulesEngine.AnalyzeTransaction(ctx, createdTx)

	if errors.Is(err, core.ErrTransactionRefused) {
		ctx.JSON(http.StatusForbidden, gin.H{"status": "transaction_refusee", "raison": err.Error()})
		return
	}
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Erreur lors de l'analyse: " + err.Error()})
		return
	}

	// 3. On traite le résultat
	if len(alertes) > 0 {
		var savedAlertes []*models.Alerte
		for _, alerte := range alertes {
			saved, _ := server.store.CreateAlerte(ctx, alerte)
			savedAlertes = append(savedAlertes, saved)
		}
		ctx.JSON(http.StatusCreated, gin.H{"status": "anomalie_detectee", "alertes": savedAlertes})
	} else {
		ctx.JSON(http.StatusOK, gin.H{"status": "aucune_anomalie_detectee"})
	}
}

// @Summary      Lister toutes les alertes
// @Description  Récupère la liste de toutes les alertes générées par le système.
// @Tags         Analystes - Alertes
// @Produce      json
// @Success      200  {array}   models.Alerte
// @Failure      401  {object}  map[string]string
// @Security     BearerAuth
// @Router       /alertes [get]
func (server *Server) handleGetAlertes(ctx *gin.Context) {
	alertes, err := server.store.GetAlertes(ctx)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Impossible de récupérer les alertes"})
		return
	}
	ctx.JSON(http.StatusOK, alertes)
}

// @Summary      Mettre à jour le statut d'une alerte
// @Description  Permet à un analyste de changer le statut d'une alerte (ex: 'Traité', 'Faux positif').
// @Tags         Analystes - Alertes
// @Accept       json
// @Produce      json
// @Param        id      path      string                  true  "ID de l'alerte"
// @Param        status  body      object{statut=string}  true  "Nouveau statut de l'alerte"
// @Success      200     {object}  models.Alerte
// @Failure      401     {object}  map[string]string
// @Security     BearerAuth
// @Router       /alertes/{id}/status [put]
func (server *Server) handleUpdateAlerteStatus(ctx *gin.Context) {
	alerteID := ctx.Param("id")
	var req struct {
		Statut string `json:"statut" binding:"required"`
	}
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Le champ 'statut' est requis"})
		return
	}
	analysteID, _ := ctx.Get("userID")
	alerte, err := server.store.UpdateAlerteStatus(ctx, alerteID, analysteID.(string), req.Statut)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Impossible de mettre à jour l'alerte"})
		return
	}
	ctx.JSON(http.StatusOK, alerte)
}

// @Summary      Créer une entrée dans la blacklist
// @Description  Ajoute une nouvelle entité à la liste de surveillance. Nécessite des droits admin.
// @Tags         Administration - Blacklist
// @Accept       json
// @Produce      json
// @Param        blacklistEntry  body      models.EntiteBlacklist  true  "Détails de l'entité à ajouter"
// @Success      201             {object}  models.EntiteBlacklist
// @Failure      403             {object}  map[string]string
// @Security     BearerAuth
// @Router       /blacklist [post]
func (server *Server) handleCreateBlacklistEntry(ctx *gin.Context) {
	var req models.EntiteBlacklist
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "JSON invalide"})
		return
	}
	entry, err := server.store.CreateBlacklistEntry(ctx, &req)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Impossible de créer l'entrée"})
		return
	}
	ctx.JSON(http.StatusCreated, entry)
}

// @Summary      Lister les entrées de la blacklist
// @Description  Récupère toutes les entités de la liste de surveillance. Nécessite des droits admin.
// @Tags         Administration - Blacklist
// @Produce      json
// @Success      200  {array}   models.EntiteBlacklist
// @Failure      403  {object}  map[string]string
// @Security     BearerAuth
// @Router       /blacklist [get]
func (server *Server) handleGetBlacklistEntries(ctx *gin.Context) {
	entries, err := server.store.GetBlacklistEntries(ctx)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Impossible de récupérer les entrées"})
		return
	}
	ctx.JSON(http.StatusOK, entries)
}

// @Summary      Mettre à jour une entrée de la blacklist
// @Description  Modifie une entité de la liste de surveillance. Nécessite des droits admin.
// @Tags         Administration - Blacklist
// @Accept       json
// @Produce      json
// @Param        id              path      string                  true  "ID de l'entité"
// @Param        blacklistEntry  body      models.EntiteBlacklist  true  "Nouvelles données de l'entité"
// @Success      200             {object}  models.EntiteBlacklist
// @Failure      403             {object}  map[string]string
// @Security     BearerAuth
// @Router       /blacklist/{id} [put]
func (server *Server) handleUpdateBlacklistEntry(ctx *gin.Context) {
	id := ctx.Param("id")
	var req models.EntiteBlacklist
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "JSON invalide"})
		return
	}
	updatedEntry, err := server.store.UpdateBlacklistEntry(ctx, id, &req)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Impossible de mettre à jour l'entrée"})
		return
	}
	ctx.JSON(http.StatusOK, updatedEntry)
}

// @Summary      Supprimer une entrée de la blacklist
// @Description  Supprime une entité de la liste de surveillance par son ID. Nécessite des droits admin.
// @Tags         Administration - Blacklist
// @Produce      json
// @Param        id  path      string  true  "ID de l'entité"
// @Success      204 "No Content"
// @Failure      403 {object}  map[string]string
// @Security     BearerAuth
// @Router       /blacklist/{id} [delete]
func (server *Server) handleDeleteBlacklistEntry(ctx *gin.Context) {
	id := ctx.Param("id")
	if err := server.store.DeleteBlacklistEntry(ctx, id); err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Impossible de supprimer l'entrée"})
		return
	}
	ctx.Status(http.StatusNoContent)
}

// @Summary      Créer un profil client
// @Description  Crée un nouveau profil client. Nécessite des droits admin.
// @Tags         Administration - Profils
// @Accept       json
// @Produce      json
// @Param        profil  body      models.ProfilClient  true  "Détails du profil"
// @Success      201     {object}  models.ProfilClient
// @Failure      403     {object}  map[string]string
// @Security     BearerAuth
// @Router       /profils [post]
func (server *Server) handleCreateProfilClient(ctx *gin.Context) {
	var req models.ProfilClient
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "JSON invalide"})
		return
	}
	profil, err := server.store.CreateProfilClient(ctx, &req)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Impossible de créer le profil"})
		return
	}
	ctx.JSON(http.StatusCreated, profil)
}

// @Summary      Lister les profils clients
// @Description  Récupère tous les profils clients. Nécessite des droits admin.
// @Tags         Administration - Profils
// @Produce      json
// @Success      200 {array}   models.ProfilClient
// @Failure      403 {object}  map[string]string
// @Security     BearerAuth
// @Router       /profils [get]
func (server *Server) handleGetProfilsClient(ctx *gin.Context) {
	profils, err := server.store.GetProfilsClient(ctx)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Impossible de récupérer les profils"})
		return
	}
	ctx.JSON(http.StatusOK, profils)
}

// @Summary      Mettre à jour un profil client
// @Description  Modifie un profil client. Nécessite des droits admin.
// @Tags         Administration - Profils
// @Accept       json
// @Produce      json
// @Param        id      path      string               true  "ID du profil"
// @Param        profil  body      models.ProfilClient  true  "Nouvelles données du profil"
// @Success      200     {object}  models.ProfilClient
// @Failure      403     {object}  map[string]string
// @Security     BearerAuth
// @Router       /profils/{id} [put]
func (server *Server) handleUpdateProfilClient(ctx *gin.Context) {
	id := ctx.Param("id")
	var req models.ProfilClient
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "JSON invalide"})
		return
	}
	profil, err := server.store.UpdateProfilClient(ctx, id, &req)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Impossible de mettre à jour le profil"})
		return
	}
	ctx.JSON(http.StatusOK, profil)
}

// @Summary      Supprimer un profil client
// @Description  Supprime un profil client. Nécessite des droits admin.
// @Tags         Administration - Profils
// @Produce      json
// @Param        id  path      string  true  "ID du profil"
// @Success      204 "No Content"
// @Failure      403 {object}  map[string]string
// @Security     BearerAuth
// @Router       /profils/{id} [delete]
func (server *Server) handleDeleteProfilClient(ctx *gin.Context) {
	id := ctx.Param("id")
	if err := server.store.DeleteProfilClient(ctx, id); err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Impossible de supprimer le profil: " + err.Error()})
		return
	}
	ctx.Status(http.StatusNoContent)
}

// @Summary      Créer une règle de contrôle
// @Description  Crée une nouvelle règle de contrôle. Nécessite des droits admin.
// @Tags         Administration - Règles
// @Accept       json
// @Produce      json
// @Param        regle  body      models.RegleControle  true  "Détails de la règle"
// @Success      201    {object}  models.RegleControle
// @Failure      403    {object}  map[string]string
// @Security     BearerAuth
// @Router       /regles [post]
func (server *Server) handleCreateRegleControle(ctx *gin.Context) {
	var req models.RegleControle
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "JSON invalide"})
		return
	}
	regle, err := server.store.CreateRegleControle(ctx, &req)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Impossible de créer la règle"})
		return
	}
	ctx.JSON(http.StatusCreated, regle)
}

// @Summary      Lister les règles d'un profil
// @Description  Récupère toutes les règles associées à un profil client. Nécessite des droits admin.
// @Tags         Administration - Règles
// @Produce      json
// @Param        id  path      string  true  "ID du Profil Client"
// @Success      200 {array}   models.RegleControle
// @Failure      403 {object}  map[string]string
// @Security     BearerAuth
// @Router       /profils/{id}/regles [get]
func (server *Server) handleGetReglesByProfil(ctx *gin.Context) {
	profilID := ctx.Param("id")
	regles, err := server.store.GetReglesByProfilID(ctx, profilID)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Impossible de récupérer les règles"})
		return
	}
	ctx.JSON(http.StatusOK, regles)
}

// @Summary      Mettre à jour une règle de contrôle
// @Description  Modifie une règle de contrôle. Nécessite des droits admin.
// @Tags         Administration - Règles
// @Accept       json
// @Produce      json
// @Param        id     path      string                true  "ID de la règle"
// @Param        regle  body      models.RegleControle  true  "Nouvelles données de la règle"
// @Success      200    {object}  models.RegleControle
// @Failure      403    {object}  map[string]string
// @Security     BearerAuth
// @Router       /regles/{id} [put]
func (server *Server) handleUpdateRegleControle(ctx *gin.Context) {
	id := ctx.Param("id")
	var req models.RegleControle
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "JSON invalide"})
		return
	}
	regle, err := server.store.UpdateRegleControle(ctx, id, &req)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Impossible de mettre à jour la règle"})
		return
	}
	ctx.JSON(http.StatusOK, regle)
}

// @Summary      Supprimer une règle de contrôle
// @Description  Supprime une règle de contrôle. Nécessite des droits admin.
// @Tags         Administration - Règles
// @Produce      json
// @Param        id  path      string  true  "ID de la règle"
// @Success      204 "No Content"
// @Failure      403 {object}  map[string]string
// @Security     BearerAuth
// @Router       /regles/{id} [delete]
func (server *Server) handleDeleteRegleControle(ctx *gin.Context) {
	id := ctx.Param("id")
	if err := server.store.DeleteRegleControle(ctx, id); err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Impossible de supprimer la règle"})
		return
	}
	ctx.Status(http.StatusNoContent)
}

// @Summary      Créer un client
// @Description  Crée un nouveau client et l'associe à un profil. Nécessite des droits admin.
// @Tags         Administration - Clients
// @Accept       json
// @Produce      json
// @Param        client  body      models.Client  true  "Détails du client"
// @Success      201     {object}  models.Client
// @Failure      403     {object}  map[string]string
// @Security     BearerAuth
// @Router       /clients [post]
func (server *Server) handleCreateClient(ctx *gin.Context) {
	var req models.Client
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "JSON invalide"})
		return
	}
	client, err := server.store.CreateClient(ctx, &req)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Impossible de créer le client"})
		return
	}
	ctx.JSON(http.StatusCreated, client)
}

// Middlewares...
func (server *Server) authMiddleware() gin.HandlerFunc {
	return func(ctx *gin.Context) {
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
		claims := &jwt.RegisteredClaims{}
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
		ctx.Set("userID", claims.Subject)
		ctx.Next()
	}
}

func (server *Server) adminMiddleware() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		userID, exists := ctx.Get("userID")
		if !exists {
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Utilisateur non authentifié"})
			return
		}
		user, err := server.store.GetUserByID(ctx, userID.(string))
		if err != nil {
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Utilisateur non trouvé"})
			return
		}
		if user.Role != "admin" {
			ctx.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "Accès refusé. Droits administrateur requis."})
			return
		}
		ctx.Next()
	}
}
