package core

import (
	"anomaly-detector/internal/models"
	"anomaly-detector/internal/store"
	"context"
	"errors"
	"fmt"
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
		return nil, err
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

	return alertesGenerees, nil
}

// applyRulesForAccount est une fonction helper qui applique les règles pour un numéro de compte donné.
func (re *RulesEngine) applyRulesForAccount(ctx context.Context, tx *models.Transaction, numeroCompte string) ([]*models.Alerte, error) {
	var alertes []*models.Alerte
	regles, err := re.store.GetReglesForNumeroCompte(ctx, numeroCompte)
	if err != nil {
		return nil, fmt.Errorf("impossible de récupérer les règles pour le compte %s: %w", numeroCompte, err)
	}

	for _, regle := range regles {
		var violation = false
		var raison = ""

		switch regle.TypeRegle {
		case "montant":
			if regle.PeriodeReference == "7d" {
				total, _ := re.store.GetTransactionSumForPeriod(ctx, numeroCompte, 7)
				if total > regle.Seuil {
					violation, raison = true, fmt.Sprintf("Cumul sur 7 jours de %.2f dépassant le seuil de %.2f.", total, regle.Seuil)
				}
			}
		case "frequence":
			if regle.PeriodeReference == "24h" {
				count, _ := re.store.GetTransactionCountForPeriod(ctx, numeroCompte, 1)
				if count > int64(regle.Seuil) {
					violation, raison = true, fmt.Sprintf("Fréquence de %d transactions dépassant le seuil de %d sur 24h.", count, int(regle.Seuil))
				}
			}
		case "pays_risque":
			isRisky, _ := re.store.IsPaysARisque(ctx, tx.PaysPaye)
			if isRisky {
				violation, raison = true, fmt.Sprintf("Transaction vers un pays à haut risque: %s", tx.PaysPaye)
			}
		}

		if violation {
			alerte := &models.Alerte{
				TransactionID:     tx.ID,
				RegleDeclencheeID: regle.ID,
				NiveauCriticite:   regle.Criticite,
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
	case "Interdits", "Blacklist":
		// Règle : Refuser la transaction et renseigner un suivi (même si refusé).
		re.store.CreateSuiviBlacklist(ctx, tx.ID, tx.NumeroComptePayeur, statusPayeur)
		return fmt.Errorf("%w: le payeur est sur la liste '%s'", ErrTransactionRefused, statusPayeur)
	case "PEP", "Surveillés":
		// Règle : Valider et renseigner un état de suivi.
		re.store.CreateSuiviBlacklist(ctx, tx.ID, tx.NumeroComptePayeur, statusPayeur)
	}

	// Analyser le payé (s'il existe)
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
