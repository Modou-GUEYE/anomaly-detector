package store

import (
	"anomaly-detector/internal/models"
	"context"
)

// ===================== PROFILS CLIENTS =====================

// GetProfilClientIDByCode retourne l'ID du profil client à partir du code type client.
func (s *Store) GetProfilClientIDByCode(ctx context.Context, code string) (string, error) {
	var nomProfil string
	switch code {
	case "P":
		nomProfil = "Particulier"
	case "M":
		nomProfil = "Entreprise"
	case "C":
		nomProfil = "Commerçant"
	case "B":
		nomProfil = "Business"
	default:
		nomProfil = "Particulier"
	}
	var profil models.ProfilClient
	result := s.gormDB.WithContext(ctx).Where("nom_profil = ?", nomProfil).First(&profil)
	if result.Error != nil {
		return "", result.Error
	}
	return profil.ID, nil
}

// CreateProfilClient insère un nouveau profil client dans la base de données.
func (s *Store) CreateProfilClient(ctx context.Context, profil *models.ProfilClient) (*models.ProfilClient, error) {
	result := s.gormDB.WithContext(ctx).Create(profil)
	if result.Error != nil {
		return nil, result.Error
	}
	return profil, nil
}

// GetProfilsClient récupère tous les profils clients.
func (s *Store) GetProfilsClient(ctx context.Context) ([]models.ProfilClient, error) {
	var profils []models.ProfilClient
	result := s.gormDB.WithContext(ctx).Find(&profils)
	if result.Error != nil {
		return nil, result.Error
	}
	return profils, nil
}

// UpdateProfilClient met à jour un profil client existant.
func (s *Store) UpdateProfilClient(ctx context.Context, id string, profil *models.ProfilClient) (*models.ProfilClient, error) {
	profil.ID = id
	result := s.gormDB.WithContext(ctx).Model(&models.ProfilClient{}).Where("id = ?", id).
		Updates(map[string]interface{}{
			"nom_profil":  profil.NomProfil,
			"description": profil.Description,
		})
	if result.Error != nil {
		return nil, result.Error
	}
	// Recharger le profil mis à jour
	var updated models.ProfilClient
	if err := s.gormDB.WithContext(ctx).First(&updated, "id = ?", id).Error; err != nil {
		return nil, err
	}
	*profil = updated
	return profil, nil
}

// DeleteProfilClient supprime un profil client par son ID.
func (s *Store) DeleteProfilClient(ctx context.Context, id string) error {
	result := s.gormDB.WithContext(ctx).Delete(&models.ProfilClient{}, "id = ?", id)
	return result.Error
}
