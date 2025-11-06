package store

import (
	"anomaly-detector/internal/models"
	"context"
	"errors"
	"fmt"

	"gorm.io/gorm"
)

// ===================== UTILISATEURS =====================

// CreateUser insère un nouvel utilisateur dans la base de données.
func (s *Store) CreateUser(ctx context.Context, user *models.Utilisateur) (string, error) {
	// --- GORM ---
	// Vérifier si l'email existe déjà
	var existing models.Utilisateur
	err := s.gormDB.WithContext(ctx).Where("email = ?", user.Email).First(&existing).Error
	if err == nil {
		// Utilisateur déjà existant
		return "", fmt.Errorf("un utilisateur avec cet email existe déjà")
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		// Erreur inattendue
		return "", err
	}
	// Créer l'utilisateur
	result := s.gormDB.WithContext(ctx).Create(user)
	if result.Error != nil {
		return "", result.Error
	}
	return user.ID, nil
}

// GetUserByEmail récupère un utilisateur par son adresse email.
func (s *Store) GetUserByEmail(ctx context.Context, email string) (*models.Utilisateur, error) {
	var user models.Utilisateur
	result := s.gormDB.WithContext(ctx).Where("email = ?", email).First(&user)
	if result.Error != nil {
		return nil, result.Error
	}
	return &user, nil
}

// GetUserByID récupère un utilisateur par son ID.
func (s *Store) GetUserByID(ctx context.Context, id string) (*models.Utilisateur, error) {
	var user models.Utilisateur
	result := s.gormDB.WithContext(ctx).First(&user, "id = ?", id)
	if result.Error != nil {
		return nil, result.Error
	}
	return &user, nil
}
