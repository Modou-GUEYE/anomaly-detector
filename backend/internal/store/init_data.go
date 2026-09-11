package store

import (
	"context"
	"fmt"
	"log"
	"time"

	"anomaly-detector/internal/models"

	"golang.org/x/crypto/bcrypt"
)

// InitializeRequiredData initialise les données essentielles au démarrage
func (s *Store) InitializeRequiredData(ctx context.Context) error {
	log.Println("Initialisation des données requises...")

	// 1. Initialiser les profils clients
	if err := s.initializeProfilsClients(ctx); err != nil {
		return fmt.Errorf("erreur lors de l'initialisation des profils clients: %w", err)
	}

	// 2. Initialiser les méta-données d'alertes
	if err := s.initializeAlertMeta(ctx); err != nil {
		return fmt.Errorf("erreur lors de l'initialisation des méta-données d'alertes: %w", err)
	}

	// 3. Initialiser un participant BIS si non présent
	var countBIS int64
	err := s.Gorm().WithContext(ctx).Model(&models.SystemeExterne{}).Where("id LIKE ?", "bis-%").Count(&countBIS).Error
	if err != nil {
		return fmt.Errorf("erreur lors de la vérification du participant BIS: %w", err)
	}
	if countBIS == 0 {
		_, err := s.CreateSystemeExterne(ctx, "bis", "Participant BIS", "system", true)
		if err != nil {
			return fmt.Errorf("erreur lors de la création du participant BIS: %w", err)
		}
		log.Println("  ✓ Participant BIS créé")
	} else {
		log.Println("✓ Participant BIS déjà existant")
	}

	// 4. Initialiser les utilisateurs par défaut
	if err := s.initializeDefaultUsers(ctx); err != nil {
		return fmt.Errorf("erreur lors de l'initialisation des utilisateurs par défaut: %w", err)
	}

	// 5. Charger l'ensemble des données de référence dans le cache mémoire
	if err := s.RefreshAll(ctx); err != nil {
		log.Printf("Avertissement: échec partiel du chargement du cache mémoire: %v", err)
	} else {
		log.Println("✓ Cache mémoire de référence initialisé")
	}

	log.Println("✓ Données requises initialisées avec succès")
	return nil
}

// initializeProfilsClients insère les 4 profils clients par défaut s'ils n'existent pas
func (s *Store) initializeProfilsClients(ctx context.Context) error {
	var count int64
	if err := s.gormDB.WithContext(ctx).Model(&models.ProfilClient{}).Count(&count).Error; err != nil {
		return err
	}

	if count > 0 {
		log.Printf("✓ Profils clients déjà existants (%d profils)", count)
		return nil
	}

	log.Println("→ Insertion des profils clients par défaut...")

	profils := []models.ProfilClient{
		{NomProfil: "Particulier", Description: "Client particulier"},
		{NomProfil: "Entreprise", Description: "Client entreprise"},
		{NomProfil: "Commerçant", Description: "Client Commerçant(e)"},
		{NomProfil: "Business", Description: "Client business"},
	}

	for _, profil := range profils {
		// Vérifier si le profil existe déjà (par nom)
		var existing models.ProfilClient
		err := s.gormDB.WithContext(ctx).Where("LOWER(nom_profil) = LOWER(?)", profil.NomProfil).First(&existing).Error
		if err == nil {
			log.Printf("  - Profil '%s' déjà existant, skip", profil.NomProfil)
			continue
		}

		// Créer le profil
		if err := s.gormDB.WithContext(ctx).Create(&profil).Error; err != nil {
			return fmt.Errorf("erreur lors de la création du profil '%s': %w", profil.NomProfil, err)
		}
		log.Printf("  ✓ Profil '%s' créé", profil.NomProfil)
	}

	return nil
}

// initializeAlertMeta insère les méta-données d'alertes par défaut si elles n'existent pas
func (s *Store) initializeAlertMeta(ctx context.Context) error {
	var count int64
	if err := s.gormDB.WithContext(ctx).Table("alertes_meta").Count(&count).Error; err != nil {
		return err
	}

	if count > 0 {
		log.Printf("✓ Méta-données d'alertes déjà existantes (%d méta-données)", count)
		return nil
	}

	log.Println("→ Insertion des méta-données d'alertes par défaut...")

	metas := []models.AlertMeta{
		{Code: "A001", Type: "Montant", Statut: "En cours", Message: "Cumul sur 7 jours dépassant le seuil autorisé.", NiveauCriticite: "Haute"},
		{Code: "A002", Type: "frequence", Statut: "En cours", Message: "Nombre de transactions sur 24h dépassant le seuil autorisé.", NiveauCriticite: "Haute"},
		{Code: "A003", Type: "Pays à risque", Statut: "En cours", Message: "Transaction vers ou depuis un pays à risque.", NiveauCriticite: "Haute"},
		{Code: "A004A", Type: "Interdit", Statut: "En cours", Message: "Compte interdit : transaction refusée.", NiveauCriticite: "Haute"},
		{Code: "A004B", Type: "Blacklist", Statut: "En cours", Message: "Compte blacklisté : transaction validée sous surveillance.", NiveauCriticite: "Haute"},
		{Code: "A004C", Type: "PEP", Statut: "En cours", Message: "Personne Politiquement Exposée : transaction validée sous surveillance.", NiveauCriticite: "Moyenne"},
		{Code: "A004D", Type: "Surveillés", Statut: "En cours", Message: "Compte sous surveillance : transaction validée avec suivi renforcé.", NiveauCriticite: "Moyenne"},
	}

	for _, meta := range metas {
		// Vérifier si la méta-donnée existe déjà (par code)
		var existing models.AlertMeta
		err := s.gormDB.WithContext(ctx).Table("alertes_meta").Where("code = ?", meta.Code).First(&existing).Error
		if err == nil {
			log.Printf("  - Méta-donnée '%s' (%s) déjà existante, skip", meta.Code, meta.Type)
			continue
		}

		// Créer la méta-donnée
		if err := s.gormDB.WithContext(ctx).Table("alertes_meta").Create(&meta).Error; err != nil {
			return fmt.Errorf("erreur lors de la création de la méta-donnée '%s': %w", meta.Code, err)
		}
		log.Printf("  ✓ Méta-donnée '%s' (%s) créée", meta.Code, meta.Type)
	}

	return nil
}

// initializeDefaultUsers insère les utilisateurs par défaut s'ils n'existent pas
func (s *Store) initializeDefaultUsers(ctx context.Context) error {
	users := []struct{ email, username, password, role string }{
		{"admin@gsie.sn", "admin", "admin123", "admin"},
		{"analyste@gsie.sn", "analyste", "analyste123", "analyst"},
	}
	for _, u := range users {
		if _, err := s.GetUserByEmail(ctx, u.email); err == nil {
			continue
		}
		hashed, err := bcrypt.GenerateFromPassword([]byte(u.password), 14)
		if err != nil {
			log.Printf("unable to hash default user %s: %v", u.email, err)
			continue
		}
		if _, err := s.CreateUser(ctx, &models.Utilisateur{
			NomUtilisateur: u.username,
			Email:          u.email,
			MotDePasseHash: string(hashed),
			Role:           u.role,
			Actif:          true,
			DateCreation:   time.Now(),
		}); err != nil {
			log.Printf("unable to create default user %s: %v", u.email, err)
		}
	}
	return nil
}
