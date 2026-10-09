package store

import (
	"context"
	"errors"
	"fmt"
	"time"

	"gorm.io/gorm"

	"anomaly-detector/internal/models"
)

// ===================== COMPTES =====================

// CreateCompte insère un nouveau compte dans la base de données.
func (s *Store) CreateCompte(ctx context.Context, compte *models.Compte, userID string) (*models.Compte, error) {
	// Remplissage automatique du profil si vide
	if (compte.ProfilClientID == nil || *compte.ProfilClientID == "") && compte.TypeClient != "" {
		if profilID, err := s.GetProfilClientIDByCode(ctx, compte.TypeClient); err == nil && profilID != nil {
			compte.ProfilClientID = profilID
		}
	}
	// Initialisation automatique de la date de création si vide
	if compte.DateCreation.IsZero() {
		compte.DateCreation = time.Now()
	}

	if err := s.gormDB.WithContext(ctx).Create(compte).Error; err != nil {
		return nil, err
	}

	// Stocker immédiatement dans le cache mémoire
	s.cacheAccount(compte)

	// Logging d'audit asynchrone pour ne pas bloquer le chemin critique de traitement
	go func(c models.Compte) {
		actionDesc := fmt.Sprintf("Création du compte '%s'", c.NumeroCompte)
		details := fmt.Sprintf("Création du compte: %s (%s)", c.NumeroCompte, c.NomClient)
		_ = s.LogAction(context.Background(), "MDA", actionDesc, "Comptes", c.ID, details, "succès")
	}(*compte)

	return compte, nil
}

// GetCompteByCompte recherche un client par son numéro de compte unique.
func (s *Store) GetCompteByCompte(ctx context.Context, numeroCompte string) (*models.Compte, error) {
	// 1. Recherche dans le cache mémoire (0ms)
	if val, ok := s.accountCache.Load(numeroCompte); ok {
		if c, ok := val.(*models.Compte); ok && c != nil {
			return c, nil
		}
	}

	// 2. Recherche en base de données
	var compte models.Compte
	err := s.gormDB.WithContext(ctx).Where("numero_compte = ?", numeroCompte).First(&compte).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return &models.Compte{}, nil // Aucun compte trouvé, retourne un compte vide
		}
		return nil, err
	}

	// Mettre en cache pour les transactions suivantes
	s.cacheAccount(&compte)
	return &compte, nil
}

// cacheAccount ajoute un compte au cache mémoire. Le cache est borné (ACCOUNT_CACHE_MAX) :
// au-delà, il est vidé pour éviter une croissance mémoire illimitée sur la durée.
func (s *Store) cacheAccount(c *models.Compte) {
	if _, loaded := s.accountCache.LoadOrStore(c.NumeroCompte, c); loaded {
		return
	}
	if s.accountCount.Add(1) > s.accountMax {
		s.accountCache.Clear()
		s.accountCount.Store(0)
	}
}

// GetReglesForCompte récupère les règles actives associées au profil client d'un compte donné.
func (s *Store) GetReglesForCompte(ctx context.Context, numeroCompte string) ([]models.RegleControle, error) {
	// 1. Filtrage ultra-rapide en mémoire si le cache est actif (0ms, aucun JOIN SQL)
	if s.refCache != nil {
		s.refCache.mu.RLock()
		if s.refCache.initialized {
			activeRules := s.refCache.activeRules
			s.refCache.mu.RUnlock()

			compte, err := s.GetCompteByCompte(ctx, numeroCompte)
			if err == nil && compte != nil && compte.ID != "" {
				matchedRules := make([]models.RegleControle, 0, len(activeRules))
				for _, r := range activeRules {
					// Règle universelle (sans profil spécifique) ou règle liée au profil du compte
					if r.ProfilClientID == nil || *r.ProfilClientID == "" ||
						(compte.ProfilClientID != nil && *r.ProfilClientID == *compte.ProfilClientID) {
						matchedRules = append(matchedRules, r)
					}
				}
				return matchedRules, nil
			}
		} else {
			s.refCache.mu.RUnlock()
		}
	}

	// 2. Repli SQL classique si le cache n'est pas prêt
	var regles []models.RegleControle
	err := s.gormDB.WithContext(ctx).
		Table("regles_controle AS r").
		Select("r.*").
		Joins("JOIN comptes AS c ON c.profil_client_id::text = r.profil_client_id OR r.profil_client_id IS NULL OR r.profil_client_id = ''").
		Where("c.numero_compte = ? AND r.active = ?", numeroCompte, true).
		Find(&regles).Error
	return regles, err
}

// GetAllComptes retourne la liste de tous les comptes clients enregistrés.
func (s *Store) GetAllComptes(ctx context.Context) ([]*models.Compte, error) {
	var comptes []*models.Compte
	err := s.gormDB.WithContext(ctx).
		Order("date_creation DESC").
		Find(&comptes).Error
	return comptes, err
}

// GetComptesSurveillesCount retourne le nombre de comptes uniques dans la table suivi_blacklist.
func (s *Store) GetComptesSurveillesCount(ctx context.Context) (int64, error) {
	var count int64
	err := s.gormDB.WithContext(ctx).
		Table("suivi_blacklist").
		Select("COUNT(DISTINCT numero_compte)").
		Scan(&count).Error
	return count, err
}

// GetComptesSurveilles retourne la liste des comptes uniques dans suivi_blacklist.
func (s *Store) GetComptesSurveilles(ctx context.Context) ([]string, error) {
	var comptes []string
	err := s.gormDB.WithContext(ctx).
		Table("suivi_blacklist").
		Distinct("numero_compte").
		Pluck("numero_compte", &comptes).Error
	return comptes, err
}
