package main

import (
	"anomaly-detector/internal/api"
	"anomaly-detector/internal/models"
	"anomaly-detector/internal/store"
	"context"
	"fmt"
	"log"
	"os"

	"time"

	"github.com/joho/godotenv"
	"golang.org/x/crypto/bcrypt"
)

// @title           API du Moteur de Détection d'Anomalies
// @version         1.0
// @description     Backend pour le projet MDA de GSIE Technology, conçu pour la détection de fraudes LBC/FT.
// @contact.name    Modou GUEYE
// @contact.email   modou.gueye@gsietechnology.com
// @license.name    MIT
// @host            localhost:8080
// @BasePath        /api/v1
// @securityDefinitions.apikey  BearerAuth
// @in                          header
// @name                        Authorization

func main() {
	loadDotEnv()

	connStr := buildDatabaseURL()
	dbStore, err := store.NewStore(connStr)
	if err != nil {
		log.Fatalf("Impossible d'initialiser le store: %v", err)
	}
	defer dbStore.Close()

	if os.Getenv("DB_CHECK_SIGNATURE") == "true" {
		if err := checkDatabaseSignature(dbStore); err != nil {
			log.Fatalf("❌ Database signature check failed: %v", err)
		}
		log.Println("✅ Database signature check passed")
	}

	if os.Getenv("DB_AUTO_MIGRATE") == "true" {
		log.Println("Running AutoMigrate...")
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
		seedDefaultUsers(dbStore)
	}

	server := api.NewServer(dbStore)
	if err := server.Start(":8080"); err != nil {
		log.Fatalf("Impossible de démarrer le serveur: %v", err)
	}
}

func buildDatabaseURL() string {
	if connStr := os.Getenv("DATABASE_URL"); connStr != "" {
		return connStr
	}

	dbName := getEnv("DEV_DB_NAME", "anomaly_db")
	dbUser := getEnv("DEV_DB_USER", "anomaly_user")
	dbPassword := getEnv("DEV_DB_PASSWORD", "passer")
	dbHost := getEnv("DEV_DB_HOST", "localhost")
	dbPort := getEnv("DEV_DB_PORT", "5432")
	sslMode := "require"
	if os.Getenv("DB_SSL_DISABLE") == "true" {
		sslMode = "disable"
	}

	return fmt.Sprintf("postgresql://%s:%s@%s:%s/%s?sslmode=%s",
		dbUser, dbPassword, dbHost, dbPort, dbName, sslMode)
}

func loadDotEnv() {
	candidates := []string{"backend/.env", ".env"}
	for _, p := range candidates {
		if err := godotenv.Load(p); err == nil {
			log.Printf("Loaded .env from %s", p)
			return
		}
	}
	log.Println(".env not loaded (ok in prod), continuing")
}

func seedDefaultUsers(s *store.Store) {
	users := []struct{ email, username, password, role string }{
		{getEnv("DEFAULT_ADMIN_EMAIL", "admin@gsie.sn"), "admin", getEnv("DEFAULT_ADMIN_PASSWORD", "admin123"), "admin"},
		{getEnv("DEFAULT_ANALYST_EMAIL", "analyste@gsie.sn"), "analyste", getEnv("DEFAULT_ANALYST_PASSWORD", "analyste123"), "analyst"},
	}

	ctx := context.Background()
	for _, u := range users {
		if _, err := s.GetUserByEmail(ctx, u.email); err == nil {
			continue
		}
		hashed, _ := bcrypt.GenerateFromPassword([]byte(u.password), 14)
		s.CreateUser(ctx, &models.Utilisateur{
			NomUtilisateur: u.username,
			Email:          u.email,
			MotDePasseHash: string(hashed),
			Role:           u.role,
			Actif:          true,
			DateCreation:   time.Now(),
		})
	}
}

func getEnv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func checkDatabaseSignature(s *store.Store) error {
	tables := []string{"utilisateurs", "comptes", "transactions", "entites_blacklist", "profils_client", "regles_controle", "pays_risque", "alertes"}
	for _, table := range tables {
		if !s.Gorm().Migrator().HasTable(table) {
			return fmt.Errorf("table manquante: %s", table)
		}
	}
	return nil
}
