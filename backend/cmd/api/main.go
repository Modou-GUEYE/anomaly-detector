package main

import (
	"anomaly-detector/internal/api"
	"anomaly-detector/internal/models"
	"anomaly-detector/internal/store"
	"context"
	"fmt"
	"log"
	"os"

	"github.com/joho/godotenv"
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

	ctx := context.Background()

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
			&models.SystemeExterne{},
		); err != nil {
			log.Fatalf("AutoMigrate failed: %v", err)
		}
		// L'initialisation des utilisateurs par défaut est désormais dans InitializeRequiredData
	}

	// Initialiser les données requises (profils clients + méta-données d'alertes + utilisateurs)
	if err := dbStore.InitializeRequiredData(ctx); err != nil {
		log.Fatalf("Impossible d'initialiser les données requises: %v", err)
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
