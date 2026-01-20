package store

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"anomaly-detector/internal/models"
)

// ===================== ALERTES =====================

// GetAlertes récupère toutes les alertes du système.
func (s *Store) GetAlertes(ctx context.Context) ([]models.Alerte, error) {
	var alertes []models.Alerte
	if err := s.gormDB.WithContext(ctx).Order("date_generation DESC").Find(&alertes).Error; err != nil {
		return nil, err
	}
	return alertes, nil
}

// GetAlertCountsByMonthAndType retourne le nombre d'alertes par type pour les derniers 12 mois.
// Retourne une map: type -> [{year: YYYY, month: M, count: N}, ...]
func (s *Store) GetAlertCountsByMonthAndType(ctx context.Context, months int) (map[string][]map[string]int64, error) {
	// On utilise une requête SQL brute pour grouper par type, année et mois
	type Row struct {
		Type  string
		Year  int
		Month int
		Count int64
	}
	rows := []Row{}
	// Calcul de la date de début
	since := time.Now().AddDate(0, -months, 0)

	// Utiliser GORM avec Raw pour PostgreSQL EXTRACT qui n'a pas d'équivalent direct GORM
	if err := s.gormDB.WithContext(ctx).
		Table("alertes").
		Select("type, EXTRACT(YEAR FROM date_generation)::int AS year, EXTRACT(MONTH FROM date_generation)::int AS month, COUNT(*) as count").
		Where("date_generation >= ?", since).
		Group("type, EXTRACT(YEAR FROM date_generation), EXTRACT(MONTH FROM date_generation)").
		Order("type, EXTRACT(YEAR FROM date_generation), EXTRACT(MONTH FROM date_generation)").
		Scan(&rows).Error; err != nil {
		return nil, err
	}
	out := make(map[string][]map[string]int64)
	for _, r := range rows {
		key := r.Type
		out[key] = append(out[key], map[string]int64{"year": int64(r.Year), "month": int64(r.Month), "count": r.Count})
	}
	return out, nil
}

// CreateAlerte insère une nouvelle alerte dans la base de données.
func (s *Store) CreateAlerte(ctx context.Context, alerte *models.Alerte, userID string) (*models.Alerte, error) {
	// Si l'alerte est créée avec un statut "traité", définir automatiquement la date de traitement
	if strings.Contains(strings.ToLower(alerte.Statut), "trait") && alerte.DateTraitement == nil {
		now := time.Now()
		alerte.DateTraitement = &now
	}

	if err := s.gormDB.WithContext(ctx).Create(alerte).Error; err != nil {
		return nil, err
	}

	// Log synchrone de création d'alerte
	details := fmt.Sprintf("Nouvelle alerte créée - Type: %s, Action: %s, Code: %s",
		alerte.Type, alerte.Action, alerte.Code)

	actionDescription := fmt.Sprintf("Génération alerte '%s'", alerte.Type)

	// Déterminer le résultat selon l'action de l'alerte
	resultat := map[string]string{"bloquer": "bloqué", "surveiller": "surveillance"}[strings.ToLower(alerte.Action)]
	if resultat == "" {
		resultat = "surveillance"
	}

	if err := s.LogAction(ctx, "MDA", actionDescription, "Alertes", alerte.ID, details, resultat); err != nil {
		log.Printf("Erreur lors de l'enregistrement du log d'audit: %v", err)
	}

	return alerte, nil
}

// FixAlertesTaitesesSansDateTraitement corrige les alertes traitées qui n'ont pas de date_traitement
func (s *Store) FixAlertesTaitesesSansDateTraitement(ctx context.Context) error {
	// Trouver toutes les alertes avec statut "traité" mais sans date_traitement
	var alertes []models.Alerte
	err := s.gormDB.WithContext(ctx).
		Where("statut ILIKE ?", "%trait%").
		Where("date_traitement IS NULL").
		Find(&alertes).Error

	if err != nil {
		return err
	}

	log.Printf("Correction de %d alertes traitées sans date_traitement", len(alertes))

	// Pour chaque alerte, définir date_traitement = date_generation
	for _, alerte := range alertes {
		err = s.gormDB.WithContext(ctx).
			Model(&models.Alerte{}).
			Where("id = ?", alerte.ID).
			Update("date_traitement", alerte.DateGeneration).Error

		if err != nil {
			log.Printf("Erreur lors de la correction de l'alerte %s: %v", alerte.ID, err)
			return err
		}
	}

	log.Printf("Correction terminée pour %d alertes", len(alertes))
	return nil
}

// UpdateAlerteStatus met à jour le statut d'une alerte et lui assigne un analyste.
func (s *Store) UpdateAlerteStatus(ctx context.Context, alerteID string, analysteID string, newStatus string) (*models.Alerte, error) {
	var alerte models.Alerte

	// Récupérer l'alerte actuelle pour le log
	if err := s.gormDB.WithContext(ctx).First(&alerte, "id = ?", alerteID).Error; err != nil {
		return nil, err
	}

	oldStatus := alerte.Statut

	// Préparer les mises à jour
	updates := map[string]interface{}{
		"statut":              newStatus,
		"analyste_assigne_id": analysteID,
	}

	// Si le statut indique que l'alerte est traitée, ajouter la date de traitement
	if strings.Contains(strings.ToLower(newStatus), "trait") {
		updates["date_traitement"] = time.Now()
	}

	// Met à jour le statut, l'analyste et potentiellement la date de traitement
	if err := s.gormDB.WithContext(ctx).Model(&models.Alerte{}).Where("id = ?", alerteID).Updates(updates).Error; err != nil {
		return nil, err
	}

	// Recharge l'alerte mise à jour
	if err := s.gormDB.WithContext(ctx).First(&alerte, "id = ?", alerteID).Error; err != nil {
		return nil, err
	}

	// Log synchrone de mise à jour de statut
	details := fmt.Sprintf("Statut alerte modifié - De: %s vers: %s, Analyste: %s",
		oldStatus, newStatus, analysteID)

	actionDescription := fmt.Sprintf("Mise à jour alerte '%s' vers '%s'", alerte.Type, newStatus)

	if err := s.LogAction(ctx, analysteID, actionDescription, "Alertes", alerteID, details, "succès"); err != nil {
		log.Printf("Erreur lors de l'enregistrement du log d'audit: %v", err)
	}

	return &alerte, nil
}

// GetAlertMetaByType récupère la meta-donnée d'alerte pour un type donné (ex: Interdits, Blacklist, PEP, Surveillés)
func (s *Store) GetAlertMetaByType(ctx context.Context, alertType string) (*models.AlertMeta, error) {
	normalizedType := strings.TrimSpace(alertType)
	var meta models.AlertMeta
	result := s.gormDB.WithContext(ctx).Table("alertes_meta").Where("type ILIKE ?", normalizedType).Limit(1).First(&meta)
	if result.Error != nil {
		return nil, result.Error
	}
	return &meta, nil
}

// GetAlertesBloquerActivesByCompte retourne les alertes actives (statut 'En cours') de type 'bloquer' pour un compte donné, avec filtrage optionnel par code.
func (s *Store) GetAlertesBloquerActivesByCompte(ctx context.Context, numeroCompte string, code ...string) ([]models.Alerte, error) {
	var alertes []models.Alerte
	// Récupérer les IDs des transactions liées à ce compte
	var transactionIDs []string
	var transactions []models.Transaction
	if err := s.gormDB.WithContext(ctx).
		Where("numero_compte_payeur = ? OR numero_compte_paye = ?", numeroCompte, numeroCompte).
		Find(&transactions).Error; err != nil {
		return nil, err
	}
	transactionIDs = make([]string, 0, len(transactions))
	for _, tx := range transactions {
		transactionIDs = append(transactionIDs, tx.ID)
	}
	if len(transactionIDs) == 0 {
		return alertes, nil
	}
	query := s.gormDB.WithContext(ctx).
		Where("action = ? AND statut = ? AND transaction_id IN ?", "bloquer", "En cours", transactionIDs)
	if len(code) > 0 && code[0] != "" {
		query = query.Where("code = ?", code[0])
	}
	if err := query.Order("date_generation DESC").Find(&alertes).Error; err != nil {
		return nil, err
	}
	return alertes, nil
}
