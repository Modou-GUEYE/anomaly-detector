package core

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	"anomaly-detector/internal/models"
	"anomaly-detector/internal/store"
)

// ===================== CONSTANTES & TYPES =====================

var ErrTransactionRefused = errors.New("transaction refusée")

// ErrMissingPartnerBankCode is returned when the required environment variable is not set.
var ErrMissingPartnerBankCode = errors.New("PARTNER_BANK_CODE environment variable not set")

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

	// Code banque du partenaire principal configurable via variable d'environnement PARTNER_BANK_CODE
	partnerBankCode := os.Getenv("PARTNER_BANK_CODE")
	if partnerBankCode == "" {
		// Exiger la variable d'environnement : si absente, on retourne une erreur explicite
		return nil, fmt.Errorf("%w: PARTNER_BANK_CODE must be set", ErrMissingPartnerBankCode)
	}
	// Utiliser HasPrefix pour éviter les slices hors limites et supporter des longueurs variables
	isPayerPartner := strings.HasPrefix(tx.NumeroComptePayeur, partnerBankCode)
	isPayeePartner := strings.HasPrefix(tx.NumeroComptePaye, partnerBankCode)

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
						alerte := makeAlerte(meta, regleID, tx, tx.NumeroComptePayeur, action, calculerScoreRisque(reglePaysRisque, tx), &paysRisque)
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
						alerte := makeAlerte(meta, regleID, tx, tx.NumeroComptePaye, action, calculerScoreRisque(reglePaysRisque, tx), &paysRisque)
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

// makeAlerte construit une alerte avec des valeurs communes, en évitant les chaînes vides pour les UUID optionnels.
func makeAlerte(meta *models.AlertMeta, regleID string, tx *models.Transaction, numeroCompte string, action string, score float64, paysRisque *string) *models.Alerte {
	var reglePtr *string
	if regleID != "" {
		reglePtr = &regleID
	}

	return &models.Alerte{
		TransactionID:   nil,
		Code:            meta.Code,
		Type:            meta.Type,
		Statut:          meta.Statut,
		Message:         meta.Message,
		NiveauCriticite: meta.NiveauCriticite,
		ScoreRisque:     score,
		DateGeneration:  tx.DateOperation,
		Action:          action,
		RegleDeclencheeID: func() *string {
			if reglePtr != nil && *reglePtr != "" {
				return reglePtr
			}
			return nil
		}(),
		NumeroCompte: numeroCompte,
		PaysRisque:   paysRisque,
	}
}

// makeAlerteWithPeriode crée une alerte en adaptant le message selon la période
func makeAlerteWithPeriode(meta *models.AlertMeta, regleID string, tx *models.Transaction, numeroCompte string, action string, score float64, paysRisque *string, periode string) *models.Alerte {
	var reglePtr *string
	if regleID != "" {
		reglePtr = &regleID
	}

	// Adapter le message selon la période pour les règles de montant et fréquence
	message := meta.Message
	if (meta.Type == "Montant" || meta.Type == "montant" || meta.Type == "frequence" || meta.Type == "Frequence") && periode != "" {
		// Remplacer la période dans le message existant
		if meta.Type == "Montant" || meta.Type == "montant" {
			switch periode {
			case "24h":
				message = "Cumul sur 24h dépassant le seuil autorisé."
			case "7d":
				message = "Cumul sur 7 jours dépassant le seuil autorisé."
			case "30d":
				message = "Cumul sur 30 jours dépassant le seuil autorisé."
			}
		} else if meta.Type == "frequence" || meta.Type == "Frequence" {
			switch periode {
			case "24h":
				message = "Nombre de transactions sur 24h dépassant le seuil autorisé."
			case "7d":
				message = "Nombre de transactions sur 7 jours dépassant le seuil autorisé."
			case "30d":
				message = "Nombre de transactions sur 30 jours dépassant le seuil autorisé."
			}
		}
	}

	return &models.Alerte{
		TransactionID:   nil,
		Code:            meta.Code,
		Type:            meta.Type,
		Statut:          meta.Statut,
		Message:         message,
		NiveauCriticite: meta.NiveauCriticite,
		ScoreRisque:     score,
		DateGeneration:  tx.DateOperation,
		Action:          action,
		RegleDeclencheeID: func() *string {
			if reglePtr != nil && *reglePtr != "" {
				return reglePtr
			}
			return nil
		}(),
		NumeroCompte: numeroCompte,
		PaysRisque:   paysRisque,
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
				days = 7 // Par défaut: semaine
			}

			total, _ := re.store.GetTransactionSumForPeriod(ctx, numeroCompte, days)
			// Inclure le montant de la transaction courante dans le calcul du cumul
			total += tx.Montant
			if total > regle.Seuil {
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
				alerte := makeAlerteWithPeriode(meta, regleID, tx, numeroCompte, action, calculerScoreRisque(&regle, tx), nil, regle.PeriodeReference)
				alertes = append(alertes, alerte)
				// Si l'action est "bloquer", refuser la transaction
				if action == "bloquer" {
					return alertes, fmt.Errorf("%w: transaction bloquée - règle de montant dépassée", ErrTransactionRefused)
				}
				continue
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
			// Inclure la transaction courante dans le comptage
			count += 1
			if count > int64(regle.Seuil) {
				violation = true
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
			var alerte *models.Alerte
			// Pour les types "frequence", adapter le message selon la période
			if regle.TypeRegle == "frequence" {
				alerte = makeAlerteWithPeriode(meta, regleID, tx, numeroCompte, action, calculerScoreRisque(&regle, tx), nil, regle.PeriodeReference)
			} else {
				alerte = makeAlerte(meta, regleID, tx, numeroCompte, action, calculerScoreRisque(&regle, tx), nil)
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
				var action string
				switch statusPayeur {
				case "Interdit":
					scoreRisque = 1.0
					action = "bloquer" // Obligation légale OFAC/SWIFT
				case "Blacklist":
					scoreRisque = 1.0
					action = "surveiller" // Liste interne → Valider + Suivi
				case "PEP", "Surveillés":
					scoreRisque = 0.7
					action = "surveiller" // Valider + Suivi
				}
				regleVal := ""
				if regleID != nil {
					regleVal = *regleID
				}
				alerte := makeAlerte(meta, regleVal, tx, tx.NumeroComptePayeur, action, scoreRisque, nil)
				alertes = append(alertes, alerte)
			}
			// Seul "Interdit" bloque la transaction (obligation légale OFAC/SWIFT)
			if statusPayeur == "Interdit" {
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
				var action string
				switch statusPaye {
				case "Interdit":
					scoreRisque = 1.0
					action = "bloquer" // Obligation légale OFAC/SWIFT
				case "Blacklist":
					scoreRisque = 1.0
					action = "surveiller" // Liste interne → Valider + Suivi
				case "PEP", "Surveillés":
					scoreRisque = 0.7
					action = "surveiller" // Valider + Suivi
				}
				regleVal := ""
				if regleID != nil {
					regleVal = *regleID
				}
				alerte := makeAlerte(meta, regleVal, tx, tx.NumeroComptePaye, action, scoreRisque, nil)
				alertes = append(alertes, alerte)
			}
			// Seul "Interdit" bloque la transaction (obligation légale OFAC/SWIFT)
			if statusPaye == "Interdit" {
				if meta == nil {
					return alertes, fmt.Errorf("%w: meta-donnée d'alerte introuvable pour %s", ErrTransactionRefused, statusPaye)
				}
				return alertes, fmt.Errorf("%w: %s", ErrTransactionRefused, meta.Message)
			}
		}
	}
	return alertes, nil
}
