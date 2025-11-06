package store

import (
	"context"
	"time"

	"anomaly-detector/internal/models"
)

// ===================== AUDIT ET CONFORMITÉ =====================

// CreateLogAudit enregistre une action d'audit dans le système
func (s *Store) CreateLogAudit(ctx context.Context, log *models.LogAudit) error {
	if log.DateHeure.IsZero() {
		log.DateHeure = time.Now()
	}

	result := s.gormDB.WithContext(ctx).Create(log)
	return result.Error
}

// LogAction enregistre automatiquement une action avec les détails minimaux
func (s *Store) LogAction(ctx context.Context, utilisateurID, action, typeAction, entityID, details, resultat string) error {
	// Récupérer le rôle de l'utilisateur selon les 4 types d'intervenants
	role := "unknown"
	switch utilisateurID {
	case "MDA":
		role = "system" // Moteur de Détection d'Anomalie
	case "app", "app_bis_01": // Applications externes
		role = "application"
	default:
		// Récupérer le rôle depuis la base de données (admin/analyst/etc.)
		// Tous les vrais utilisateurs (admin et analyst) passent ici
		user, err := s.GetUserByID(ctx, utilisateurID)
		if err == nil && user != nil {
			role = user.Role // "admin" ou "analyst" depuis la table utilisateurs
		}
	}

	log := &models.LogAudit{
		UtilisateurID:   utilisateurID,
		ActionEffectuee: action,
		TypeAction:      typeAction,
		EntityID:        entityID,
		Details:         details,
		Resultat:        resultat,
		Role:            role,
		DateHeure:       time.Now(),
	}

	return s.CreateLogAudit(ctx, log)
}

// LogActionWithRole enregistre une action avec un rôle spécifique
func (s *Store) LogActionWithRole(ctx context.Context, utilisateurID, action, typeAction, entityID, details, resultat, role string) error {
	log := &models.LogAudit{
		UtilisateurID:   utilisateurID,
		ActionEffectuee: action,
		TypeAction:      typeAction,
		EntityID:        entityID,
		Details:         details,
		Resultat:        resultat,
		Role:            role,
		DateHeure:       time.Now(),
	}

	return s.CreateLogAudit(ctx, log)
}

// GetLogsAudit récupère les logs d'audit avec filtres
func (s *Store) GetLogsAudit(ctx context.Context, filters map[string]interface{}) ([]models.LogAudit, error) {
	var logs []models.LogAudit
	query := s.gormDB.WithContext(ctx).Model(&models.LogAudit{})

	// Appliquer les filtres
	if typeAction, ok := filters["type_action"]; ok && typeAction != "" {
		query = query.Where("type_action = ?", typeAction)
	}

	if utilisateurID, ok := filters["utilisateur_id"]; ok && utilisateurID != "" {
		query = query.Where("utilisateur_id = ?", utilisateurID)
	}

	if action, ok := filters["action_effectuee"]; ok && action != "" {
		query = query.Where("action_effectuee ILIKE ?", "%"+action.(string)+"%")
	}

	if resultat, ok := filters["resultat"]; ok && resultat != "" {
		query = query.Where("resultat = ?", resultat)
	}

	// Filtres de date
	if dateDebut, ok := filters["date_debut"]; ok && dateDebut != "" {
		if date, err := time.Parse("2006-01-02", dateDebut.(string)); err == nil {
			query = query.Where("date_heure >= ?", date)
		}
	}

	if dateFin, ok := filters["date_fin"]; ok && dateFin != "" {
		if date, err := time.Parse("2006-01-02", dateFin.(string)); err == nil {
			// Ajouter 24h pour inclure toute la journée
			dateFin := date.Add(24 * time.Hour)
			query = query.Where("date_heure < ?", dateFin)
		}
	}

	// Limitation et tri
	limit := 1000 // Limite par défaut
	if limitVal, ok := filters["limit"]; ok {
		if l, ok := limitVal.(int); ok && l > 0 && l < 5000 {
			limit = l
		}
	}

	result := query.Order("date_heure DESC").Limit(limit).Find(&logs)
	return logs, result.Error
}

// GetAuditStatistics retourne des statistiques d'audit pour la conformité
func (s *Store) GetAuditStatistics(ctx context.Context, periode int) (map[string]interface{}, error) {
	stats := make(map[string]interface{})
	since := time.Now().AddDate(0, 0, -periode)

	// Nombre total d'actions auditées
	var totalActions int64
	err := s.gormDB.WithContext(ctx).
		Model(&models.LogAudit{}).
		Where("date_heure >= ?", since).
		Count(&totalActions).Error
	if err != nil {
		return nil, err
	}
	stats["total_actions"] = totalActions

	// Actions par type
	var actionsByType []struct {
		TypeAction string `json:"type_action"`
		Count      int64  `json:"count"`
	}
	err = s.gormDB.WithContext(ctx).
		Model(&models.LogAudit{}).
		Select("type_action, COUNT(*) as count").
		Where("date_heure >= ?", since).
		Group("type_action").
		Scan(&actionsByType).Error
	if err != nil {
		return nil, err
	}
	stats["actions_by_type"] = actionsByType

	// Actions échouées
	var actionsFailed int64
	err = s.gormDB.WithContext(ctx).
		Model(&models.LogAudit{}).
		Where("date_heure >= ? AND resultat IN (?)", since, []string{"echec", "erreur", "rejeté"}).
		Count(&actionsFailed).Error
	if err != nil {
		return nil, err
	}
	stats["actions_failed"] = actionsFailed

	// Utilisateurs actifs
	var utilisateursActifs int64
	err = s.gormDB.WithContext(ctx).
		Model(&models.LogAudit{}).
		Distinct("utilisateur_id").
		Where("date_heure >= ?", since).
		Count(&utilisateursActifs).Error
	if err != nil {
		return nil, err
	}
	stats["utilisateurs_actifs"] = utilisateursActifs

	return stats, nil
}

// GetComplianceMetrics retourne les métriques de conformité LBC/FT
func (s *Store) GetComplianceMetrics(ctx context.Context) (map[string]interface{}, error) {
	metrics := make(map[string]interface{})

	// 1. Déclarations de soupçon (alertes avec action "declarer")
	var declarationsSoupcon int64
	err := s.gormDB.WithContext(ctx).
		Model(&models.Alerte{}).
		Where("action = ? OR statut ILIKE ?", "declarer", "%declaration%").
		Count(&declarationsSoupcon).Error
	if err != nil {
		return nil, err
	}
	metrics["declarations_soupcon"] = declarationsSoupcon

	// 2. Entités bloquées (alertes avec action "bloquer")
	var entitesBloquees int64
	err = s.gormDB.WithContext(ctx).
		Model(&models.Alerte{}).
		Where("action = ?", "bloquer").
		Count(&entitesBloquees).Error
	if err != nil {
		return nil, err
	}
	metrics["entites_bloquees"] = entitesBloquees

	// 3. Comptes surveillés
	surveilleesCount, err := s.GetComptesSurveillesCount(ctx)
	if err != nil {
		return nil, err
	}
	metrics["entites_surveillees"] = surveilleesCount

	// 4. Transactions avec pays à risque (dernier mois) - utiliser les données raw_data
	var transactionsPaysRisque int64 = 0
	// Compter seulement les transactions existantes pour l'instant
	err = s.gormDB.WithContext(ctx).
		Model(&models.Transaction{}).
		Where("date_operation >= ?", time.Now().AddDate(0, -1, 0)).
		Count(&transactionsPaysRisque).Error
	if err != nil {
		return nil, err
	}
	metrics["transactions_pays_risque"] = transactionsPaysRisque

	// 5. Montant total des transactions à risque (dernier mois)
	var montantPaysRisque float64 = 0
	err = s.gormDB.WithContext(ctx).
		Model(&models.Transaction{}).
		Where("date_operation >= ?", time.Now().AddDate(0, -1, 0)).
		Select("COALESCE(SUM(montant), 0)").
		Row().Scan(&montantPaysRisque)
	if err != nil {
		return nil, err
	}
	metrics["montant_pays_risque"] = montantPaysRisque

	// 6. Taux de déclaration (pourcentage d'alertes déclarées)
	var totalAlertes int64
	err = s.gormDB.WithContext(ctx).
		Model(&models.Alerte{}).
		Count(&totalAlertes).Error
	if err != nil {
		return nil, err
	}

	tauxDeclaration := 0.0
	if totalAlertes > 0 {
		tauxDeclaration = (float64(declarationsSoupcon) / float64(totalAlertes)) * 100
		tauxDeclaration = float64(int(tauxDeclaration*10+0.5)) / 10 // Arrondir à 1 décimale
	}
	metrics["taux_declaration"] = tauxDeclaration
	metrics["total_alertes"] = totalAlertes

	return metrics, nil
}
