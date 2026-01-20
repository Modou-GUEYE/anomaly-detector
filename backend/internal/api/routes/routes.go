package routes

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"anomaly-detector/internal/api/handlers"
	"anomaly-detector/internal/api/middleware"
	"anomaly-detector/internal/auth"
	"anomaly-detector/internal/models"
	"anomaly-detector/internal/store"

	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	"github.com/xuri/excelize/v2"
	"golang.org/x/crypto/bcrypt"
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
		// Public signup endpoint: creates a user inactif (requires admin activation)
		authRoutes.POST("/signup", func(ctx *gin.Context) {
			var req struct {
				NomUtilisateur string `json:"nomUtilisateur" binding:"required"`
				Email          string `json:"email" binding:"required,email"`
				Password       string `json:"password" binding:"required,min=6"`
				Role           string `json:"role" binding:"required,oneof=admin analyst"`
			}

			if err := ctx.ShouldBindJSON(&req); err != nil {
				ctx.JSON(400, gin.H{"error": "Données invalides", "details": err.Error()})
				return
			}

			// Check existing user
			existing, _ := store.GetUserByEmail(ctx, req.Email)
			if existing != nil {
				ctx.JSON(409, gin.H{"error": "Un utilisateur avec cet email existe déjà"})
				return
			}

			// Hash password
			hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
			if err != nil {
				ctx.JSON(500, gin.H{"error": "Erreur lors du hachage du mot de passe"})
				return
			}

			user := &models.Utilisateur{
				NomUtilisateur: req.NomUtilisateur,
				Email:          req.Email,
				MotDePasseHash: string(hashedPassword),
				Role:           req.Role,
				Actif:          false,
				DateCreation:   time.Now(),
			}

			id, err := store.CreateUser(ctx, user)
			if err != nil {
				ctx.JSON(500, gin.H{"error": "Impossible de créer l'utilisateur", "details": err.Error()})
				return
			}

			createdUser, err := store.GetUserByID(ctx, id)
			if err != nil {
				ctx.JSON(500, gin.H{"error": "Utilisateur créé mais impossible de le récupérer"})
				return
			}

			resp := gin.H{
				"id":             createdUser.ID,
				"nomUtilisateur": createdUser.NomUtilisateur,
				"email":          createdUser.Email,
				"role":           createdUser.Role,
				"actif":          createdUser.Actif,
				"dateCreation":   createdUser.DateCreation,
			}

			ctx.JSON(201, gin.H{"message": "Utilisateur créé (inactif). Un administrateur doit l'activer.", "user": resp})
		})
	}
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
		// GET - Récupérer tous les pays à risque
		paysRisqueRoutes.GET("", func(ctx *gin.Context) {
			pays, err := store.GetAllPaysARisque(ctx)
			if err != nil {
				log.Printf("[ERROR] GET /api/v1/pays-risque -> GetAllPaysARisque: %v", err)
				ctx.JSON(500, gin.H{"error": "Impossible de récupérer les pays à risque"})
				return
			}
			ctx.JSON(200, pays)
		})

		// POST - Ajouter un pays à risque
		paysRisqueRoutes.POST("", func(ctx *gin.Context) {
			var request struct {
				NomPays string `json:"nom_pays" binding:"required"`
				Code    string `json:"code" binding:"required"` // Code ISO alpha-2 OBLIGATOIRE
			}
			if err := ctx.ShouldBindJSON(&request); err != nil {
				ctx.JSON(400, gin.H{"error": "Nom du pays et code ISO alpha-2 requis"})
				return
			}

			// Valider le code (2 caractères exactement)
			code := strings.ToUpper(strings.TrimSpace(request.Code))
			if len(code) != 2 {
				ctx.JSON(400, gin.H{"error": "Le code pays doit contenir exactement 2 caractères (ISO alpha-2)"})
				return
			}

			// Récupérer l'utilisateur connecté
			userID, exists := ctx.Get("userID")
			if !exists {
				ctx.JSON(500, gin.H{"error": "Utilisateur non identifié"})
				return
			}

			// Créer le pays avec le code
			pays := &models.PaysRisque{
				NomPays: request.NomPays,
				Code:    code,
				Active:  true,
			}

			err := store.AddPaysARisqueWithCode(ctx, pays, userID.(string))
			if err != nil {
				ctx.JSON(500, gin.H{"error": "Impossible d'ajouter le pays à risque"})
				return
			}
			ctx.JSON(201, gin.H{"message": "Pays ajouté avec succès"})
		})

		// PUT - Modifier un pays à risque
		paysRisqueRoutes.PUT("/:ancien_nom", func(ctx *gin.Context) {
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

			// Valider le code (2 caractères exactement)
			code := strings.ToUpper(strings.TrimSpace(request.Code))
			if len(code) != 2 {
				ctx.JSON(400, gin.H{"error": "Le code pays doit contenir exactement 2 caractères (ISO alpha-2)"})
				return
			}

			// Utiliser nom_pays ou nouveau_nom selon ce qui est fourni
			nouveauNom := request.NomPays
			if request.NouveauNom != "" {
				nouveauNom = request.NouveauNom
			}

			// Récupérer l'utilisateur connecté
			userID, exists := ctx.Get("userID")
			if !exists {
				ctx.JSON(500, gin.H{"error": "Utilisateur non identifié"})
				return
			}

			err := store.UpdatePaysARisqueWithCode(ctx, ancienNom, nouveauNom, code, userID.(string))
			if err != nil {
				ctx.JSON(500, gin.H{"error": "Impossible de modifier le pays à risque"})
				return
			}
			ctx.JSON(200, gin.H{"message": "Pays modifié avec succès"})
		})

		// DELETE - Supprimer un pays à risque
		paysRisqueRoutes.DELETE("/:nom_pays", func(ctx *gin.Context) {
			nomPays := ctx.Param("nom_pays")

			// Récupérer l'utilisateur connecté
			userID, exists := ctx.Get("userID")
			if !exists {
				ctx.JSON(500, gin.H{"error": "Utilisateur non identifié"})
				return
			}

			err := store.DeletePaysARisque(ctx, nomPays, userID.(string))
			if err != nil {
				ctx.JSON(500, gin.H{"error": "Impossible de supprimer le pays à risque"})
				return
			}
			ctx.JSON(200, gin.H{"message": "Pays supprimé avec succès"})
		})

		// POST - Importer des pays à risque depuis un fichier CSV/JSON/Excel
		paysRisqueRoutes.POST("/import", func(ctx *gin.Context) {
			file, err := ctx.FormFile("file")
			if err != nil {
				// debug: failed to read uploaded file (logging retained in response)
				ctx.JSON(400, gin.H{"error": "Fichier manquant"})
				return
			}
			// file received (info removed to reduce verbosity)

			// Récupérer l'utilisateur connecté
			userID, exists := ctx.Get("userID")
			if !exists {
				ctx.JSON(500, gin.H{"error": "Utilisateur non identifié"})
				return
			}

			// Structure pour stocker les pays avec leurs codes
			type PaysData struct {
				NomPays string
				Code    string
			}
			var paysToImport []PaysData
			filename := strings.ToLower(file.Filename)

			// Parser selon le format
			if strings.HasSuffix(filename, ".xlsx") || strings.HasSuffix(filename, ".xls") {
				// Format Excel
				// processing excel file (info removed)
				src, err := file.Open()
				if err != nil {
					// excel open error (handled by response)
					ctx.JSON(500, gin.H{"error": "Impossible d'ouvrir le fichier Excel"})
					return
				}
				defer src.Close()

				xlFile, err := excelize.OpenReader(src)
				if err != nil {
					// excel read error (handled by response)
					ctx.JSON(400, gin.H{"error": fmt.Sprintf("Erreur lecture Excel: %v", err)})
					return
				}
				defer xlFile.Close()

				// Lire la première feuille
				sheets := xlFile.GetSheetList()
				if len(sheets) == 0 {
					// no sheets found in excel
					ctx.JSON(400, gin.H{"error": "Le fichier Excel ne contient aucune feuille"})
					return
				}
				// sheets found (info removed)

				rows, err := xlFile.GetRows(sheets[0])
				if err != nil {
					// error reading rows (handled by response)
					ctx.JSON(400, gin.H{"error": fmt.Sprintf("Erreur lecture des lignes: %v", err)})
					return
				}
				// row count info removed

				// Parser les lignes (ignorer l'en-tête)
				for i, row := range rows {
					// per-line debug removed

					// Ignorer l'en-tête
					if i == 0 && len(row) >= 1 {
						firstCell := strings.ToLower(row[0])
						if strings.Contains(firstCell, "nom") || strings.Contains(firstCell, "pays") {
							// header detected, skipping line
							continue
						}
					}

					// Vérifier qu'on a au moins 2 colonnes
					if len(row) >= 2 {
						nomPays := strings.TrimSpace(row[0])
						code := strings.TrimSpace(row[1])
						if nomPays != "" && code != "" {
							// country added (info removed)
							paysToImport = append(paysToImport, PaysData{NomPays: nomPays, Code: code})
						} else {
							// ignored line with empty fields
						}
					} else {
						// ignored line with insufficient columns
					}
				}

			} else if strings.HasSuffix(filename, ".json") {
				// Format JSON
				src, err := file.Open()
				if err != nil {
					ctx.JSON(500, gin.H{"error": "Impossible d'ouvrir le fichier"})
					return
				}
				defer src.Close()

				content := make([]byte, file.Size)
				if _, err := src.Read(content); err != nil {
					ctx.JSON(500, gin.H{"error": "Impossible de lire le fichier"})
					return
				}

				var dataWithCodes []struct {
					NomPays string `json:"nomPays"`
					Code    string `json:"code"`
				}
				if err := json.Unmarshal(content, &dataWithCodes); err == nil {
					// JSON format detected (info removed)
					for _, d := range dataWithCodes {
						if d.NomPays != "" && d.Code != "" {
							paysToImport = append(paysToImport, PaysData{NomPays: d.NomPays, Code: d.Code})
						}
					}
				} else {
					// tried alternative json format
					// Essayer aussi le format avec nom_pays au lieu de nomPays
					var dataWithCodesAlt []struct {
						NomPays string `json:"nom_pays"`
						Code    string `json:"code"`
					}
					if err := json.Unmarshal(content, &dataWithCodesAlt); err == nil {
						for _, d := range dataWithCodesAlt {
							if d.NomPays != "" && d.Code != "" {
								paysToImport = append(paysToImport, PaysData{NomPays: d.NomPays, Code: d.Code})
							}
						}
					} else {
						ctx.JSON(400, gin.H{"error": "Format JSON invalide. Utilisez [{\"nomPays\":\"France\",\"code\":\"FR\"}, ...] ou [{\"nom_pays\":\"France\",\"code\":\"FR\"}, ...]"})
						return
					}
				}

			} else if strings.HasSuffix(filename, ".csv") || strings.HasSuffix(filename, ".txt") {
				// Format CSV
				src, err := file.Open()
				if err != nil {
					ctx.JSON(500, gin.H{"error": "Impossible d'ouvrir le fichier"})
					return
				}
				defer src.Close()

				content := make([]byte, file.Size)
				if _, err := src.Read(content); err != nil {
					ctx.JSON(500, gin.H{"error": "Impossible de lire le fichier"})
					return
				}

				// Format CSV: nom_pays,code
				lines := strings.Split(string(content), "\n")
				hasHeader := false
				for i, line := range lines {
					line = strings.TrimSpace(line)
					// Ignorer ligne vide
					if line == "" {
						continue
					}
					// Détecter et ignorer la ligne d'en-tête
					if i == 0 && (strings.Contains(strings.ToLower(line), "nom") || strings.Contains(strings.ToLower(line), "pays") || strings.Contains(strings.ToLower(line), "code")) {
						hasHeader = true
						continue
					}
					// Parser la ligne: nom_pays,code
					parts := strings.Split(line, ",")
					if len(parts) >= 2 {
						nomPays := strings.TrimSpace(parts[0])
						code := strings.TrimSpace(parts[1])
						if nomPays != "" && code != "" {
							paysToImport = append(paysToImport, PaysData{NomPays: nomPays, Code: code})
						}
					} else if len(parts) == 1 && !hasHeader {
						// Format sans code - retourner une erreur
						ctx.JSON(400, gin.H{"error": "Le fichier CSV doit contenir deux colonnes: nom_pays,code (ISO alpha-2). Exemple: France,FR"})
						return
					}
				}
			} else {
				ctx.JSON(400, gin.H{"error": "Format de fichier non supporté. Utilisez CSV, JSON ou Excel (.xlsx, .xls)"})
				return
			}

			if len(paysToImport) == 0 {
				// no countries found in uploaded file
				ctx.JSON(400, gin.H{"error": "Aucun pays trouvé dans le fichier. Assurez-vous que le fichier contient des données au format: nom_pays,code"})
				return
			}

			// extracted countries count info removed

			// Importer les pays avec leurs codes
			imported := 0
			var errors []string
			for _, paysData := range paysToImport {
				pays := &models.PaysRisque{
					NomPays: paysData.NomPays,
					Code:    paysData.Code,
					Active:  true,
				}
				if err := store.AddPaysARisqueWithCode(ctx, pays, userID.(string)); err != nil {
					errors = append(errors, fmt.Sprintf("Erreur pour '%s' (%s): %v", paysData.NomPays, paysData.Code, err))
				} else {
					imported++
				}
			}

			ctx.JSON(200, gin.H{
				"message":  fmt.Sprintf("%d/%d pays importés avec succès", imported, len(paysToImport)),
				"imported": imported,
				"total":    len(paysToImport),
				"errors":   errors,
			})
		})
	}
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
		userRoutes.GET("", func(ctx *gin.Context) {
			userID, _ := ctx.Get("userID")
			user, err := store.GetUserByID(ctx, userID.(string))
			if err != nil {
				ctx.JSON(500, gin.H{"error": "Impossible de récupérer le profil"})
				return
			}
			ctx.JSON(200, user)
		})

		// Mise à jour du profil de l'utilisateur connecté (nomUtilisateur, email, password)
		userRoutes.PUT("", func(ctx *gin.Context) {
			log.Printf("PUT /api/v1/profile called - Authorization header present: %v", ctx.GetHeader("Authorization") != "")

			userID, _ := ctx.Get("userID")
			var req struct {
				NomUtilisateur *string `json:"nomUtilisateur" binding:"omitempty"`
				Email          *string `json:"email" binding:"omitempty,email"`
				Password       *string `json:"password" binding:"omitempty,min=6"`
				OldPassword    *string `json:"oldPassword" binding:"omitempty"`
			}

			if err := ctx.ShouldBindJSON(&req); err != nil {
				ctx.JSON(400, gin.H{"error": "Données invalides", "details": err.Error()})
				return
			}

			updates := make(map[string]interface{})
			if req.NomUtilisateur != nil {
				updates["nom_utilisateur"] = *req.NomUtilisateur
			}
			if req.Email != nil {
				// vérifier unicité email
				existing, _ := store.GetUserByEmail(ctx, *req.Email)
				if existing != nil && existing.ID != userID.(string) {
					ctx.JSON(409, gin.H{"error": "Un utilisateur avec cet email existe déjà"})
					return
				}
				updates["email"] = *req.Email
			}
			if req.Password != nil {
				// Vérifier que l'ancien mot de passe est fourni
				if req.OldPassword == nil || *req.OldPassword == "" {
					ctx.JSON(400, gin.H{"error": "L'ancien mot de passe est requis pour changer le mot de passe"})
					return
				}

				// Récupérer l'utilisateur actuel pour vérifier l'ancien mot de passe
				user, err := store.GetUserByID(ctx, userID.(string))
				if err != nil {
					ctx.JSON(500, gin.H{"error": "Impossible de récupérer l'utilisateur"})
					return
				}

				// Vérifier si l'ancien mot de passe est correct
				err = bcrypt.CompareHashAndPassword([]byte(user.MotDePasseHash), []byte(*req.OldPassword))
				if err != nil {
					ctx.JSON(401, gin.H{"error": "L'ancien mot de passe est incorrect"})
					return
				}

				hashed, err := bcrypt.GenerateFromPassword([]byte(*req.Password), bcrypt.DefaultCost)
				if err != nil {
					ctx.JSON(500, gin.H{"error": "Erreur lors du hachage du mot de passe"})
					return
				}
				updates["mot_de_passe_hash"] = string(hashed)
			}

			if len(updates) == 0 {
				ctx.JSON(400, gin.H{"error": "Aucune donnée à mettre à jour"})
				return
			}

			updated, err := store.UpdateUser(ctx, userID.(string), updates)
			if err != nil {
				ctx.JSON(500, gin.H{"error": "Impossible de mettre à jour le profil", "details": err.Error()})
				return
			}
			ctx.JSON(200, updated)
		})
	}

	// ========== UTILISATEURS (Protégé) ==========
	usersRoutes := apiV1.Group("/users")
	usersRoutes.Use(middleware.JwtAuthMiddleware(store), middleware.RoleMiddleware("admin"))
	{
		usersRoutes.POST("", func(ctx *gin.Context) {
			var req struct {
				NomUtilisateur string `json:"nomUtilisateur" binding:"required"`
				Email          string `json:"email" binding:"required,email"`
				Password       string `json:"password" binding:"required,min=6"`
				Role           string `json:"role" binding:"required,oneof=admin analyst"`
				Actif          bool   `json:"actif"`
			}

			if err := ctx.ShouldBindJSON(&req); err != nil {
				ctx.JSON(400, gin.H{"error": "Données invalides", "details": err.Error()})
				return
			}

			// Vérifier si l'utilisateur existe déjà
			existingUser, _ := store.GetUserByEmail(ctx, req.Email)
			if existingUser != nil {
				ctx.JSON(409, gin.H{"error": "Un utilisateur avec cet email existe déjà"})
				return
			}

			// Hacher le mot de passe
			hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
			if err != nil {
				ctx.JSON(500, gin.H{"error": "Erreur lors du hachage du mot de passe"})
				return
			}

			// Créer l'utilisateur
			user := &models.Utilisateur{
				NomUtilisateur: req.NomUtilisateur,
				Email:          req.Email,
				MotDePasseHash: string(hashedPassword),
				Role:           req.Role,
				Actif:          req.Actif,
				DateCreation:   time.Now(),
			}

			createdUserID, err := store.CreateUser(ctx, user)
			if err != nil {
				ctx.JSON(500, gin.H{"error": "Impossible de créer l'utilisateur", "details": err.Error()})
				return
			}

			// Récupérer l'utilisateur créé pour le retourner
			createdUser, err := store.GetUserByID(ctx, createdUserID)
			if err != nil {
				ctx.JSON(500, gin.H{"error": "Utilisateur créé mais impossible de le récupérer"})
				return
			}

			// Retourner l'utilisateur sans le mot de passe
			response := gin.H{
				"id":             createdUser.ID,
				"nomUtilisateur": createdUser.NomUtilisateur,
				"email":          createdUser.Email,
				"role":           createdUser.Role,
				"actif":          createdUser.Actif,
				"dateCreation":   createdUser.DateCreation,
			}

			ctx.JSON(201, gin.H{"message": "Utilisateur créé avec succès", "user": response})
		})

		// Lister tous les utilisateurs (admin uniquement)
		usersRoutes.GET("", func(ctx *gin.Context) {
			users, err := store.GetAllUsers(ctx)
			if err != nil {
				ctx.JSON(500, gin.H{"error": "Impossible de récupérer la liste des utilisateurs"})
				return
			}
			ctx.JSON(200, users)
		})

		// Mise à jour partielle d'un utilisateur (ex: activation/désactivation)
		usersRoutes.PUT("/:id", func(ctx *gin.Context) {
			id := ctx.Param("id")
			var req map[string]interface{}
			if err := ctx.ShouldBindJSON(&req); err != nil {
				ctx.JSON(400, gin.H{"error": "Données invalides", "details": err.Error()})
				return
			}
			updated, err := store.UpdateUser(ctx, id, req)
			if err != nil {
				ctx.JSON(500, gin.H{"error": "Impossible de mettre à jour l'utilisateur", "details": err.Error()})
				return
			}
			ctx.JSON(200, updated)
		})

		// Supprimer un utilisateur
		usersRoutes.DELETE("/:id", func(ctx *gin.Context) {
			id := ctx.Param("id")
			userID, exists := ctx.Get("userID")
			if !exists {
				ctx.JSON(500, gin.H{"error": "Utilisateur non identifié"})
				return
			}
			if id == userID {
				ctx.JSON(403, gin.H{"error": "Vous ne pouvez pas désactiver ou supprimer votre propre compte."})
				return
			}
			// On modifie DeleteUser pour retourner une erreur spéciale si désactivation
			err := store.DeleteUser(ctx, id)
			if err != nil {
				// Si l'erreur contient le mot clé "désactivé" on retourne 409
				if err.Error() == "blocked:disabled" {
					ctx.JSON(409, gin.H{"error": "Impossible de supprimer l'utilisateur : il est référencé par d'autres données (logs, audit, ...). Vous pouvez le désactiver à la place.", "blocked": true})
					return
				}
				ctx.JSON(500, gin.H{"error": "Impossible de supprimer l'utilisateur", "details": err.Error()})
				return
			}
			ctx.JSON(200, gin.H{"message": "Utilisateur supprimé avec succès"})
		})

		usersRoutes.GET("/:id", func(ctx *gin.Context) {
			userID := ctx.Param("id")
			user, err := store.GetUserByID(ctx, userID)
			if err != nil {
				ctx.JSON(500, gin.H{"error": "Impossible de récupérer l'utilisateur"})
				return
			}
			ctx.JSON(200, user)
		})
	}
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
		reportsRoutes.GET("/efficiency", func(ctx *gin.Context) {
			period := ctx.DefaultQuery("period", "year")

			// Valider le paramètre period
			validPeriod, err := validateReportPeriod(period)
			if err != nil {
				ctx.JSON(400, gin.H{"error": err.Error()})
				return
			}

			// Récupérer les métriques d'efficacité depuis la base de données
			metrics, err := store.GetEfficiencyMetrics(ctx, validPeriod)
			if err != nil {
				log.Printf("Erreur lors de la récupération des métriques d'efficacité: %v", err)
				ctx.JSON(500, gin.H{"error": "Impossible de récupérer les métriques d'efficacité"})
				return
			}

			// Transformer en camelCase cohérent avec frontend
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
		})
		reportsRoutes.GET("/rule-violations", func(ctx *gin.Context) {
			period := ctx.DefaultQuery("period", "year")

			// Valider le paramètre period
			validPeriod, err := validateReportPeriod(period)
			if err != nil {
				ctx.JSON(400, gin.H{"error": err.Error()})
				return
			}

			// Récupérer les violations de règles depuis la base de données
			violations, err := store.GetRuleViolationsByPeriod(ctx, validPeriod)
			if err != nil {
				log.Printf("Erreur lors de la récupération des violations de règles: %v", err)
				ctx.JSON(500, gin.H{"error": "Impossible de récupérer les violations de règles"})
				return
			}

			// Uniformiser en camelCase (ruleName, count, percentage déjà corrects dans struct store.RuleViolation)
			ctx.JSON(200, violations)
		})
		reportsRoutes.GET("/audit", func(ctx *gin.Context) {
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

			logs, err := store.GetLogsAudit(ctx, filters)
			if err != nil {
				ctx.JSON(500, gin.H{"error": "Impossible de récupérer les logs d'audit"})
				return
			}

			// Enrichir TOUS les logs avec les noms d'utilisateur pour un affichage uniforme
			enrichedLogs := make([]gin.H, len(logs))
			for i, log := range logs {
				// Déterminer le nom d'utilisateur à afficher de manière uniforme
				nomUtilisateur := "Système"
				switch log.UtilisateurID {
				case "MDA":
					nomUtilisateur = "Moteur de Détection"
				case "app", "app_bis_01":
					nomUtilisateur = "Application Externe"
				default:
					// Pour TOUS les utilisateurs humains (admin/analyst), récupérer le nom depuis la base
					user, err := store.GetUserByID(ctx, log.UtilisateurID)
					if err == nil && user != nil {
						nomUtilisateur = user.NomUtilisateur
					} else {
						// Fallback uniforme pour tous les cas d'échec
						if len(log.UtilisateurID) > 8 {
							nomUtilisateur = "Utilisateur " + log.UtilisateurID[:8] + "..."
						} else {
							nomUtilisateur = "Utilisateur " + log.UtilisateurID
						}
					}
				}

				enrichedLogs[i] = gin.H{
					"id":               log.ID,
					"date_heure":       log.DateHeure,
					"utilisateur_id":   log.UtilisateurID,
					"nom_utilisateur":  nomUtilisateur,
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
		})

		// Export XLSX des logs d'audit
		reportsRoutes.GET("/audit/export", func(ctx *gin.Context) {
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
			if v := ctx.Query("limit"); v != "" {
				filters["limit"] = v
			}

			logs, err := store.GetLogsAudit(ctx, filters)
			if err != nil {
				ctx.JSON(500, gin.H{"error": "Impossible de récupérer les logs d'audit"})
				return
			}

			file := excelize.NewFile()
			sheet := file.GetSheetName(0)

			// Styles
			headerStyle, _ := file.NewStyle(&excelize.Style{
				Font:      &excelize.Font{Bold: true, Color: "#FFFFFF"},
				Fill:      excelize.Fill{Type: "pattern", Color: []string{"#4F81BD"}, Pattern: 1},
				Alignment: &excelize.Alignment{Horizontal: "center"},
			})
			rowAltStyle, _ := file.NewStyle(&excelize.Style{
				Fill: excelize.Fill{Type: "pattern", Color: []string{"#F2F2F2"}, Pattern: 1},
			})
			dateStyle, _ := file.NewStyle(&excelize.Style{
				NumFmt: 22, // yyyy-mm-dd h:mm
			})

			// En-têtes
			headers := []string{"Date", "Utilisateur", "Événement", "Description", "Détails", "Résultat"}
			for i, h := range headers {
				col, _ := excelize.ColumnNumberToName(i + 1)
				cell := fmt.Sprintf("%s1", col)
				file.SetCellValue(sheet, cell, h)
			}
			file.SetCellStyle(sheet, "A1", "F1", headerStyle)

			// Lignes de données
			rowIndex := 2
			for i := len(logs) - 1; i >= 0; i-- {
				logItem := logs[i]

				// Nom utilisateur enrichi
				nomUtilisateur := "Système"
				switch logItem.UtilisateurID {
				case "MDA":
					nomUtilisateur = "Moteur de Détection"
				case "app", "app_bis_01":
					nomUtilisateur = "Application Externe"
				default:
					if logItem.UtilisateurID != "" {
						if user, err := store.GetUserByID(ctx, logItem.UtilisateurID); err == nil && user != nil && user.NomUtilisateur != "" {
							nomUtilisateur = user.NomUtilisateur
						} else if len(logItem.UtilisateurID) > 8 {
							nomUtilisateur = "Utilisateur " + logItem.UtilisateurID[:8] + "..."
						} else {
							nomUtilisateur = "Utilisateur " + logItem.UtilisateurID
						}
					}
				}

				values := []interface{}{
					logItem.DateHeure,
					nomUtilisateur,
					logItem.TypeAction,
					logItem.ActionEffectuee,
					logItem.Details,
					logItem.Resultat,
				}
				for j, v := range values {
					col, _ := excelize.ColumnNumberToName(j + 1)
					cell := fmt.Sprintf("%s%d", col, rowIndex)
					file.SetCellValue(sheet, cell, v)
				}

				// Style date
				file.SetCellStyle(sheet, fmt.Sprintf("A%d", rowIndex), fmt.Sprintf("A%d", rowIndex), dateStyle)

				// Alternance ligne
				if rowIndex%2 == 0 {
					file.SetCellStyle(sheet, fmt.Sprintf("A%d", rowIndex), fmt.Sprintf("F%d", rowIndex), rowAltStyle)
				}
				rowIndex++
			}

			// Largeur des colonnes
			file.SetColWidth(sheet, "A", "A", 20)
			file.SetColWidth(sheet, "B", "B", 20)
			file.SetColWidth(sheet, "C", "C", 20)
			file.SetColWidth(sheet, "D", "D", 45)
			file.SetColWidth(sheet, "E", "E", 45)
			file.SetColWidth(sheet, "F", "F", 20)

			// Figer l'en-tête
			file.SetPanes(sheet, &excelize.Panes{
				Freeze:      true,
				Split:       false,
				TopLeftCell: "A2",
				ActivePane:  "topLeft",
			})

			filename := "audit_export_" + time.Now().Format("20060102_150405") + ".xlsx"
			ctx.Header("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
			ctx.Header("Content-Disposition", "attachment; filename="+filename)
			if err := file.Write(ctx.Writer); err != nil {
				ctx.JSON(500, gin.H{"error": "Erreur écriture fichier XLSX"})
				return
			}
		})
	}
}
