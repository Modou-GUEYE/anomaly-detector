package core

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"anomaly-detector/internal/models"
	"anomaly-detector/internal/store"
)

// ===================== CONSTANTES & TYPES =====================

var ErrTransactionRefused = errors.New("transaction refusée")

// RulesEngine centralise l'application des règles métier et des contrôles sur les transactions.
type RulesEngine struct {
	store *store.Store
}

// NewRulesEngine crée une nouvelle instance du moteur de règles métier.
func NewRulesEngine(store *store.Store) *RulesEngine {
	return &RulesEngine{store: store}
}

// ===================== MÉTHODES PUBLIQUES =====================

// AnalyzeTransaction orchestre l'analyse complète d'une transaction (blacklist, règles métier, pays à risque).
func (re *RulesEngine) AnalyzeTransaction(ctx context.Context, tx *models.Transaction) ([]*models.Alerte, error) {
	var alertesGenerees []*models.Alerte

	// Code banque du partenaire principal (à configurer via base de données)
	const partnerBankCode = "SN079"
	isPayerPartner := len(tx.NumeroComptePayeur) >= 5 && tx.NumeroComptePayeur[:5] == partnerBankCode
	isPayeePartner := len(tx.NumeroComptePaye) >= 5 && tx.NumeroComptePaye[:5] == partnerBankCode

	// --- 1. CONTRÔLE BLACKLIST  ---
	if isPayerPartner {
		alertesBlacklist, err := re.checkBlacklist(ctx, tx, true, false)
		if err != nil {
			// Préserver les alertes déjà générées même en cas de refus
			return append(alertesGenerees, alertesBlacklist...), err
		}
		alertesGenerees = append(alertesGenerees, alertesBlacklist...)
	}
	if isPayeePartner && tx.NumeroComptePaye != "" {
		alertesBlacklist, err := re.checkBlacklist(ctx, tx, false, true)
		if err != nil {
			// Préserver les alertes déjà générées même en cas de refus
			return append(alertesGenerees, alertesBlacklist...), err
		}
		alertesGenerees = append(alertesGenerees, alertesBlacklist...)
	}

	// --- 2. APPLICATION DES RÈGLES MÉTIER ---
	if isPayerPartner {
		alertesPayeur, err := re.applyRulesForAccount(ctx, tx, tx.NumeroComptePayeur)
		if err != nil {
			return append(alertesGenerees, alertesPayeur...), err
		}
		alertesGenerees = append(alertesGenerees, alertesPayeur...)
	}
	if isPayeePartner && tx.NumeroComptePaye != "" {
		alertesPaye, err := re.applyRulesForAccount(ctx, tx, tx.NumeroComptePaye)
		if err != nil {
			return append(alertesGenerees, alertesPaye...), err
		}
		alertesGenerees = append(alertesGenerees, alertesPaye...)
	}

	// --- 3. REGLE PAYS_A_RISQUE ---
	reglePaysRisque, err := re.store.GetReglePaysRisqueActive(ctx)
	if err != nil || reglePaysRisque == nil {
		// Règle pas active, pas d'alerte
	} else {
		meta, err := re.store.GetAlertMetaByType(ctx, "Pays à risque")
		if err != nil || meta == nil {
		} else {
			// Récupérer les pays à risque depuis la nouvelle table
			paysARisque, err := re.store.GetAllPaysARisque(ctx)
			if err != nil {
				// Erreur de récupération, continue
			} else {
				for _, paysRecord := range paysARisque {
					// Comparer avec le CODE pays (ISO alpha-2), pas le nom
					codePays := paysRecord.Code
					if isPayerPartner && strings.EqualFold(tx.PaysPayeur, codePays) {
						paysRisque := paysRecord.NomPays // Pour le message d'alerte
						regleID := reglePaysRisque.ID
						// Utiliser l'action de la règle si définie, sinon fallback sur criticité
						action := reglePaysRisque.Action
						if action == "" {
							action = actionFromCriticite(meta.NiveauCriticite)
						}
						alerte := &models.Alerte{
							TransactionID:     tx.ID,
							Code:              meta.Code,
							Type:              meta.Type,
							Statut:            meta.Statut,
							Message:           meta.Message,
							NiveauCriticite:   meta.NiveauCriticite,
							ScoreRisque:       calculerScoreRisque(reglePaysRisque, tx),
							DateGeneration:    tx.DateOperation,
							Action:            action,
							NumeroCompte:      tx.NumeroComptePayeur,
							PaysRisque:        &paysRisque,
							RegleDeclencheeID: &regleID,
						}
						alertesGenerees = append(alertesGenerees, alerte)
						// Si l'action est "bloquer", refuser la transaction
						if action == "bloquer" {
							return alertesGenerees, fmt.Errorf("%w: transaction bloquée - pays à risque détecté (%s)", ErrTransactionRefused, paysRisque)
						}
					}
					if isPayeePartner && strings.EqualFold(tx.PaysPaye, codePays) {
						paysRisque := paysRecord.NomPays // Pour le message d'alerte
						regleID := reglePaysRisque.ID
						// Utiliser l'action de la règle si définie, sinon fallback sur criticité
						action := reglePaysRisque.Action
						if action == "" {
							action = actionFromCriticite(meta.NiveauCriticite)
						}
						alerte := &models.Alerte{
							TransactionID:     tx.ID,
							Code:              meta.Code,
							Type:              meta.Type,
							Statut:            meta.Statut,
							Message:           meta.Message,
							NiveauCriticite:   meta.NiveauCriticite,
							ScoreRisque:       calculerScoreRisque(reglePaysRisque, tx),
							DateGeneration:    tx.DateOperation,
							Action:            action,
							NumeroCompte:      tx.NumeroComptePaye,
							PaysRisque:        &paysRisque,
							RegleDeclencheeID: &regleID,
						}
						alertesGenerees = append(alertesGenerees, alerte)
						// Si l'action est "bloquer", refuser la transaction
						if action == "bloquer" {
							return alertesGenerees, fmt.Errorf("%w: transaction bloquée - pays à risque détecté (%s)", ErrTransactionRefused, paysRisque)
						}
					}
				}
			}
		}
	}

	return alertesGenerees, nil
}

// ===================== HELPERS INTERNES =====================

// calculerScoreRisque calcule le score de risque pour une règle et une transaction donnée.
func calculerScoreRisque(regle *models.RegleControle, _ *models.Transaction) float64 {

	switch regle.TypeRegle {
	case "montant", "pays_risque", "Interdit", "Blacklist":
		return 1.0
	case "frequence", "PEP", "Surveillés":
		return 0.7
	default:
		return 0.5
	}
}

// actionFromCriticite retourne l'action à prendre selon la criticité.
func actionFromCriticite(criticite string) string {
	switch strings.ToLower(criticite) {
	case "haute":
		return "bloquer"
	case "moyenne", "faible":
		return "surveiller"
	default:
		return "surveiller"
	}
}

// applyRulesForAccount applique les règles métier pour un compte spécifique.
func (re *RulesEngine) applyRulesForAccount(ctx context.Context, tx *models.Transaction, numeroCompte string) ([]*models.Alerte, error) {
	var alertes []*models.Alerte
	regles, err := re.store.GetReglesForCompte(ctx, numeroCompte)
	if err != nil {
		return nil, err
	}
	for _, regle := range regles {
		var violation = false
		regleID := regle.ID

		switch regle.TypeRegle {
		case "montant":
			if regle.PeriodeReference == "7d" {
				total, _ := re.store.GetTransactionSumForPeriod(ctx, numeroCompte, 7)
				if total >= regle.Seuil {
					violation = true
				}
				if violation {
					meta, err := re.store.GetAlertMetaByType(ctx, regle.TypeRegle)
					if err != nil || meta == nil {
						return nil, fmt.Errorf("meta-donnée d'alerte introuvable pour le type '%s'", regle.TypeRegle)
					}
					// Utiliser l'action de la règle si définie, sinon fallback sur criticité
					action := regle.Action
					if action == "" {
						action = actionFromCriticite(meta.NiveauCriticite)
					}
					alerte := &models.Alerte{
						TransactionID:     tx.ID,
						Code:              meta.Code,
						Type:              meta.Type,
						Statut:            meta.Statut,
						Message:           meta.Message,
						NiveauCriticite:   meta.NiveauCriticite,
						ScoreRisque:       calculerScoreRisque(&regle, tx),
						DateGeneration:    tx.DateOperation,
						RegleDeclencheeID: &regleID,
						Action:            action,
						NumeroCompte:      numeroCompte,
					}
					alertes = append(alertes, alerte)
					// Si l'action est "bloquer", refuser la transaction
					if action == "bloquer" {
						return alertes, fmt.Errorf("%w: transaction bloquée - règle de montant dépassée", ErrTransactionRefused)
					}
					continue
				}
			}
		case "frequence":
			// Déterminer la période basée sur periodeReference
			var days int
			switch regle.PeriodeReference {
			case "24h":
				days = 1
			case "7d":
				days = 7
			case "30d":
				days = 30
			default:
				days = 1
			}

			count, _ := re.store.GetTransactionCountForPeriod(ctx, numeroCompte, days)
			if count >= int64(regle.Seuil) {
				violation = true
				// Ajout du suivi_blacklist pour violation de la règle de fréquence
				meta, err := re.store.GetAlertMetaByType(ctx, regle.TypeRegle)
				if err == nil && meta != nil {
					// Utiliser l'action de la règle si définie, sinon fallback sur criticité
					action := regle.Action
					if action == "" {
						action = actionFromCriticite(meta.NiveauCriticite)
					}
					_ = re.store.CreateSuiviBlacklist(ctx, tx.ID, numeroCompte, action, regle.TypeRegle)
				}
			}
		}

		if violation && regle.TypeRegle != "montant" {
			meta, err := re.store.GetAlertMetaByType(ctx, regle.TypeRegle)
			if err != nil || meta == nil {
				return nil, fmt.Errorf("meta-donnée d'alerte introuvable pour le type '%s'", regle.TypeRegle)
			}
			// Utiliser l'action de la règle si définie, sinon fallback sur criticité
			action := regle.Action
			if action == "" {
				action = actionFromCriticite(meta.NiveauCriticite)
			}
			alerte := &models.Alerte{
				TransactionID:     tx.ID,
				Code:              meta.Code,
				Type:              meta.Type,
				Statut:            meta.Statut,
				Message:           meta.Message,
				NiveauCriticite:   meta.NiveauCriticite,
				ScoreRisque:       calculerScoreRisque(&regle, tx),
				DateGeneration:    tx.DateOperation,
				Action:            action,
				RegleDeclencheeID: &regleID,
				NumeroCompte:      numeroCompte,
			}
			alertes = append(alertes, alerte)
			// Si l'action est "bloquer", refuser la transaction
			if action == "bloquer" {
				return alertes, fmt.Errorf("%w: transaction bloquée - règle de %s dépassée", ErrTransactionRefused, regle.TypeRegle)
			}
		}
	}
	return alertes, nil
}

// checkBlacklist centralise la logique de la blacklist conformément aux spécifications.
// Seul le compte BIS sélectionné (payeur ou payé) est vérifié selon les flags.
func (re *RulesEngine) checkBlacklist(ctx context.Context, tx *models.Transaction, checkPayeur bool, checkPaye bool) ([]*models.Alerte, error) {
	var alertes []*models.Alerte
	if checkPayeur {
		statusPayeur, err := re.store.IsAccountOnBlacklist(ctx, tx.NumeroComptePayeur)
		if err != nil {
			return nil, err
		}
		switch statusPayeur {
		case "Interdit", "Blacklist", "PEP", "Surveillés":
			meta, err := re.store.GetAlertMetaByType(ctx, statusPayeur)
			var regleID *string
			regle, _ := re.store.GetRegleByType(ctx, statusPayeur)
			if regle != nil {
				regleID = &regle.ID
			}
			if err == nil && meta != nil {
				var scoreRisque float64 = 0
				switch statusPayeur {
				case "Interdit", "Blacklist":
					scoreRisque = 1.0
				case "PEP", "Surveillés":
					scoreRisque = 0.7
				}
				alerte := &models.Alerte{
					TransactionID:     tx.ID,
					Code:              meta.Code,
					Type:              meta.Type,
					Statut:            meta.Statut,
					Message:           meta.Message,
					NiveauCriticite:   meta.NiveauCriticite,
					ScoreRisque:       scoreRisque,
					DateGeneration:    tx.DateOperation,
					Action:            actionFromCriticite(meta.NiveauCriticite),
					RegleDeclencheeID: regleID,
					NumeroCompte:      tx.NumeroComptePayeur,
				}
				alertes = append(alertes, alerte)
			}
			if statusPayeur == "Interdit" || statusPayeur == "Blacklist" {
				if meta == nil {
					return alertes, fmt.Errorf("%w: meta-donnée d'alerte introuvable pour %s", ErrTransactionRefused, statusPayeur)
				}
				return alertes, fmt.Errorf("%w: %s", ErrTransactionRefused, meta.Message)
			}
		}
	}
	if checkPaye && tx.NumeroComptePaye != "" {
		statusPaye, err := re.store.IsAccountOnBlacklist(ctx, tx.NumeroComptePaye)
		if err != nil {
			return nil, err
		}
		switch statusPaye {
		case "Interdit", "Blacklist", "PEP", "Surveillés":
			meta, err := re.store.GetAlertMetaByType(ctx, statusPaye)
			var regleID *string
			regle, _ := re.store.GetRegleByType(ctx, statusPaye)
			if regle != nil {
				regleID = &regle.ID
			}
			if err == nil && meta != nil {
				var scoreRisque float64 = 0
				switch statusPaye {
				case "Interdit", "Blacklist":
					scoreRisque = 1.0
				case "PEP", "Surveillés":
					scoreRisque = 0.7
				}
				alerte := &models.Alerte{
					TransactionID:     tx.ID,
					Code:              meta.Code,
					Type:              meta.Type,
					Statut:            meta.Statut,
					Message:           meta.Message,
					NiveauCriticite:   meta.NiveauCriticite,
					ScoreRisque:       scoreRisque,
					DateGeneration:    tx.DateOperation,
					Action:            actionFromCriticite(meta.NiveauCriticite),
					RegleDeclencheeID: regleID,
					NumeroCompte:      tx.NumeroComptePaye,
				}
				alertes = append(alertes, alerte)
			}
			if statusPaye == "Interdit" || statusPaye == "Blacklist" {
				if meta == nil {
					return alertes, fmt.Errorf("%w: meta-donnée d'alerte introuvable pour %s", ErrTransactionRefused, statusPaye)
				}
				return alertes, fmt.Errorf("%w: %s", ErrTransactionRefused, meta.Message)
			}
		}
	}
	return alertes, nil
}
