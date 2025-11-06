package store

import (
	"context"
	"fmt"
	"strings"
	"time"

	"anomaly-detector/internal/models"
)

// ===================== EFFICIENCY METRICS (Données Réelles) =====================

// EfficiencyMetric représente une métrique d'efficacité pour une période donnée
type EfficiencyMetric struct {
	Period            string    `json:"period"`
	AvgProcessingTime float64   `json:"avg_processing_time"`
	ResolutionRate    float64   `json:"resolution_rate"`
	Date              time.Time `json:"date"`
}

// GetEfficiencyMetrics calcule les vraies métriques d'efficacité depuis la base de données
func (s *Store) GetEfficiencyMetrics(ctx context.Context, period string) ([]EfficiencyMetric, error) {
	var metrics []EfficiencyMetric
	var err error

	switch period {
	case "month":
		metrics, err = s.getWeeklyEfficiencyMetrics(ctx)
	case "quarter":
		metrics, err = s.getMonthlyEfficiencyMetrics(ctx, 3)
	default: // year
		metrics, err = s.getMonthlyEfficiencyMetrics(ctx, 12)
	}

	return metrics, err
}

// getWeeklyEfficiencyMetrics calcule les métriques par semaine (dernier mois)
func (s *Store) getWeeklyEfficiencyMetrics(ctx context.Context) ([]EfficiencyMetric, error) {
	// Récupérer toutes les alertes des 4 dernières semaines avec GORM
	var alertes []models.Alerte
	err := s.gormDB.WithContext(ctx).
		Where("date_generation IS NOT NULL").
		Where("date_generation >= ?", time.Now().AddDate(0, 0, -28)).
		Find(&alertes).Error

	if err != nil {
		return nil, err
	}

	// Grouper les alertes par semaine et calculer les métriques
	weeklyData := make(map[string]struct {
		TotalAlertes    int
		AlertesTraitees int
		TotalJours      float64
		CountJours      int
	})

	for _, alerte := range alertes {
		// Calculer le début de la semaine (lundi)
		year, week := alerte.DateGeneration.ISOWeek()
		weekKey := fmt.Sprintf("%d-W%02d", year, week)

		data := weeklyData[weekKey]
		data.TotalAlertes++

		// Vérifier si l'alerte est traitée ET a une date de traitement réelle
		if strings.Contains(strings.ToLower(alerte.Statut), "trait") && alerte.DateTraitement != nil {
			data.AlertesTraitees++

			// Calculer le temps de traitement réel en jours
			heures := alerte.DateTraitement.Sub(alerte.DateGeneration).Hours()
			jours := heures / 24.0 // Convertir en jours
			data.TotalJours += jours
			data.CountJours++
		}

		weeklyData[weekKey] = data
	}

	// Générer les métriques pour les 4 dernières semaines
	var metrics []EfficiencyMetric
	now := time.Now()

	for i := 3; i >= 0; i-- {
		weekDate := now.AddDate(0, 0, -i*7)
		year, week := weekDate.ISOWeek()
		weekKey := fmt.Sprintf("%d-W%02d", year, week)

		if data, exists := weeklyData[weekKey]; exists {
			avgTime := 0.0
			if data.CountJours > 0 {
				avgTime = data.TotalJours / float64(data.CountJours)
				// Arrondir le temps de traitement à 1 décimale (jours)
				avgTime = float64(int(avgTime*10+0.5)) / 10
			}

			resolutionRate := 0.0
			if data.TotalAlertes > 0 {
				resolutionRate = (float64(data.AlertesTraitees) / float64(data.TotalAlertes)) * 100
				// Arrondir le pourcentage à 1 décimale
				resolutionRate = float64(int(resolutionRate*10+0.5)) / 10
			}

			metrics = append(metrics, EfficiencyMetric{
				Period:            fmt.Sprintf("Sem %d", 4-i),
				AvgProcessingTime: avgTime,
				ResolutionRate:    resolutionRate,
				Date:              weekDate,
			})
		} else {
			// Aucune donnée pour cette semaine
			metrics = append(metrics, EfficiencyMetric{
				Period:            fmt.Sprintf("Sem %d", 4-i),
				AvgProcessingTime: 0.0,
				ResolutionRate:    0.0,
				Date:              weekDate,
			})
		}
	}

	return metrics, nil
}

// getMonthlyEfficiencyMetrics calcule les métriques par mois
func (s *Store) getMonthlyEfficiencyMetrics(ctx context.Context, months int) ([]EfficiencyMetric, error) {
	// Récupérer toutes les alertes des derniers mois avec GORM
	var alertes []models.Alerte
	err := s.gormDB.WithContext(ctx).
		Where("date_generation IS NOT NULL").
		Where("date_generation >= ?", time.Now().AddDate(0, -months, 0)).
		Find(&alertes).Error

	if err != nil {
		return nil, err
	}

	// Grouper les alertes par mois et calculer les métriques
	monthlyData := make(map[string]struct {
		TotalAlertes    int
		AlertesTraitees int
		TotalJours      float64 // Renommé pour clarifier l'unité
		CountJours      int     // Renommé pour clarifier l'unité
	})

	for _, alerte := range alertes {
		monthKey := alerte.DateGeneration.Format("2006-01")

		data := monthlyData[monthKey]
		data.TotalAlertes++

		// Vérifier si l'alerte est traitée ET a une date de traitement réelle
		if strings.Contains(strings.ToLower(alerte.Statut), "trait") && alerte.DateTraitement != nil {
			data.AlertesTraitees++

			// Calculer le temps de traitement réel en jours
			heures := alerte.DateTraitement.Sub(alerte.DateGeneration).Hours()
			jours := heures / 24.0 // Convertir en jours
			data.TotalJours += jours
			data.CountJours++
		}

		monthlyData[monthKey] = data
	}

	// Générer les 12 derniers mois même s'il n'y a pas de données
	var metrics []EfficiencyMetric
	now := time.Now()

	for i := months - 1; i >= 0; i-- {
		monthDate := now.AddDate(0, -i, 0)
		monthDate = time.Date(monthDate.Year(), monthDate.Month(), 1, 0, 0, 0, 0, monthDate.Location())
		monthKey := monthDate.Format("2006-01")

		if data, exists := monthlyData[monthKey]; exists {
			avgTime := 0.0
			if data.CountJours > 0 {
				avgTime = data.TotalJours / float64(data.CountJours)
				// Arrondir le temps de traitement à 1 décimale (jours)
				avgTime = float64(int(avgTime*10+0.5)) / 10
			}

			resolutionRate := 0.0
			if data.TotalAlertes > 0 {
				resolutionRate = (float64(data.AlertesTraitees) / float64(data.TotalAlertes)) * 100
				// Arrondir le pourcentage à 1 décimale
				resolutionRate = float64(int(resolutionRate*10+0.5)) / 10
			}

			// Formatage du nom du mois en français
			monthName := s.formatMonthName(monthDate)

			metrics = append(metrics, EfficiencyMetric{
				Period:            monthName,
				AvgProcessingTime: avgTime,
				ResolutionRate:    resolutionRate,
				Date:              monthDate,
			})
		} else {
			// Aucune donnée pour ce mois, créer une entrée avec des valeurs par défaut
			monthName := s.formatMonthName(monthDate)
			metrics = append(metrics, EfficiencyMetric{
				Period:            monthName,
				AvgProcessingTime: 0.0,
				ResolutionRate:    0.0,
				Date:              monthDate,
			})
		}
	}

	return metrics, nil
}

// formatMonthName convertit une date en nom de mois français
func (s *Store) formatMonthName(date time.Time) string {
	months := []string{
		"Janvier", "Février", "Mars", "Avril", "Mai", "Juin",
		"Juillet", "Août", "Septembre", "Octobre", "Novembre", "Décembre",
	}

	month := int(date.Month())
	if month < 1 || month > 12 {
		return "Mois invalide"
	}

	return months[month-1]
}
