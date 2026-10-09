package store

import (
	"anomaly-detector/internal/models"
	"context"
	"errors"
	"fmt"
	"time"

	"gorm.io/gorm"
)

const whereUserIDEquals = "id = ?"

// userCacheEntry conserve un utilisateur lu en base pour une durée limitée.
type userCacheEntry struct {
	user    *models.Utilisateur
	expires time.Time
}

// GetUserByIDCached retourne l'utilisateur depuis un cache mémoire à durée de vie courte
// (USER_CACHE_TTL, 30s par défaut). Utilisé par le middleware JWT et l'audit, qui
// interrogeaient la table utilisateurs à chaque requête authentifiée.
// Le cache est invalidé immédiatement par UpdateUser et DeleteUser.
func (s *Store) GetUserByIDCached(ctx context.Context, id string) (*models.Utilisateur, error) {
	if val, ok := s.userCache.Load(id); ok {
		if entry := val.(userCacheEntry); time.Now().Before(entry.expires) {
			return entry.user, nil
		}
	}
	user, err := s.GetUserByID(ctx, id)
	if err != nil {
		return nil, err
	}
	s.userCache.Store(id, userCacheEntry{user: user, expires: time.Now().Add(s.userCacheTTL)})
	return user, nil
}

// ===================== UTILISATEURS =====================

// CreateUser insère un nouvel utilisateur dans la base de données.
func (s *Store) CreateUser(ctx context.Context, user *models.Utilisateur) (string, error) {
	// --- GORM ---
	// Vérifier si l'email existe déjà
	var existing models.Utilisateur
	err := s.gormDB.WithContext(ctx).Where("email = ?", user.Email).First(&existing).Error
	if err == nil {
		// Utilisateur déjà existant
		return "", fmt.Errorf("un utilisateur avec cet email existe déjà")
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		// Erreur inattendue
		return "", err
	}
	// Créer l'utilisateur
	result := s.gormDB.WithContext(ctx).Create(user)
	if result.Error != nil {
		return "", result.Error
	}
	return user.ID, nil
}

// GetUserByEmail récupère un utilisateur par son adresse email.
func (s *Store) GetUserByEmail(ctx context.Context, email string) (*models.Utilisateur, error) {
	var user models.Utilisateur
	// Recherche case-insensitive pour l'email
	result := s.gormDB.WithContext(ctx).Where("LOWER(email) = LOWER(?)", email).First(&user)
	if result.Error != nil {
		return nil, result.Error
	}
	return &user, nil
}

// GetUserByID récupère un utilisateur par son ID.
func (s *Store) GetUserByID(ctx context.Context, id string) (*models.Utilisateur, error) {
	var user models.Utilisateur
	result := s.gormDB.WithContext(ctx).First(&user, whereUserIDEquals, id)
	if result.Error != nil {
		return nil, result.Error
	}
	return &user, nil
}

// GetAllUsers récupère tous les utilisateurs (sans mot de passe)
func (s *Store) GetAllUsers(ctx context.Context) ([]models.Utilisateur, error) {
	var users []models.Utilisateur
	result := s.gormDB.WithContext(ctx).Select("id, nom_utilisateur, email, role, actif, date_creation").Find(&users)
	if result.Error != nil {
		return nil, result.Error
	}
	return users, nil
}

// UpdateUser met à jour certains champs d'un utilisateur (actif, role, nomUtilisateur)
func (s *Store) UpdateUser(ctx context.Context, id string, updates map[string]interface{}) (*models.Utilisateur, error) {
	var user models.Utilisateur
	if err := s.gormDB.WithContext(ctx).First(&user, whereUserIDEquals, id).Error; err != nil {
		return nil, err
	}

	if err := s.gormDB.WithContext(ctx).Model(&user).Updates(updates).Error; err != nil {
		return nil, err
	}

	s.userCache.Delete(id)

	// Recharger
	if err := s.gormDB.WithContext(ctx).First(&user, whereUserIDEquals, id).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

// DeleteUser supprime un utilisateur s'il n'a aucune trace, sinon le désactive
func (s *Store) DeleteUser(ctx context.Context, id string) error {
	// Vérifier s'il existe des logs liés à cet utilisateur
	var count int64
	err := s.gormDB.WithContext(ctx).Model(&models.LogAudit{}).Where("utilisateur_id = ?", id).Count(&count).Error
	if err != nil {
		return err
	}
	if count > 0 {
		// Désactiver si des traces existent
		err := s.gormDB.WithContext(ctx).Model(&models.Utilisateur{}).Where(whereUserIDEquals, id).Update("actif", false).Error
		s.userCache.Delete(id)
		if err != nil {
			return err
		}
		// Retourne une erreur spéciale pour signaler au handler qu'il faut afficher le bouton Désactiver
		return fmt.Errorf("blocked:disabled")
	}
	// Sinon, supprimer physiquement
	err = s.gormDB.WithContext(ctx).Delete(&models.Utilisateur{}, whereUserIDEquals, id).Error
	s.userCache.Delete(id)
	return err
}
