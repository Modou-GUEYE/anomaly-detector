package store

import (
	"context"
	"errors"
	"fmt"
	"log"
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
	if err := s.gormDB.WithContext(ctx).Create(entry).Error; err != nil {
		if userID != "" {
			_ = s.LogAction(ctx, userID, fmt.Sprintf("Ajout d'entité '%s' à la liste de surveillance", entry.TypeEntite),
				"Blacklist", "", fmt.Sprintf("Échec - Nom: %s, Type: %s, Erreur: %v", entry.Nom, entry.TypeEntite, err), "échec")
		}
		return nil, err
	}

	if userID != "" {
		if err := s.LogAction(ctx, userID, fmt.Sprintf("Ajout d'entité '%s' à la liste de surveillance", entry.TypeEntite),
			"Blacklist", entry.ID, fmt.Sprintf("Nouvelle entité - Nom: %s, Type: %s", entry.Nom, entry.TypeEntite), "succès"); err != nil {
			log.Printf("Erreur logging audit blacklist %s: %v", entry.ID, err)
		}
	}
	return entry, nil
}

// GetBlacklistEntries récupère toutes les entités de la blacklist.
func (s *Store) GetBlacklistEntries(ctx context.Context) ([]models.EntiteBlacklist, error) {
	var entries []models.EntiteBlacklist
	if err := s.gormDB.WithContext(ctx).Find(&entries).Error; err != nil {
		return nil, err
	}
	return entries, nil
}

// UpdateBlacklistEntry met à jour une entité existante dans la blacklist.
func (s *Store) UpdateBlacklistEntry(ctx context.Context, id string, entry *models.EntiteBlacklist, userID string) (*models.EntiteBlacklist, error) {
	entry.ID = id
	db := s.gormDB.WithContext(ctx)
	if err := db.Model(&models.EntiteBlacklist{}).Where("id = ?", id).Updates(map[string]interface{}{
		"nom":                          entry.Nom,
		"type_entite":                  entry.TypeEntite,
		"informations_supplementaires": entry.InformationsSupplementaires,
	}).Error; err != nil {
		if userID != "" {
			_ = s.LogAction(ctx, userID, fmt.Sprintf("Modification d'entité '%s' de la liste de surveillance", entry.TypeEntite),
				"Blacklist", id, fmt.Sprintf("Échec - ID: %s, Erreur: %v", id, err), "échec")
		}
		return nil, err
	}

	var updated models.EntiteBlacklist
	if err := db.First(&updated, "id = ?", id).Error; err != nil {
		if userID != "" {
			_ = s.LogAction(ctx, userID, fmt.Sprintf("Modification d'entité '%s' de la liste de surveillance", entry.TypeEntite),
				"Blacklist", id, fmt.Sprintf("Échec rechargement - ID: %s, Erreur: %v", id, err), "échec")
		}
		return nil, err
	}
	*entry = updated

	if userID != "" {
		if err := s.LogAction(ctx, userID, fmt.Sprintf("Modification d'entité '%s' de la liste de surveillance", updated.TypeEntite),
			"Blacklist", id, fmt.Sprintf("Mise à jour - Nom: %s, Type: %s", updated.Nom, updated.TypeEntite), "succès"); err != nil {
			log.Printf("Erreur logging audit blacklist %s: %v", id, err)
		}
	}
	return entry, nil
}

// DeleteBlacklistEntry supprime une entité de la blacklist par son ID.
func (s *Store) DeleteBlacklistEntry(ctx context.Context, id string, userID string) error {
	var entry models.EntiteBlacklist
	db := s.gormDB.WithContext(ctx)

	if userID != "" {
		if err := db.First(&entry, "id = ?", id).Error; err != nil {
			_ = s.LogAction(ctx, userID, "Suppression d'entité de la liste de surveillance",
				"Blacklist", id, fmt.Sprintf("Échec - ID introuvable: %s, Erreur: %v", id, err), "échec")
			return err
		}
	}

	if err := db.Delete(&models.EntiteBlacklist{}, "id = ?", id).Error; err != nil {
		if userID != "" {
			_ = s.LogAction(ctx, userID, fmt.Sprintf("Suppression d'entité '%s' de la liste de surveillance", entry.TypeEntite),
				"Blacklist", id, fmt.Sprintf("Échec - Nom: %s, Type: %s, Erreur: %v", entry.Nom, entry.TypeEntite, err), "échec")
		}
		return err
	}

	if userID != "" {
		if err := s.LogAction(ctx, userID, fmt.Sprintf("Suppression d'entité '%s' de la liste de surveillance", entry.TypeEntite),
			"Blacklist", id, fmt.Sprintf("Supprimé - Nom: %s, Type: %s", entry.Nom, entry.TypeEntite), "succès"); err != nil {
			log.Printf("Erreur logging audit blacklist %s: %v", id, err)
		}
	}
	return nil
}

// GetBlacklistStatusForCompte vérifie si un compte est dans la blacklist et retourne le type d'entité s'il y est.
func (s *Store) GetBlacklistStatusForCompte(ctx context.Context, compteID string) (string, error) {
	var entite models.EntiteBlacklist
	err := s.gormDB.WithContext(ctx).Where("numero_compte = ?", compteID).First(&entite).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	return entite.TypeEntite, nil
}

// IsAccountOnBlacklist vérifie si un KYC ID est sur la blacklist.
func (s *Store) IsAccountOnBlacklist(ctx context.Context, kycID string) (string, error) {
	var entite models.EntiteBlacklist
	db := s.gormDB.WithContext(ctx)

	if err := db.Where("numero_compte = ?", kycID).First(&entite).Error; err == nil {
		return entite.TypeEntite, nil
	}

	err := db.Where("UPPER(numero_compte) = UPPER(?)", kycID).First(&entite).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	return entite.TypeEntite, nil
}
