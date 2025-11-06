package store

import (
	"context"
	"time"

	"anomaly-detector/internal/models"
)

// ===================== SUIVI BLACKLIST =====================

// CreateSuiviBlacklist enregistre une trace lorsqu'une transaction implique une entité sous surveillance (GORM).
func (s *Store) CreateSuiviBlacklist(ctx context.Context, transactionID, numeroCompte, statutSuivi, raison string) error {
	suivi := &models.SuiviBlacklist{
		TransactionID: transactionID,
		NumeroCompte:  numeroCompte,
		StatutSuivi:   statutSuivi,
		Raison:        raison,
		DateDetection: time.Now(),
	}
	result := s.gormDB.WithContext(ctx).Create(suivi)
	return result.Error
}

// GetSuiviBlacklist retourne la liste complète des entrées de suivi_blacklist.
func (s *Store) GetSuiviBlacklist(ctx context.Context) ([]*models.SuiviBlacklist, error) {
	var suivis []*models.SuiviBlacklist
	err := s.gormDB.WithContext(ctx).
		Order("date_detection DESC").
		Find(&suivis).Error
	return suivis, err
}
