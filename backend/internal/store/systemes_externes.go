package store

import (
	"anomaly-detector/internal/models"
	"context"
	"fmt"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type SystemesExternesStore struct {
	db *gorm.DB
}

func NewSystemesExternesStore(db *gorm.DB) *SystemesExternesStore {
	return &SystemesExternesStore{db: db}
}

func (s *Store) IsSystemIDAllowed(ctx context.Context, id string, role string) (bool, error) {
	var sys models.SystemeExterne
	err := s.Gorm().WithContext(ctx).Where("id = ? AND actif = ?", id, true).First(&sys).Error
	if err != nil {
		if err.Error() == "record not found" || err.Error() == "gorm: record not found" {
			return false, nil
		}
		return false, err
	}
	if sys.Role != role {
		return false, nil
	}
	return true, nil
}

func (s *Store) CreateSystemeExterne(ctx context.Context, prefix, nom, role string, actif bool) (*models.SystemeExterne, error) {
	id := fmt.Sprintf("%s-%s", prefix, uuid.New().String())
	sys := &models.SystemeExterne{
		ID:    id,
		Nom:   nom,
		Role:  role,
		Actif: actif,
	}
	if err := s.Gorm().WithContext(ctx).Create(sys).Error; err != nil {
		return nil, err
	}
	return sys, nil
}
