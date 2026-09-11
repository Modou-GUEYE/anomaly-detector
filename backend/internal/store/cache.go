package store

import (
	"context"
	"strings"
	"sync"

	"anomaly-detector/internal/models"
)

// RefCache maintient en mémoire les tables de référence et données quasi-statiques
// pour éliminer les requêtes SQL séquentielles dans le chemin critique des transactions.
type RefCache struct {
	mu sync.RWMutex

	// Méta-données d'alertes : clé = type normalisé (minuscules)
	alertMeta map[string]*models.AlertMeta

	// Pays à risque actifs
	paysRisqueList []models.PaysRisque
	paysRisqueMap  map[string]models.PaysRisque // clé = code pays ISO en majuscules

	// Règle pays_risque active
	reglePaysRisque *models.RegleControle

	// Profils clients : clé = code ou nom normalisé en minuscules -> UUID profil
	profilIDs map[string]string

	// Règles de contrôle actives
	activeRules []models.RegleControle
	rulesByType map[string]*models.RegleControle // clé = type_regle en minuscules

	// Entités de surveillance / Blacklist : clé = numéro de compte majuscule -> type_entite
	blacklist map[string]string

	initialized bool
}

func newRefCache() *RefCache {
	return &RefCache{
		alertMeta:      make(map[string]*models.AlertMeta),
		paysRisqueMap:  make(map[string]models.PaysRisque),
		profilIDs:      make(map[string]string),
		rulesByType:    make(map[string]*models.RegleControle),
		blacklist:      make(map[string]string),
		paysRisqueList: make([]models.PaysRisque, 0),
		activeRules:    make([]models.RegleControle, 0),
	}
}

// RefreshAll recharge l'ensemble des caches de référence depuis la base de données.
func (s *Store) RefreshAll(ctx context.Context) error {
	s.refCache.mu.Lock()
	defer s.refCache.mu.Unlock()

	// 1. Méta-données d'alertes
	var metas []models.AlertMeta
	if err := s.gormDB.WithContext(ctx).Table("alertes_meta").Find(&metas).Error; err == nil {
		s.refCache.alertMeta = make(map[string]*models.AlertMeta, len(metas))
		for i := range metas {
			m := metas[i]
			s.refCache.alertMeta[strings.ToLower(strings.TrimSpace(m.Type))] = &m
			s.refCache.alertMeta[strings.ToLower(strings.TrimSpace(m.Code))] = &m
		}
	}

	// 2. Pays à risque
	var pays []models.PaysRisque
	if err := s.gormDB.WithContext(ctx).Where("active = ?", true).Order("nom_pays ASC").Find(&pays).Error; err == nil {
		s.refCache.paysRisqueList = pays
		s.refCache.paysRisqueMap = make(map[string]models.PaysRisque, len(pays))
		for _, p := range pays {
			if p.Code != "" {
				s.refCache.paysRisqueMap[strings.ToUpper(strings.TrimSpace(p.Code))] = p
			}
		}
	}

	// 3. Règle pays_risque active
	var reglePays models.RegleControle
	if err := s.gormDB.WithContext(ctx).Where("type_regle = ? AND active = ?", "pays_risque", true).First(&reglePays).Error; err == nil {
		s.refCache.reglePaysRisque = &reglePays
	} else {
		s.refCache.reglePaysRisque = nil
	}

	// 4. Profils clients
	var profils []models.ProfilClient
	if err := s.gormDB.WithContext(ctx).Find(&profils).Error; err == nil {
		s.refCache.profilIDs = make(map[string]string, len(profils)*2)
		for _, p := range profils {
			normNom := strings.ToLower(strings.TrimSpace(p.NomProfil))
			s.refCache.profilIDs[normNom] = p.ID
			// Alias standard
			switch normNom {
			case "particulier":
				s.refCache.profilIDs["p"] = p.ID
			case "entreprise":
				s.refCache.profilIDs["m"] = p.ID
			case "commerçant", "commercant":
				s.refCache.profilIDs["c"] = p.ID
			case "business":
				s.refCache.profilIDs["b"] = p.ID
			}
		}
	}

	// 5. Règles de contrôle actives
	var rules []models.RegleControle
	if err := s.gormDB.WithContext(ctx).Where("active = ?", true).Find(&rules).Error; err == nil {
		s.refCache.activeRules = rules
		s.refCache.rulesByType = make(map[string]*models.RegleControle, len(rules))
		for i := range rules {
			r := rules[i]
			s.refCache.rulesByType[strings.ToLower(strings.TrimSpace(r.TypeRegle))] = &r
		}
	}

	// 6. Blacklist
	var blacklists []models.EntiteBlacklist
	if err := s.gormDB.WithContext(ctx).Find(&blacklists).Error; err == nil {
		s.refCache.blacklist = make(map[string]string, len(blacklists))
		for _, b := range blacklists {
			compte := strings.ToUpper(strings.TrimSpace(b.NumeroCompte))
			if compte != "" {
				s.refCache.blacklist[compte] = b.TypeEntite
			}
		}
	}

	s.refCache.initialized = true
	return nil
}

// InvalidateRules recharge le cache des règles
func (s *Store) InvalidateRules(ctx context.Context) {
	s.refCache.mu.Lock()
	defer s.refCache.mu.Unlock()

	var rules []models.RegleControle
	if err := s.gormDB.WithContext(ctx).Where("active = ?", true).Find(&rules).Error; err == nil {
		s.refCache.activeRules = rules
		s.refCache.rulesByType = make(map[string]*models.RegleControle, len(rules))
		for i := range rules {
			r := rules[i]
			s.refCache.rulesByType[strings.ToLower(strings.TrimSpace(r.TypeRegle))] = &r
		}
	}

	var reglePays models.RegleControle
	if err := s.gormDB.WithContext(ctx).Where("type_regle = ? AND active = ?", "pays_risque", true).First(&reglePays).Error; err == nil {
		s.refCache.reglePaysRisque = &reglePays
	} else {
		s.refCache.reglePaysRisque = nil
	}
}

// InvalidateCountries recharge le cache des pays à risque
func (s *Store) InvalidateCountries(ctx context.Context) {
	s.refCache.mu.Lock()
	defer s.refCache.mu.Unlock()

	var pays []models.PaysRisque
	if err := s.gormDB.WithContext(ctx).Where("active = ?", true).Order("nom_pays ASC").Find(&pays).Error; err == nil {
		s.refCache.paysRisqueList = pays
		s.refCache.paysRisqueMap = make(map[string]models.PaysRisque, len(pays))
		for _, p := range pays {
			if p.Code != "" {
				s.refCache.paysRisqueMap[strings.ToUpper(strings.TrimSpace(p.Code))] = p
			}
		}
	}
}

// InvalidateBlacklist recharge le cache de la blacklist
func (s *Store) InvalidateBlacklist(ctx context.Context) {
	s.refCache.mu.Lock()
	defer s.refCache.mu.Unlock()

	var blacklists []models.EntiteBlacklist
	if err := s.gormDB.WithContext(ctx).Find(&blacklists).Error; err == nil {
		s.refCache.blacklist = make(map[string]string, len(blacklists))
		for _, b := range blacklists {
			compte := strings.ToUpper(strings.TrimSpace(b.NumeroCompte))
			if compte != "" {
				s.refCache.blacklist[compte] = b.TypeEntite
			}
		}
	}
}

// InvalidateProfiles recharge le cache des profils
func (s *Store) InvalidateProfiles(ctx context.Context) {
	s.refCache.mu.Lock()
	defer s.refCache.mu.Unlock()

	var profils []models.ProfilClient
	if err := s.gormDB.WithContext(ctx).Find(&profils).Error; err == nil {
		s.refCache.profilIDs = make(map[string]string, len(profils)*2)
		for _, p := range profils {
			normNom := strings.ToLower(strings.TrimSpace(p.NomProfil))
			s.refCache.profilIDs[normNom] = p.ID
			switch normNom {
			case "particulier":
				s.refCache.profilIDs["p"] = p.ID
			case "entreprise":
				s.refCache.profilIDs["m"] = p.ID
			case "commerçant", "commercant":
				s.refCache.profilIDs["c"] = p.ID
			case "business":
				s.refCache.profilIDs["b"] = p.ID
			}
		}
	}
}
