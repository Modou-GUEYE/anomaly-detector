package main

import (
	"anomaly-detector/internal/api"
	"anomaly-detector/internal/models"
	"anomaly-detector/internal/store"
	"context"
	"fmt"
	"log"

	"github.com/go-faker/faker/v4"
)

func main() {
	connStr := "postgresql://anomaly_user:passer@123@localhost:5432/anomaly_db?sslmode=disable"
	dbStore, err := store.NewStore(connStr)
	if err != nil {
		log.Fatalf("❌ Impossible d'initialiser le store: %v", err)
	}
	defer dbStore.Close()

	ctx := context.Background()

	fmt.Println("Nettoyage de la base de données...")
	cleanDatabase(ctx, dbStore)

	fmt.Println("Création des données de test...")
	seedData(ctx, dbStore)

	fmt.Println("✅ Base de données peuplée avec succès !")
}

func cleanDatabase(ctx context.Context, s *store.Store) {
	s.DB().Exec(ctx, "DELETE FROM alertes")
	s.DB().Exec(ctx, "DELETE FROM transactions")
	s.DB().Exec(ctx, "DELETE FROM regles_controle")
	s.DB().Exec(ctx, "DELETE FROM clients")
	s.DB().Exec(ctx, "DELETE FROM profils_clients")
	s.DB().Exec(ctx, "DELETE FROM entites_blacklist")
	s.DB().Exec(ctx, "DELETE FROM utilisateurs")
}

func seedData(ctx context.Context, s *store.Store) {
	// --- Création des utilisateurs ---
	adminPass, _ := api.HashPassword("admin123")
	s.CreateUser(ctx, &models.Utilisateur{NomUtilisateur: "admin", Email: "admin@gsie.sn", MotDePasseHash: adminPass, Role: "admin", Actif: true})
	fmt.Println("- Utilisateur Admin créé")

	analystePass, _ := api.HashPassword("analyste123")
	s.CreateUser(ctx, &models.Utilisateur{NomUtilisateur: "analyste", Email: "analyste@gsie.sn", MotDePasseHash: analystePass, Role: "analyste", Actif: true})
	fmt.Println("- Utilisateur Analyste créé")

	// --- Création de la configuration ---
	p1, _ := s.CreateProfilClient(ctx, &models.ProfilClient{NomProfil: "Particulier Standard", Description: "Client normal"})
	p2, _ := s.CreateProfilClient(ctx, &models.ProfilClient{NomProfil: "Entreprise VIP", Description: "Client à haut volume"})
	fmt.Println("- 2 Profils Clients créés")

	s.CreateRegleControle(ctx, &models.RegleControle{NomRegle: "Fréquence > 5 en 24h", TypeRegle: "frequence", Seuil: 5, PeriodeReference: "24h", Criticite: "Moyenne", Active: true, ProfilClientID: p1.ID})
	s.CreateRegleControle(ctx, &models.RegleControle{NomRegle: "Cumul Hebdomadaire > 1 000 000 FCFA", TypeRegle: "montant", Seuil: 1000000.0, PeriodeReference: "7d", Criticite: "Élevée", Active: true, ProfilClientID: p2.ID})
	fmt.Println("- 2 Règles de Contrôle créées")

	// --- Peuplement de la Blacklist ---
	s.CreateBlacklistEntry(ctx, &models.EntiteBlacklist{Nom: "KYC_INTERDIT_01", TypeEntite: "Interdit"})
	fmt.Println("- 1 entrée 'Interdit' créée dans la blacklist")

	// --- Création des Clients ---
	// Créer un client entreprise pour la règle de cumul
	clientEntreprise, _ := s.CreateClient(ctx, &models.Client{KycID: "KYC_ENTREPRISE_TEST", TypeClient: "Entreprise", ProfilClientID: p2.ID})
	fmt.Println("- 1 Client Entreprise créé")

	// Créer 10 clients particuliers
	for i := 0; i < 10; i++ {
		s.CreateClient(ctx, &models.Client{KycID: faker.UUIDHyphenated(), TypeClient: "Particulier", ProfilClientID: p1.ID})
	}
	fmt.Println("- 10 Clients Particuliers créés")

	// --- Création de Transactions ---
	fmt.Println("Création d'un historique de transactions...")
	// Créons 2 transactions pour notre client Entreprise
	s.CreateTransaction(ctx, &models.Transaction{
		NumeroComptePayeur: clientEntreprise.KycID,
		Montant:            150000.0,
		Portee:             "NATIONAL",
		PaysPaye:           "Sénégal",
	})
	s.CreateTransaction(ctx, &models.Transaction{
		NumeroComptePayeur: clientEntreprise.KycID,
		Montant:            250000.0,
		Portee:             "NATIONAL",
		PaysPaye:           "Sénégal",
	})
	fmt.Println("- 2 transactions créées pour le client Entreprise")
}
