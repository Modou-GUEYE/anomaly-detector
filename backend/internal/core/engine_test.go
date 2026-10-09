package core

import (
	"reflect"
	"testing"

	"anomaly-detector/internal/models"
	"anomaly-detector/internal/store"
)

func TestActivityPeriods(t *testing.T) {
	regles := []models.RegleControle{
		{TypeRegle: "montant", PeriodeReference: "24h"},
		{TypeRegle: "montant", PeriodeReference: "30d"},
		{TypeRegle: "frequence", PeriodeReference: "24h"}, // même période que la 1re : pas de doublon
		{TypeRegle: "montant"},                            // défaut montant : 7 jours
		{TypeRegle: "pays_risque"},                        // n'a pas besoin d'historique
	}
	got := activityPeriods(regles)
	want := []int{1, 30, 7}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("activityPeriods() = %v, want %v", got, want)
	}

	if got := activityPeriods([]models.RegleControle{{TypeRegle: "pays_risque"}}); len(got) != 0 {
		t.Fatalf("aucune période attendue sans règle montant/fréquence, got %v", got)
	}
}

func TestIsRuleViolated(t *testing.T) {
	activity := map[int]store.PeriodStats{
		1:  {Sum: 40_000_000, Count: 19},
		30: {Sum: 250_000_000, Count: 100},
	}
	tests := []struct {
		name    string
		montant float64
		regle   models.RegleControle
		want    bool
	}{
		{"montant 24h sous le seuil", 5_000_000, models.RegleControle{TypeRegle: "montant", PeriodeReference: "24h", Seuil: 50_000_000}, false},
		{"montant 24h au-dessus du seuil", 15_000_000, models.RegleControle{TypeRegle: "montant", PeriodeReference: "24h", Seuil: 50_000_000}, true},
		{"montant 30j au-dessus du seuil", 60_000_000, models.RegleControle{TypeRegle: "montant", PeriodeReference: "30d", Seuil: 300_000_000}, true},
		{"fréquence atteint le seuil sans le dépasser", 1, models.RegleControle{TypeRegle: "frequence", PeriodeReference: "24h", Seuil: 20}, false},
		{"fréquence dépasse le seuil", 1, models.RegleControle{TypeRegle: "frequence", PeriodeReference: "24h", Seuil: 19}, true},
		{"période sans historique", 1, models.RegleControle{TypeRegle: "montant", PeriodeReference: "7d", Seuil: 10}, false},
		{"type non évalué ici", 1e12, models.RegleControle{TypeRegle: "pays_risque"}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tx := &models.Transaction{Montant: tt.montant}
			if got := isRuleViolated(tx, tt.regle, activity); got != tt.want {
				t.Fatalf("isRuleViolated() = %v, want %v", got, tt.want)
			}
		})
	}
}
