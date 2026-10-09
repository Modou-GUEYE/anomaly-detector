package handlers

import (
	"anomaly-detector/internal/core"
	"anomaly-detector/internal/models"
	"anomaly-detector/internal/store"
	"errors"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
)

// AnomaliesHandler gère les opérations liées aux anomalies et alertes
type AnomaliesHandler struct {
	store *store.Store
}

type accountPayload struct {
	nomClient    string
	numeroCompte string
	typeCompte   string
	typeClient   string
	categorie    string
	pays         string
	ville        string
	adresse      string
}

func partnerBankCodeFromEnv() (string, error) {
	partnerBankCode := strings.TrimSpace(os.Getenv("PARTNER_BANK_CODE"))
	if partnerBankCode == "" {
		return "", core.ErrMissingPartnerBankCode
	}
	return partnerBankCode, nil
}

func isPartnerAccount(numeroCompte string, partnerBankCode string) bool {
	return strings.HasPrefix(numeroCompte, partnerBankCode)
}

func hasPartnerAccount(tx *models.Transaction, partnerBankCode string) bool {
	return isPartnerAccount(tx.NumeroComptePayeur, partnerBankCode) || isPartnerAccount(tx.NumeroComptePaye, partnerBankCode)
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

	// Vérifier que l'appelant est identifié (les écritures sont auditées sous l'acteur système "MDA")
	if _, exists := ctx.Get("userID"); !exists {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Utilisateur non identifié"})
		return
	}

	payeur := accountPayload{
		nomClient:    tx.NomPayeur,
		numeroCompte: tx.NumeroComptePayeur,
		typeCompte:   tx.TypeComptePayeur,
		typeClient:   tx.TypeClientPayeur,
		categorie:    tx.CategoriePayeur,
		pays:         tx.PaysPayeur,
		ville:        tx.VillePayeur,
		adresse:      tx.AdressePayeur,
	}
	paye := accountPayload{
		nomClient:    tx.NomPaye,
		numeroCompte: tx.NumeroComptePaye,
		typeCompte:   tx.TypeComptePaye,
		typeClient:   tx.TypeClientPaye,
		categorie:    tx.CategoriePaye,
		pays:         tx.PaysPaye,
		ville:        tx.VillePaye,
		adresse:      tx.AdressePaye,
	}
	if err := h.ensureAccounts(ctx, payeur, paye); err != nil {
		return
	}

	partnerBankCode, err := partnerBankCodeFromEnv()
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if !hasPartnerAccount(&tx, partnerBankCode) {
		ctx.JSON(http.StatusOK, gin.H{"status": "transaction_non_analysee", "raison": "Aucun compte partenaire impliqué dans la transaction."})
		return
	}

	h.analyzeAndRespond(ctx, &tx)
}

// ensureAccounts crée les comptes payeur/payé absents en une seule transaction SQL.
func (h *AnomaliesHandler) ensureAccounts(ctx *gin.Context, payloads ...accountPayload) error {
	comptes := make([]*models.Compte, 0, len(payloads))
	for _, payload := range payloads {
		if payload.numeroCompte == "" {
			continue
		}
		profilID, errProfil := h.store.GetProfilClientIDByCode(ctx, payload.typeClient)
		if errProfil != nil {
			ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Profil client introuvable pour le type: " + payload.typeClient, "details": errProfil.Error()})
			return errProfil
		}
		comptes = append(comptes, &models.Compte{
			NomClient:      payload.nomClient,
			NumeroCompte:   payload.numeroCompte,
			TypeCompte:     payload.typeCompte,
			TypeClient:     payload.typeClient,
			Categorie:      payload.categorie,
			Pays:           payload.pays,
			Ville:          payload.ville,
			Adresse:        payload.adresse,
			ProfilClientID: profilID,
		})
	}

	if err := h.store.EnsureComptes(ctx, comptes); err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Erreur lors de la création des comptes payeur/payé", "details": err.Error()})
		return err
	}
	return nil
}

func (h *AnomaliesHandler) analyzeAndRespond(ctx *gin.Context, tx *models.Transaction) {
	rulesEngine := core.NewRulesEngine(h.store)
	alertes, err := rulesEngine.AnalyzeTransaction(ctx, tx)
	if err != nil {
		if errors.Is(err, core.ErrTransactionRefused) {
			h.handleRefusedTransaction(ctx, tx, alertes)
			return
		}
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	h.handleValidTransaction(ctx, tx, alertes)
}

// handleRefusedTransaction traite une transaction refusée
func (h *AnomaliesHandler) handleRefusedTransaction(ctx *gin.Context, tx *models.Transaction, alertes []*models.Alerte) {
	for _, alerte := range alertes {
		if strings.EqualFold(alerte.Action, "bloquer") {
			alerte.Statut = "Traité"
		}
	}
	if err := h.store.SaveAnalysis(ctx, tx, h.analysisAlerts(alertes)); err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Impossible d'enregistrer la transaction: " + err.Error()})
		return
	}
	ctx.JSON(http.StatusForbidden, gin.H{"status": "transaction_bloquee", "alertes": alertes})
}

// handleValidTransaction traite une transaction valide
func (h *AnomaliesHandler) handleValidTransaction(ctx *gin.Context, tx *models.Transaction, alertes []*models.Alerte) {
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
		h.handleRefusedTransaction(ctx, tx, bloqueAlertes)
		return
	}

	if err := h.store.SaveAnalysis(ctx, tx, h.analysisAlerts(autresAlertes)); err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Impossible d'enregistrer la transaction: " + err.Error()})
		return
	}

	if len(autresAlertes) > 0 {
		ctx.JSON(http.StatusCreated, gin.H{"status": "anomalie_detectee", "alertes": autresAlertes})
	} else {
		ctx.JSON(http.StatusOK, gin.H{"status": "aucune_anomalie_detectee"})
	}
}

// analysisAlerts indique pour chaque alerte si une entrée suivi_blacklist doit être créée.
func (h *AnomaliesHandler) analysisAlerts(alertes []*models.Alerte) []store.AnalysisAlert {
	partnerBankCode, err := partnerBankCodeFromEnv()
	result := make([]store.AnalysisAlert, 0, len(alertes))
	for _, alerte := range alertes {
		createSuivi := err == nil && alerte.NumeroCompte != "" &&
			isPartnerAccount(alerte.NumeroCompte, partnerBankCode) && h.shouldCreateSuivi(alerte)
		result = append(result, store.AnalysisAlert{Alerte: alerte, CreateSuivi: createSuivi})
	}
	return result
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
// @Description  Récupère la liste des alertes générées par le système, les plus récentes d'abord. Sans `limit`, toutes les alertes sont retournées.
// @Tags         Anomalies
// @Produce      json
// @Param        limit  query     int  false  "Nombre maximum d'alertes (max 500)"
// @Param        page   query     int  false  "Numéro de page (à partir de 1)"
// @Success      200  {array}   models.Alerte
// @Failure      401  {object}  map[string]string
// @Security     BearerAuth
// @Router       /anomalies [get]
func (h *AnomaliesHandler) HandleGetAlertes(ctx *gin.Context) {
	// Pagination optionnelle (?limit=20&page=2) : sans paramètre, toutes les alertes sont retournées.
	limit, _ := strconv.Atoi(ctx.Query("limit"))
	page, _ := strconv.Atoi(ctx.Query("page"))
	if limit > 500 {
		limit = 500
	}
	offset := 0
	if limit > 0 && page > 1 {
		offset = (page - 1) * limit
	}
	alertes, err := h.store.GetAlertesPage(ctx, limit, offset)
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
