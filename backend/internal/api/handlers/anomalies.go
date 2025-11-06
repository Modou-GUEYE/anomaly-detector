package handlers

import (
	"anomaly-detector/internal/core"
	"anomaly-detector/internal/models"
	"anomaly-detector/internal/store"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

// AnomaliesHandler gère les opérations liées aux anomalies et alertes
type AnomaliesHandler struct {
	store *store.Store
}

// NewAnomaliesHandler crée une nouvelle instance du handler d'anomalies
func NewAnomaliesHandler(store *store.Store) *AnomaliesHandler {
	return &AnomaliesHandler{store: store}
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
// @Router       /anomalies/analyze [post]
func (h *AnomaliesHandler) HandleAnalyzeTransaction(ctx *gin.Context) {
	var tx models.Transaction
	if err := ctx.ShouldBindJSON(&tx); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Format de transaction invalide: " + err.Error()})
		return
	}

	// Récupérer l'utilisateur connecté pour l'audit logging
	userID, exists := ctx.Get("userID")
	if !exists {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Utilisateur non identifié"})
		return
	}

	// Création automatique du compte payeur si inexistant
	if tx.NumeroComptePayeur != "" {
		clientPayeur, err := h.store.GetCompteByCompte(ctx, tx.NumeroComptePayeur)
		if err != nil || clientPayeur.ID == "" {
			profilID, _ := h.store.GetProfilClientIDByCode(ctx, tx.TypeClientPayeur)
			_, errCreate := h.store.CreateCompte(ctx, &models.Compte{
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
			}, userID.(string))
			if errCreate != nil {
				ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Erreur lors de la création du client payeur", "details": errCreate.Error()})
				return
			}
		}
	}

	// Création automatique du compte payé si inexistant
	if tx.NumeroComptePaye != "" {
		clientPaye, err := h.store.GetCompteByCompte(ctx, tx.NumeroComptePaye)
		if err != nil || clientPaye.ID == "" {
			profilID, _ := h.store.GetProfilClientIDByCode(ctx, tx.TypeClientPaye)
			_, errCreate := h.store.CreateCompte(ctx, &models.Compte{
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
			}, userID.(string))
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
	rulesEngine := core.NewRulesEngine(h.store)
	alertes, err := rulesEngine.AnalyzeTransaction(ctx, &tx)
	if err != nil {
		if errors.Is(err, core.ErrTransactionRefused) {
			h.handleRefusedTransaction(ctx, &tx, alertes, userID.(string))
			return
		}
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	h.handleValidTransaction(ctx, &tx, alertes, userID.(string))
}

// handleRefusedTransaction traite une transaction refusée
func (h *AnomaliesHandler) handleRefusedTransaction(ctx *gin.Context, tx *models.Transaction, alertes []*models.Alerte, userID string) {
	createdTx, txErr := h.store.CreateTransaction(ctx, tx)
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
		saved, _ := h.store.CreateAlerte(ctx, alerte, userID)
		if saved != nil {
			savedBloqueAlertes = append(savedBloqueAlertes, saved)
		}
		h.createSuiviIfNeeded(ctx, createdTx.ID, alerte)
	}
	ctx.JSON(http.StatusForbidden, gin.H{"status": "transaction_bloquee", "alertes": savedBloqueAlertes})
}

// handleValidTransaction traite une transaction valide
func (h *AnomaliesHandler) handleValidTransaction(ctx *gin.Context, tx *models.Transaction, alertes []*models.Alerte, userID string) {
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
		h.handleRefusedTransaction(ctx, tx, bloqueAlertes, userID)
		return
	}

	createdTx, err := h.store.CreateTransaction(ctx, tx)
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
		saved, _ := h.store.CreateAlerte(ctx, alerte, userID)
		savedAlertes = append(savedAlertes, saved)
		h.createSuiviIfNeeded(ctx, createdTx.ID, alerte)
	}

	if len(savedAlertes) > 0 {
		ctx.JSON(http.StatusCreated, gin.H{"status": "anomalie_detectee", "alertes": savedAlertes})
	} else {
		ctx.JSON(http.StatusOK, gin.H{"status": "aucune_anomalie_detectee"})
	}
}

// createSuiviIfNeeded ajoute à suivi_blacklist si nécessaire
func (h *AnomaliesHandler) createSuiviIfNeeded(ctx *gin.Context, transactionID string, alerte *models.Alerte) {
	const partnerBankCode = "SN079"
	if alerte.NumeroCompte != "" && len(alerte.NumeroCompte) >= 5 && alerte.NumeroCompte[:5] == partnerBankCode {
		if h.shouldCreateSuivi(alerte) {
			_ = h.store.CreateSuiviBlacklist(ctx, transactionID, alerte.NumeroCompte, "surveiller", alerte.Type)
		}
	}
}

// shouldCreateSuivi détermine si un suivi doit être créé pour une alerte
func (h *AnomaliesHandler) shouldCreateSuivi(alerte *models.Alerte) bool {
	criticalTypes := []string{"Interdit", "Blacklist", "PEP", "Surveillés", "Pays à risque", "montant", "frequence"}

	if alerte.Action == "surveiller" {
		return true
	}

	for _, criticalType := range criticalTypes {
		if alerte.Type == criticalType {
			return true
		}
	}

	return false
}

// HandleGetAlertMetrics retourne les métriques d'alertes agrégées par mois et par type
func (h *AnomaliesHandler) HandleGetAlertMetrics(ctx *gin.Context) {
	months := 12
	if mStr := ctx.Query("months"); mStr != "" {
		if m, err := strconv.Atoi(mStr); err == nil && m > 0 {
			months = m
		}
	}
	metrics, err := h.store.GetAlertCountsByMonthAndType(ctx, months)
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
func (h *AnomaliesHandler) HandleGetAlertes(ctx *gin.Context) {
	alertes, err := h.store.GetAlertes(ctx)
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
func (h *AnomaliesHandler) HandleUpdateAlerteStatus(ctx *gin.Context) {
	alerteID := ctx.Param("id")
	var req struct {
		Statut string `json:"statut" binding:"required"`
	}
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Le champ 'statut' est requis"})
		return
	}
	analysteID, _ := ctx.Get("userID")
	alerte, err := h.store.UpdateAlerteStatus(ctx, alerteID, analysteID.(string), req.Statut)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Impossible de mettre à jour l'alerte", "details": err.Error()})
		return
	}
	ctx.JSON(http.StatusOK, alerte)
}
