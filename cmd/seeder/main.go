package main

import (
	"anomaly-detector/internal/api"
	"anomaly-detector/internal/models"
	"anomaly-detector/internal/store"
	"context"
	"fmt"
	"log"
)

func main() {
	connStr := "postgresql://anomaly_user:passer@localhost:5432/anomaly_db?sslmode=disable"
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
	s.DB().Exec(ctx, "DELETE FROM suivi_blacklist")
	s.DB().Exec(ctx, "DELETE FROM transactions")
	s.DB().Exec(ctx, "DELETE FROM regles_controle")
	s.DB().Exec(ctx, "DELETE FROM clients")
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

	// --- Peuplement de la Blacklist ---
	s.CreateBlacklistEntry(ctx, &models.EntiteBlacklist{
		Nom:          "Madiambal Diagne",
		NumeroCompte: "SN01100010012345678901234",
		TypeEntite:   "Interdits",
	})
	s.CreateBlacklistEntry(ctx, &models.EntiteBlacklist{
		Nom:          "Fatou Diop",
		NumeroCompte: "SN01100020098765432109876",
		TypeEntite:   "Blacklist",
	})
	s.CreateBlacklistEntry(ctx, &models.EntiteBlacklist{
		Nom:          "Ousmane Sonko",
		NumeroCompte: "SN01100030011223344556677",
		TypeEntite:   "PEP",
	})
	s.CreateBlacklistEntry(ctx, &models.EntiteBlacklist{
		Nom:          "Aminata Fall",
		NumeroCompte: "SN01100040099887766554433",
		TypeEntite:   "Surveillés",
	})

	fmt.Println("- 4 entrées créées dans la blacklist (Interdits, Blacklist, PEP, Surveillés)")

	// --- Création de Transactions ---
	fmt.Println("Création d'un historique de transactions...")
	s.CreateTransaction(ctx, &models.Transaction{
		NumeroComptePayeur: "SN01100020098765432109876", // Fatou Diop (Blacklist)
		NomPayeur:          "Fatou Diop",
		TypeComptePayeur:   "Particulier",
		CategoriePayeur:    "Commerçant",
		PaysPayeur:         "Sénégal",
		VillePayeur:        "Dakar",
		AdressePayeur:      "Rue 12",
		NumeroComptePaye:   "SN01100040099887766554433", // Aminata Fall (Surveillés)
		NomPaye:            "Aminata Fall",
		TypeComptePaye:     "Entreprise",
		CategoriePaye:      "Importateur",
		PaysPaye:           "Sénégal",
		VillePaye:          "Thiès",
		AdressePaye:        "Avenue 7",
		Montant:            150000.0,
		Portee:             "NATIONAL",
	})
	s.CreateTransaction(ctx, &models.Transaction{
		NumeroComptePayeur: "SN01100010012345678901234", // Madiambal Diagne (Interdits)
		NomPayeur:          "Madiambal Diagne",
		TypeComptePayeur:   "Particulier",
		CategoriePayeur:    "Journaliste",
		PaysPayeur:         "Sénégal",
		VillePayeur:        "Saint-Louis",
		AdressePayeur:      "Rue 5",
		NumeroComptePaye:   "SN01100030011223344556677", // Ousmane Sonko (PEP)
		NomPaye:            "Ousmane Sonko",
		TypeComptePaye:     "Particulier",
		CategoriePaye:      "Politicien",
		PaysPaye:           "Sénégal",
		VillePaye:          "Ziguinchor",
		AdressePaye:        "Boulevard 3",
		Montant:            250000.0,
		Portee:             "NATIONAL",
	})
	fmt.Println("- 2 transactions créées pour le client Entreprise")
}
