// ===================== PACKAGE & IMPORTS =====================
package main

import (
	"anomaly-detector/internal/models"
	"anomaly-detector/internal/store"
	"context"
	"log"
	"os"

	"github.com/joho/godotenv"

	"path/filepath"
)

// Ce binaire initialise la base de données en exécutant AutoMigrate sur les modèles.
func main() {
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
		&models.SystemeExterne{},
	); err != nil {
		log.Fatalf("AutoMigrate failed: %v", err)
	}

	log.Println("AutoMigrate completed successfully")

	// Initialise toutes les données requises (profils, alertes, utilisateurs, BIS)
	if err := dbStore.InitializeRequiredData(context.Background()); err != nil {
		log.Fatalf("Erreur lors de l'initialisation des données requises: %v", err)
	}
	log.Println("✓ Données requises initialisées avec succès")
}
