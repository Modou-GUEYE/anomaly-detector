package store

import (
	"context"
	"fmt"
	"time"

	"anomaly-detector/internal/models"
)

// GetRuleViolationsByPeriod récupère les violations de règles par période
func (s *Store) GetRuleViolationsByPeriod(ctx context.Context, period string) ([]RuleViolation, error) {
	var violations []RuleViolation
	var err error

	switch period {
	case "month":
		violations, err = s.getRuleViolationsForMonth(ctx)
	case "quarter":
		violations, err = s.getRuleViolationsForQuarter(ctx)
	case "year":
	default:
		violations, err = s.getRuleViolationsForYear(ctx)
	}

	if err != nil {
		return nil, fmt.Errorf("erreur lors de la récupération des violations de règles pour la période %s: %w", period, err)
	}

	// S'assurer qu'on retourne toujours un tableau, jamais nil
	if violations == nil {
		violations = []RuleViolation{}
	}

	// Calculer les pourcentages
	total := 0
	for _, v := range violations {
		total += v.Count
	}

	for i := range violations {
		if total > 0 {
			violations[i].Percentage = float64(violations[i].Count) / float64(total) * 100
		}
	}

	return violations, nil
}

// getRuleViolationsForMonth calcule les violations sur le dernier mois
func (s *Store) getRuleViolationsForMonth(ctx context.Context) ([]RuleViolation, error) {
	startDate := time.Now().AddDate(0, -1, 0) // Dernier mois
	return s.getRuleViolationsForDateRange(ctx, startDate, time.Now())
}

// getRuleViolationsForQuarter calcule les violations sur le dernier trimestre
func (s *Store) getRuleViolationsForQuarter(ctx context.Context) ([]RuleViolation, error) {
	startDate := time.Now().AddDate(0, -3, 0) // Dernier trimestre
	return s.getRuleViolationsForDateRange(ctx, startDate, time.Now())
}

// getRuleViolationsForYear calcule les violations sur la dernière année
// Si aucune donnée sur l'année, utilise les données du mois
func (s *Store) getRuleViolationsForYear(ctx context.Context) ([]RuleViolation, error) {
	startDate := time.Now().AddDate(-1, 0, 0) // Dernière année
	violations, err := s.getRuleViolationsForDateRange(ctx, startDate, time.Now())

	// Si aucune donnée sur l'année, utiliser les données du mois comme fallback
	if err == nil && len(violations) == 0 {
		return s.getRuleViolationsForMonth(ctx)
	}

	return violations, err
}

// getRuleViolationsForDateRange calcule les violations pour une plage de dates
func (s *Store) getRuleViolationsForDateRange(_ context.Context, startDate, endDate time.Time) ([]RuleViolation, error) {
	var results []struct {
		Type  string `gorm:"column:type"`
		Count int    `gorm:"column:count"`
	}

	err := s.gormDB.Model(&models.Alerte{}).
		Select("type, COUNT(*) as count").
		Where("date_generation >= ? AND date_generation <= ?", startDate, endDate).
		Group("type").
		Order("count DESC").
		Scan(&results).Error
	if err != nil {
		return nil, fmt.Errorf("erreur lors de l'exécution de la requête des violations: %w", err)
	}

	violations := make([]RuleViolation, len(results))
	for i, result := range results {
		violations[i] = RuleViolation{
			RuleName: result.Type,
			Count:    result.Count,
		}
	}

	return violations, nil
}
