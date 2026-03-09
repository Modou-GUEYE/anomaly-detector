package store

import (
	"anomaly-detector/internal/models"
	"context"
	"time"

	"gorm.io/gorm"
)

const (
	whereDateGenerationGTE = "date_generation >= ?"
	whereDateGenerationLTE = "date_generation <= ?"
)

type dashboardDateRange struct {
	dateDebutParsed time.Time
	dateFinParsed   time.Time
	hasDateDebut    bool
	hasDateFin      bool
}

func parseDashboardDateRange(dateDebut string, dateFin string) (*dashboardDateRange, error) {
	rangeFilter := &dashboardDateRange{
		hasDateDebut: dateDebut != "",
		hasDateFin:   dateFin != "",
	}

	if rangeFilter.hasDateDebut {
		parsed, err := time.Parse("2006-01-02", dateDebut)
		if err != nil {
			return nil, err
		}
		rangeFilter.dateDebutParsed = parsed
	}

	if rangeFilter.hasDateFin {
		parsed, err := time.Parse("2006-01-02", dateFin)
		if err != nil {
			return nil, err
		}
		rangeFilter.dateFinParsed = parsed.Add(23*time.Hour + 59*time.Minute + 59*time.Second)
	}

	return rangeFilter, nil
}

func applyDateFilters(query *gorm.DB, rangeFilter *dashboardDateRange, fieldName string) *gorm.DB {
	if rangeFilter.hasDateDebut {
		query = query.Where(fieldName+" >= ?", rangeFilter.dateDebutParsed)
	}
	if rangeFilter.hasDateFin {
		query = query.Where(fieldName+" <= ?", rangeFilter.dateFinParsed)
	}
	return query
}

func applyDateGenerationFilters(query *gorm.DB, rangeFilter *dashboardDateRange) *gorm.DB {
	if rangeFilter.hasDateDebut {
		query = query.Where(whereDateGenerationGTE, rangeFilter.dateDebutParsed)
	}
	if rangeFilter.hasDateFin {
		query = query.Where(whereDateGenerationLTE, rangeFilter.dateFinParsed)
	}
	return query
}

// GetDashboardSummary retourne un résumé complet du dashboard avec filtrage optionnel par dates
func (s *Store) GetDashboardSummary(ctx context.Context, dateDebut, dateFin string) (map[string]interface{}, error) {
	summary := make(map[string]interface{})

	rangeFilter, err := parseDashboardDateRange(dateDebut, dateFin)
	if err != nil {
		return nil, err
	}

	// Compter les transactions
	var transactionCount int64
	txQuery := applyDateFilters(s.gormDB.WithContext(ctx).Model(&models.Transaction{}), rangeFilter, "date_operation")
	if err := txQuery.Count(&transactionCount).Error; err != nil {
		return nil, err
	}
	summary["transactions_count"] = transactionCount

	// Compter les alertes
	var alertCount int64
	alertQuery := applyDateGenerationFilters(s.gormDB.WithContext(ctx).Model(&models.Alerte{}), rangeFilter)
	if err := alertQuery.Count(&alertCount).Error; err != nil {
		return nil, err
	}
	summary["anomalies_count"] = alertCount

	// Compter les alertes traitées
	var treatedCount int64
	treatedQuery := applyDateGenerationFilters(s.gormDB.WithContext(ctx).Model(&models.Alerte{}).Where("statut ILIKE ?", "%trait%"), rangeFilter)
	if err := treatedQuery.Count(&treatedCount).Error; err != nil {
		return nil, err
	}
	summary["anomalies_treated"] = treatedCount

	// Compter les transactions bloquées
	var blockedTransactions int64
	blockedQuery := applyDateGenerationFilters(s.gormDB.WithContext(ctx).Model(&models.Alerte{}).Where("action = ? AND transaction_id IS NOT NULL", "bloquer"), rangeFilter)
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
	recentAlertQuery := applyDateGenerationFilters(s.gormDB.WithContext(ctx), rangeFilter)
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
	typeQuery := applyDateGenerationFilters(s.gormDB.WithContext(ctx).Model(&models.Alerte{}), rangeFilter)
	if err := typeQuery.Select("type, COUNT(*) as count").Group("type").Scan(&types).Error; err != nil {
		return nil, err
	}
	for _, t := range types {
		typeCountMap[t.Type] = t.Count
	}
	summary["anomalies_by_type"] = typeCountMap

	return summary, nil
}
