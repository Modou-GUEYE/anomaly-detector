package routes

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"log"
	"math/big"
	"mime/multipart"
	"os"
	"strings"
	"time"

	"anomaly-detector/internal/api/handlers"
	"anomaly-detector/internal/api/middleware"
	"anomaly-detector/internal/auth"
	"anomaly-detector/internal/email"
	"anomaly-detector/internal/models"
	"anomaly-detector/internal/store"

	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	"github.com/xuri/excelize/v2"
	"golang.org/x/crypto/bcrypt"
)

const (
	invalidDataMessage        = "Données invalides"
	unableToFetchUserMessage  = "Impossible de récupérer l'utilisateur"
	passwordHashErrorMessage  = "Erreur lors du hachage du mot de passe"
	unidentifiedUserMessage   = "Utilisateur non identifié"
	duplicateUserEmailMessage = "Un utilisateur avec cet email existe déjà"
	userDisplayNamePrefix     = "Utilisateur "
)

// AllHandlers regroupe tous les handlers du système
type AllHandlers struct {
	Auth      *handlers.AuthHandler
	Anomalies *handlers.AnomaliesHandler
	Entities  *handlers.EntitiesHandler
	Rules     *handlers.RulesHandler
}

// NewAllHandlers crée toutes les instances de handlers
func NewAllHandlers(store *store.Store) *AllHandlers {
	return &AllHandlers{
		Auth:      handlers.NewAuthHandler(store),
		Anomalies: handlers.NewAnomaliesHandler(store),
		Entities:  handlers.NewEntitiesHandler(store),
		Rules:     handlers.NewRulesHandler(store),
	}
}

// SetupRoutes configure toutes les routes de l'API
func SetupRoutes(router *gin.Engine, handlers *AllHandlers, store *store.Store) {
	// Mesure du temps de traitement serveur (en-tête X-Response-Time)
	router.Use(middleware.ResponseTimeMiddleware())

	// Configuration du middleware CORS
	router.Use(middleware.CorsMiddleware())

	// Route pour la documentation Swagger UI
	router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	// --- Routes API v1 (STANDARD ET NEUTRE) ---
	apiV1 := router.Group("/api/v1")

	// Endpoint runtime config utilisé par le frontend
	// Retourne uniquement les éléments nécessaires côté client :
	// - le nom du rôle système (KEY_ROLE_SYSTEM)
	// - la TTL des tokens d'accès (KEY_JWT_ACCESS_TTL_MIN)
	apiV1.GET("/config", func(ctx *gin.Context) {
		systemRole := os.Getenv("KEY_ROLE_SYSTEM")
		if systemRole == "" {
			systemRole = "system"
		}
		jwtTtl := os.Getenv("KEY_JWT_ACCESS_TTL_MIN")
		// renvoyer tel quel (chaine vide signifie pas d'exp si KEY_JWT_ACCESS_TTL_MIN=="0" ou vide)
		ctx.JSON(200, gin.H{
			"system":      systemRole,
			"jwt_ttl_min": jwtTtl,
		})
	})

	// JWKS endpoint exposant la clé publique pour la validation des tokens RS256
	// Permet au frontend / clients de récupérer la clé publique (JWK) via
	// /.well-known/jwks.json
	apiV1.GET("/.well-known/jwks.json", func(ctx *gin.Context) {
		jwks, err := auth.GetJwks()
		if err != nil {
			// Loguer l'erreur détaillée pour faciliter le diagnostic en production
			log.Printf("jwks error: %v", err)
			ctx.JSON(500, gin.H{"error": "jwks unavailable"})
			return
		}
		ctx.JSON(200, jwks)
	})

	// Configuration des routes par domaine
	setupAuthRoutes(apiV1, handlers.Auth, store)
	setupAnomaliesRoutes(apiV1, handlers.Anomalies, store)
	setupEntitiesRoutes(apiV1, handlers.Entities, store)
	setupRulesRoutes(apiV1, handlers.Rules, store)
	setupDashboardRoutes(apiV1, store)
	setupProfilesRoutes(apiV1, store)
	setupAccountsRoutes(apiV1, store)
	setupSuiviRoutes(apiV1, store)
	setupPaysRisqueRoutes(apiV1, store)
	setupTransactionsRoutes(apiV1, store)
	setupUserRoutes(apiV1, store)
	setupReportsRoutes(apiV1, store)
}

// setupAuthRoutes configure les routes d'authentification
func setupAuthRoutes(apiV1 *gin.RouterGroup, authHandler *handlers.AuthHandler, store *store.Store) {
	// ========== AUTHENTIFICATION (Public) ==========
	authRoutes := apiV1.Group("/auth")
	{
		authRoutes.POST("/login", authHandler.HandleLogin)
		authRoutes.POST("/system-token", authHandler.HandleGetSystemToken)
		authRoutes.POST("/forgot-password", authHandler.HandleForgotPassword)
		authRoutes.POST("/reset-password", authHandler.HandleResetPassword)
		authRoutes.POST("/first-password-change", middleware.JwtAuthMiddleware(store), firstPasswordChangeHandler(store))
	}
}

type firstPasswordChangeRequest struct {
	OldPassword string `json:"oldPassword" binding:"required"`
	NewPassword string `json:"newPassword" binding:"required,min=6"`
}

func firstPasswordChangeHandler(store *store.Store) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		userID, ok := userIDFromContext(ctx)
		if !ok {
			ctx.JSON(500, gin.H{"error": unidentifiedUserMessage})
			return
		}

		req, ok := bindFirstPasswordChangeRequest(ctx)
		if !ok {
			return
		}

		user, ok := fetchUserByID(ctx, store, userID)
		if !ok {
			return
		}

		if !verifyOldPassword(ctx, user.MotDePasseHash, req.OldPassword) {
			return
		}

		if !updateFirstPassword(ctx, store, userID, req.NewPassword) {
			return
		}

		newToken, accessTTL, ok := issueFirstPasswordChangedToken(ctx, user)
		if !ok {
			return
		}

		ctx.JSON(200, gin.H{
			"message":      "Mot de passe changé avec succès",
			"access_token": newToken,
			"expires_in":   int(accessTTL.Seconds()),
		})
	}
}

func userIDFromContext(ctx *gin.Context) (string, bool) {
	userID, exists := ctx.Get("userID")
	if !exists {
		return "", false
	}
	value, ok := userID.(string)
	if !ok || value == "" {
		return "", false
	}
	return value, true
}

func bindFirstPasswordChangeRequest(ctx *gin.Context) (*firstPasswordChangeRequest, bool) {
	var req firstPasswordChangeRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(400, gin.H{"error": invalidDataMessage, "details": err.Error()})
		return nil, false
	}
	return &req, true
}

func fetchUserByID(ctx *gin.Context, st *store.Store, userID string) (*models.Utilisateur, bool) {
	user, err := st.GetUserByID(ctx, userID)
	if err != nil {
		ctx.JSON(500, gin.H{"error": unableToFetchUserMessage})
		return nil, false
	}
	return user, true
}

func verifyOldPassword(ctx *gin.Context, hash string, oldPassword string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(oldPassword))
	if err != nil {
		ctx.JSON(401, gin.H{"error": "L'ancien mot de passe est incorrect"})
		return false
	}
	return true
}

func updateFirstPassword(ctx *gin.Context, st *store.Store, userID string, newPassword string) bool {
	hashed, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		ctx.JSON(500, gin.H{"error": passwordHashErrorMessage})
		return false
	}

	updates := map[string]interface{}{
		"mot_de_passe_hash":         string(hashed),
		"doit_changer_mot_de_passe": false,
	}

	if _, err = st.UpdateUser(ctx, userID, updates); err != nil {
		ctx.JSON(500, gin.H{"error": "Impossible de mettre à jour le mot de passe"})
		return false
	}
	return true
}

func issueFirstPasswordChangedToken(ctx *gin.Context, user *models.Utilisateur) (string, time.Duration, bool) {
	accessTTL := time.Duration(0)
	if v := os.Getenv("KEY_JWT_ACCESS_TTL_MIN"); v != "" && v != "0" {
		if d, parseErr := time.ParseDuration(v + "m"); parseErr == nil {
			accessTTL = d
		}
	}

	roles := []string{user.Role}
	newToken, err := auth.NewSignedTokenFromEnv(user.ID, roles, accessTTL, false)
	if err != nil {
		log.Printf("error signing new token for user %s: %v", user.ID, err)
		ctx.JSON(500, gin.H{"error": "Mot de passe changé mais impossible de générer un nouveau token"})
		return "", 0, false
	}

	return newToken, accessTTL, true
}

// setupAnomaliesRoutes configure les routes d'anomalies et transactions
func setupAnomaliesRoutes(apiV1 *gin.RouterGroup, anomaliesHandler *handlers.AnomaliesHandler, store *store.Store) {
	// ========== ANOMALIES / TRANSACTIONS (Protégé) ==========
	anomalyRoutes := apiV1.Group("/anomalies")
	anomalyRoutes.Use(middleware.JwtAuthMiddleware(store), middleware.RoleMiddleware("admin", "analyst", "system"))
	{
		anomalyRoutes.POST("/analyze", anomaliesHandler.HandleAnalyzeTransaction)
		anomalyRoutes.GET("", anomaliesHandler.HandleGetAlertes)
		anomalyRoutes.GET("/metrics", anomaliesHandler.HandleGetAlertMetrics)
		anomalyRoutes.GET("/:id", anomaliesHandler.HandleGetAlertes)
		anomalyRoutes.PUT("/:id/status", anomaliesHandler.HandleUpdateAlerteStatus)
	}
}

// setupEntitiesRoutes configure les routes d'entités et blacklist
func setupEntitiesRoutes(apiV1 *gin.RouterGroup, entitiesHandler *handlers.EntitiesHandler, store *store.Store) {
	// ========== ENTITÉS / BLACKLIST (Vérification publique) ==========
	entityPublicRoutes := apiV1.Group("/entities")
	{
		entityPublicRoutes.GET("/:identifier/status", func(ctx *gin.Context) {
			identifier := ctx.Param("identifier")
			typeEntite, err := store.GetBlacklistStatusForCompte(ctx, identifier)
			if err != nil {
				ctx.JSON(500, gin.H{"error": "Impossible de récupérer le statut"})
				return
			}
			if typeEntite == "" {
				typeEntite = "clean"
			}
			ctx.JSON(200, gin.H{"status": typeEntite})
		})
	}

	// ========== ENTITÉS / BLACKLIST (Protégé) ==========
	entityRoutes := apiV1.Group("/entities")
	entityRoutes.Use(middleware.JwtAuthMiddleware(store), middleware.RoleMiddleware("admin", "analyst"))
	{
		entityRoutes.GET("", entitiesHandler.HandleGetBlacklistEntries)
		entityRoutes.POST("", entitiesHandler.HandleCreateBlacklistEntry)
		entityRoutes.PUT("/:id", entitiesHandler.HandleUpdateBlacklistEntry)
		entityRoutes.DELETE("/:id", entitiesHandler.HandleDeleteBlacklistEntry)
		entityRoutes.POST("/:identifier/verify", entitiesHandler.HandleCheckBlacklist)
	}
}

// setupRulesRoutes configure les routes de règles de contrôle
func setupRulesRoutes(apiV1 *gin.RouterGroup, rulesHandler *handlers.RulesHandler, store *store.Store) {
	// ========== RÈGLES DE CONTRÔLE (Protégé) ==========
	rulesRoutes := apiV1.Group("/rules")
	rulesRoutes.Use(middleware.JwtAuthMiddleware(store), middleware.RoleMiddleware("admin", "analyst"))
	{
		rulesRoutes.GET("", rulesHandler.HandleGetAllRegles)
		rulesRoutes.GET("/:id/profiles", rulesHandler.HandleGetReglesByProfil)
		rulesRoutes.GET("/:id", rulesHandler.HandleGetRegleByID)
		rulesRoutes.POST("", middleware.CheckRole("admin"), rulesHandler.HandleCreateRegleControle)
		rulesRoutes.PUT("/:id", middleware.CheckRole("admin"), rulesHandler.HandleUpdateRegleControle)
		rulesRoutes.PATCH("/:id/status", middleware.CheckRole("admin"), rulesHandler.HandleToggleRegleStatus)
		rulesRoutes.DELETE("/:id", middleware.CheckRole("admin"), rulesHandler.HandleDeleteRegleControle)
	}
}

// setupDashboardRoutes configure les routes du tableau de bord
func setupDashboardRoutes(apiV1 *gin.RouterGroup, store *store.Store) {
	dashboardRoutes := apiV1.Group("/dashboard")
	dashboardRoutes.Use(middleware.JwtAuthMiddleware(store), middleware.RoleMiddleware("admin", "analyst", "system"))
	{
		dashboardRoutes.GET("/summary", func(ctx *gin.Context) {
			dateDebut := ctx.Query("dateDebut")
			dateFin := ctx.Query("dateFin")

			summary, err := store.GetDashboardSummary(ctx, dateDebut, dateFin)
			if err != nil {
				ctx.JSON(500, gin.H{"error": "Impossible de récupérer le résumé"})
				return
			}
			ctx.JSON(200, summary)
		})
	}
}

// setupProfilesRoutes configure les routes des profils clients
func setupProfilesRoutes(apiV1 *gin.RouterGroup, store *store.Store) {
	// ========== PROFILS CLIENTS (Protégé) ==========
	profileRoutes := apiV1.Group("/profiles")
	profileRoutes.Use(middleware.JwtAuthMiddleware(store), middleware.RoleMiddleware("admin", "analyst"))
	{
		// GET - Récupérer tous les profils clients
		profileRoutes.GET("", func(ctx *gin.Context) {
			profils, err := store.GetProfilsClient(ctx)
			if err != nil {
				ctx.JSON(500, gin.H{"error": "Impossible de récupérer les profils clients"})
				return
			}
			ctx.JSON(200, profils)
		})
	}
}

// setupAccountsRoutes configure les routes des comptes clients
func setupAccountsRoutes(apiV1 *gin.RouterGroup, store *store.Store) {
	// ========== COMPTES CLIENTS (Protégé) ==========
	accountRoutes := apiV1.Group("/accounts")
	accountRoutes.Use(middleware.JwtAuthMiddleware(store), middleware.RoleMiddleware("admin", "analyst"))
	{
		accountRoutes.GET("", func(ctx *gin.Context) {
			comptes, err := store.GetAllComptes(ctx)
			if err != nil {
				ctx.JSON(500, gin.H{"error": "Impossible de récupérer les comptes"})
				return
			}
			ctx.JSON(200, comptes)
		})
		accountRoutes.GET("/monitored", func(ctx *gin.Context) {
			count, err := store.GetComptesSurveillesCount(ctx)
			if err != nil {
				ctx.JSON(500, gin.H{"error": "Impossible de récupérer le nombre de comptes surveillés"})
				return
			}
			ctx.JSON(200, gin.H{"count": count})
		})
	}
}

// setupSuiviRoutes configure les routes de surveillance/suivi
func setupSuiviRoutes(apiV1 *gin.RouterGroup, store *store.Store) {
	// ========== SURVEILLANCE - SUIVI BLACKLIST (Protégé) ==========
	suiviRoutes := apiV1.Group("/suivi-blacklist")
	suiviRoutes.Use(middleware.JwtAuthMiddleware(store), middleware.RoleMiddleware("admin", "analyst"))
	{
		suiviRoutes.GET("", func(ctx *gin.Context) {
			suivis, err := store.GetSuiviBlacklist(ctx)
			if err != nil {
				ctx.JSON(500, gin.H{"error": "Impossible de récupérer les suivis"})
				return
			}
			ctx.JSON(200, suivis)
		})
	}
}

// setupPaysRisqueRoutes configure les routes des pays à risque
func setupPaysRisqueRoutes(apiV1 *gin.RouterGroup, store *store.Store) {
	// ========== PAYS À RISQUE (Protégé) ==========
	paysRisqueRoutes := apiV1.Group("/pays-risque")
	paysRisqueRoutes.Use(middleware.JwtAuthMiddleware(store), middleware.RoleMiddleware("admin", "analyst"))
	{
		paysRisqueRoutes.GET("", paysRisqueListHandler(store))
		paysRisqueRoutes.POST("", paysRisqueCreateHandler(store))
		paysRisqueRoutes.PUT("/:ancien_nom", paysRisqueUpdateHandler(store))
		paysRisqueRoutes.DELETE("/:nom_pays", paysRisqueDeleteHandler(store))
		paysRisqueRoutes.POST("/import", paysRisqueImportHandler(store))
	}
}

type paysImportData struct {
	NomPays string
	Code    string
}

func requireAuthenticatedUserID(ctx *gin.Context) (string, bool) {
	userID, ok := userIDFromContext(ctx)
	if !ok {
		ctx.JSON(500, gin.H{"error": unidentifiedUserMessage})
		return "", false
	}
	return userID, true
}

func normalizeISOCountryCode(ctx *gin.Context, code string) (string, bool) {
	normalized := strings.ToUpper(strings.TrimSpace(code))
	if len(normalized) != 2 {
		ctx.JSON(400, gin.H{"error": "Le code pays doit contenir exactement 2 caractères (ISO alpha-2)"})
		return "", false
	}
	return normalized, true
}

func paysRisqueListHandler(st *store.Store) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		pays, err := st.GetAllPaysARisque(ctx)
		if err != nil {
			log.Printf("[ERROR] GET /api/v1/pays-risque -> GetAllPaysARisque: %v", err)
			ctx.JSON(500, gin.H{"error": "Impossible de récupérer les pays à risque"})
			return
		}
		ctx.JSON(200, pays)
	}
}

func paysRisqueCreateHandler(st *store.Store) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		var request struct {
			NomPays string `json:"nom_pays" binding:"required"`
			Code    string `json:"code" binding:"required"`
		}
		if err := ctx.ShouldBindJSON(&request); err != nil {
			ctx.JSON(400, gin.H{"error": "Nom du pays et code ISO alpha-2 requis"})
			return
		}

		code, ok := normalizeISOCountryCode(ctx, request.Code)
		if !ok {
			return
		}

		userID, ok := requireAuthenticatedUserID(ctx)
		if !ok {
			return
		}

		pays := &models.PaysRisque{NomPays: request.NomPays, Code: code, Active: true}
		if err := st.AddPaysARisqueWithCode(ctx, pays, userID); err != nil {
			ctx.JSON(500, gin.H{"error": "Impossible d'ajouter le pays à risque"})
			return
		}
		ctx.JSON(201, gin.H{"message": "Pays ajouté avec succès"})
	}
}

func paysRisqueUpdateHandler(st *store.Store) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		ancienNom := ctx.Param("ancien_nom")
		var request struct {
			NomPays    string `json:"nom_pays" binding:"required"`
			NouveauNom string `json:"nouveau_nom"`
			Code       string `json:"code" binding:"required"`
		}
		if err := ctx.ShouldBindJSON(&request); err != nil {
			ctx.JSON(400, gin.H{"error": "Nom du pays et code ISO alpha-2 requis", "details": err.Error()})
			return
		}

		code, ok := normalizeISOCountryCode(ctx, request.Code)
		if !ok {
			return
		}

		nouveauNom := request.NomPays
		if request.NouveauNom != "" {
			nouveauNom = request.NouveauNom
		}

		userID, ok := requireAuthenticatedUserID(ctx)
		if !ok {
			return
		}

		if err := st.UpdatePaysARisqueWithCode(ctx, ancienNom, nouveauNom, code, userID); err != nil {
			ctx.JSON(500, gin.H{"error": "Impossible de modifier le pays à risque"})
			return
		}
		ctx.JSON(200, gin.H{"message": "Pays modifié avec succès"})
	}
}

func paysRisqueDeleteHandler(st *store.Store) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		nomPays := ctx.Param("nom_pays")
		userID, ok := requireAuthenticatedUserID(ctx)
		if !ok {
			return
		}
		if err := st.DeletePaysARisque(ctx, nomPays, userID); err != nil {
			ctx.JSON(500, gin.H{"error": "Impossible de supprimer le pays à risque"})
			return
		}
		ctx.JSON(200, gin.H{"message": "Pays supprimé avec succès"})
	}
}

func paysRisqueImportHandler(st *store.Store) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		file, err := ctx.FormFile("file")
		if err != nil {
			ctx.JSON(400, gin.H{"error": "Fichier manquant"})
			return
		}

		userID, ok := requireAuthenticatedUserID(ctx)
		if !ok {
			return
		}

		paysToImport, status, message := parsePaysImportFile(file)
		if status != 0 {
			ctx.JSON(status, gin.H{"error": message})
			return
		}
		if len(paysToImport) == 0 {
			ctx.JSON(400, gin.H{"error": "Aucun pays trouvé dans le fichier. Assurez-vous que le fichier contient des données au format: nom_pays,code"})
			return
		}

		imported, errors := importPaysRisqueData(ctx, st, userID, paysToImport)
		ctx.JSON(200, gin.H{
			"message":  fmt.Sprintf("%d/%d pays importés avec succès", imported, len(paysToImport)),
			"imported": imported,
			"total":    len(paysToImport),
			"errors":   errors,
		})
	}
}

func parsePaysImportFile(file *multipart.FileHeader) ([]paysImportData, int, string) {
	filename := strings.ToLower(file.Filename)
	switch {
	case strings.HasSuffix(filename, ".xlsx") || strings.HasSuffix(filename, ".xls"):
		items, err := parsePaysFromExcel(file)
		if err != nil {
			return nil, 400, err.Error()
		}
		return items, 0, ""
	case strings.HasSuffix(filename, ".json"):
		items, err := parsePaysFromJSON(file)
		if err != nil {
			return nil, 400, err.Error()
		}
		return items, 0, ""
	case strings.HasSuffix(filename, ".csv") || strings.HasSuffix(filename, ".txt"):
		items, err := parsePaysFromCSV(file)
		if err != nil {
			return nil, 400, err.Error()
		}
		return items, 0, ""
	default:
		return nil, 400, "Format de fichier non supporté. Utilisez CSV, JSON ou Excel (.xlsx, .xls)"
	}
}

func parsePaysFromExcel(file *multipart.FileHeader) ([]paysImportData, error) {
	src, err := file.Open()
	if err != nil {
		return nil, fmt.Errorf("Impossible d'ouvrir le fichier Excel")
	}
	defer src.Close()

	xlFile, err := excelize.OpenReader(src)
	if err != nil {
		return nil, fmt.Errorf("Erreur lecture Excel: %v", err)
	}
	defer xlFile.Close()

	sheets := xlFile.GetSheetList()
	if len(sheets) == 0 {
		return nil, fmt.Errorf("Le fichier Excel ne contient aucune feuille")
	}

	rows, err := xlFile.GetRows(sheets[0])
	if err != nil {
		return nil, fmt.Errorf("Erreur lecture des lignes: %v", err)
	}

	result := make([]paysImportData, 0, len(rows))
	for i, row := range rows {
		if shouldSkipPaysHeaderRow(i, row) {
			continue
		}
		if len(row) < 2 {
			continue
		}
		nomPays := strings.TrimSpace(row[0])
		code := strings.TrimSpace(row[1])
		if nomPays != "" && code != "" {
			result = append(result, paysImportData{NomPays: nomPays, Code: code})
		}
	}
	return result, nil
}

func parsePaysFromJSON(file *multipart.FileHeader) ([]paysImportData, error) {
	content, err := readUploadedFileContent(file)
	if err != nil {
		return nil, err
	}

	var camelCase []struct {
		NomPays string `json:"nomPays"`
		Code    string `json:"code"`
	}
	if err := json.Unmarshal(content, &camelCase); err == nil {
		return normalizeJSONPaysCamel(camelCase), nil
	}

	var snakeCase []struct {
		NomPays string `json:"nom_pays"`
		Code    string `json:"code"`
	}
	if err := json.Unmarshal(content, &snakeCase); err == nil {
		return normalizeJSONPaysSnake(snakeCase), nil
	}

	return nil, fmt.Errorf("Format JSON invalide. Utilisez [{\"nomPays\":\"France\",\"code\":\"FR\"}, ...] ou [{\"nom_pays\":\"France\",\"code\":\"FR\"}, ...]")
}

func parsePaysFromCSV(file *multipart.FileHeader) ([]paysImportData, error) {
	content, err := readUploadedFileContent(file)
	if err != nil {
		return nil, err
	}

	lines := strings.Split(string(content), "\n")
	result := make([]paysImportData, 0, len(lines))
	hasHeader := false
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		if isCSVPaysHeader(i, trimmed) {
			hasHeader = true
			continue
		}
		parts := strings.Split(trimmed, ",")
		if len(parts) >= 2 {
			nomPays := strings.TrimSpace(parts[0])
			code := strings.TrimSpace(parts[1])
			if nomPays != "" && code != "" {
				result = append(result, paysImportData{NomPays: nomPays, Code: code})
			}
			continue
		}
		if len(parts) == 1 && !hasHeader {
			return nil, fmt.Errorf("Le fichier CSV doit contenir deux colonnes: nom_pays,code (ISO alpha-2). Exemple: France,FR")
		}
	}
	return result, nil
}

func importPaysRisqueData(ctx *gin.Context, st *store.Store, userID string, paysToImport []paysImportData) (int, []string) {
	imported := 0
	var errors []string
	for _, paysData := range paysToImport {
		pays := &models.PaysRisque{NomPays: paysData.NomPays, Code: paysData.Code, Active: true}
		if err := st.AddPaysARisqueWithCode(ctx, pays, userID); err != nil {
			errors = append(errors, fmt.Sprintf("Erreur pour '%s' (%s): %v", paysData.NomPays, paysData.Code, err))
			continue
		}
		imported++
	}
	return imported, errors
}

func readUploadedFileContent(file *multipart.FileHeader) ([]byte, error) {
	src, err := file.Open()
	if err != nil {
		return nil, fmt.Errorf("Impossible d'ouvrir le fichier")
	}
	defer src.Close()

	content := make([]byte, file.Size)
	if _, err := src.Read(content); err != nil {
		return nil, fmt.Errorf("Impossible de lire le fichier")
	}
	return content, nil
}

func shouldSkipPaysHeaderRow(index int, row []string) bool {
	if index != 0 || len(row) < 1 {
		return false
	}
	firstCell := strings.ToLower(row[0])
	return strings.Contains(firstCell, "nom") || strings.Contains(firstCell, "pays")
}

func isCSVPaysHeader(index int, line string) bool {
	if index != 0 {
		return false
	}
	lower := strings.ToLower(line)
	return strings.Contains(lower, "nom") || strings.Contains(lower, "pays") || strings.Contains(lower, "code")
}

func normalizeJSONPaysCamel(items []struct {
	NomPays string `json:"nomPays"`
	Code    string `json:"code"`
}) []paysImportData {
	result := make([]paysImportData, 0, len(items))
	for _, item := range items {
		if item.NomPays != "" && item.Code != "" {
			result = append(result, paysImportData{NomPays: item.NomPays, Code: item.Code})
		}
	}
	return result
}

func normalizeJSONPaysSnake(items []struct {
	NomPays string `json:"nom_pays"`
	Code    string `json:"code"`
}) []paysImportData {
	result := make([]paysImportData, 0, len(items))
	for _, item := range items {
		if item.NomPays != "" && item.Code != "" {
			result = append(result, paysImportData{NomPays: item.NomPays, Code: item.Code})
		}
	}
	return result
}

// setupTransactionsRoutes configure les routes des transactions
func setupTransactionsRoutes(apiV1 *gin.RouterGroup, store *store.Store) {
	// ========== TRANSACTIONS (Protégé) ==========
	transactionRoutes := apiV1.Group("/transactions")
	transactionRoutes.Use(middleware.JwtAuthMiddleware(store), middleware.RoleMiddleware("admin", "analyst"))
	{
		transactionRoutes.GET("", func(ctx *gin.Context) {
			transactions, err := store.GetTransactions(ctx)
			if err != nil {
				ctx.JSON(500, gin.H{"error": "Impossible de récupérer les transactions"})
				return
			}
			ctx.JSON(200, transactions)
		})
	}
}

// setupUserRoutes configure les routes utilisateur/profil
func setupUserRoutes(apiV1 *gin.RouterGroup, store *store.Store) {
	// ========== PROFIL UTILISATEUR (Protégé) ==========
	userRoutes := apiV1.Group("/profile")
	userRoutes.Use(middleware.JwtAuthMiddleware(store))
	{
		userRoutes.GET("", profileGetHandler(store))
		userRoutes.PUT("", profileUpdateHandler(store))
	}

	// ========== UTILISATEURS (Protégé) ==========
	usersRoutes := apiV1.Group("/users")
	usersRoutes.Use(middleware.JwtAuthMiddleware(store), middleware.RoleMiddleware("admin"))
	{
		usersRoutes.POST("", usersCreateHandler(store))
		usersRoutes.GET("", usersListHandler(store))
		usersRoutes.PUT("/:id", usersUpdateHandler(store))
		usersRoutes.DELETE("/:id", usersDeleteHandler(store))
		usersRoutes.GET("/:id", usersGetByIDHandler(store))
	}
}

type profileUpdateRequest struct {
	NomUtilisateur *string `json:"nomUtilisateur" binding:"omitempty"`
	Email          *string `json:"email" binding:"omitempty,email"`
	Password       *string `json:"password" binding:"omitempty,min=6"`
	OldPassword    *string `json:"oldPassword" binding:"omitempty"`
}

type userCreateRequest struct {
	NomUtilisateur string `json:"nomUtilisateur" binding:"required"`
	Email          string `json:"email" binding:"required,email"`
	Role           string `json:"role" binding:"required,oneof=admin analyst"`
}

type userUpdateRequest struct {
	NomUtilisateur        *string `json:"nomUtilisateur"`
	Email                 *string `json:"email"`
	Role                  *string `json:"role"`
	Actif                 *bool   `json:"actif"`
	DoitChangerMotDePasse *bool   `json:"doitChangerMotDePasse"`
}

func profileGetHandler(st *store.Store) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		userID, ok := requireAuthenticatedUserID(ctx)
		if !ok {
			return
		}
		user, err := st.GetUserByID(ctx, userID)
		if err != nil {
			ctx.JSON(500, gin.H{"error": "Impossible de récupérer le profil"})
			return
		}
		ctx.JSON(200, user)
	}
}

func profileUpdateHandler(st *store.Store) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		userID, ok := requireAuthenticatedUserID(ctx)
		if !ok {
			return
		}

		var req profileUpdateRequest
		if err := ctx.ShouldBindJSON(&req); err != nil {
			ctx.JSON(400, gin.H{"error": invalidDataMessage, "details": err.Error()})
			return
		}

		updates, ok := buildProfileUpdates(ctx, st, userID, req)
		if !ok {
			return
		}
		if len(updates) == 0 {
			ctx.JSON(400, gin.H{"error": "Aucune donnée à mettre à jour"})
			return
		}

		updated, err := st.UpdateUser(ctx, userID, updates)
		if err != nil {
			ctx.JSON(500, gin.H{"error": "Impossible de mettre à jour le profil", "details": err.Error()})
			return
		}
		ctx.JSON(200, updated)
	}
}

func buildProfileUpdates(ctx *gin.Context, st *store.Store, userID string, req profileUpdateRequest) (map[string]interface{}, bool) {
	updates := make(map[string]interface{})
	if req.NomUtilisateur != nil {
		updates["nom_utilisateur"] = *req.NomUtilisateur
	}
	if req.Email != nil {
		existing, _ := st.GetUserByEmail(ctx, *req.Email)
		if existing != nil && existing.ID != userID {
			ctx.JSON(409, gin.H{"error": duplicateUserEmailMessage})
			return nil, false
		}
		updates["email"] = *req.Email
	}
	if req.Password != nil {
		hashedPassword, ok := hashUpdatedPassword(ctx, st, userID, req.OldPassword, *req.Password)
		if !ok {
			return nil, false
		}
		updates["mot_de_passe_hash"] = hashedPassword
	}
	return updates, true
}

func hashUpdatedPassword(ctx *gin.Context, st *store.Store, userID string, oldPassword *string, newPassword string) (string, bool) {
	if oldPassword == nil || *oldPassword == "" {
		ctx.JSON(400, gin.H{"error": "L'ancien mot de passe est requis pour changer le mot de passe"})
		return "", false
	}

	user, ok := fetchUserByID(ctx, st, userID)
	if !ok {
		return "", false
	}
	if !verifyOldPassword(ctx, user.MotDePasseHash, *oldPassword) {
		return "", false
	}

	hashed, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		ctx.JSON(500, gin.H{"error": passwordHashErrorMessage})
		return "", false
	}
	return string(hashed), true
}

func usersCreateHandler(st *store.Store) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		var req userCreateRequest
		if err := ctx.ShouldBindJSON(&req); err != nil {
			ctx.JSON(400, gin.H{"error": invalidDataMessage, "details": err.Error()})
			return
		}

		existingUser, _ := st.GetUserByEmail(ctx, req.Email)
		if existingUser != nil {
			ctx.JSON(409, gin.H{"error": duplicateUserEmailMessage})
			return
		}

		generatedPassword, err := generateTemporaryPassword(12)
		if err != nil {
			ctx.JSON(500, gin.H{"error": "Impossible de générer le mot de passe utilisateur"})
			return
		}

		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(generatedPassword), bcrypt.DefaultCost)
		if err != nil {
			ctx.JSON(500, gin.H{"error": passwordHashErrorMessage})
			return
		}

		user := &models.Utilisateur{
			NomUtilisateur:        req.NomUtilisateur,
			Email:                 req.Email,
			MotDePasseHash:        string(hashedPassword),
			Role:                  req.Role,
			Actif:                 false,
			DoitChangerMotDePasse: true,
			DateCreation:          time.Now(),
		}

		createdUserID, err := st.CreateUser(ctx, user)
		if err != nil {
			ctx.JSON(500, gin.H{"error": "Impossible de créer l'utilisateur", "details": err.Error()})
			return
		}

		createdUser, err := st.GetUserByID(ctx, createdUserID)
		if err != nil {
			ctx.JSON(500, gin.H{"error": "Utilisateur créé mais impossible de le récupérer"})
			return
		}

		sendProvisioningEmail(createdUser, generatedPassword)
		ctx.JSON(201, gin.H{"message": "Utilisateur créé avec succès (compte en attente d'activation)", "user": userResponsePayload(createdUser)})
	}
}

func usersListHandler(st *store.Store) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		users, err := st.GetAllUsers(ctx)
		if err != nil {
			ctx.JSON(500, gin.H{"error": "Impossible de récupérer la liste des utilisateurs"})
			return
		}
		ctx.JSON(200, users)
	}
}

func usersUpdateHandler(st *store.Store) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		id := ctx.Param("id")
		var req userUpdateRequest
		if err := ctx.ShouldBindJSON(&req); err != nil {
			ctx.JSON(400, gin.H{"error": invalidDataMessage, "details": err.Error()})
			return
		}

		updates, ok := buildAdminUserUpdates(ctx, st, id, req)
		if !ok {
			return
		}
		if len(updates) == 0 {
			ctx.JSON(400, gin.H{"error": "Aucune donnée à mettre à jour"})
			return
		}

		userBefore, err := st.GetUserByID(ctx, id)
		if err != nil {
			ctx.JSON(404, gin.H{"error": "Utilisateur non trouvé"})
			return
		}

		updated, err := st.UpdateUser(ctx, id, updates)
		if err != nil {
			ctx.JSON(500, gin.H{"error": "Impossible de mettre à jour l'utilisateur", "details": err.Error()})
			return
		}

		sendActivationEmailIfNeeded(req.Actif, userBefore.Actif, updated)
		ctx.JSON(200, updated)
	}
}

func buildAdminUserUpdates(ctx *gin.Context, st *store.Store, id string, req userUpdateRequest) (map[string]interface{}, bool) {
	updates := make(map[string]interface{})
	if req.NomUtilisateur != nil {
		updates["nom_utilisateur"] = *req.NomUtilisateur
	}
	if req.Email != nil {
		existing, _ := st.GetUserByEmail(ctx, *req.Email)
		if existing != nil && existing.ID != id {
			ctx.JSON(409, gin.H{"error": duplicateUserEmailMessage})
			return nil, false
		}
		updates["email"] = *req.Email
	}
	if req.Role != nil {
		role := strings.ToLower(strings.TrimSpace(*req.Role))
		if role != "admin" && role != "analyst" {
			ctx.JSON(400, gin.H{"error": "Rôle invalide"})
			return nil, false
		}
		updates["role"] = role
	}
	if req.Actif != nil {
		updates["actif"] = *req.Actif
	}
	if req.DoitChangerMotDePasse != nil {
		updates["doit_changer_mot_de_passe"] = *req.DoitChangerMotDePasse
	}
	return updates, true
}

func usersDeleteHandler(st *store.Store) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		id := ctx.Param("id")
		userID, ok := requireAuthenticatedUserID(ctx)
		if !ok {
			return
		}
		if id == userID {
			ctx.JSON(403, gin.H{"error": "Vous ne pouvez pas désactiver ou supprimer votre propre compte."})
			return
		}
		err := st.DeleteUser(ctx, id)
		if err != nil {
			if err.Error() == "blocked:disabled" {
				ctx.JSON(409, gin.H{"error": "Impossible de supprimer l'utilisateur : il est référencé par d'autres données (logs, audit, ...). Vous pouvez le désactiver à la place.", "blocked": true})
				return
			}
			ctx.JSON(500, gin.H{"error": "Impossible de supprimer l'utilisateur", "details": err.Error()})
			return
		}
		ctx.JSON(200, gin.H{"message": "Utilisateur supprimé avec succès"})
	}
}

func usersGetByIDHandler(st *store.Store) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		id := ctx.Param("id")
		user, err := st.GetUserByID(ctx, id)
		if err != nil {
			ctx.JSON(500, gin.H{"error": unableToFetchUserMessage})
			return
		}
		ctx.JSON(200, user)
	}
}

func sendProvisioningEmail(createdUser *models.Utilisateur, generatedPassword string) {
	emailService := email.NewService()
	if err := emailService.SendUserProvisioningEmail(createdUser.Email, createdUser.NomUtilisateur, generatedPassword, createdUser.Role); err != nil {
		log.Printf("User provisioning email failed for admin-created user %s: %v", createdUser.Email, err)
	}
}

func userResponsePayload(createdUser *models.Utilisateur) gin.H {
	return gin.H{
		"id":             createdUser.ID,
		"nomUtilisateur": createdUser.NomUtilisateur,
		"email":          createdUser.Email,
		"role":           createdUser.Role,
		"actif":          createdUser.Actif,
		"dateCreation":   createdUser.DateCreation,
	}
}

func sendActivationEmailIfNeeded(actif *bool, wasActive bool, updated *models.Utilisateur) {
	if actif == nil || !*actif || wasActive {
		return
	}
	emailService := email.NewService()
	if err := emailService.SendAccountActivatedEmail(updated.Email, updated.NomUtilisateur); err != nil {
		log.Printf("Email activation failed for user %s: %v", updated.Email, err)
	}
}

func generateTemporaryPassword(length int) (string, error) {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	if length <= 0 {
		return "", fmt.Errorf("invalid password length")
	}

	password := make([]byte, length)
	max := big.NewInt(int64(len(charset)))
	for i := 0; i < length; i++ {
		n, err := rand.Int(rand.Reader, max)
		if err != nil {
			return "", err
		}
		password[i] = charset[n.Int64()]
	}

	return string(password), nil
}

// validateReportPeriod valide et normalise le paramètre period pour les rapports
func validateReportPeriod(period string) (string, error) {
	normalized := strings.ToLower(strings.TrimSpace(period))
	switch normalized {
	case "month", "quarter", "year":
		return normalized, nil
	default:
		return "", fmt.Errorf("période invalide: %s (autorisées: month, quarter, year)", period)
	}
}

// setupReportsRoutes configure les routes de rapports et statistiques
func setupReportsRoutes(apiV1 *gin.RouterGroup, store *store.Store) {
	// ========== RAPPORTS & STATISTIQUES (Protégé) ==========
	reportsRoutes := apiV1.Group("/reports")
	reportsRoutes.Use(middleware.JwtAuthMiddleware(store), middleware.RoleMiddleware("admin", "analyst"))
	{
		reportsRoutes.GET("/efficiency", reportEfficiencyHandler(store))
		reportsRoutes.GET("/rule-violations", reportRuleViolationsHandler(store))
		reportsRoutes.GET("/audit", reportAuditHandler(store))
		reportsRoutes.GET("/audit/export", reportAuditExportHandler(store))
	}
}

func reportEfficiencyHandler(st *store.Store) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		period := ctx.DefaultQuery("period", "year")
		validPeriod, err := validateReportPeriod(period)
		if err != nil {
			ctx.JSON(400, gin.H{"error": err.Error()})
			return
		}

		metrics, err := st.GetEfficiencyMetrics(ctx, validPeriod)
		if err != nil {
			log.Printf("Erreur lors de la récupération des métriques d'efficacité: %v", err)
			ctx.JSON(500, gin.H{"error": "Impossible de récupérer les métriques d'efficacité"})
			return
		}

		response := make([]map[string]interface{}, len(metrics))
		for i, metric := range metrics {
			response[i] = map[string]interface{}{
				"period":            metric.Period,
				"avgProcessingTime": metric.AvgProcessingTime,
				"resolutionRate":    metric.ResolutionRate,
				"date":              metric.Date,
			}
		}

		ctx.JSON(200, response)
	}
}

func reportRuleViolationsHandler(st *store.Store) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		period := ctx.DefaultQuery("period", "year")
		validPeriod, err := validateReportPeriod(period)
		if err != nil {
			ctx.JSON(400, gin.H{"error": err.Error()})
			return
		}

		violations, err := st.GetRuleViolationsByPeriod(ctx, validPeriod)
		if err != nil {
			log.Printf("Erreur lors de la récupération des violations de règles: %v", err)
			ctx.JSON(500, gin.H{"error": "Impossible de récupérer les violations de règles"})
			return
		}

		ctx.JSON(200, violations)
	}
}

func buildAuditFilters(ctx *gin.Context, includeLimit bool) map[string]interface{} {
	filters := map[string]interface{}{}
	if v := ctx.Query("module"); v != "" {
		filters["module"] = v
	}
	if v := ctx.Query("statut"); v != "" {
		filters["statut"] = v
	}
	if v := ctx.Query("utilisateur_id"); v != "" {
		filters["utilisateur_id"] = v
	}
	if v := ctx.Query("action"); v != "" {
		filters["action"] = v
	}
	if v := ctx.Query("date_debut"); v != "" {
		filters["date_debut"] = v
	}
	if v := ctx.Query("date_fin"); v != "" {
		filters["date_fin"] = v
	}
	if includeLimit {
		if v := ctx.Query("limit"); v != "" {
			filters["limit"] = v
		}
	}
	return filters
}

func resolveAuditUserDisplayName(ctx *gin.Context, st *store.Store, userID string) string {
	switch userID {
	case "MDA":
		return "Moteur de Détection"
	case "app", "app_bis_01":
		return "Application Externe"
	}

	if user, err := st.GetUserByID(ctx, userID); err == nil && user != nil && user.NomUtilisateur != "" {
		return user.NomUtilisateur
	}

	if len(userID) > 8 {
		return userDisplayNamePrefix + userID[:8] + "..."
	}
	return userDisplayNamePrefix + userID
}

func reportAuditHandler(st *store.Store) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		logs, err := st.GetLogsAudit(ctx, buildAuditFilters(ctx, false))
		if err != nil {
			ctx.JSON(500, gin.H{"error": "Impossible de récupérer les logs d'audit"})
			return
		}

		enrichedLogs := make([]gin.H, len(logs))
		for i, log := range logs {
			enrichedLogs[i] = gin.H{
				"id":               log.ID,
				"date_heure":       log.DateHeure,
				"utilisateur_id":   log.UtilisateurID,
				"nom_utilisateur":  resolveAuditUserDisplayName(ctx, st, log.UtilisateurID),
				"action_effectuee": log.ActionEffectuee,
				"details":          log.Details,
				"resultat":         log.Resultat,
				"type_action":      log.TypeAction,
				"entity_id":        log.EntityID,
				"adresse_ip":       log.AdresseIP,
				"role":             log.Role,
			}
		}

		ctx.JSON(200, enrichedLogs)
	}
}

func reportAuditExportHandler(st *store.Store) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		logs, err := st.GetLogsAudit(ctx, buildAuditFilters(ctx, true))
		if err != nil {
			ctx.JSON(500, gin.H{"error": "Impossible de récupérer les logs d'audit"})
			return
		}

		file := excelize.NewFile()
		sheet := file.GetSheetName(0)

		if err := buildAuditExportWorkbook(ctx, st, file, sheet, logs); err != nil {
			ctx.JSON(500, gin.H{"error": err.Error()})
			return
		}

		filename := "audit_export_" + time.Now().Format("20060102_150405") + ".xlsx"
		ctx.Header("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
		ctx.Header("Content-Disposition", "attachment; filename="+filename)
		if err := file.Write(ctx.Writer); err != nil {
			ctx.JSON(500, gin.H{"error": "Erreur écriture fichier XLSX"})
			return
		}
	}
}

func buildAuditExportWorkbook(ctx *gin.Context, st *store.Store, file *excelize.File, sheet string, logs []models.LogAudit) error {
	headerStyle, rowAltStyle, dateStyle, err := buildAuditExportStyles(file)
	if err != nil {
		return fmt.Errorf("erreur création styles XLSX")
	}

	if err := writeAuditExportHeaders(file, sheet, headerStyle); err != nil {
		return err
	}

	if err := writeAuditExportRows(ctx, st, file, sheet, logs, rowAltStyle, dateStyle); err != nil {
		return err
	}

	configureAuditExportSheet(file, sheet)
	return nil
}

func buildAuditExportStyles(file *excelize.File) (int, int, int, error) {
	headerStyle, err := file.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true, Color: "#FFFFFF"},
		Fill:      excelize.Fill{Type: "pattern", Color: []string{"#4F81BD"}, Pattern: 1},
		Alignment: &excelize.Alignment{Horizontal: "center"},
	})
	if err != nil {
		return 0, 0, 0, err
	}

	rowAltStyle, err := file.NewStyle(&excelize.Style{
		Fill: excelize.Fill{Type: "pattern", Color: []string{"#F2F2F2"}, Pattern: 1},
	})
	if err != nil {
		return 0, 0, 0, err
	}

	dateStyle, err := file.NewStyle(&excelize.Style{NumFmt: 22})
	if err != nil {
		return 0, 0, 0, err
	}

	return headerStyle, rowAltStyle, dateStyle, nil
}

func writeAuditExportHeaders(file *excelize.File, sheet string, headerStyle int) error {
	headers := []string{"Date", "Utilisateur", "Événement", "Description", "Détails", "Résultat"}
	for i, h := range headers {
		col, err := excelize.ColumnNumberToName(i + 1)
		if err != nil {
			return err
		}
		cell := fmt.Sprintf("%s1", col)
		if err := file.SetCellValue(sheet, cell, h); err != nil {
			return err
		}
	}
	return file.SetCellStyle(sheet, "A1", "F1", headerStyle)
}

func writeAuditExportRows(ctx *gin.Context, st *store.Store, file *excelize.File, sheet string, logs []models.LogAudit, rowAltStyle int, dateStyle int) error {
	rowIndex := 2
	for i := len(logs) - 1; i >= 0; i-- {
		logItem := logs[i]
		nomUtilisateur := resolveAuditExportUserName(ctx, st, logItem)
		values := auditExportRowValues(logItem, nomUtilisateur)
		if err := writeAuditExportValueCells(file, sheet, rowIndex, values); err != nil {
			return err
		}
		if err := styleAuditExportRow(file, sheet, rowIndex, rowAltStyle, dateStyle); err != nil {
			return err
		}
		rowIndex++
	}
	return nil
}

func resolveAuditExportUserName(ctx *gin.Context, st *store.Store, logItem models.LogAudit) string {
	if logItem.UtilisateurID == "" {
		return "Système"
	}
	return resolveAuditUserDisplayName(ctx, st, logItem.UtilisateurID)
}

func auditExportRowValues(logItem models.LogAudit, nomUtilisateur string) []interface{} {
	return []interface{}{
		logItem.DateHeure,
		nomUtilisateur,
		logItem.TypeAction,
		logItem.ActionEffectuee,
		logItem.Details,
		logItem.Resultat,
	}
}

func writeAuditExportValueCells(file *excelize.File, sheet string, rowIndex int, values []interface{}) error {
	for j, v := range values {
		col, err := excelize.ColumnNumberToName(j + 1)
		if err != nil {
			return err
		}
		cell := fmt.Sprintf("%s%d", col, rowIndex)
		if err := file.SetCellValue(sheet, cell, v); err != nil {
			return err
		}
	}
	return nil
}

func styleAuditExportRow(file *excelize.File, sheet string, rowIndex int, rowAltStyle int, dateStyle int) error {
	if err := file.SetCellStyle(sheet, fmt.Sprintf("A%d", rowIndex), fmt.Sprintf("A%d", rowIndex), dateStyle); err != nil {
		return err
	}
	if rowIndex%2 == 0 {
		if err := file.SetCellStyle(sheet, fmt.Sprintf("A%d", rowIndex), fmt.Sprintf("F%d", rowIndex), rowAltStyle); err != nil {
			return err
		}
	}
	return nil
}

func configureAuditExportSheet(file *excelize.File, sheet string) {
	file.SetColWidth(sheet, "A", "A", 20)
	file.SetColWidth(sheet, "B", "B", 20)
	file.SetColWidth(sheet, "C", "C", 20)
	file.SetColWidth(sheet, "D", "D", 45)
	file.SetColWidth(sheet, "E", "E", 45)
	file.SetColWidth(sheet, "F", "F", 20)

	file.SetPanes(sheet, &excelize.Panes{
		Freeze:      true,
		Split:       false,
		TopLeftCell: "A2",
		ActivePane:  "topLeft",
	})
}
