package store

import (
	"context"
	"time"

	"anomaly-detector/internal/models"
)

// ===================== TRANSACTIONS =====================

// CreateTransaction insère la nouvelle transaction riche dans la base de données.
func (s *Store) CreateTransaction(ctx context.Context, tx *models.Transaction) (*models.Transaction, error) {
	// Initialisation automatique de la date d'opération si vide
	if tx.DateOperation.IsZero() {
		tx.DateOperation = time.Now()
	}
	// --- GORM ---
	result := s.gormDB.WithContext(ctx).Create(tx)
	if result.Error != nil {
		return nil, result.Error
	}
	return tx, nil
}

// GetTransactions récupère toutes les transactions enregistrées dans la base.
func (s *Store) GetTransactions(ctx context.Context) ([]*models.Transaction, error) {
	var transactions []*models.Transaction
	result := s.gormDB.WithContext(ctx).Find(&transactions)
	if result.Error != nil {
		return nil, result.Error
	}
	return transactions, nil
}

// GetTransactionSumForPeriod calcule le montant total des transactions pour un numéro de compte avec GORM.
func (s *Store) GetTransactionSumForPeriod(ctx context.Context, numeroCompte string, jours int) (float64, error) {
	var total float64
	since := time.Now().AddDate(0, 0, -jours)
	result := s.gormDB.WithContext(ctx).
		Model(&models.Transaction{}).
		Where("(numero_compte_payeur = ? OR numero_compte_paye = ?) AND date_operation >= ?", numeroCompte, numeroCompte, since).
		Select("COALESCE(SUM(montant), 0)").
		Row()
	err := result.Scan(&total)
	return total, err
}

// GetTransactionCountForPeriod compte le nombre de transactions pour un numéro de compte avec GORM.
func (s *Store) GetTransactionCountForPeriod(ctx context.Context, numeroCompte string, jours int) (int64, error) {
	var count int64
	since := time.Now().AddDate(0, 0, -jours)
	result := s.gormDB.WithContext(ctx).
		Model(&models.Transaction{}).
		Where("(numero_compte_payeur = ? OR numero_compte_paye = ?) AND date_operation >= ?", numeroCompte, numeroCompte, since).
		Count(&count)
	return count, result.Error
}
