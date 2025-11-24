// ===================== PACKAGE & IMPORTS =====================
package main

import (
	"anomaly-detector/internal/models"
	"anomaly-detector/internal/store"
	"context"
	"log"
	"time"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

// HashPassword hache un mot de passe avec bcrypt
func HashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), 14)
	return string(bytes), err
}

// ===================== FONCTION PRINCIPALE =====================
// main initialise le store, nettoie la base et peuple les données de test.
func main() {
	// Connexion à la base Postgres
	connStr := "postgresql://anomaly_user:passer@localhost:5432/anomaly_db?sslmode=disable"
	dbStore, err := store.NewStore(connStr)
	if err != nil {
		log.Fatalf("Impossible d'initialiser le store: %v", err)
	}
	defer dbStore.Close()

	ctx := context.Background()

	cleanDatabase(ctx, dbStore)
	seedData(ctx, dbStore)
}

// ===================== FONCTIONS DE NETTOYAGE =====================
// cleanDatabase supprime toutes les données des tables principales pour repartir sur une base propre.
func cleanDatabase(ctx context.Context, s *store.Store) {
	s.Gorm().WithContext(ctx).Session(&gorm.Session{AllowGlobalUpdate: true}).Delete(&models.Alerte{})
	s.Gorm().WithContext(ctx).Session(&gorm.Session{AllowGlobalUpdate: true}).Delete(&models.SuiviBlacklist{})
	s.Gorm().WithContext(ctx).Session(&gorm.Session{AllowGlobalUpdate: true}).Delete(&models.Transaction{})
	s.Gorm().WithContext(ctx).Session(&gorm.Session{AllowGlobalUpdate: true}).Delete(&models.RegleControle{}) // optionnel
	s.Gorm().WithContext(ctx).Session(&gorm.Session{AllowGlobalUpdate: true}).Delete(&models.Compte{})
	s.Gorm().WithContext(ctx).Session(&gorm.Session{AllowGlobalUpdate: true}).Delete(&models.EntiteBlacklist{})
	s.Gorm().WithContext(ctx).Session(&gorm.Session{AllowGlobalUpdate: true}).Delete(&models.Utilisateur{})
}

// ===================== FONCTIONS DE PEUPLEMENT =====================
// seedData insère des utilisateurs, des entrées blacklist et des comptes de test.
func seedData(ctx context.Context, s *store.Store) {

	// --- Création des utilisateurs ---
	// Ajout d'un utilisateur admin
	adminPass, _ := HashPassword("admin123")
	s.CreateUser(ctx, &models.Utilisateur{NomUtilisateur: "admin", Email: "admin@gsie.sn", MotDePasseHash: adminPass, Role: "admin", Actif: true, DateCreation: time.Now()})

	// Ajout d'un utilisateur analyste
	analystePass, _ := HashPassword("analyste123")
	s.CreateUser(ctx, &models.Utilisateur{NomUtilisateur: "analyste", Email: "analyste@gsie.sn", MotDePasseHash: analystePass, Role: "analyst", Actif: true, DateCreation: time.Now()})

	// Ajout d'un utilisateur modou (analyste)
	modouPass, _ := HashPassword("modou123")
	s.CreateUser(ctx, &models.Utilisateur{NomUtilisateur: "modou GUEYE", Email: "modouG@gsie.sn", MotDePasseHash: modouPass, Role: "admin", Actif: true, DateCreation: time.Now()})

	// --- Peuplement de la Blacklist ---
	// Ajout de 4 entités à la blacklist avec différents types
	s.CreateBlacklistEntry(ctx, &models.EntiteBlacklist{
		Nom:          "Madiambal Diagne",
		NumeroCompte: "SN07900010012345678901234",
		TypeEntite:   "Interdit",
	}, "system")
	s.CreateBlacklistEntry(ctx, &models.EntiteBlacklist{
		Nom:          "Fatou Diop",
		NumeroCompte: "SN07900020098765432109876",
		TypeEntite:   "Blacklist",
	}, "system")
	s.CreateBlacklistEntry(ctx, &models.EntiteBlacklist{
		Nom:          "Ousmane Sonko",
		NumeroCompte: "SN07900030011223344556677",
		TypeEntite:   "PEP",
	}, "system")
	s.CreateBlacklistEntry(ctx, &models.EntiteBlacklist{
		Nom:          "Aminata Fall",
		NumeroCompte: "SN07900040099887766554433",
		TypeEntite:   "Surveillés",
	}, "system")

	// --- Création de comptes partenaires (code banque SN079) ---

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
	}, "system")
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
	}, "system")
}
