package handlers

import (
	"anomaly-detector/internal/models"
	"anomaly-detector/internal/store"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// EntitiesHandler gère les opérations liées aux entités blacklistées
type EntitiesHandler struct {
	store *store.Store
}

// NewEntitiesHandler crée une nouvelle instance du handler d'entités
func NewEntitiesHandler(store *store.Store) *EntitiesHandler {
	return &EntitiesHandler{store: store}
}

// @Summary      Lister les entrées de la blacklist
// @Description  Récupère toutes les entités de la liste de surveillance.
// @Tags         Entités
// @Produce      json
// @Success      200  {array}   models.EntiteBlacklist
// @Failure      403  {object}  map[string]string
// @Security     BearerAuth
// @Router       /entities [get]
func (h *EntitiesHandler) HandleGetBlacklistEntries(ctx *gin.Context) {
	entries, err := h.store.GetBlacklistEntries(ctx)
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
func (h *EntitiesHandler) HandleCreateBlacklistEntry(ctx *gin.Context) {
	var req models.EntiteBlacklist
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

	entry, err := h.store.CreateBlacklistEntry(ctx, &req, userID.(string))
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
func (h *EntitiesHandler) HandleUpdateBlacklistEntry(ctx *gin.Context) {
	id := ctx.Param("id")
	var req models.EntiteBlacklist
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

	updatedEntry, err := h.store.UpdateBlacklistEntry(ctx, id, &req, userID.(string))
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
func (h *EntitiesHandler) HandleDeleteBlacklistEntry(ctx *gin.Context) {
	id := ctx.Param("id")

	// Récupérer l'utilisateur connecté
	userID, exists := ctx.Get("userID")
	if !exists {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Utilisateur non identifié"})
		return
	}

	if err := h.store.DeleteBlacklistEntry(ctx, id, userID.(string)); err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Impossible de supprimer l'entrée"})
		return
	}
	ctx.Status(http.StatusNoContent)
}

// HandleCheckBlacklist vérifie si un compte est dans la blacklist
func (h *EntitiesHandler) HandleCheckBlacklist(ctx *gin.Context) {
	var req struct {
		NumeroCompte string `json:"numero_compte" binding:"required"`
	}
	if err := ctx.ShouldBindJSON(&req); err != nil || strings.TrimSpace(req.NumeroCompte) == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Numéro de compte manquant ou invalide"})
		return
	}
	status, err := h.store.IsAccountOnBlacklist(ctx, req.NumeroCompte)
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
