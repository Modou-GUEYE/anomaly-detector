package store

import (
	"anomaly-detector/internal/models"
	"context"
	"time"
)

// GetDashboardSummary retourne un résumé complet du dashboard avec filtrage optionnel par dates
func (s *Store) GetDashboardSummary(ctx context.Context, dateDebut, dateFin string) (map[string]interface{}, error) {
	summary := make(map[string]interface{})

	// Parse des dates si fournies
	var dateDebutParsed, dateFinParsed time.Time
	var err error

	if dateDebut != "" {
		dateDebutParsed, err = time.Parse("2006-01-02", dateDebut)
		if err != nil {
			return nil, err
		}
	}

	if dateFin != "" {
		dateFinParsed, err = time.Parse("2006-01-02", dateFin)
		if err != nil {
			return nil, err
		}
		// Ajouter 23:59:59 pour inclure toute la journée
		dateFinParsed = dateFinParsed.Add(23*time.Hour + 59*time.Minute + 59*time.Second)
	}

	// Compter les transactions
	var transactionCount int64
	txQuery := s.gormDB.WithContext(ctx).Model(&models.Transaction{})
	if dateDebut != "" {
		txQuery = txQuery.Where("date_operation >= ?", dateDebutParsed)
	}
	if dateFin != "" {
		txQuery = txQuery.Where("date_operation <= ?", dateFinParsed)
	}
	if err := txQuery.Count(&transactionCount).Error; err != nil {
		return nil, err
	}
	summary["transactions_count"] = transactionCount

	// Compter les alertes
	var alertCount int64
	alertQuery := s.gormDB.WithContext(ctx).Model(&models.Alerte{})
	if dateDebut != "" {
		alertQuery = alertQuery.Where("date_generation >= ?", dateDebutParsed)
	}
	if dateFin != "" {
		alertQuery = alertQuery.Where("date_generation <= ?", dateFinParsed)
	}
	if err := alertQuery.Count(&alertCount).Error; err != nil {
		return nil, err
	}
	summary["anomalies_count"] = alertCount

	// Compter les alertes traitées
	var treatedCount int64
	treatedQuery := s.gormDB.WithContext(ctx).Model(&models.Alerte{}).Where("statut ILIKE ?", "%trait%")
	if dateDebut != "" {
		treatedQuery = treatedQuery.Where("date_generation >= ?", dateDebutParsed)
	}
	if dateFin != "" {
		treatedQuery = treatedQuery.Where("date_generation <= ?", dateFinParsed)
	}
	if err := treatedQuery.Count(&treatedCount).Error; err != nil {
		return nil, err
	}
	summary["anomalies_treated"] = treatedCount

	// Compter les transactions bloquées
	var blockedTransactions int64
	blockedQuery := s.gormDB.WithContext(ctx).Model(&models.Alerte{}).Where("action = ? AND transaction_id IS NOT NULL", "bloquer")
	if dateDebut != "" {
		blockedQuery = blockedQuery.Where("date_generation >= ?", dateDebutParsed)
	}
	if dateFin != "" {
		blockedQuery = blockedQuery.Where("date_generation <= ?", dateFinParsed)
	}
	if err := blockedQuery.Count(&blockedTransactions).Error; err != nil {
		return nil, err
	}
	summary["transactions_blocked"] = blockedTransactions

	// Compter les comptes surveillés
	monitoredCount, err := s.GetComptesSurveillesCount(ctx)
	if err != nil {
		return nil, err
	}
	summary["monitored_accounts"] = monitoredCount

	// Alertes récentes
	var recentAnomalies []models.Alerte
	recentAlertQuery := s.gormDB.WithContext(ctx)
	if dateDebut != "" {
		recentAlertQuery = recentAlertQuery.Where("date_generation >= ?", dateDebutParsed)
	}
	if dateFin != "" {
		recentAlertQuery = recentAlertQuery.Where("date_generation <= ?", dateFinParsed)
	}
	if err := recentAlertQuery.Order("date_generation DESC").Find(&recentAnomalies).Error; err != nil {
		return nil, err
	}
	summary["recent_anomalies"] = recentAnomalies

	// Comptages par type d'anomalie
	typeCountMap := make(map[string]int64)
	var types []struct {
		Type  string
		Count int64
	}
	typeQuery := s.gormDB.WithContext(ctx).Model(&models.Alerte{})
	if dateDebut != "" {
		typeQuery = typeQuery.Where("date_generation >= ?", dateDebutParsed)
	}
	if dateFin != "" {
		typeQuery = typeQuery.Where("date_generation <= ?", dateFinParsed)
	}
	if err := typeQuery.Select("type, COUNT(*) as count").Group("type").Scan(&types).Error; err != nil {
		return nil, err
	}
	for _, t := range types {
		typeCountMap[t.Type] = t.Count
	}
	summary["anomalies_by_type"] = typeCountMap

	return summary, nil
}
