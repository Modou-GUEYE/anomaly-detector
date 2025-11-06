package store

import (
	"context"
	"errors"
	"fmt"
	"time"

	"gorm.io/gorm"

	"anomaly-detector/internal/models"
)

// ===================== BLACKLIST =====================

// CreateBlacklistEntry insère une nouvelle entité dans la blacklist.
func (s *Store) CreateBlacklistEntry(ctx context.Context, entry *models.EntiteBlacklist, userID string) (*models.EntiteBlacklist, error) {
	if entry.DateAjout.IsZero() {
		entry.DateAjout = time.Now()
	}
	result := s.gormDB.WithContext(ctx).Create(entry)
	if result.Error != nil {
		// Logger l'échec si userID fourni
		if userID != "" {
			details := fmt.Sprintf("Échec lors de l'ajout d'entité à la blacklist - Nom: %s, Type: %s, Erreur: %v",
				entry.Nom, entry.TypeEntite, result.Error)
			actionDescription := fmt.Sprintf("Ajout d'entité '%s' à la liste de surveillance", entry.TypeEntite)
			_ = s.LogAction(ctx, userID, actionDescription, "Blacklist", "", details, "échec")
		}
		return nil, result.Error
	}

	// Log d'ajout à la blacklist si userID fourni
	if userID != "" {
		details := fmt.Sprintf("Nouvelle entité ajoutée à la blacklist - Nom: %s, Type: %s",
			entry.Nom, entry.TypeEntite)
		actionDescription := fmt.Sprintf("Ajout d'entité '%s' à la liste de surveillance", entry.TypeEntite)

		logErr := s.LogAction(ctx, userID, actionDescription, "Blacklist", entry.ID, details, "succès")
		if logErr != nil {
			fmt.Printf("Erreur lors du logging d'audit pour l'ajout blacklist %s: %v\n", entry.ID, logErr)
		}
	}

	return entry, nil
}

// GetBlacklistEntries récupère toutes les entités de la blacklist.
func (s *Store) GetBlacklistEntries(ctx context.Context) ([]models.EntiteBlacklist, error) {
	var entries []models.EntiteBlacklist
	result := s.gormDB.WithContext(ctx).Find(&entries)
	if result.Error != nil {
		return nil, result.Error
	}
	return entries, nil
}

// UpdateBlacklistEntry met à jour une entité existante dans la blacklist.
func (s *Store) UpdateBlacklistEntry(ctx context.Context, id string, entry *models.EntiteBlacklist, userID string) (*models.EntiteBlacklist, error) {
	// GORM : mise à jour par ID
	entry.ID = id
	result := s.gormDB.WithContext(ctx).Model(&models.EntiteBlacklist{}).Where("id = ?", id).
		Updates(map[string]interface{}{
			"nom":                          entry.Nom,
			"type_entite":                  entry.TypeEntite,
			"informations_supplementaires": entry.InformationsSupplementaires,
		})
	if result.Error != nil {
		// Logger l'échec
		if userID != "" {
			details := fmt.Sprintf("Échec lors de la modification d'entité blacklist - ID: %s, Erreur: %v", id, result.Error)
			actionDescription := fmt.Sprintf("Modification d'entité '%s' de la liste de surveillance", entry.TypeEntite)
			_ = s.LogAction(ctx, userID, actionDescription, "Blacklist", id, details, "échec")
		}
		return nil, result.Error
	}
	// Recharger l'entité mise à jour pour obtenir la dateAjout
	var updated models.EntiteBlacklist
	if err := s.gormDB.WithContext(ctx).First(&updated, "id = ?", id).Error; err != nil {
		// Logger l'échec de rechargement
		if userID != "" {
			details := fmt.Sprintf("Échec lors du rechargement d'entité blacklist - ID: %s, Erreur: %v", id, err)
			actionDescription := fmt.Sprintf("Modification d'entité '%s' de la liste de surveillance", entry.TypeEntite)
			_ = s.LogAction(ctx, userID, actionDescription, "Blacklist", id, details, "échec")
		}
		return nil, err
	}
	*entry = updated

	// Log d'audit si userID fourni
	if userID != "" {
		details := fmt.Sprintf("Entité blacklist mise à jour - Nom: %s, Type: %s",
			updated.Nom, updated.TypeEntite)
		actionDescription := fmt.Sprintf("Modification d'entité '%s' de la liste de surveillance", updated.TypeEntite)

		logErr := s.LogAction(ctx, userID, actionDescription, "Blacklist", id, details, "succès")
		if logErr != nil {
			fmt.Printf("Erreur lors du logging d'audit pour la modification blacklist %s: %v\n", id, logErr)
		}
	}

	return entry, nil
}

// DeleteBlacklistEntry supprime une entité de la blacklist par son ID.
func (s *Store) DeleteBlacklistEntry(ctx context.Context, id string, userID string) error {
	// Récupérer les informations de l'entité avant suppression pour le logging
	var entry models.EntiteBlacklist
	if userID != "" {
		if err := s.gormDB.WithContext(ctx).First(&entry, "id = ?", id).Error; err != nil {
			// Logger l'échec de récupération
			details := fmt.Sprintf("Échec lors de la suppression d'entité blacklist - ID introuvable: %s, Erreur: %v", id, err)
			actionDescription := "Suppression d'entité de la liste de surveillance"
			_ = s.LogAction(ctx, userID, actionDescription, "Blacklist", id, details, "échec")
			return err
		}
	}

	result := s.gormDB.WithContext(ctx).Delete(&models.EntiteBlacklist{}, "id = ?", id)
	if result.Error != nil {
		// Logger l'échec de suppression
		if userID != "" {
			details := fmt.Sprintf("Échec lors de la suppression d'entité blacklist - Nom: %s, Type: %s, Erreur: %v",
				entry.Nom, entry.TypeEntite, result.Error)
			actionDescription := fmt.Sprintf("Suppression d'entité '%s' de la liste de surveillance", entry.TypeEntite)
			_ = s.LogAction(ctx, userID, actionDescription, "Blacklist", id, details, "échec")
		}
		return result.Error
	}

	// Log d'audit si userID fourni
	if userID != "" {
		details := fmt.Sprintf("Entité blacklist supprimée - Nom: %s, Type: %s",
			entry.Nom, entry.TypeEntite)
		actionDescription := fmt.Sprintf("Suppression d'entité '%s' de la liste de surveillance", entry.TypeEntite)

		logErr := s.LogAction(ctx, userID, actionDescription, "Blacklist", id, details, "succès")
		if logErr != nil {
			fmt.Printf("Erreur lors du logging d'audit pour la suppression blacklist %s: %v\n", id, logErr)
		}
	}

	return nil
}

// GetBlacklistStatusForCompte vérifie si un compte est dans la blacklist et retourne le type d'entité s'il y est.
func (s *Store) GetBlacklistStatusForCompte(ctx context.Context, compteID string) (string, error) {

	var entite models.EntiteBlacklist
	result := s.gormDB.WithContext(ctx).Where("numero_compte = ?", compteID).First(&entite)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return "", nil
		}
		return "", result.Error
	}
	return entite.TypeEntite, nil
}

// IsAccountOnBlacklist vérifie si un KYC ID est sur la blacklist.
func (s *Store) IsAccountOnBlacklist(ctx context.Context, kycID string) (string, error) {
	var entite models.EntiteBlacklist

	// Recherche exacte d'abord
	result := s.gormDB.WithContext(ctx).Where("numero_compte = ?", kycID).First(&entite)
	if result.Error == nil {
		// Trouvé avec correspondance exacte
		return entite.TypeEntite, nil
	}

	// Si pas trouvé avec correspondance exacte, essayer recherche insensible à la casse
	result = s.gormDB.WithContext(ctx).Where("UPPER(numero_compte) = UPPER(?)", kycID).First(&entite)
	if result.Error == nil {
		// Trouvé avec correspondance insensible à la casse
		return entite.TypeEntite, nil
	}

	// Si toujours pas trouvé
	if errors.Is(result.Error, gorm.ErrRecordNotFound) {
		// Pas trouvé dans la blacklist
		return "", nil
	}

	// Erreur de base de données
	return "", result.Error
}
