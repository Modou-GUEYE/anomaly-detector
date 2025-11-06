package main

import (
	"anomaly-detector/internal/models"
	"anomaly-detector/internal/store"
	"context"
	"log"
)

func main() {
	// Connexion à la base Postgres
	connStr := "postgresql://anomaly_user:passer@localhost:5432/anomaly_db?sslmode=disable"
	dbStore, err := store.NewStore(connStr)
	if err != nil {
		log.Fatalf("Impossible d'initialiser le store: %v", err)
	}
	defer dbStore.Close()

	ctx := context.Background()

	// Créer les profils clients de base
	profils := []models.ProfilClient{
		{
			ID:          "35010251-6c98-4a49-9671-5fbdfdc3f4ff",
			NomProfil:   "Particulier",
			Description: "Profil pour les clients particuliers",
		},
		{
			ID:          "78f1853e-2c40-4f0d-8536-917c4889e7c2",
			NomProfil:   "Commerçant",
			Description: "Profil pour les clients commerçants",
		},
		{
			ID:          "9663462a-e1c7-427a-b317-2a8b1568b5b7",
			NomProfil:   "Business",
			Description: "Profil pour les petites et moyennes entreprises",
		},
		{
			ID:          "c944b60d-7f1b-4298-bcbf-2fc3dc4695fd",
			NomProfil:   "Entreprise",
			Description: "Profil pour les grandes entreprises",
		},
	}

	for _, profil := range profils {
		// Vérifier si le profil existe déjà
		var existingProfil models.ProfilClient
		result := dbStore.Gorm().WithContext(ctx).Where("id = ?", profil.ID).First(&existingProfil)

		if result.Error != nil {
			// Le profil n'existe pas, le créer
			_, err := dbStore.CreateProfilClient(ctx, &profil)
			if err != nil {
				log.Printf("Erreur lors de la création du profil %s: %v", profil.NomProfil, err)
			} else {
				log.Printf("Profil créé avec succès: %s (ID: %s)", profil.NomProfil, profil.ID)
			}
		} else {
			log.Printf("Profil déjà existant: %s (ID: %s)", profil.NomProfil, profil.ID)
		}
	}

	log.Println("Initialisation des profils clients terminée")
}
