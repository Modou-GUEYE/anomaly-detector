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

const (
	statusInterdit   = "Interdit"
	statusBlacklist  = "Blacklist"
	statusPEP        = "PEP"
	statusSurveilles = "Surveillés"
	actionBloquer    = "bloquer"
)

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
	payeeAccountPresent := tx.NumeroComptePaye != ""

	// --- 1. CONTRÔLE BLACKLIST  ---
	if err := re.appendBlacklistAlerts(ctx, tx, isPayerPartner, isPayeePartner && payeeAccountPresent, &alertesGenerees); err != nil {
		return alertesGenerees, err
	}

	// --- 2. APPLICATION DES RÈGLES MÉTIER ---
	if err := re.appendRulesAlerts(ctx, tx, isPayerPartner, isPayeePartner && payeeAccountPresent, &alertesGenerees); err != nil {
		return alertesGenerees, err
	}

	// --- 3. REGLE PAYS_A_RISQUE ---
	if err := re.appendPaysRisqueAlerts(ctx, tx, isPayerPartner, isPayeePartner && payeeAccountPresent, &alertesGenerees); err != nil {
		return alertesGenerees, err
	}

	return alertesGenerees, nil
}

func (re *RulesEngine) appendBlacklistAlerts(ctx context.Context, tx *models.Transaction, checkPayeur bool, checkPaye bool, alertesGenerees *[]*models.Alerte) error {
	if checkPayeur {
		alertesBlacklist, err := re.checkBlacklist(ctx, tx, true, false)
		*alertesGenerees = append(*alertesGenerees, alertesBlacklist...)
		if err != nil {
			return err
		}
	}
	if checkPaye {
		alertesBlacklist, err := re.checkBlacklist(ctx, tx, false, true)
		*alertesGenerees = append(*alertesGenerees, alertesBlacklist...)
		if err != nil {
			return err
		}
	}
	return nil
}

func (re *RulesEngine) appendRulesAlerts(ctx context.Context, tx *models.Transaction, isPayerPartner bool, checkPayee bool, alertesGenerees *[]*models.Alerte) error {
	if isPayerPartner {
		alertesPayeur, err := re.applyRulesForAccount(ctx, tx, tx.NumeroComptePayeur)
		*alertesGenerees = append(*alertesGenerees, alertesPayeur...)
		if err != nil {
			return err
		}
	}
	if checkPayee {
		alertesPaye, err := re.applyRulesForAccount(ctx, tx, tx.NumeroComptePaye)
		*alertesGenerees = append(*alertesGenerees, alertesPaye...)
		if err != nil {
			return err
		}
	}
	return nil
}

func (re *RulesEngine) appendPaysRisqueAlerts(ctx context.Context, tx *models.Transaction, isPayerPartner bool, checkPayee bool, alertesGenerees *[]*models.Alerte) error {
	reglePaysRisque, err := re.store.GetReglePaysRisqueActive(ctx)
	if err != nil || reglePaysRisque == nil {
		return nil
	}

	meta, err := re.store.GetAlertMetaByType(ctx, "Pays à risque")
	if err != nil || meta == nil {
		return nil
	}

	paysARisque, err := re.store.GetAllPaysARisque(ctx)
	if err != nil {
		return nil
	}

	context := paysRisqueAlertContext{
		tx:              tx,
		meta:            meta,
		reglePaysRisque: reglePaysRisque,
		alertesGenerees: alertesGenerees,
	}

	for _, paysRecord := range paysARisque {
		if err := context.appendIfMatch(isPayerPartner, tx.PaysPayeur, tx.NumeroComptePayeur, paysRecord); err != nil {
			return err
		}
		if err := context.appendIfMatch(checkPayee, tx.PaysPaye, tx.NumeroComptePaye, paysRecord); err != nil {
			return err
		}
	}

	return nil
}

type paysRisqueAlertContext struct {
	tx              *models.Transaction
	meta            *models.AlertMeta
	reglePaysRisque *models.RegleControle
	alertesGenerees *[]*models.Alerte
}

func (c paysRisqueAlertContext) appendIfMatch(accountEligible bool, txPays string, numeroCompte string, paysRecord models.PaysRisque) error {
	if !accountEligible || !strings.EqualFold(txPays, paysRecord.Code) {
		return nil
	}

	paysRisque := paysRecord.NomPays
	action := c.reglePaysRisque.Action
	if action == "" {
		action = actionFromCriticite(c.meta.NiveauCriticite)
	}

	alerte := makeAlerte(c.meta, c.reglePaysRisque.ID, c.tx, numeroCompte, action, calculerScoreRisque(c.reglePaysRisque, c.tx), &paysRisque)
	*c.alertesGenerees = append(*c.alertesGenerees, alerte)

	if action == actionBloquer {
		return fmt.Errorf("%w: transaction bloquée - pays à risque détecté (%s)", ErrTransactionRefused, paysRisque)
	}
	return nil
}

// ===================== HELPERS INTERNES =====================

// calculerScoreRisque calcule le score de risque pour une règle et une transaction donnée.
func calculerScoreRisque(regle *models.RegleControle, _ *models.Transaction) float64 {

	switch regle.TypeRegle {
	case "montant", "pays_risque", statusInterdit, statusBlacklist:
		return 1.0
	case "frequence", statusPEP, statusSurveilles:
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
func makeAlerteWithPeriode(meta *models.AlertMeta, regleID string, tx *models.Transaction, numeroCompte string, action string, score float64, periode string) *models.Alerte {
	alerte := makeAlerte(meta, regleID, tx, numeroCompte, action, score, nil)
	alerte.Message = messageForPeriode(meta.Type, periode, meta.Message)
	return alerte
}

func messageForPeriode(alertType string, periode string, fallback string) string {
	if periode == "" {
		return fallback
	}

	normalizedType := strings.ToLower(alertType)
	switch normalizedType {
	case "montant":
		switch periode {
		case "24h":
			return "Cumul sur 24h dépassant le seuil autorisé."
		case "7d":
			return "Cumul sur 7 jours dépassant le seuil autorisé."
		case "30d":
			return "Cumul sur 30 jours dépassant le seuil autorisé."
		}
	case "frequence":
		switch periode {
		case "24h":
			return "Nombre de transactions sur 24h dépassant le seuil autorisé."
		case "7d":
			return "Nombre de transactions sur 7 jours dépassant le seuil autorisé."
		case "30d":
			return "Nombre de transactions sur 30 jours dépassant le seuil autorisé."
		}
	}

	return fallback
}

// applyRulesForAccount applique les règles métier pour un compte spécifique.
func (re *RulesEngine) applyRulesForAccount(ctx context.Context, tx *models.Transaction, numeroCompte string) ([]*models.Alerte, error) {
	var alertes []*models.Alerte
	regles, err := re.store.GetReglesForCompte(ctx, numeroCompte)
	if err != nil {
		return nil, err
	}

	for _, regle := range regles {
		violation, err := re.isRuleViolated(ctx, tx, numeroCompte, regle)
		if err != nil {
			return nil, err
		}
		if !violation {
			continue
		}

		alerte, action, err := re.buildRuleAlerte(ctx, tx, numeroCompte, regle)
		if err != nil {
			return nil, err
		}
		alertes = append(alertes, alerte)

		if action == actionBloquer {
			if regle.TypeRegle == "montant" {
				return alertes, fmt.Errorf("%w: transaction bloquée - règle de montant dépassée", ErrTransactionRefused)
			}
			return alertes, fmt.Errorf("%w: transaction bloquée - règle de %s dépassée", ErrTransactionRefused, regle.TypeRegle)
		}
	}
	return alertes, nil
}

func (re *RulesEngine) isRuleViolated(ctx context.Context, tx *models.Transaction, numeroCompte string, regle models.RegleControle) (bool, error) {
	switch regle.TypeRegle {
	case "montant":
		return re.isMontantRuleViolated(ctx, tx, numeroCompte, regle), nil
	case "frequence":
		return re.isFrequenceRuleViolated(ctx, numeroCompte, regle), nil
	default:
		return false, nil
	}
}

func (re *RulesEngine) isMontantRuleViolated(ctx context.Context, tx *models.Transaction, numeroCompte string, regle models.RegleControle) bool {
	days := periodDays(regle.PeriodeReference, 7)
	total, _ := re.store.GetTransactionSumForPeriod(ctx, numeroCompte, days)
	total += tx.Montant
	return total > regle.Seuil
}

func (re *RulesEngine) isFrequenceRuleViolated(ctx context.Context, numeroCompte string, regle models.RegleControle) bool {
	days := periodDays(regle.PeriodeReference, 1)
	count, _ := re.store.GetTransactionCountForPeriod(ctx, numeroCompte, days)
	count += 1
	return count > int64(regle.Seuil)
}

func periodDays(periodeReference string, defaultDays int) int {
	switch periodeReference {
	case "24h":
		return 1
	case "7d":
		return 7
	case "30d":
		return 30
	default:
		return defaultDays
	}
}

func (re *RulesEngine) buildRuleAlerte(ctx context.Context, tx *models.Transaction, numeroCompte string, regle models.RegleControle) (*models.Alerte, string, error) {
	meta, err := re.store.GetAlertMetaByType(ctx, regle.TypeRegle)
	if err != nil || meta == nil {
		return nil, "", fmt.Errorf("meta-donnée d'alerte introuvable pour le type '%s'", regle.TypeRegle)
	}

	action := regle.Action
	if action == "" {
		action = actionFromCriticite(meta.NiveauCriticite)
	}

	score := calculerScoreRisque(&regle, tx)
	if regle.TypeRegle == "frequence" || regle.TypeRegle == "montant" {
		return makeAlerteWithPeriode(meta, regle.ID, tx, numeroCompte, action, score, regle.PeriodeReference), action, nil
	}

	return makeAlerte(meta, regle.ID, tx, numeroCompte, action, score, nil), action, nil
}

// checkBlacklist centralise la logique de la blacklist conformément aux spécifications.
// Seul le compte BIS sélectionné (payeur ou payé) est vérifié selon les flags.
func (re *RulesEngine) checkBlacklist(ctx context.Context, tx *models.Transaction, checkPayeur bool, checkPaye bool) ([]*models.Alerte, error) {
	var alertes []*models.Alerte
	if checkPayeur {
		alertesPayeur, err := re.checkBlacklistForAccount(ctx, tx, tx.NumeroComptePayeur)
		alertes = append(alertes, alertesPayeur...)
		if err != nil {
			return alertes, err
		}
	}
	if checkPaye && tx.NumeroComptePaye != "" {
		alertesPaye, err := re.checkBlacklistForAccount(ctx, tx, tx.NumeroComptePaye)
		alertes = append(alertes, alertesPaye...)
		if err != nil {
			return alertes, err
		}
	}
	return alertes, nil
}

func (re *RulesEngine) checkBlacklistForAccount(ctx context.Context, tx *models.Transaction, account string) ([]*models.Alerte, error) {
	status, err := re.store.IsAccountOnBlacklist(ctx, account)
	if err != nil {
		return nil, err
	}
	if !isMonitoredBlacklistStatus(status) {
		return nil, nil
	}

	meta, alert, err := re.buildBlacklistAlert(ctx, tx, account, status)
	if err != nil {
		return nil, err
	}

	var alertes []*models.Alerte
	if alert != nil {
		alertes = append(alertes, alert)
	}

	if status != statusInterdit {
		return alertes, nil
	}
	if meta == nil {
		return alertes, fmt.Errorf("%w: meta-donnée d'alerte introuvable pour %s", ErrTransactionRefused, status)
	}
	return alertes, fmt.Errorf("%w: %s", ErrTransactionRefused, meta.Message)
}

func (re *RulesEngine) buildBlacklistAlert(ctx context.Context, tx *models.Transaction, account string, status string) (*models.AlertMeta, *models.Alerte, error) {
	meta, err := re.store.GetAlertMetaByType(ctx, status)
	regle, regleErr := re.store.GetRegleByType(ctx, status)
	if regleErr != nil {
		return nil, nil, regleErr
	}

	if err != nil || meta == nil {
		return meta, nil, nil
	}

	regleID := ""
	if regle != nil {
		regleID = regle.ID
	}

	scoreRisque, action := blacklistRiskAndAction(status)
	alerte := makeAlerte(meta, regleID, tx, account, action, scoreRisque, nil)
	return meta, alerte, nil
}

func isMonitoredBlacklistStatus(status string) bool {
	switch status {
	case statusInterdit, statusBlacklist, statusPEP, statusSurveilles:
		return true
	default:
		return false
	}
}

func blacklistRiskAndAction(status string) (float64, string) {
	switch status {
	case statusInterdit:
		return 1.0, actionBloquer
	case statusBlacklist:
		return 1.0, "surveiller"
	case statusPEP, statusSurveilles:
		return 0.7, "surveiller"
	default:
		return 0.5, "surveiller"
	}
}
