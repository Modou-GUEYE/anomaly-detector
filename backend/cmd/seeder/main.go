// ===================== PACKAGE & IMPORTS =====================
package main

import (
	"anomaly-detector/internal/api"
	"anomaly-detector/internal/models"
	"anomaly-detector/internal/store"
	"context"
	"fmt"
	"log"
	"time"
)

// ===================== FONCTION PRINCIPALE =====================
// main initialise le store, nettoie la base et peuple les données de test.
func main() {
	// Connexion à la base Postgres
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

// ===================== FONCTIONS DE NETTOYAGE =====================
// cleanDatabase supprime toutes les données des tables principales pour repartir sur une base propre.
func cleanDatabase(ctx context.Context, s *store.Store) {
	s.Gorm().WithContext(ctx).Exec("DELETE FROM alertes")
	s.Gorm().WithContext(ctx).Exec("DELETE FROM suivi_blacklist")
	s.Gorm().WithContext(ctx).Exec("DELETE FROM transactions")
	//s.Gorm().WithContext(ctx).Exec("DELETE FROM regles_controle")
	s.Gorm().WithContext(ctx).Exec("DELETE FROM comptes")
	s.Gorm().WithContext(ctx).Exec("DELETE FROM entites_blacklist")
	//s.Gorm().WithContext(ctx).Exec("DELETE FROM utilisateurs")
}

// ===================== FONCTIONS DE PEUPLEMENT =====================
// seedData insère des utilisateurs, des entrées blacklist et des comptes de test.
func seedData(ctx context.Context, s *store.Store) {

	// --- Création des utilisateurs ---
	// Ajout d'un utilisateur admin
	adminPass, _ := api.HashPassword("admin123")
	s.CreateUser(ctx, &models.Utilisateur{NomUtilisateur: "admin", Email: "admin@gsie.sn", MotDePasseHash: adminPass, Role: "admin", Actif: true, DateCreation: time.Now()})
	fmt.Println("- Utilisateur Admin créé")

	// Ajout d'un utilisateur analyste
	analystePass, _ := api.HashPassword("analyste123")
	s.CreateUser(ctx, &models.Utilisateur{NomUtilisateur: "analyste", Email: "analyste@gsie.sn", MotDePasseHash: analystePass, Role: "analyste", Actif: true, DateCreation: time.Now()})
	fmt.Println("- Utilisateur Analyste créé")

	// Ajout d'un utilisateur modou (analyste)
	modouPass, _ := api.HashPassword("modou123")
	s.CreateUser(ctx, &models.Utilisateur{NomUtilisateur: "modou", Email: "modou@gsie.sn", MotDePasseHash: modouPass, Role: "analyste", Actif: true, DateCreation: time.Now()})
	fmt.Println("- Utilisateur Modou créé")

	// --- Peuplement de la Blacklist ---
	// Ajout de 4 entités à la blacklist avec différents types
	s.CreateBlacklistEntry(ctx, &models.EntiteBlacklist{
		Nom:          "Madiambal Diagne",
		NumeroCompte: "SN07900010012345678901234",
		TypeEntite:   "Interdit",
	})
	s.CreateBlacklistEntry(ctx, &models.EntiteBlacklist{
		Nom:          "Fatou Diop",
		NumeroCompte: "SN07900020098765432109876",
		TypeEntite:   "Blacklist",
	})
	s.CreateBlacklistEntry(ctx, &models.EntiteBlacklist{
		Nom:          "Ousmane Sonko",
		NumeroCompte: "SN07900030011223344556677",
		TypeEntite:   "PEP",
	})
	s.CreateBlacklistEntry(ctx, &models.EntiteBlacklist{
		Nom:          "Aminata Fall",
		NumeroCompte: "SN07900040099887766554433",
		TypeEntite:   "Surveillée",
	})

	fmt.Println("- 4 entrées créées dans la blacklist (Interdit, Blacklist, PEP, Surveillée)")

	// --- Création de comptes partenaires (code banque SN079) ---
	fmt.Println("Création de comptes partenaires (code banque SN079)...")

	// Récupère l'ID du profil client pour les comptes partenaires
	profilID, _ := s.GetProfilClientIDByCode(ctx, "M")
	s.CreateCompte(ctx, &models.Compte{
		NumeroCompte:   "SN07900010000000000001",
		NomClient:      "Partenaire 1",
		TypeClient:     "M",
		TypeCompte:     "Entreprise",
		Categorie:      "Banque",
		Pays:           "Sénégal",
		Ville:          "Dakar",
		Adresse:        "Avenue Partenaire 1",
		ProfilClientID: profilID,
		DateCreation:   time.Now(),
	})
	s.CreateCompte(ctx, &models.Compte{
		NumeroCompte:   "SN07900020000000000002",
		NomClient:      "Partenaire 2",
		TypeClient:     "M",
		TypeCompte:     "Entreprise",
		Categorie:      "Banque",
		Pays:           "Sénégal",
		Ville:          "Dakar",
		Adresse:        "Avenue Partenaire 2",
		ProfilClientID: profilID,
		DateCreation:   time.Now(),
	})
	fmt.Println("- 2 comptes partenaires créés pour les tests")
}
