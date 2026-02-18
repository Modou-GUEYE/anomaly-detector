package handlers

import (
	"anomaly-detector/internal/models"
	"anomaly-detector/internal/store"
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// RulesHandler gère les opérations liées aux règles de contrôle
type RulesHandler struct {
	store *store.Store
}

// NewRulesHandler crée une nouvelle instance du handler de règles
func NewRulesHandler(store *store.Store) *RulesHandler {
	return &RulesHandler{store: store}
}

const (
	defaultMontantPeriode   = "7d"
	defaultFrequencePeriode = "24h"
)

// allowedTypes répertorie les types de règles autorisées (évolutif)
var allowedTypes = map[string]struct{}{
	"montant":     {},
	"frequence":   {},
	"pays_risque": {},
}

var allowedActions = map[string]struct{}{
	"bloquer":    {},
	"surveiller": {},
	"notifier":   {},
}

// normalizeAndValidateRegle applique les valeurs par défaut et valide les champs selon le type.
// Elle ne persiste rien : uniquement transformation/validation.
func normalizeAndValidateRegle(r *models.RegleControle) error {
	r.TypeRegle = strings.TrimSpace(strings.ToLower(r.TypeRegle))
	if _, ok := allowedTypes[r.TypeRegle]; !ok {
		return fmt.Errorf("typeRegle invalide: %s", r.TypeRegle)
	}

	// Action par défaut si vide
	if strings.TrimSpace(r.Action) == "" {
		r.Action = "surveiller"
	}
	// Normaliser action et valider
	r.Action = strings.TrimSpace(strings.ToLower(r.Action))
	if _, ok := allowedActions[r.Action]; !ok {
		r.Action = "surveiller"
	}

	// Nom et description basiques requis
	r.NomRegle = strings.TrimSpace(r.NomRegle)
	r.Description = strings.TrimSpace(r.Description)
	if r.NomRegle == "" {
		return fmt.Errorf("nomRegle est obligatoire")
	}
	if r.Description == "" {
		return fmt.Errorf("description est obligatoire")
	}

	switch r.TypeRegle {
	case "montant":
		// Périodes autorisées pour montant
		allowed := map[string]struct{}{"24h": {}, "7d": {}, "30d": {}}
		if r.PeriodeReference == "" {
			r.PeriodeReference = defaultMontantPeriode
		} else {
			p := strings.ToLower(strings.TrimSpace(r.PeriodeReference))
			if _, ok := allowed[p]; !ok {
				// Valeur non autorisée -> défaut sécurisé
				r.PeriodeReference = defaultMontantPeriode
			} else {
				r.PeriodeReference = p
			}
		}
		if r.Seuil <= 0 {
			return fmt.Errorf("seuil doit être > 0 pour une règle de type montant")
		}
	case "frequence":
		// Périodes autorisées pour frequence
		allowed := map[string]struct{}{"24h": {}, "7d": {}, "30d": {}}
		if r.PeriodeReference == "" {
			r.PeriodeReference = defaultFrequencePeriode
		} else {
			p := strings.ToLower(strings.TrimSpace(r.PeriodeReference))
			if _, ok := allowed[p]; !ok {
				r.PeriodeReference = defaultFrequencePeriode
			} else {
				r.PeriodeReference = p
			}
		}
		if r.Seuil <= 0 {
			return fmt.Errorf("seuil doit être > 0 pour une règle de type frequence")
		}
	default:
		// Pour pays_risque: pas de periodeReference, pas de seuil obligatoire
		r.PeriodeReference = ""
	}
	return nil
}

// HandleGetAllRegles liste toutes les règles de contrôle
func (h *RulesHandler) HandleGetAllRegles(ctx *gin.Context) {
	regles, err := h.store.GetAllRegles(ctx)
	if err != nil {
		ctx.JSON(500, gin.H{"error": "Impossible de récupérer les règles"})
		return
	}
	ctx.JSON(200, regles)
}

// HandleGetRegleByID retourne une règle unique par son ID
func (h *RulesHandler) HandleGetRegleByID(ctx *gin.Context) {
	id := ctx.Param("id")
	regle, err := h.store.GetRegleByID(ctx, id)
	if err != nil {
		ctx.JSON(404, gin.H{"error": "Règle introuvable"})
		return
	}
	ctx.JSON(200, regle)
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
func (h *RulesHandler) HandleCreateRegleControle(ctx *gin.Context) {
	var req models.RegleControle
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "JSON invalide"})
		return
	}

	// Récupérer l'utilisateur connecté
	userID, exists := ctx.Get("userID")
	if !exists {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Utilisateur non identifié"})
		return
	}

	if err := normalizeAndValidateRegle(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	regle, err := h.store.CreateRegleControle(ctx, &req, userID.(string))
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
func (h *RulesHandler) HandleUpdateRegleControle(ctx *gin.Context) {
	id := ctx.Param("id")
	var req models.RegleControle
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "JSON invalide"})
		return
	}

	// Récupérer l'utilisateur connecté
	userID, exists := ctx.Get("userID")
	if !exists {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Utilisateur non identifié"})
		return
	}

	if err := normalizeAndValidateRegle(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	regle, err := h.store.UpdateRegleControle(ctx, id, &req, userID.(string))
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
func (h *RulesHandler) HandleDeleteRegleControle(ctx *gin.Context) {
	id := ctx.Param("id")

	// Récupérer l'ID utilisateur depuis le token
	userID, exists := ctx.Get("userID")
	if !exists {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Utilisateur non identifié"})
		return
	}

	if err := h.store.DeleteRegleControle(ctx, id, userID.(string)); err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Impossible de supprimer la règle",
			"details": err.Error(),
		})
		return
	}
	ctx.Status(http.StatusNoContent)
}

// @Summary      Activer/Désactiver une règle de contrôle
// @Description  Change le statut actif/inactif d'une règle de contrôle.
// @Tags         Règles
// @Accept       json
// @Produce      json
// @Param        id     path      string                 true  "ID de la règle"
// @Param        status body      object{active=bool}   true  "Nouveau statut"
// @Success      200    {object}  models.RegleControle
// @Failure      403    {object}  map[string]string
// @Security     BearerAuth
// @Router       /rules/{id}/status [patch]
func (h *RulesHandler) HandleToggleRegleStatus(ctx *gin.Context) {
	id := ctx.Param("id")
	var req struct {
		Active bool `json:"active"`
	}
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("JSON invalide: %v", err)})
		return
	}

	// Récupérer l'utilisateur connecté
	userID, exists := ctx.Get("userID")
	if !exists {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Utilisateur non identifié"})
		return
	}

	// Convertir le booléen en string pour la méthode existante
	status := "Inactif"
	if req.Active {
		status = "Actif"
	}

	// Met à jour le statut avec logging d'audit
	updatedRegle, err := h.store.UpdateRegleControleStatus(ctx, id, status, userID.(string))
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Impossible de mettre à jour le statut: %v", err)})
		return
	}

	ctx.JSON(http.StatusOK, updatedRegle)
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
func (h *RulesHandler) HandleGetReglesByProfil(ctx *gin.Context) {
	profilID := ctx.Param("id")
	regles, err := h.store.GetReglesByProfilID(ctx, profilID)
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
func (h *RulesHandler) HandleGetReglesForCompte(ctx *gin.Context) {
	numeroCompte := ctx.Param("numeroCompte")
	regles, err := h.store.GetReglesForCompte(ctx, numeroCompte)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Impossible de récupérer les règles pour ce compte"})
		return
	}
	ctx.JSON(http.StatusOK, regles)
}
