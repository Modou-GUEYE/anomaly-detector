package store

import (
	"context"
	"errors"
	"fmt"
	"time"

	"gorm.io/gorm"

	"anomaly-detector/internal/models"
)

// ===================== COMPTES =====================

// CreateCompte insère un nouveau compte dans la base de données.
func (s *Store) CreateCompte(ctx context.Context, compte *models.Compte, userID string) (*models.Compte, error) {
	// Remplissage automatique du profil si vide
	if compte.ProfilClientID == "" && compte.TypeClient != "" {
		profilID, err := s.GetProfilClientIDByCode(ctx, compte.TypeClient)
		if err == nil && profilID != "" {
			compte.ProfilClientID = profilID
		}
	}
	// Initialisation automatique de la date de création si vide
	if compte.DateCreation.IsZero() {
		compte.DateCreation = time.Now()
	}
	// --- GORM ---
	result := s.gormDB.WithContext(ctx).Create(compte)
	if result.Error != nil {
		return nil, result.Error
	}

	// Logging d'audit avec l'utilisateur fourni
	actionDescription := fmt.Sprintf("Création du compte '%s'", compte.NumeroCompte)
	err := s.LogAction(ctx, "MDA", actionDescription, "Comptes", compte.ID, fmt.Sprintf("Création du compte: %s (%s)", compte.NumeroCompte, compte.NomClient), "succès")
	if err != nil {
		// Ne pas faire échouer la création si le logging échoue, mais logger l'erreur
		fmt.Printf("Erreur lors du logging d'audit pour la création de compte %s: %v\n", compte.ID, err)
	}

	return compte, nil
}

// GetCompteByCompte recherche un client par son numéro de compte unique.
func (s *Store) GetCompteByCompte(ctx context.Context, numeroCompte string) (*models.Compte, error) {

	var compte models.Compte
	result := s.gormDB.WithContext(ctx).Where("numero_compte = ?", numeroCompte).First(&compte)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			// Aucun compte trouvé, retourne un compte vide
			return &models.Compte{}, nil
		}
		return nil, result.Error
	}
	return &compte, nil
}

// GetReglesForCompte récupère les règles actives associées au profil client d'un compte donné.
func (s *Store) GetReglesForCompte(ctx context.Context, numeroCompte string) ([]models.RegleControle, error) {
	// On récupère le profil client du compte
	var compte models.Compte
	if err := s.gormDB.WithContext(ctx).Where("numero_compte = ?", numeroCompte).First(&compte).Error; err != nil {
		return nil, err
	}
	var regles []models.RegleControle
	result := s.gormDB.WithContext(ctx).Where("profil_client_id = ? AND active = ?", compte.ProfilClientID, true).Find(&regles)
	if result.Error != nil {
		return nil, result.Error
	}
	return regles, nil
}

// GetAllComptes retourne la liste de tous les comptes clients enregistrés.
func (s *Store) GetAllComptes(ctx context.Context) ([]*models.Compte, error) {
	var comptes []*models.Compte
	err := s.gormDB.WithContext(ctx).
		Order("date_creation DESC").
		Find(&comptes).Error
	return comptes, err
}

// GetComptesSurveillesCount retourne le nombre de comptes uniques dans la table suivi_blacklist.
func (s *Store) GetComptesSurveillesCount(ctx context.Context) (int64, error) {
	var count int64
	err := s.gormDB.WithContext(ctx).
		Table("suivi_blacklist").
		Select("COUNT(DISTINCT numero_compte)").
		Scan(&count).Error
	return count, err
}

// GetComptesSurveilles retourne la liste des comptes uniques dans suivi_blacklist.
func (s *Store) GetComptesSurveilles(ctx context.Context) ([]string, error) {
	var comptes []string
	err := s.gormDB.WithContext(ctx).
		Table("suivi_blacklist").
		Distinct("numero_compte").
		Pluck("numero_compte", &comptes).Error
	return comptes, err
}
