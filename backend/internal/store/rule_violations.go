package store

import (
	"context"
	"fmt"
	"time"

	"anomaly-detector/internal/models"
)

// Helpers pour bornes de périodes calendaires (UTC)
func startOfMonthUTC(t time.Time) time.Time {
	tt := t.UTC()
	return time.Date(tt.Year(), tt.Month(), 1, 0, 0, 0, 0, time.UTC)
}

func startOfQuarterUTC(t time.Time) time.Time {
	tt := t.UTC()
	qStartMonth := ((int(tt.Month())-1)/3)*3 + 1 // 1,4,7,10
	return time.Date(tt.Year(), time.Month(qStartMonth), 1, 0, 0, 0, 0, time.UTC)
}

func startOfYearUTC(t time.Time) time.Time {
	tt := t.UTC()
	return time.Date(tt.Year(), time.January, 1, 0, 0, 0, 0, time.UTC)
}

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
		violations, err = s.getRuleViolationsForYear(ctx)
	default:
		// Par sécurité, fallback sur la dernière année
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
	now := time.Now().UTC()
	startDate := startOfMonthUTC(now) // Mois calendaire courant (1er du mois)
	return s.getRuleViolationsForDateRange(ctx, startDate, now)
}

// getRuleViolationsForQuarter calcule les violations sur le dernier trimestre
func (s *Store) getRuleViolationsForQuarter(ctx context.Context) ([]RuleViolation, error) {
	now := time.Now().UTC()
	startDate := startOfQuarterUTC(now) // Trimestre calendaire courant (Q1/Q2/Q3/Q4)
	return s.getRuleViolationsForDateRange(ctx, startDate, now)
}

// getRuleViolationsForYear calcule les violations sur la dernière année
func (s *Store) getRuleViolationsForYear(ctx context.Context) ([]RuleViolation, error) {
	now := time.Now().UTC()
	startDate := startOfYearUTC(now) // Année calendaire courante (1er janvier)
	return s.getRuleViolationsForDateRange(ctx, startDate, now)
}

// getRuleViolationsForDateRange calcule les violations pour une plage de dates
func (s *Store) getRuleViolationsForDateRange(ctx context.Context, startDate, endDate time.Time) ([]RuleViolation, error) {
	var results []struct {
		Type  string `gorm:"column:type"`
		Count int    `gorm:"column:count"`
	}

	err := s.gormDB.WithContext(ctx).Model(&models.Alerte{}).
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
