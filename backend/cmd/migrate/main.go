// ===================== PACKAGE & IMPORTS =====================
package main

import (
	"anomaly-detector/internal/models"
	"anomaly-detector/internal/store"
	"context"
	"log"
	"os"
	"time"

	"github.com/joho/godotenv"
	"golang.org/x/crypto/bcrypt"

	"path/filepath"
)

// Ce binaire initialise la base de données en exécutant AutoMigrate sur les modèles.
// Usage:
//   - définir DATABASE_URL si nécessaire
//   - exécuter `./migrate` ou lancer le container avec la commande correspondante
func main() {
	// Charger .env en dev si présent
	//_ = godotenv.Load("backend/.env", ".env")
	wd, _ := os.Getwd()
	_ = godotenv.Load(filepath.Join(wd, "backend", ".env"))

	connStr := os.Getenv("DATABASE_URL")
	if connStr == "" {
		connStr = "postgresql://anomaly_user:passer@localhost:5432/anomaly_db?sslmode=disable"
	}

	dbStore, err := store.NewStore(connStr)
	if err != nil {
		log.Fatalf("Impossible d'initialiser le store: %v", err)
	}
	defer dbStore.Close()

	log.Println("Running AutoMigrate for models...")
	if err := dbStore.Gorm().AutoMigrate(
		&models.Utilisateur{},
		&models.Compte{},
		&models.Transaction{},
		&models.EntiteBlacklist{},
		&models.ProfilClient{},
		&models.RegleControle{},
		&models.PaysRisque{},
		&models.Alerte{},
		&models.AlertMeta{},
		&models.SuiviBlacklist{},
		&models.LogAudit{},
	); err != nil {
		log.Fatalf("AutoMigrate failed: %v", err)
	}

	log.Println("AutoMigrate completed successfully")

	// Seed default users if requested (or use sensible defaults)
	seedDefaultUsers(dbStore)
}

// seedDefaultUsers crée deux utilisateurs par défaut (admin et analyst) s'ils n'existent pas.
// Les emails/mots de passe peuvent être fournis via les variables d'env :
// DEFAULT_ADMIN_EMAIL, DEFAULT_ADMIN_PASSWORD, DEFAULT_ANALYST_EMAIL, DEFAULT_ANALYST_PASSWORD
// Si elles ne sont pas fournies, des valeurs par défaut limitées sont utilisées (changez les en prod).
func seedDefaultUsers(s *store.Store) {
	ctx := context.Background()

	adminEmail := os.Getenv("DEFAULT_ADMIN_EMAIL")
	if adminEmail == "" {
		adminEmail = "admin@gsie.sn"
	}
	adminPass := os.Getenv("DEFAULT_ADMIN_PASSWORD")
	if adminPass == "" {
		adminPass = "admin123"
	}

	analystEmail := os.Getenv("DEFAULT_ANALYST_EMAIL")
	if analystEmail == "" {
		analystEmail = "analyste@gsie.sn"
	}
	analystPass := os.Getenv("DEFAULT_ANALYST_PASSWORD")
	if analystPass == "" {
		analystPass = "analyste123"
	}

	// helper to create user if missing
	createIfMissing := func(email, username, password, role string) {
		if _, err := s.GetUserByEmail(ctx, email); err == nil {
			log.Printf("user %s already exists, skipping", email)
			return
		}
		hashed, err := bcrypt.GenerateFromPassword([]byte(password), 14)
		if err != nil {
			log.Printf("failed to hash password for %s: %v", email, err)
			return
		}
		user := &models.Utilisateur{
			NomUtilisateur: username,
			Email:          email,
			MotDePasseHash: string(hashed),
			Role:           role,
			Actif:          true,
			DateCreation:   time.Now(),
		}
		if _, err := s.CreateUser(ctx, user); err != nil {
			log.Printf("failed to create user %s: %v", email, err)
			return
		}
		log.Printf("created default user %s (%s)", email, role)
	}

	createIfMissing(adminEmail, "admin", adminPass, "admin")
	createIfMissing(analystEmail, "analyste", analystPass, "analyst")
}
