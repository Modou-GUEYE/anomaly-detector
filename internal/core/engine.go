package core

import (
	"anomaly-detector/internal/models"
	"anomaly-detector/internal/store"
	"context"
	"errors"
	"fmt"
	"strings"
)

var ErrTransactionRefused = errors.New("transaction refusée")

type RulesEngine struct{ store *store.Store }

func NewRulesEngine(store *store.Store) *RulesEngine { return &RulesEngine{store: store} }

// AnalyzeTransaction est la fonction principale qui orchestre toute l'analyse.
func (re *RulesEngine) AnalyzeTransaction(ctx context.Context, tx *models.Transaction) ([]*models.Alerte, error) {
	var alertesGenerees []*models.Alerte

	// --- 1. CONTRÔLE BLACKLIST  ---
	err := re.checkBlacklist(ctx, tx)
	if err != nil {
		return nil, fmt.Errorf("impossible d'analyser la transaction: %w", err)
	}

	// --- 2. APPLICATION DES RÈGLES MÉTIER ---
	alertesPayeur, err := re.applyRulesForAccount(ctx, tx, tx.NumeroComptePayeur)
	if err != nil {
		return nil, err
	}
	alertesGenerees = append(alertesGenerees, alertesPayeur...)

	if tx.NumeroComptePaye != "" {
		alertesPaye, err := re.applyRulesForAccount(ctx, tx, tx.NumeroComptePaye)
		if err != nil {
			return nil, err
		}
		alertesGenerees = append(alertesGenerees, alertesPaye...)
	}

	// --- 3. REGLE PAYS_A_RISQUE ---
	reglePaysRisque, err := re.store.GetReglePaysRisqueActive(ctx)
	if err != nil || reglePaysRisque == nil {
		fmt.Printf("Erreur récupération règle pays_risque: %v\n", err)
	} else {
		for _, pays := range reglePaysRisque.ListePays {
			if strings.EqualFold(tx.PaysPayeur, pays) {
				alerte := &models.Alerte{
					TransactionID:     tx.ID,
					RegleDeclencheeID: reglePaysRisque.ID,
					NiveauCriticite:   reglePaysRisque.Criticite,
					ScoreRisque:       calculerScoreRisque(reglePaysRisque, tx),
					RaisonDetection:   fmt.Sprintf("Transaction depuis un pays à haut risque: %s", tx.PaysPayeur),
				}
				alertesGenerees = append(alertesGenerees, alerte)
			}
			if strings.EqualFold(tx.PaysPaye, pays) {
				alerte := &models.Alerte{
					TransactionID:     tx.ID,
					RegleDeclencheeID: reglePaysRisque.ID,
					NiveauCriticite:   reglePaysRisque.Criticite,
					ScoreRisque:       calculerScoreRisque(reglePaysRisque, tx),
					RaisonDetection:   fmt.Sprintf("Transaction vers un pays à haut risque: %s", tx.PaysPaye),
				}
				alertesGenerees = append(alertesGenerees, alerte)
			}
		}
	}

	return alertesGenerees, nil
}

// calculerScoreRisque calcule le score de risque pour une règle et une transaction donnée.
func calculerScoreRisque(regle *models.RegleControle, tx *models.Transaction) float64 {
	switch regle.TypeRegle {
	case "montant":
		return 0.8
	case "frequence":
		return 0.7
	case "pays_risque":
		return 1.0
	default:
		return 0.5
	}
}

// applyRulesForAccount applique les règles métier pour un compte spécifique.
func (re *RulesEngine) applyRulesForAccount(ctx context.Context, tx *models.Transaction, numeroCompte string) ([]*models.Alerte, error) {
	var alertes []*models.Alerte
	regles, err := re.store.GetReglesForNumeroCompte(ctx, numeroCompte)
	if err != nil {
		return nil, err
	}

	for _, regle := range regles {
		var violation = false
		var raison = ""

		switch regle.TypeRegle {

		case "montant":
			// Vérification du seuil sur la période de référence
			// Exemple: Si la période de référence est "7d", on vérifie le cumul sur 7 jours
			if regle.PeriodeReference == "7d" {
				total, _ := re.store.GetTransactionSumForPeriod(ctx, numeroCompte, 7)
				if total > regle.Seuil {
					violation, raison = true, fmt.Sprintf("Cumul sur 7 jours de %.2f dépassant le seuil de %.2f.", total, regle.Seuil)
				}
			}
		case "frequence":
			// Vérification de la fréquence sur la période de référence
			// Exemple: Si la période de référence est "24h", on vérifie le nombre de transactions sur 24h
			if regle.PeriodeReference == "24h" {
				count, _ := re.store.GetTransactionCountForPeriod(ctx, numeroCompte, 1)
				if count > int64(regle.Seuil) {
					violation, raison = true, fmt.Sprintf("Fréquence de %d transactions dépassant le seuil de %d sur 24h.", count, int(regle.Seuil))
				}
			}
		case "pays_risque":
			// Vérification des pays à risque
			// Exemple: Si le pays du payeur est à risque on génère une alerte mais on ne bloque pas la transaction
			isRiskyPayeur, _ := re.store.IsPaysARisque(ctx, tx.PaysPayeur)
			isRiskyPaye, _ := re.store.IsPaysARisque(ctx, tx.PaysPaye)
			if isRiskyPayeur {
				violation, raison = true, fmt.Sprintf("Transaction depuis un pays à haut risque: %s", tx.PaysPayeur)
			} else if isRiskyPaye {
				violation, raison = true, fmt.Sprintf("Transaction vers un pays à haut risque: %s", tx.PaysPaye)
			}
		}

		if violation {
			alerte := &models.Alerte{
				TransactionID:     tx.ID,
				RegleDeclencheeID: regle.ID,
				NiveauCriticite:   regle.Criticite,
				ScoreRisque:       calculerScoreRisque(&regle, tx),
				RaisonDetection:   raison,
			}
			alertes = append(alertes, alerte)
		}
	}
	return alertes, nil
}

// checkBlacklist centralise la logique de la blacklist conformément aux spécifications.
func (re *RulesEngine) checkBlacklist(ctx context.Context, tx *models.Transaction) error {
	// Analyser le payeur
	statusPayeur, err := re.store.IsKycOnBlacklist(ctx, tx.NumeroComptePayeur)
	if err != nil {
		return err
	}

	switch statusPayeur {
	// Refuser immédiatement si "Interdits" ou "Blacklist" et ajouter une entrée de suivi
	case "Interdits", "Blacklist":
		re.store.CreateSuiviBlacklist(ctx, tx.ID, tx.NumeroComptePayeur, statusPayeur)
		return fmt.Errorf("%w: le payeur est sur la liste '%s'", ErrTransactionRefused, statusPayeur)
		// Ajouter une entrée de suivi si "PEP" ou "Surveillés" et ne pas bloquer la transaction
	case "PEP", "Surveillés":
		re.store.CreateSuiviBlacklist(ctx, tx.ID, tx.NumeroComptePayeur, statusPayeur)
	}

	// Analyser le payé
	if tx.NumeroComptePaye != "" {
		statusPaye, err := re.store.IsKycOnBlacklist(ctx, tx.NumeroComptePaye)
		if err != nil {
			return err
		}

		switch statusPaye {
		case "Interdits", "Blacklist":
			re.store.CreateSuiviBlacklist(ctx, tx.ID, tx.NumeroComptePaye, statusPaye)
			return fmt.Errorf("%w: le destinataire est sur la liste '%s'", ErrTransactionRefused, statusPaye)
		case "PEP", "Surveillés":
			re.store.CreateSuiviBlacklist(ctx, tx.ID, tx.NumeroComptePaye, statusPaye)
		}
	}

	return nil
}
