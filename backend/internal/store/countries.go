package store

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"gorm.io/gorm"

	"anomaly-detector/internal/models"
)

// ===================== GESTION PAYS À RISQUE (Nouvelle Architecture) =====================

// GetAllPaysARisque récupère tous les pays à risque actifs
func (s *Store) GetAllPaysARisque(ctx context.Context) ([]models.PaysRisque, error) {
	var pays []models.PaysRisque
	result := s.gormDB.WithContext(ctx).Where("active = ?", true).Order("nom_pays ASC").Find(&pays)
	if result.Error != nil {
		return nil, result.Error
	}
	return pays, nil
}

// IsPaysARisque vérifie si un pays est dans la liste des pays à risque actifs (par nom)
func (s *Store) IsPaysARisque(ctx context.Context, nomPays string) (bool, error) {
	var count int64
	result := s.gormDB.WithContext(ctx).Model(&models.PaysRisque{}).
		Where("UPPER(nom_pays) = ? AND active = ?", strings.ToUpper(nomPays), true).
		Count(&count)

	if result.Error != nil {
		return false, result.Error
	}

	return count > 0, nil
}

// IsPaysARisqueByCode vérifie si un pays est dans la liste des pays à risque actifs (par code ISO alpha-2)
func (s *Store) IsPaysARisqueByCode(ctx context.Context, codePays string) (bool, error) {
	var count int64
	result := s.gormDB.WithContext(ctx).Model(&models.PaysRisque{}).
		Where("UPPER(code) = ? AND active = ?", strings.ToUpper(codePays), true).
		Count(&count)

	if result.Error != nil {
		return false, result.Error
	}

	return count > 0, nil
}

// AddPaysARisque ajoute un pays à la liste des pays à risque
func (s *Store) AddPaysARisque(ctx context.Context, nomPays string, userID string) error {
	pays := &models.PaysRisque{
		NomPays: nomPays,
		Active:  true,
	}
	return s.AddPaysARisqueWithCode(ctx, pays, userID)
}

// AddPaysARisqueWithCode ajoute un pays à risque avec son code ISO obligatoire
func (s *Store) AddPaysARisqueWithCode(ctx context.Context, pays *models.PaysRisque, userID string) error {
	// Validation: le code est obligatoire
	if pays.Code == "" || len(pays.Code) != 2 {
		return fmt.Errorf("le code pays ISO alpha-2 est obligatoire (2 caractères)")
	}

	// Vérifier si le pays existe déjà
	var existing models.PaysRisque
	result := s.gormDB.WithContext(ctx).Where("UPPER(nom_pays) = ?", strings.ToUpper(pays.NomPays)).First(&existing)

	if result.Error == nil {
		// Le pays existe déjà, on le réactive et met à jour le code
		updates := map[string]interface{}{
			"active": true,
			"code":   pays.Code,
		}
		if err := s.gormDB.WithContext(ctx).Model(&existing).Updates(updates).Error; err != nil {
			return err
		}
		return nil // Déjà présent
	}

	if !errors.Is(result.Error, gorm.ErrRecordNotFound) {
		return result.Error
	}

	// Créer le nouveau pays avec le code
	if err := s.gormDB.WithContext(ctx).Select("NomPays", "Active", "Code").Create(pays).Error; err != nil {
		// Logger l'échec si userID fourni
		if userID != "" {
			_ = s.LogAction(ctx, userID, fmt.Sprintf("Échec ajout pays '%s'", pays.NomPays), "Pays à Risque", "", fmt.Sprintf("Erreur: %v", err), "échec")
		}
		return err
	}

	// Lier explicitement au rule 'pays_risque' active si disponible
	var regle models.RegleControle
	regleResult := s.gormDB.WithContext(ctx).Where("type_regle = ? AND active = ?", "pays_risque", true).First(&regle)
	if regleResult.Error == nil {
		// Mettre à jour la FK
		if err := s.gormDB.WithContext(ctx).Model(pays).Update("regle_controle_id", regle.ID).Error; err != nil {
			fmt.Printf("Warn: impossible de lier le pays %s à la règle pays_risque: %v\n", pays.NomPays, err)
		} else {
			pays.RegleControleID = &regle.ID
		}
	} else if !errors.Is(regleResult.Error, gorm.ErrRecordNotFound) {
		fmt.Printf("Warn: erreur lors de la récupération de la règle pays_risque active: %v\n", regleResult.Error)
	}

	// Logging d'audit
	if userID != "" {
		actionDescription := fmt.Sprintf("Ajout du pays à risque '%s'", pays.NomPays)
		if err := s.LogAction(ctx, userID, actionDescription, "Pays à Risque", pays.ID, fmt.Sprintf("Ajout du pays: %s", pays.NomPays), "succès"); err != nil {
			fmt.Printf("Erreur lors du logging d'audit pour l'ajout du pays %s: %v\n", pays.NomPays, err)
		}
	}

	return nil
}

// LinkPaysRisqueToRegle associe toutes les entrées pays_risque sans liaison à la règle 'pays_risque' active
func (s *Store) LinkPaysRisqueToRegle(ctx context.Context) error {
	var regle models.RegleControle
	if err := s.gormDB.WithContext(ctx).Where("type_regle = ? AND active = ?", "pays_risque", true).First(&regle).Error; err != nil {
		return fmt.Errorf("règle pays_risque active introuvable: %w", err)
	}

	// Mettre à jour uniquement les pays sans FK définie
	result := s.gormDB.WithContext(ctx).Model(&models.PaysRisque{}).Where("regle_controle_id IS NULL").Update("regle_controle_id", regle.ID)
	if result.Error != nil {
		return result.Error
	}
	return nil
}

// DeletePaysARisque supprime (désactive) un pays de la liste des pays à risque
func (s *Store) DeletePaysARisque(ctx context.Context, nomPays string, userID string) error {
	// Trouver le pays
	var pays models.PaysRisque
	result := s.gormDB.WithContext(ctx).Where("UPPER(nom_pays) = ?", strings.ToUpper(nomPays)).First(&pays)
	if result.Error != nil {
		return result.Error
	}

	// Désactiver au lieu de supprimer (soft delete logique)
	if err := s.gormDB.WithContext(ctx).Model(&pays).Update("active", false).Error; err != nil {
		return err
	}

	// Logging d'audit
	if userID != "" {
		actionDescription := fmt.Sprintf("Suppression du pays à risque '%s'", nomPays)
		if err := s.LogAction(ctx, userID, actionDescription, "Pays à Risque", pays.ID, fmt.Sprintf("Suppression du pays: %s", nomPays), "succès"); err != nil {
			fmt.Printf("Erreur lors du logging d'audit pour la suppression du pays %s: %v\n", nomPays, err)
		}
	}

	return nil
}

// UpdatePaysARisque met à jour le nom d'un pays dans la liste
func (s *Store) UpdatePaysARisque(ctx context.Context, ancienNom, nouveauNom string, userID string) error {
	return s.UpdatePaysARisqueWithCode(ctx, ancienNom, nouveauNom, "", userID)
}

// UpdatePaysARisqueWithCode met à jour le nom et le code d'un pays dans la liste
func (s *Store) UpdatePaysARisqueWithCode(ctx context.Context, ancienNom, nouveauNom, code string, userID string) error {
	// Validation: le code est obligatoire
	if code == "" || len(code) != 2 {
		return fmt.Errorf("le code pays ISO alpha-2 est obligatoire (2 caractères)")
	}

	// Trouver le pays
	var pays models.PaysRisque
	result := s.gormDB.WithContext(ctx).Where("UPPER(nom_pays) = ?", strings.ToUpper(ancienNom)).First(&pays)
	if result.Error != nil {
		return result.Error
	}

	// Mettre à jour le nom et le code
	updates := map[string]interface{}{
		"nom_pays": nouveauNom,
		"code":     code,
	}

	if err := s.gormDB.WithContext(ctx).Model(&pays).Updates(updates).Error; err != nil {
		return err
	}

	// Logging d'audit
	if userID != "" {
		actionDescription := fmt.Sprintf("Modification du pays à risque '%s' en '%s'", ancienNom, nouveauNom)
		details := fmt.Sprintf("Modification du pays: %s → %s", ancienNom, nouveauNom)
		if code != "" {
			details += fmt.Sprintf(", code: %s", code)
		}
		if err := s.LogAction(ctx, userID, actionDescription, "Pays à Risque", pays.ID, details, "succès"); err != nil {
			fmt.Printf("Erreur lors du logging d'audit pour la modification du pays %s: %v\n", ancienNom, err)
		}
	}

	return nil
}

// GetReglePaysRisqueActive retourne la règle pays_risque active (pour compatibilité)
// Note: Maintenant que les pays sont dans une table séparée, cette méthode
// est principalement utilisée pour vérifier si la règle est active
func (s *Store) GetReglePaysRisqueActive(ctx context.Context) (*models.RegleControle, error) {
	var regle models.RegleControle
	result := s.gormDB.WithContext(ctx).Where("type_regle = ? AND active = ?", "pays_risque", true).First(&regle)
	if result.Error != nil {
		return nil, result.Error
	}
	return &regle, nil
}
