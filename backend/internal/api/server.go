// ===================== PACKAGE & IMPORTS =====================
package api

import (
	"anomaly-detector/internal/core"
	"anomaly-detector/internal/models"
	"anomaly-detector/internal/store"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	_ "anomaly-detector/docs"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	"golang.org/x/crypto/bcrypt"
)

// ===================== CONSTANTES & VARIABLES GLOBALES =====================

var jwtKey = []byte("ma_super_cle_secrete_qui_doit_etre_longue_et_complexe")

// ===================== STRUCTURES PRINCIPALES =====================

// Server sert les requêtes HTTP pour notre service.
type Server struct {
	store  *store.Store
	router *gin.Engine
}

// ===================== MIDDLEWARES D'AUTHENTIFICATION & AUTORISATION =====================

// jwtAuthMiddleware authentifie les requêtes via un token JWT passé dans le header Authorization.
// Il vérifie la présence et la validité du token, extrait les claims utiles (userID, role)
// et les injecte dans le contexte Gin pour les middlewares/handlers suivants.
// Si le token est absent, mal formé ou invalide, la requête est stoppée avec un code 401.
func jwtAuthMiddleware() gin.HandlerFunc {
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

// roleMiddleware restreint l'accès à certaines routes selon le rôle de l'utilisateur.
// Vérifie que le claim 'role' du token JWT correspond à l'un des rôles autorisés.
func roleMiddleware(allowedRoles ...string) gin.HandlerFunc {
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

// checkRole est un middleware qui vérifie si l'utilisateur a un rôle spécifique.
// À utiliser pour les contrôles de rôle supplémentaires au niveau des routes.
func checkRole(requiredRole string) gin.HandlerFunc {
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

// ===================== HANDLERS PUBLICS - AUTHENTIFICATION =====================

// @Summary      Connecter un utilisateur
// @Description  Authentifie un utilisateur avec email et mot de passe et renvoie un token JWT.
// @Tags         Authentification
// @Accept       json
// @Produce      json
// @Param        credentials  body      object{email=string,password=string}  true  "Identifiants de l'utilisateur"
// @Success      200          {object}  map[string]string "Renvoie un token JWT"
// @Failure      401          {object}  map[string]string
// @Router       /auth/login [post]
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

// handleGetSystemToken génère un token JWT pour un système externe (partenaire)
// Cette fonction est neutre et générique, utilisable par n'importe quel système.
func handleGetSystemToken(ctx *gin.Context) {
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
	tokenString, err := token.SignedString(jwtKey)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Impossible de générer le token"})
		return
	}
	ctx.JSON(http.StatusOK, gin.H{"token": tokenString})
}

// ===================== HANDLERS PROTÉGÉS - ANOMALIES/TRANSACTIONS =====================

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
// @Router       /anomalies/analyze [post]
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

	// Code banque du partenaire principal
	const partnerBankCode = "SN079"
	isPayeurPartenaire := len(tx.NumeroComptePayeur) >= 5 && tx.NumeroComptePayeur[:5] == partnerBankCode
	isPayePartenaire := len(tx.NumeroComptePaye) >= 5 && tx.NumeroComptePaye[:5] == partnerBankCode
	if !isPayeurPartenaire && !isPayePartenaire {
		ctx.JSON(http.StatusOK, gin.H{"status": "transaction_non_analysee", "raison": "Aucun compte partenaire (code banque SN079) impliqué dans la transaction."})
		return
	}

	// Analyse de la transaction
	rulesEngine := core.NewRulesEngine(server.store)
	alertes, err := rulesEngine.AnalyzeTransaction(ctx, &tx)
	if err != nil {
		if errors.Is(err, core.ErrTransactionRefused) {
			createdTx, txErr := server.store.CreateTransaction(ctx, &tx)
			if txErr != nil {
				ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Impossible d'enregistrer la transaction: " + txErr.Error()})
				return
			}
			savedBloqueAlertes := make([]*models.Alerte, 0)
			for _, alerte := range alertes {
				if strings.EqualFold(alerte.Action, "bloquer") {
					alerte.Statut = "Traité"
				}
				alerte.TransactionID = createdTx.ID
				if alerte.DateGeneration.IsZero() {
					alerte.DateGeneration = time.Now()
				}
				saved, _ := server.store.CreateAlerte(ctx, alerte)
				if saved != nil {
					savedBloqueAlertes = append(savedBloqueAlertes, saved)
				}
			}
			msg := err.Error()
			if idx := strings.Index(msg, ": "); idx != -1 {
				msg = msg[idx+2:]
			}
			ctx.JSON(http.StatusForbidden, gin.H{"status": "transaction_bloquee", "error": msg, "alertes": savedBloqueAlertes})
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
		createdTx, err := server.store.CreateTransaction(ctx, &tx)
		if err != nil {
			ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Impossible d'enregistrer la transaction: " + err.Error()})
			return
		}
		var savedBloqueAlertes []*models.Alerte
		for _, alerte := range bloqueAlertes {
			alerte.TransactionID = createdTx.ID
			if strings.EqualFold(alerte.Action, "bloquer") {
				alerte.Statut = "Traité"
			}
			if alerte.DateGeneration.IsZero() {
				alerte.DateGeneration = time.Now()
			}
			saved, _ := server.store.CreateAlerte(ctx, alerte)
			savedBloqueAlertes = append(savedBloqueAlertes, saved)
		}
		ctx.JSON(http.StatusForbidden, gin.H{"status": "transaction_bloquee", "alertes": savedBloqueAlertes})
		return
	}

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
			const partnerBankCode = "SN079"
			if alerte.NumeroCompte != "" && len(alerte.NumeroCompte) >= 5 && alerte.NumeroCompte[:5] == partnerBankCode {
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

// handleGetAlertMetrics retourne les métriques d'alertes agrégées par mois et par type.
func (server *Server) handleGetAlertMetrics(ctx *gin.Context) {
	months := 12
	if mStr := ctx.Query("months"); mStr != "" {
		if m, err := strconv.Atoi(mStr); err == nil && m > 0 {
			months = m
		}
	}
	metrics, err := server.store.GetAlertCountsByMonthAndType(ctx, months)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Impossible de récupérer les métriques d'alertes"})
		return
	}
	ctx.JSON(http.StatusOK, gin.H{"metrics": metrics})
}

// @Summary      Lister toutes les alertes
// @Description  Récupère la liste de toutes les alertes générées par le système.
// @Tags         Anomalies
// @Produce      json
// @Success      200  {array}   models.Alerte
// @Failure      401  {object}  map[string]string
// @Security     BearerAuth
// @Router       /anomalies [get]
func (server *Server) handleGetAlertes(ctx *gin.Context) {
	alertes, err := server.store.GetAlertes(ctx)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Impossible de récupérer les alertes"})
		return
	}
	ctx.JSON(http.StatusOK, alertes)
}

// @Summary      Mettre à jour le statut d'une alerte
// @Description  Permet à un analyste de changer le statut d'une alerte.
// @Tags         Anomalies
// @Accept       json
// @Produce      json
// @Param        id      path      string                  true  "ID de l'alerte"
// @Param        status  body      object{statut=string}  true  "Nouveau statut de l'alerte"
// @Success      200     {object}  models.Alerte
// @Failure      401     {object}  map[string]string
// @Security     BearerAuth
// @Router       /anomalies/{id}/status [put]
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
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Impossible de mettre à jour l'alerte", "details": err.Error()})
		return
	}
	ctx.JSON(http.StatusOK, alerte)
}

// ===================== HANDLERS PROTÉGÉS - ENTITÉS/BLACKLIST =====================

// @Summary      Lister les entrées de la blacklist
// @Description  Récupère toutes les entités de la liste de surveillance.
// @Tags         Entités
// @Produce      json
// @Success      200  {array}   models.EntiteBlacklist
// @Failure      403  {object}  map[string]string
// @Security     BearerAuth
// @Router       /entities [get]
func (server *Server) handleGetBlacklistEntries(ctx *gin.Context) {
	entries, err := server.store.GetBlacklistEntries(ctx)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Impossible de récupérer les entrées"})
		return
	}
	ctx.JSON(http.StatusOK, entries)
}

// @Summary      Créer une entrée dans la blacklist
// @Description  Ajoute une nouvelle entité à la liste de surveillance.
// @Tags         Entités
// @Accept       json
// @Produce      json
// @Param        blacklistEntry  body      models.EntiteBlacklist  true  "Détails de l'entité à ajouter"
// @Success      201             {object}  models.EntiteBlacklist
// @Failure      403             {object}  map[string]string
// @Security     BearerAuth
// @Router       /entities [post]
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

// @Summary      Mettre à jour une entrée de la blacklist
// @Description  Modifie une entité de la liste de surveillance.
// @Tags         Entités
// @Accept       json
// @Produce      json
// @Param        id              path      string                  true  "ID de l'entité"
// @Param        blacklistEntry  body      models.EntiteBlacklist  true  "Nouvelles données de l'entité"
// @Success      200             {object}  models.EntiteBlacklist
// @Failure      403             {object}  map[string]string
// @Security     BearerAuth
// @Router       /entities/{id} [put]
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
// @Description  Supprime une entité de la liste de surveillance par son ID.
// @Tags         Entités
// @Produce      json
// @Param        id  path      string  true  "ID de l'entité"
// @Success      204 "No Content"
// @Failure      403 {object}  map[string]string
// @Security     BearerAuth
// @Router       /entities/{id} [delete]
func (server *Server) handleDeleteBlacklistEntry(ctx *gin.Context) {
	id := ctx.Param("id")
	if err := server.store.DeleteBlacklistEntry(ctx, id); err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Impossible de supprimer l'entrée"})
		return
	}
	ctx.Status(http.StatusNoContent)
}

// Handler pour vérifier si un compte est dans la blacklist
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
	switch status {
	case "Interdit", "Blacklist", "PEP", "Surveillés":
		ctx.JSON(http.StatusOK, gin.H{"statut": status})
	default:
		ctx.JSON(http.StatusOK, gin.H{"statut": "Aucun"})
	}
}

// ===================== HANDLERS PROTÉGÉS - RÈGLES DE CONTRÔLE =====================

// Handler pour lister toutes les règles de contrôle
func (server *Server) handleGetAllRegles(ctx *gin.Context) {
	regles, err := server.store.GetAllRegles(ctx)
	if err != nil {
		ctx.JSON(500, gin.H{"error": "Impossible de récupérer les règles"})
		return
	}
	ctx.JSON(200, regles)
}

// @Summary      Créer une règle de contrôle
// @Description  Crée une nouvelle règle de contrôle.
// @Tags         Règles
// @Accept       json
// @Produce      json
// @Param        regle  body      models.RegleControle  true  "Détails de la règle"
// @Success      201    {object}  models.RegleControle
// @Failure      403    {object}  map[string]string
// @Security     BearerAuth
// @Router       /rules [post]
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

// @Summary      Mettre à jour une règle de contrôle
// @Description  Modifie une règle de contrôle.
// @Tags         Règles
// @Accept       json
// @Produce      json
// @Param        id     path      string                true  "ID de la règle"
// @Param        regle  body      models.RegleControle  true  "Nouvelles données de la règle"
// @Success      200    {object}  models.RegleControle
// @Failure      403    {object}  map[string]string
// @Security     BearerAuth
// @Router       /rules/{id} [put]
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
// @Description  Supprime une règle de contrôle.
// @Tags         Règles
// @Produce      json
// @Param        id  path      string  true  "ID de la règle"
// @Success      204 "No Content"
// @Failure      403 {object}  map[string]string
// @Security     BearerAuth
// @Router       /rules/{id} [delete]
func (server *Server) handleDeleteRegleControle(ctx *gin.Context) {
	id := ctx.Param("id")
	if err := server.store.DeleteRegleControle(ctx, id); err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Impossible de supprimer la règle"})
		return
	}
	ctx.Status(http.StatusNoContent)
}

// @Summary      Lister les règles d'un profil
// @Description  Récupère toutes les règles associées à un profil client.
// @Tags         Règles
// @Produce      json
// @Param        id  path      string  true  "ID du Profil Client"
// @Success      200 {array}   models.RegleControle
// @Failure      403 {object}  map[string]string
// @Security     BearerAuth
// @Router       /rules/{id}/profiles [get]
func (server *Server) handleGetReglesByProfil(ctx *gin.Context) {
	profilID := ctx.Param("id")
	regles, err := server.store.GetReglesByProfilID(ctx, profilID)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Impossible de récupérer les règles"})
		return
	}
	ctx.JSON(http.StatusOK, regles)
}

// @Summary      Lister les règles actives d'un compte
// @Description  Récupère toutes les règles actives associées au profil client d'un compte donné.
// @Tags         Règles
// @Produce      json
// @Param        numeroCompte  path      string  true  "Numéro de compte"
// @Success      200 {array}   models.RegleControle
// @Failure      403 {object}  map[string]string
// @Security     BearerAuth
// @Router       /accounts/{number}/rules [get]
func (server *Server) handleGetReglesForCompte(ctx *gin.Context) {
	numeroCompte := ctx.Param("numeroCompte")
	regles, err := server.store.GetReglesForCompte(ctx, numeroCompte)
	if err != nil {
		ctx.JSON(500, gin.H{"error": "Impossible de récupérer les règles pour ce compte", "details": err.Error()})
		return
	}
	ctx.JSON(200, regles)
}

// ===================== HANDLERS PROTÉGÉS - PROFILS CLIENTS =====================

// @Summary      Lister les profils clients
// @Description  Récupère tous les profils clients.
// @Tags         Profils
// @Produce      json
// @Success      200 {array}   models.ProfilClient
// @Failure      403 {object}  map[string]string
// @Security     BearerAuth
// @Router       /profiles [get]
func (server *Server) handleGetProfilsClient(ctx *gin.Context) {
	profils, err := server.store.GetProfilsClient(ctx)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Impossible de récupérer les profils"})
		return
	}
	ctx.JSON(http.StatusOK, profils)
}

// @Summary      Créer un profil client
// @Description  Crée un nouveau profil client.
// @Tags         Profils
// @Accept       json
// @Produce      json
// @Param        profil  body      models.ProfilClient  true  "Détails du profil"
// @Success      201     {object}  models.ProfilClient
// @Failure      403     {object}  map[string]string
// @Security     BearerAuth
// @Router       /profiles [post]
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

// @Summary      Mettre à jour un profil client
// @Description  Modifie un profil client.
// @Tags         Profils
// @Accept       json
// @Produce      json
// @Param        id      path      string               true  "ID du profil"
// @Param        profil  body      models.ProfilClient  true  "Nouvelles données du profil"
// @Success      200     {object}  models.ProfilClient
// @Failure      403     {object}  map[string]string
// @Security     BearerAuth
// @Router       /profiles/{id} [put]
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
// @Description  Supprime un profil client.
// @Tags         Profils
// @Produce      json
// @Param        id  path      string  true  "ID du profil"
// @Success      204 "No Content"
// @Failure      403 {object}  map[string]string
// @Security     BearerAuth
// @Router       /profiles/{id} [delete]
func (server *Server) handleDeleteProfilClient(ctx *gin.Context) {
	id := ctx.Param("id")
	if err := server.store.DeleteProfilClient(ctx, id); err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Impossible de supprimer le profil: " + err.Error()})
		return
	}
	ctx.Status(http.StatusNoContent)
}

// ===================== HANDLERS PROTÉGÉS - COMPTES CLIENTS =====================

// @Summary      Lister toutes les transactions
// @Description  Récupère la liste de toutes les transactions enregistrées.
// @Tags         Comptes
// @Produce      json
// @Success      200  {array}   models.Transaction
// @Failure      401  {object}  map[string]string
// @Security     BearerAuth
// @Router       /accounts [get]
func (server *Server) handleGetTransactions(ctx *gin.Context) {
	transactions, err := server.store.GetTransactions(ctx)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Impossible de récupérer les transactions"})
		return
	}
	ctx.JSON(http.StatusOK, transactions)
}

// @Summary      Résumé du Dashboard (Optimisé)
// @Description  Retourne toutes les données nécessaires au dashboard en UNE SEULE requête.
// @Tags         Dashboard
// @Produce      json
// @Success      200  {object}  gin.H
// @Failure      500  {object}  gin.H
// @Router       /api/v1/dashboard/summary [get]
func (server *Server) handleGetDashboardSummary(ctx *gin.Context) {
	summary, err := server.store.GetDashboardSummary(ctx)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Impossible de récupérer le résumé du dashboard"})
		return
	}
	ctx.JSON(http.StatusOK, summary)
}

// @Summary      Créer un client
// @Description  Crée un nouveau client et l'associe à un profil.
// @Tags         Comptes
// @Accept       json
// @Produce      json
// @Param        compte  body      models.Compte  true  "Détails du compte"
// @Success      201     {object}  models.Compte
// @Failure      403     {object}  map[string]string
// @Security     BearerAuth
// @Router       /accounts [post]
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

// @Summary      Obtenir le nombre de comptes surveillés
// @Description  Retourne le nombre de comptes uniques présents dans la table suivi_blacklist.
// @Tags         Surveillance
// @Produce      json
// @Success      200  {object}  object{count=int,comptes=[]string}
// @Failure      500  {object}  map[string]string
// @Security     BearerAuth
// @Router       /accounts/monitored [get]
func (server *Server) handleGetComptesSurveilles(ctx *gin.Context) {
	count, err := server.store.GetComptesSurveillesCount(ctx)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Impossible de récupérer le nombre de comptes surveillés", "details": err.Error()})
		return
	}
	comptes, err := server.store.GetComptesSurveilles(ctx)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Impossible de récupérer la liste des comptes surveillés", "details": err.Error()})
		return
	}
	ctx.JSON(http.StatusOK, gin.H{"count": count, "comptes": comptes})
}

// ===================== HANDLERS PROTÉGÉS - CONFIGURATION =====================

// @Summary      Obtenir la liste des pays à risque
// @Description  Récupère la liste actuelle des pays à risque.
// @Tags         Configuration
// @Produce      json
// @Success      200   {object}  object{liste_pays=[]string}
// @Failure      500   {object}  map[string]string
// @Security     BearerAuth
// @Router       /configuration/risk-countries [get]
func (server *Server) handleGetPaysRisque(ctx *gin.Context) {
	regle, err := server.store.GetReglePaysRisqueActive(ctx)
	if err != nil || regle == nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Impossible de récupérer la liste des pays à risque", "details": err.Error()})
		return
	}
	ctx.JSON(http.StatusOK, gin.H{"liste_pays": regle.ListePays})
}

// @Summary      Ajouter un pays à la liste des pays à risque
// @Description  Ajoute un pays à la liste sans écraser les autres.
// @Tags         Configuration
// @Accept       json
// @Produce      json
// @Param        pays  body      object{pays=string}  true  "Pays à ajouter"
// @Success      200   {object}  map[string]string
// @Failure      400   {object}  map[string]string
// @Failure      500   {object}  map[string]string
// @Security     BearerAuth
// @Router       /configuration/risk-countries [post]
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

// @Summary      Mettre à jour la liste des pays à risque
// @Description  Modifie la liste des pays à risque.
// @Tags         Configuration
// @Accept       json
// @Produce      json
// @Param        pays  body      object{liste_pays=[]string}  true  "Nouvelle liste de pays"
// @Success      200   {object}  map[string]string
// @Failure      400   {object}  map[string]string
// @Failure      500   {object}  map[string]string
// @Security     BearerAuth
// @Router       /configuration/risk-countries/{country} [put]
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

// @Summary      Retirer un pays de la liste des pays à risque
// @Description  Retire un pays de la liste sans toucher aux autres.
// @Tags         Configuration
// @Accept       json
// @Produce      json
// @Param        pays  body      object{pays=string}  true  "Pays à retirer"
// @Success      200   {object}  map[string]string
// @Failure      400   {object}  map[string]string
// @Failure      500   {object}  map[string]string
// @Security     BearerAuth
// @Router       /configuration/risk-countries/{country} [delete]
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

// ===================== HANDLERS PROTÉGÉS - PROFIL UTILISATEUR =====================

// @Summary      Obtenir le profil de l'utilisateur connecté
// @Description  Renvoie les informations de l'utilisateur actuellement authentifié.
// @Tags         Profil
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

// ===================== INITIALISATION SERVEUR & ROUTES =====================

// NewServer crée une nouvelle instance de serveur avec toutes les routes configurées.
func NewServer(store *store.Store) *Server {
	server := &Server{store: store}
	router := gin.Default()

	// Route pour la documentation Swagger UI
	router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	// --- Routes API v1 (STANDARD ET NEUTRE) ---
	apiV1 := router.Group("/api/v1")

	// ========== AUTHENTIFICATION (Public) ==========
	authRoutes := apiV1.Group("/auth")
	{
		authRoutes.POST("/login", server.handleLogin)
		authRoutes.POST("/token", handleGetSystemToken)
	}

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
			ctx.JSON(200, gin.H{"status": typeEntite})
		})
	}

	// ========== ANOMALIES / TRANSACTIONS (Protégé) ==========
	anomalyRoutes := apiV1.Group("/anomalies")
	anomalyRoutes.Use(jwtAuthMiddleware(), roleMiddleware("admin", "analyst", "system"))
	{
		anomalyRoutes.POST("/analyze", server.handleAnalyzeTransaction)
		anomalyRoutes.GET("", server.handleGetAlertes)
		anomalyRoutes.GET("/metrics", server.handleGetAlertMetrics)
		anomalyRoutes.GET("/:id", server.handleGetAlertes)
		anomalyRoutes.PUT("/:id/status", server.handleUpdateAlerteStatus)
	}

	// ========== ENTITÉS / BLACKLIST (Protégé) ==========
	entityRoutes := apiV1.Group("/entities")
	entityRoutes.Use(jwtAuthMiddleware(), roleMiddleware("admin", "analyst"))
	{
		entityRoutes.GET("", server.handleGetBlacklistEntries)
		entityRoutes.POST("", server.handleCreateBlacklistEntry)
		entityRoutes.PUT("/:id", server.handleUpdateBlacklistEntry)
		entityRoutes.DELETE("/:id", server.handleDeleteBlacklistEntry)
		entityRoutes.POST("/:identifier/verify", server.handleCheckBlacklist)
	}

	// ========== RÈGLES DE CONTRÔLE (Protégé) ==========
	rulesRoutes := apiV1.Group("/rules")
	rulesRoutes.Use(jwtAuthMiddleware(), roleMiddleware("admin", "analyst"))
	{
		rulesRoutes.GET("", server.handleGetAllRegles)
		rulesRoutes.GET("/:id", server.handleGetAllRegles)
		rulesRoutes.GET("/:id/profiles", server.handleGetReglesByProfil)
		rulesRoutes.POST("", checkRole("admin"), server.handleCreateRegleControle)
		rulesRoutes.PUT("/:id", checkRole("admin"), server.handleUpdateRegleControle)
		rulesRoutes.DELETE("/:id", checkRole("admin"), server.handleDeleteRegleControle)
	}

	// ========== DASHBOARD (Protégé - Optimisé) ==========
	dashboardRoutes := apiV1.Group("/dashboard")
	dashboardRoutes.Use(jwtAuthMiddleware(), roleMiddleware("admin", "analyst", "system", "reviewer"))
	{
		dashboardRoutes.GET("/summary", server.handleGetDashboardSummary)
	}

	// ========== PROFILS CLIENTS (Protégé) ==========
	profileRoutes := apiV1.Group("/profiles")
	profileRoutes.Use(jwtAuthMiddleware(), roleMiddleware("admin", "analyst"))
	{
		profileRoutes.GET("", server.handleGetProfilsClient)
		profileRoutes.GET("/:id", server.handleGetProfilsClient)
		profileRoutes.GET("/:id/rules", server.handleGetReglesByProfil)
		profileRoutes.POST("", checkRole("admin"), server.handleCreateProfilClient)
		profileRoutes.PUT("/:id", checkRole("admin"), server.handleUpdateProfilClient)
		profileRoutes.DELETE("/:id", checkRole("admin"), server.handleDeleteProfilClient)
	}

	// ========== COMPTES CLIENTS (Protégé) ==========
	accountRoutes := apiV1.Group("/accounts")
	accountRoutes.Use(jwtAuthMiddleware(), roleMiddleware("admin", "analyst"))
	{
		accountRoutes.GET("", server.handleGetTransactions)
		accountRoutes.POST("", checkRole("admin"), server.handleCreateCompte)
		accountRoutes.GET("/:number/rules", server.handleGetReglesForCompte)
		accountRoutes.GET("/monitored", server.handleGetComptesSurveilles)
	}

	// ========== CONFIGURATION SYSTÈME (Admin uniquement) ==========
	configRoutes := apiV1.Group("/configuration")
	configRoutes.Use(jwtAuthMiddleware(), roleMiddleware("admin"))
	{
		configRoutes.GET("/risk-countries", server.handleGetPaysRisque)
		configRoutes.POST("/risk-countries", server.handleAddPaysRisque)
		configRoutes.PUT("/risk-countries/:country", server.handleUpdatePaysRisque)
		configRoutes.DELETE("/risk-countries/:country", server.handleRemovePaysRisque)
	}

	// ========== PROFIL UTILISATEUR (Protégé) ==========
	userRoutes := apiV1.Group("/profile")
	userRoutes.Use(jwtAuthMiddleware())
	{
		userRoutes.GET("", server.handleGetProfile)
	}

	server.router = router
	return server
}

// ===================== DEMARRAGE SERVEUR =====================

// Start lance le serveur HTTP sur une adresse spécifique.
func (server *Server) Start(address string) error {
	return server.router.Run(address)
}

// ===================== FONCTIONS UTILITAIRES =====================

// HashPassword hache un mot de passe en utilisant l'algorithme bcrypt.
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
