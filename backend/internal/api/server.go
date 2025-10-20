// ===================== PACKAGE & IMPORTS =====================
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

// ===================== STRUCTURES PRINCIPALES =====================

// Server sert les requêtes HTTP pour notre service.
type Server struct {
	store  *store.Store
	router *gin.Engine
}

// ===================== MIDDLEWARES =====================

// ===================== MIDDLEWARES =====================

// roleMiddleware restreint l'accès à certaines routes selon le rôle de l'utilisateur.
// Il vérifie que le claim 'role' du token JWT correspond à l'un des rôles autorisés passés en argument.
// Si le rôle n'est pas trouvé ou n'est pas autorisé, la requête est stoppée avec un code 403.
// Usage typique : roleMiddleware("admin", "analyste") pour limiter l'accès à ces rôles.
func roleMiddleware(allowedRoles ...string) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		// Récupère le rôle depuis le contexte (injecté par le middleware JWT)
		role, ok := ctx.Get("role")
		if !ok {
			// Aucun rôle trouvé dans le token : accès interdit
			ctx.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "Rôle non trouvé dans le token"})
			return
		}
		roleStr, ok := role.(string)
		if !ok {
			// Le claim 'role' n'est pas une chaîne valide : accès interdit
			ctx.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "Rôle du token invalide"})
			return
		}
		// Vérifie si le rôle de l'utilisateur fait partie des rôles autorisés (insensible à la casse)
		roleStrLower := strings.ToLower(roleStr)
		for _, allowed := range allowedRoles {
			if roleStrLower == strings.ToLower(allowed) {
				ctx.Next()
				return
			}
		}
		// Rôle non autorisé : accès refusé
		ctx.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "Accès refusé pour ce rôle"})
	}
}

// jwtAuthMiddleware authentifie les requêtes via un token JWT passé dans le header Authorization.
// Il vérifie la présence et la validité du token, extrait les claims utiles (userID, role, is_bis)
// et les injecte dans le contexte Gin pour les middlewares/handlers suivants.
// Si le token est absent, mal formé ou invalide, la requête est stoppée avec un code 401.
// Ce middleware doit être appliqué à toutes les routes nécessitant une authentification.
func jwtAuthMiddleware() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		// Récupère le header Authorization (doit être de la forme 'Bearer <token>')
		authHeader := ctx.GetHeader("Authorization")
		if authHeader == "" {
			// Aucun header : accès non autorisé
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Header d'autorisation manquant"})
			return
		}
		headerParts := strings.Split(authHeader, " ")
		if len(headerParts) != 2 || headerParts[0] != "Bearer" {
			// Format incorrect du header : accès non autorisé
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Header d'autorisation mal formaté"})
			return
		}
		tokenString := headerParts[1]
		claims := jwt.MapClaims{}
		// Vérifie et parse le token JWT avec la clé secrète
		token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				// Algorithme de signature inattendu
				return nil, errors.New("méthode de signature inattendue")
			}
			return jwtKey, nil
		})
		if err != nil || !token.Valid {
			// Token invalide ou erreur de parsing : accès non autorisé
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
		if isBis, ok := claims["is_bis"].(bool); ok {
			ctx.Set("is_bis", isBis)
		}
		// Passe au handler suivant si tout est OK
		ctx.Next()
	}
}

// ===================== HANDLERS PUBLICS =====================

// Handler pour générer un token eKYC sans compte utilisateur
func handleGetEkycToken(ctx *gin.Context) {
	claims := jwt.MapClaims{
		"sub":  "ekyc-app",
		"role": "ekyc",
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString(jwtKey)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Impossible de générer le token"})
		return
	}
	ctx.JSON(http.StatusOK, gin.H{"token": tokenString})
}

// Handler pour générer un token BIS
func handleGetBisToken(ctx *gin.Context) {
	claims := jwt.MapClaims{
		"sub":    "bis-app",
		"role":   "bis",
		"is_bis": true,
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString(jwtKey)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Impossible de générer le token"})
		return
	}
	ctx.JSON(http.StatusOK, gin.H{"token": tokenString})
}

// ===================== INITIALISATION DU SERVEUR & ROUTES =====================

// NewServer crée une nouvelle instance de serveur.
func NewServer(store *store.Store) *Server {

	server := &Server{store: store}
	router := gin.Default()

	// Route pour la documentation Swagger UI
	router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	// --- Routes Publiques (tous acteurs : BIS app, BIS web, eKYC) ---
	apiV1 := router.Group("/api/v1")
	{
		apiV1.POST("/users", server.handleCreateUser)
		apiV1.POST("/login", server.handleLogin)
		// Endpoint pour obtenir un token eKYC
		apiV1.POST("/ekyc-token", handleGetEkycToken)
		// Endpoint pour obtenir un token BIS
		apiV1.POST("/bis-token", handleGetBisToken)

		// Endpoint pour vérifier le statut blacklist d'un compte
		apiV1.GET("/blacklist/status/:numeroCompte", func(ctx *gin.Context) {
			numeroCompte := ctx.Param("numeroCompte")
			typeEntite, err := store.GetBlacklistStatusForCompte(ctx, numeroCompte)
			if err != nil {
				ctx.JSON(500, gin.H{"error": "Impossible de récupérer le statut"})
				return
			}
			ctx.JSON(200, gin.H{"typeEntite": typeEntite})
		})
	}

	// --- Route d'analyse transaction (BIS app + BIS web) ---
	// Accessible à bis, admin, analyste
	analyzeRoutes := apiV1.Group("/")
	analyzeRoutes.Use(jwtAuthMiddleware(), roleMiddleware("bis", "admin", "analyste"))
	{
		// POST /transactions/analyze pour BIS app et BIS web
		analyzeRoutes.POST("/transactions/analyze", server.handleAnalyzeTransaction)
		// BIS web: GET /transactions, GET/PUT /alertes, GET /regle/pays_risque/pays, GET /profile
		analyzeRoutes.GET("/transactions", server.handleGetTransactions)
		analyzeRoutes.GET("/alertes", server.handleGetAlertes)
		analyzeRoutes.PUT("/alertes/:id/status", server.handleUpdateAlerteStatus)
		analyzeRoutes.GET("/regle/pays_risque/pays", server.handleGetPaysRisque)
		analyzeRoutes.GET("/profile", server.handleGetProfile)
	}

	// --- Routes Blacklist (BIS web: admin/analyste) ---
	blacklistRoutes := apiV1.Group("/")
	blacklistRoutes.Use(jwtAuthMiddleware(), roleMiddleware("admin", "analyste"))
	{
		// BIS web: gestion complète de la blacklist
		blacklistRoutes.POST("/blacklist", server.handleCreateBlacklistEntry)
		blacklistRoutes.GET("/blacklist", server.handleGetBlacklistEntries)
		blacklistRoutes.PUT("/blacklist/:id", server.handleUpdateBlacklistEntry)
		blacklistRoutes.DELETE("/blacklist/:id", server.handleDeleteBlacklistEntry)
	}

	// --- Routes Admin (BIS web: admin uniquement) ---
	adminRoutes := apiV1.Group("/")
	adminRoutes.Use(jwtAuthMiddleware(), roleMiddleware("admin"))
	{
		// Profils Compte (admin uniquement)
		adminRoutes.POST("/profils", server.handleCreateProfilClient)
		adminRoutes.PUT("/profils/:id", server.handleUpdateProfilClient)
		adminRoutes.DELETE("/profils/:id", server.handleDeleteProfilClient)

		// Règles (admin uniquement)
		adminRoutes.POST("/regles", server.handleCreateRegleControle)
		adminRoutes.PUT("/regles/:id", server.handleUpdateRegleControle)
		adminRoutes.DELETE("/regles/:id", server.handleDeleteRegleControle)

		// Comptes (admin uniquement)
		adminRoutes.POST("/comptes", server.handleCreateCompte)

		// Pays à risque (admin uniquement)
		adminRoutes.PUT("/regle/pays_risque/pays", server.handleUpdatePaysRisque)
		adminRoutes.POST("/regle/pays_risque/pays", server.handleAddPaysRisque)
		adminRoutes.DELETE("/regle/pays_risque/pays", server.handleRemovePaysRisque)

		// Lister toutes les règles
		adminRoutes.GET("/regles", server.handleGetAllRegles)
	}

	// --- Routes Gestion (BIS web: admin + analyste) ---
	gestionRoutes := apiV1.Group("/")
	gestionRoutes.Use(jwtAuthMiddleware(), roleMiddleware("admin", "analyste"))
	{
		// Lecture profils/règles (admin + analyste)
		gestionRoutes.GET("/profils", server.handleGetProfilsClient)
		gestionRoutes.GET("/profils/:id/regles", server.handleGetReglesByProfil)

		// Lecture des règles actives pour un compte donné
		gestionRoutes.GET("/comptes/:numeroCompte/regles", server.handleGetReglesForCompte)

	}

	// --- Route Blacklist Check (eKYC uniquement) ---
	blacklistCheckRoutes := apiV1.Group("/")
	blacklistCheckRoutes.Use(jwtAuthMiddleware(), roleMiddleware("ekyc"))
	{
		blacklistCheckRoutes.POST("/blacklist/eKYCcheck", server.handleCheckBlacklist)
	}

	server.router = router
	return server
}

// ===================== DEMARRAGE SERVEUR =====================

// Start lance le serveur HTTP sur une adresse spécifique.
func (server *Server) Start(address string) error {
	return server.router.Run(address)
}

// ===================== UTILS =====================

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

// ===================== CONSTANTES =====================

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
		DateCreation:   time.Now(),
	}
	userID, err := server.store.CreateUser(ctx, user)
	if err != nil {
		if strings.Contains(err.Error(), "existe déjà") {
			ctx.JSON(http.StatusConflict, gin.H{"error": "Impossible de créer l'utilisateur: " + err.Error()})
		} else {
			ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Impossible de créer l'utilisateur: " + err.Error()})
		}
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
	// BIS web : rôle réel (admin ou analyste), sub = user.ID, pas de is_bis
	// eKYC : rôle ekyc
	claims := jwt.MapClaims{
		"sub":  user.ID,
		"role": user.Role,
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

// @Summary      Lister toutes les transactions
// @Description  Récupère la liste de toutes les transactions enregistrées dans le système.
// @Tags         Transactions
// @Produce      json
// @Success      200  {array}   models.Transaction
// @Failure      401  {object}  map[string]string
// @Security     BearerAuth
// @Router       /transactions [get]
func (server *Server) handleGetTransactions(ctx *gin.Context) {
	transactions, err := server.store.GetTransactions(ctx)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Impossible de récupérer les transactions"})
		return
	}
	ctx.JSON(http.StatusOK, transactions)
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

	// Création automatique du compte payeur si inexistant
	if tx.NumeroComptePayeur != "" {
		clientPayeur, err := server.store.GetCompteByCompte(ctx, tx.NumeroComptePayeur)
		if err != nil || clientPayeur.ID == "" {
			profilID, _ := server.store.GetProfilClientIDByCode(ctx, tx.TypeClientPayeur)
			_, errCreate := server.store.CreateCompte(ctx, &models.Compte{
				NomClient:      tx.NomPayeur,
				NumeroCompte:   tx.NumeroComptePayeur,
				TypeCompte:     tx.TypeComptePayeur,
				TypeClient:     tx.TypeClientPayeur,
				Categorie:      tx.CategoriePayeur,
				Pays:           tx.PaysPayeur,
				Ville:          tx.VillePayeur,
				Adresse:        tx.AdressePayeur,
				ProfilClientID: profilID,
				DateCreation:   time.Now(),
			})
			if errCreate != nil {
				ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Erreur lors de la création du client payeur", "details": errCreate.Error()})
				return
			}
		}
	}
	// Création automatique du compte payé si inexistant
	if tx.NumeroComptePaye != "" {
		clientPaye, err := server.store.GetCompteByCompte(ctx, tx.NumeroComptePaye)
		if err != nil || clientPaye.ID == "" {
			profilID, _ := server.store.GetProfilClientIDByCode(ctx, tx.TypeClientPaye)
			_, errCreate := server.store.CreateCompte(ctx, &models.Compte{
				NomClient:      tx.NomPaye,
				NumeroCompte:   tx.NumeroComptePaye,
				TypeCompte:     tx.TypeComptePaye,
				TypeClient:     tx.TypeClientPaye,
				Categorie:      tx.CategoriePaye,
				Pays:           tx.PaysPaye,
				Ville:          tx.VillePaye,
				Adresse:        tx.AdressePaye,
				ProfilClientID: profilID,
				DateCreation:   time.Now(),
			})
			if errCreate != nil {
				ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Erreur lors de la création du client payé", "details": errCreate.Error()})
				return
			}
		}
	}

	// Code banque partenaire BIS
	const bisCodeBank = "SN079"
	// Vérification si le payeur ou le payé est un compte partenaire BIS
	isPayeurPartenaire := len(tx.NumeroComptePayeur) >= 5 && tx.NumeroComptePayeur[:5] == bisCodeBank
	isPayePartenaire := len(tx.NumeroComptePaye) >= 5 && tx.NumeroComptePaye[:5] == bisCodeBank
	if !isPayeurPartenaire && !isPayePartenaire {
		ctx.JSON(http.StatusOK, gin.H{"status": "transaction_non_analysee", "raison": "Aucun compte partenaire BIS (code banque SN079) impliqué dans la transaction."})
		return
	}

	// Analyse de la transaction
	rulesEngine := core.NewRulesEngine(server.store)
	alertes, err := rulesEngine.AnalyzeTransaction(ctx, &tx)
	if err != nil {
		if errors.Is(err, core.ErrTransactionRefused) {
			msg := err.Error()
			if idx := strings.Index(msg, ": "); idx != -1 {
				msg = msg[idx+2:]
			}
			ctx.JSON(http.StatusForbidden, gin.H{"error": msg})
			return
		}
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	var bloqueAlertes []*models.Alerte
	var autresAlertes []*models.Alerte
	for _, alerte := range alertes {
		if alerte.Action == "bloquer" {
			bloqueAlertes = append(bloqueAlertes, alerte)
		} else {
			autresAlertes = append(autresAlertes, alerte)
		}
	}

	if len(bloqueAlertes) > 0 {
		// Créer la transaction et l'alerte bloquante en base
		createdTx, err := server.store.CreateTransaction(ctx, &tx)
		if err != nil {
			ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Impossible d'enregistrer la transaction: " + err.Error()})
			return
		}
		var savedBloqueAlertes []*models.Alerte
		for _, alerte := range bloqueAlertes {
			alerte.TransactionID = createdTx.ID
			if alerte.DateGeneration.IsZero() {
				alerte.DateGeneration = time.Now()
			}
			saved, _ := server.store.CreateAlerte(ctx, alerte)
			savedBloqueAlertes = append(savedBloqueAlertes, saved)
		}
		ctx.JSON(http.StatusForbidden, gin.H{"status": "transaction_bloquee", "alertes": savedBloqueAlertes})
		return
	}

	// Sinon, création de la transaction et des autres alertes
	createdTx, err := server.store.CreateTransaction(ctx, &tx)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Impossible d'enregistrer la transaction: " + err.Error()})
		return
	}

	var savedAlertes []*models.Alerte
	for _, alerte := range autresAlertes {
		alerte.TransactionID = createdTx.ID
		if alerte.DateGeneration.IsZero() {
			alerte.DateGeneration = time.Now()
		}
		saved, _ := server.store.CreateAlerte(ctx, alerte)
		savedAlertes = append(savedAlertes, saved)
		if alerte.Action == "surveiller" {
			const bisCodeBank = "SN079"
			if alerte.NumeroCompte != "" && len(alerte.NumeroCompte) >= 5 && alerte.NumeroCompte[:5] == bisCodeBank {
				_ = server.store.CreateSuiviBlacklist(ctx, createdTx.ID, alerte.NumeroCompte, "surveiller")
			}
		}
	}

	if len(savedAlertes) > 0 {
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
		// On ajoute err.Error() au message de retour
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Impossible de mettre à jour l'alerte", "details": err.Error()})
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
// @Param        compte  body      models.Compte  true  "Détails du compte"
// @Success      201     {object}  models.Compte
// @Failure      403     {object}  map[string]string
// @Security     BearerAuth
// @Router       /clients [post]
func (server *Server) handleCreateCompte(ctx *gin.Context) {
	var req models.Compte
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "JSON invalide"})
		return
	}
	compte, err := server.store.CreateCompte(ctx, &req)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Impossible de créer le compte"})
		return
	}
	ctx.JSON(http.StatusCreated, compte)
}

// @Summary      Ajouter un pays à la liste des pays à risque
// @Description  Permet à un admin d'ajouter un pays à la liste sans écraser les autres.
// @Tags         Administration - Règles
// @Accept       json
// @Produce      json
// @Param        pays  body      object{pays=string}  true  "Pays à ajouter"
// @Success      200   {object}  map[string]string
// @Failure      400   {object}  map[string]string
// @Failure      500   {object}  map[string]string
// @Security     BearerAuth
// @Router       /regle/pays_risque/pays [post]
func (server *Server) handleAddPaysRisque(ctx *gin.Context) {
	var req struct {
		Pays string `json:"pays" binding:"required"`
	}
	if err := ctx.ShouldBindJSON(&req); err != nil || strings.TrimSpace(req.Pays) == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "JSON invalide ou pays manquant"})
		return
	}
	err := server.store.AddPaysRisque(ctx, req.Pays)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Impossible d'ajouter le pays", "details": err.Error()})
		return
	}
	ctx.JSON(http.StatusOK, gin.H{"message": "Pays ajouté à la liste des pays à risque"})
}

// @Summary      Retirer un pays de la liste des pays à risque
// @Description  Permet à un admin de retirer un pays de la liste sans toucher aux autres.
// @Tags         Administration - Règles
// @Accept       json
// @Produce      json
// @Param        pays  body      object{pays=string}  true  "Pays à retirer"
// @Success      200   {object}  map[string]string
// @Failure      400   {object}  map[string]string
// @Failure      500   {object}  map[string]string
// @Security     BearerAuth
// @Router       /regle/pays_risque/pays [delete]
func (server *Server) handleRemovePaysRisque(ctx *gin.Context) {
	var req struct {
		Pays string `json:"pays" binding:"required"`
	}
	if err := ctx.ShouldBindJSON(&req); err != nil || strings.TrimSpace(req.Pays) == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "JSON invalide ou pays manquant"})
		return
	}
	err := server.store.RemovePaysRisque(ctx, req.Pays)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Impossible de retirer le pays", "details": err.Error()})
		return
	}
	ctx.JSON(http.StatusOK, gin.H{"message": "Pays retiré de la liste des pays à risque"})
}

// @Summary      Mettre à jour la liste des pays à risque
// @Description  Permet à un admin de modifier la liste des pays à risque de la règle universelle.
// @Tags         Administration - Règles
// @Success      200   {object}  map[string]string
// @Failure      400   {object}  map[string]string
// @Failure      500   {object}  map[string]string
// @Security     BearerAuth
// @Router       /regle/pays_risque/pays [put]
func (server *Server) handleUpdatePaysRisque(ctx *gin.Context) {
	var req struct {
		ListePays []string `json:"liste_pays" binding:"required"`
	}
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "JSON invalide ou liste manquante"})
		return
	}
	err := server.store.UpdatePaysRisque(ctx, req.ListePays)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Impossible de mettre à jour la liste des pays à risque", "details": err.Error()})
		return
	}
	ctx.JSON(http.StatusOK, gin.H{"message": "Liste des pays à risque mise à jour avec succès"})
}

// @Summary      Obtenir la liste des pays à risque
// @Description  Récupère la liste actuelle des pays à risque de la règle universelle.
// @Tags         Administration - Règles
// @Produce      json
// @Success      200   {object}  object{liste_pays=[]string}
// @Failure      500   {object}  map[string]string
// @Security     BearerAuth
// @Router       /regle/pays_risque/pays [get]
func (server *Server) handleGetPaysRisque(ctx *gin.Context) {
	regle, err := server.store.GetReglePaysRisqueActive(ctx)
	if err != nil || regle == nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Impossible de récupérer la liste des pays à risque", "details": err.Error()})
		return
	}
	ctx.JSON(http.StatusOK, gin.H{"liste_pays": regle.ListePays})
}

// Handler pour vérifier si un compte est dans la blacklist (pour eKYC)
func (server *Server) handleCheckBlacklist(ctx *gin.Context) {
	var req struct {
		NumeroCompte string `json:"numero_compte" binding:"required"`
	}
	if err := ctx.ShouldBindJSON(&req); err != nil || strings.TrimSpace(req.NumeroCompte) == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Numéro de compte manquant ou invalide"})
		return
	}
	status, err := server.store.IsAccountOnBlacklist(ctx, req.NumeroCompte)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Erreur lors de la vérification", "details": err.Error()})
		return
	}
	// On ne retourne que les statuts
	switch status {
	case "Interdit", "Blacklist", "PEP", "Surveillés":
		ctx.JSON(http.StatusOK, gin.H{"statut": status})
	default:
		ctx.JSON(http.StatusOK, gin.H{"statut": "Aucun"})
	}
}

// Handler pour lister toutes les règles de contrôle
func (server *Server) handleGetAllRegles(ctx *gin.Context) {
	regles, err := server.store.GetAllRegles(ctx)
	if err != nil {
		ctx.JSON(500, gin.H{"error": "Impossible de récupérer les règles"})
		return
	}
	ctx.JSON(200, regles)
}

// @Summary      Lister les règles actives d'un compte
// @Description  Récupère toutes les règles actives associées au profil client d'un compte donné (par numéro de compte). Nécessite des droits admin ou analyste.
// @Tags         Règles
// @Produce      json
// @Param        numeroCompte  path      string  true  "Numéro de compte"
// @Success      200 {array}   models.RegleControle
// @Failure      403 {object}  map[string]string
// @Security     BearerAuth
// @Router       /comptes/{numeroCompte}/regles [get]
func (server *Server) handleGetReglesForCompte(ctx *gin.Context) {
	numeroCompte := ctx.Param("numeroCompte")
	regles, err := server.store.GetReglesForCompte(ctx, numeroCompte)
	if err != nil {
		ctx.JSON(500, gin.H{"error": "Impossible de récupérer les règles pour ce compte", "details": err.Error()})
		return
	}
	ctx.JSON(200, regles)
}
