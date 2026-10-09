package store

import (
	"context"
	"fmt"
	"strings"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"anomaly-detector/internal/models"
)

// ===================== CHEMIN CRITIQUE D'ANALYSE =====================
//
// Ces méthodes regroupent les accès SQL du endpoint /anomalies/analyze afin de limiter
// le nombre d'allers-retours et surtout le nombre de COMMIT (un fsync disque chacun) :
//   - EnsureComptes      : au plus 1 SELECT + 1 transaction, et rien si les comptes sont en cache
//   - GetAccountActivity : 1 seule requête d'agrégat par compte, quelle que soit la liste de règles
//   - SaveAnalysis       : transaction, alertes, audit et suivi écrits dans une seule transaction

// PeriodStats contient le cumul des montants et le nombre de transactions d'un compte sur une période.
type PeriodStats struct {
	Sum   float64
	Count int64
}

// EnsureComptes garantit l'existence en base des comptes fournis. Les comptes absents sont
// insérés (avec leur log d'audit) dans une seule transaction ; un compte créé entre-temps par
// une requête concurrente est ignoré grâce à ON CONFLICT DO NOTHING puis relu.
func (s *Store) EnsureComptes(ctx context.Context, comptes []*models.Compte) error {
	missing := make([]*models.Compte, 0, len(comptes))
	numeros := make([]string, 0, len(comptes))
	for _, c := range comptes {
		if c == nil || c.NumeroCompte == "" {
			continue
		}
		if val, ok := s.accountCache.Load(c.NumeroCompte); ok && val.(*models.Compte) != nil {
			continue
		}
		missing = append(missing, c)
		numeros = append(numeros, c.NumeroCompte)
	}
	if len(missing) == 0 {
		return nil
	}

	found, err := s.loadComptes(ctx, numeros)
	if err != nil {
		return err
	}

	toCreate := make([]*models.Compte, 0, len(missing))
	for _, c := range missing {
		if _, ok := found[c.NumeroCompte]; !ok {
			toCreate = append(toCreate, c)
		}
	}
	if len(toCreate) == 0 {
		return nil
	}

	now := time.Now()
	var conflicts []string
	err = s.gormDB.WithContext(ctx).Transaction(func(db *gorm.DB) error {
		logs := make([]models.LogAudit, 0, len(toCreate))
		for _, c := range toCreate {
			if c.DateCreation.IsZero() {
				c.DateCreation = now
			}
			res := db.Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "numero_compte"}}, DoNothing: true}).Create(c)
			if res.Error != nil {
				return res.Error
			}
			if res.RowsAffected == 0 {
				conflicts = append(conflicts, c.NumeroCompte)
				continue
			}
			logs = append(logs, models.LogAudit{
				UtilisateurID:   "MDA",
				ActionEffectuee: fmt.Sprintf("Création du compte '%s'", c.NumeroCompte),
				TypeAction:      "Comptes",
				EntityID:        c.ID,
				Details:         fmt.Sprintf("Création du compte: %s (%s)", c.NumeroCompte, c.NomClient),
				Resultat:        "succès",
				Role:            "system",
				DateHeure:       now,
			})
		}
		if len(logs) > 0 {
			return db.Create(&logs).Error
		}
		return nil
	})
	if err != nil {
		return err
	}

	for _, c := range toCreate {
		if c.ID != "" {
			s.cacheAccount(c)
		}
	}
	if len(conflicts) > 0 {
		_, err = s.loadComptes(ctx, conflicts)
	}
	return err
}

// loadComptes lit les comptes demandés en une requête et les place en cache.
func (s *Store) loadComptes(ctx context.Context, numeros []string) (map[string]*models.Compte, error) {
	var rows []*models.Compte
	if err := s.gormDB.WithContext(ctx).Where("numero_compte IN ?", numeros).Find(&rows).Error; err != nil {
		return nil, err
	}
	found := make(map[string]*models.Compte, len(rows))
	for _, c := range rows {
		found[c.NumeroCompte] = c
		s.cacheAccount(c)
	}
	return found, nil
}

// GetAccountActivity calcule en une seule requête le cumul et le nombre de transactions d'un
// compte (payeur ou payé) pour chacune des périodes demandées, exprimées en jours.
func (s *Store) GetAccountActivity(ctx context.Context, numeroCompte string, periods []int) (map[int]PeriodStats, error) {
	result := make(map[int]PeriodStats, len(periods))
	if len(periods) == 0 {
		return result, nil
	}

	now := time.Now()
	cols := make([]string, 0, len(periods)*2)
	args := make([]interface{}, 0, len(periods)*2+3)
	maxDays := 0
	for _, days := range periods {
		since := now.AddDate(0, 0, -days)
		cols = append(cols,
			"COALESCE(SUM(montant) FILTER (WHERE date_operation >= ?), 0)",
			"COUNT(*) FILTER (WHERE date_operation >= ?)")
		args = append(args, since, since)
		if days > maxDays {
			maxDays = days
		}
	}
	args = append(args, numeroCompte, numeroCompte, now.AddDate(0, 0, -maxDays))

	query := "SELECT " + strings.Join(cols, ", ") +
		" FROM transactions WHERE (numero_compte_payeur = ? OR numero_compte_paye = ?) AND date_operation >= ?"

	row := s.gormDB.WithContext(ctx).Raw(query, args...).Row()
	values := make([]PeriodStats, len(periods))
	dest := make([]interface{}, 0, len(periods)*2)
	for i := range values {
		dest = append(dest, &values[i].Sum, &values[i].Count)
	}
	if err := row.Scan(dest...); err != nil {
		return nil, err
	}
	for i, days := range periods {
		result[days] = values[i]
	}
	return result, nil
}

// AnalysisAlert associe une alerte à persister à l'indicateur de création d'un suivi blacklist.
type AnalysisAlert struct {
	Alerte      *models.Alerte
	CreateSuivi bool
}

// SaveAnalysis enregistre dans une seule transaction SQL la transaction analysée, ses alertes,
// les logs d'audit correspondants et les entrées de suivi_blacklist.
func (s *Store) SaveAnalysis(ctx context.Context, tx *models.Transaction, alerts []AnalysisAlert) error {
	now := time.Now()
	if tx.DateOperation.IsZero() {
		tx.DateOperation = now
	}

	return s.gormDB.WithContext(ctx).Transaction(func(db *gorm.DB) error {
		if err := db.Create(tx).Error; err != nil {
			return err
		}
		if len(alerts) == 0 {
			return nil
		}

		alertes := make([]*models.Alerte, 0, len(alerts))
		for _, a := range alerts {
			a.Alerte.TransactionID = &tx.ID
			if a.Alerte.DateGeneration.IsZero() {
				a.Alerte.DateGeneration = now
			}
			if strings.Contains(strings.ToLower(a.Alerte.Statut), "trait") && a.Alerte.DateTraitement == nil {
				a.Alerte.DateTraitement = &now
			}
			alertes = append(alertes, a.Alerte)
		}
		if err := db.Create(&alertes).Error; err != nil {
			return err
		}

		logs := make([]models.LogAudit, 0, len(alertes))
		suivis := make([]models.SuiviBlacklist, 0, len(alertes))
		for _, a := range alerts {
			logs = append(logs, newAlertAuditLog(a.Alerte, now))
			if a.CreateSuivi {
				suivis = append(suivis, models.SuiviBlacklist{
					TransactionID: tx.ID,
					NumeroCompte:  a.Alerte.NumeroCompte,
					StatutSuivi:   "surveiller",
					Raison:        a.Alerte.Type,
					DateDetection: now,
				})
			}
		}
		if err := db.Create(&logs).Error; err != nil {
			return err
		}
		if len(suivis) > 0 {
			return db.Create(&suivis).Error
		}
		return nil
	})
}

// newAlertAuditLog construit le log d'audit associé à la génération d'une alerte par le moteur.
func newAlertAuditLog(alerte *models.Alerte, at time.Time) models.LogAudit {
	resultat := map[string]string{"bloquer": "bloqué", "surveiller": "surveillance"}[strings.ToLower(alerte.Action)]
	if resultat == "" {
		resultat = "surveillance"
	}
	return models.LogAudit{
		UtilisateurID:   "MDA",
		ActionEffectuee: fmt.Sprintf("Génération alerte '%s'", alerte.Type),
		TypeAction:      "Alertes",
		EntityID:        alerte.ID,
		Details:         fmt.Sprintf("Nouvelle alerte créée - Type: %s, Action: %s, Code: %s", alerte.Type, alerte.Action, alerte.Code),
		Resultat:        resultat,
		Role:            "system",
		DateHeure:       at,
	}
}
