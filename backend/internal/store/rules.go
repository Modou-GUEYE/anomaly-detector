package store

import (
	"context"
	"errors"
	"fmt"

	"gorm.io/gorm"

	"anomaly-detector/internal/models"
)

// ===================== REGLES DE CONTROLE =====================

// CreateRegleControle insère une nouvelle règle de contrôle dans la base de données.
func (s *Store) CreateRegleControle(ctx context.Context, regle *models.RegleControle, userID string) (*models.RegleControle, error) {
	// --- GORM ---
	result := s.gormDB.WithContext(ctx).Create(regle)
	if result.Error != nil {
		// Logger l'échec
		if userID != "" {
			actionDescription := fmt.Sprintf("Création de la règle '%s'", regle.NomRegle)
			details := fmt.Sprintf("Échec lors de la création de la règle de contrôle: %s, Erreur: %v", regle.NomRegle, result.Error)
			_ = s.LogAction(ctx, userID, actionDescription, "Règles de Contrôle", "", details, "échec")
		}
		return nil, result.Error
	}

	// Logging d'audit automatique si userID fourni
	if userID != "" {
		actionDescription := fmt.Sprintf("Création de la règle '%s'", regle.NomRegle)
		err := s.LogAction(ctx, userID, actionDescription, "Règles de Contrôle", regle.ID, fmt.Sprintf("Création de la règle de contrôle: %s", regle.NomRegle), "succès")
		if err != nil {
			// Ne pas faire échouer la création si le logging échoue, mais logger l'erreur
			fmt.Printf("Erreur lors du logging d'audit pour la création de règle %s: %v\n", regle.ID, err)
		}
	}

	return regle, nil
}

// GetReglesByProfilID récupère toutes les règles associées à un profil client donné.
func (s *Store) GetReglesByProfilID(ctx context.Context, profilID string) ([]models.RegleControle, error) {
	var regles []models.RegleControle
	result := s.gormDB.WithContext(ctx).Where("profil_client_id = ?", profilID).Find(&regles)
	if result.Error != nil {
		return nil, result.Error
	}
	return regles, nil
}

// UpdateRegleControle met à jour une règle de contrôle existante.
func (s *Store) UpdateRegleControle(ctx context.Context, id string, regle *models.RegleControle, userID string) (*models.RegleControle, error) {
	regle.ID = id
	// Construire dynamiquement les champs à mettre à jour (éviter d'écraser les champs non fournis)
	updateFields := map[string]interface{}{
		"nom_regle":         regle.NomRegle,
		"description":       regle.Description,
		"type_regle":        regle.TypeRegle,
		"criticite":         regle.Criticite,
		"seuil":             regle.Seuil,
		"periode_reference": regle.PeriodeReference,
		"active":            regle.Active,
		"profil_client_id":  regle.ProfilClientID,
	}

	// Seulement mettre à jour liste_pays si elle est explicitement fournie (non nil)
	// Cela évite d'effacer accidentellement la liste des pays à risque
	if regle.ListePays != nil {
		updateFields["liste_pays"] = regle.ListePays
	}

	result := s.gormDB.WithContext(ctx).Model(&models.RegleControle{}).
		Where("id = ?", id).
		Updates(updateFields)
	if result.Error != nil {
		return nil, result.Error
	}
	// Recharger la règle mise à jour
	var updated models.RegleControle
	if err := s.gormDB.WithContext(ctx).First(&updated, "id = ?", id).Error; err != nil {
		return nil, err
	}
	*regle = updated

	// Logging d'audit automatique si userID fourni
	if userID != "" {
		actionDescription := fmt.Sprintf("Modification de la règle '%s'", updated.NomRegle)
		err := s.LogAction(ctx, userID, actionDescription, "Règles de Contrôle", id, fmt.Sprintf("Modification de la règle de contrôle: %s", updated.NomRegle), "succès")
		if err != nil {
			// Ne pas faire échouer la mise à jour si le logging échoue, mais logger l'erreur
			fmt.Printf("Erreur lors du logging d'audit pour la modification de règle %s: %v\n", id, err)
		}
	}

	return regle, nil
}

// DeleteRegleControle supprime une règle de contrôle par son ID.
// Note: Les alertes associées sont CONSERVÉES pour audit (regle_declenchee_id reste en place)
func (s *Store) DeleteRegleControle(ctx context.Context, id string, userID string) error {
	// Vérifier que la règle existe
	var regle models.RegleControle
	if err := s.gormDB.WithContext(ctx).First(&regle, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return fmt.Errorf("aucune règle trouvée avec l'ID: %s", id)
		}
		return fmt.Errorf("erreur lors de la recherche de la règle: %w", err)
	}

	// Supprimer la règle (les alertes conservent leur regle_declenchee_id pour audit)
	result := s.gormDB.WithContext(ctx).Delete(&regle)
	if result.Error != nil {
		return fmt.Errorf("erreur lors de la suppression de la règle: %w", result.Error)
	}

	// Vérifier que la règle a bien été supprimée
	if result.RowsAffected == 0 {
		return fmt.Errorf("erreur lors de la suppression: aucune ligne affectée")
	}

	// Logging d'audit automatique si userID fourni
	if userID != "" {
		actionDescription := fmt.Sprintf("Suppression de la règle '%s'", regle.NomRegle)
		err := s.LogAction(ctx, userID, actionDescription, "Règles de Contrôle", id, fmt.Sprintf("Suppression de la règle de contrôle: %s", regle.NomRegle), "succès")
		if err != nil {
			// Ne pas faire échouer la suppression si le logging échoue, mais logger l'erreur
			fmt.Printf("Erreur lors du logging d'audit pour la suppression de règle %s: %v\n", id, err)
		}
	}

	return nil
}

// UpdateRegleControleStatus met à jour le statut (Actif/Inactif) d'une règle de contrôle.
func (s *Store) UpdateRegleControleStatus(ctx context.Context, id string, status string, userID string) (*models.RegleControle, error) {
	// Récupérer la règle avant modification pour avoir le nom si on fait du logging
	var beforeUpdate models.RegleControle
	var nomRegle string
	if userID != "" {
		if err := s.gormDB.WithContext(ctx).First(&beforeUpdate, "id = ?", id).Error; err != nil {
			return nil, err
		}
		nomRegle = beforeUpdate.NomRegle
	}

	// Convertir le statut string en booléen (Actif = true, Inactif = false)
	isActive := status == "Actif"

	result := s.gormDB.WithContext(ctx).Model(&models.RegleControle{}).Where("id = ?", id).
		Update("active", isActive)
	if result.Error != nil {
		return nil, result.Error
	}

	// Recharger la règle mise à jour
	var updated models.RegleControle
	if err := s.gormDB.WithContext(ctx).First(&updated, "id = ?", id).Error; err != nil {
		return nil, err
	}

	// Logging d'audit automatique si userID fourni
	if userID != "" {
		actionDescription := fmt.Sprintf("%s de la règle '%s'", status, nomRegle)
		details := fmt.Sprintf("Changement statut règle '%s' vers: %s", nomRegle, status)

		err := s.LogAction(ctx, userID, actionDescription, "Règles de Contrôle", id, details, "succès")
		if err != nil {
			fmt.Printf("Erreur lors du logging d'audit pour le changement de statut de règle %s: %v\n", id, err)
		}
	}

	return &updated, nil
}

// GetAllRegles récupère toutes les règles de contrôle.
func (s *Store) GetAllRegles(ctx context.Context) ([]models.RegleControle, error) {
	var regles []models.RegleControle
	result := s.gormDB.WithContext(ctx).Find(&regles)
	if result.Error != nil {
		return nil, result.Error
	}
	return regles, nil
}

// GetRegleByType retourne la règle métier correspondant à un type donné (ex: PEP, Blacklist, etc.)
func (s *Store) GetRegleByType(ctx context.Context, typeRegle string) (*models.RegleControle, error) {
	var regle models.RegleControle
	result := s.gormDB.WithContext(ctx).Where("type_regle = ? AND active = ?", typeRegle, true).First(&regle)
	if result.Error != nil {
		return nil, result.Error
	}
	return &regle, nil
}
