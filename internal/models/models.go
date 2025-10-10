package models

import (
	"time"
)

// Utilisateur représente un acteur du système (admin, analyste).
type Utilisateur struct {
	ID             string    `json:"id"`
	NomUtilisateur string    `json:"nomUtilisateur"`
	Email          string    `json:"email"`
	MotDePasseHash string    `json:"password"`
	Role           string    `json:"role"`
	Actif          bool      `json:"actif"`
	DateCreation   time.Time `json:"dateCreation"`
}

// Client représente une personne physique ou morale.
type Client struct {
	ID             string    `json:"id"`
	NumeroCompte   string    `json:"numeroCompte"`
	NomClient      string    `json:"nomClient"`
	TypeClient     string    `json:"typeClient"`
	TypeCompte     string    `json:"typeCompte"`
	Categorie      string    `json:"categorie"`
	Pays           string    `json:"pays"`
	Ville          string    `json:"ville"`
	Adresse        string    `json:"adresse"`
	ProfilClientID string    `json:"profilClientId,omitempty"`
	DateCreation   time.Time `json:"dateCreation"`
}

// Transaction représente une opération financière.
type Transaction struct {
	ID                 string    `json:"id,omitempty"`
	DateOperation      time.Time `json:"date_operation"`
	Montant            float64   `json:"montant"`
	Portee             string    `json:"portee"`
	PaysPayeur         string    `json:"pays_payeur"`
	VillePayeur        string    `json:"ville_payeur"`
	AdressePayeur      string    `json:"adresse_payeur"`
	TypeComptePayeur   string    `json:"type_compte_payeur"`
	TypeClientPayeur   string    `json:"type_client_payeur"`
	CategoriePayeur    string    `json:"categorie_payeur"`
	NumeroComptePayeur string    `json:"numero_compte_payeur"`
	NomPayeur          string    `json:"nom_payeur"`
	PaysPaye           string    `json:"pays_paye"`
	VillePaye          string    `json:"ville_paye"`
	AdressePaye        string    `json:"adresse_paye"`
	TypeComptePaye     string    `json:"type_compte_paye"`
	TypeClientPaye     string    `json:"type_client_paye"`
	CategoriePaye      string    `json:"categorie_paye"`
	NumeroComptePaye   string    `json:"numero_compte_paye"`
	NomPaye            string    `json:"nom_paye"`
}

// EntiteBlacklist représente une entité sous surveillance.
type EntiteBlacklist struct {
	ID                          string    `json:"id"`
	Nom                         string    `json:"nom"`
	NumeroCompte                string    `json:"numeroCompte"` // Numéro de compte pour vérification
	TypeEntite                  string    `json:"typeEntite"`   // "Interdit", "Blacklisté", "PEP", "Surveillé"
	InformationsSupplementaires string    `json:"informationsSupplementaires"`
	DateAjout                   time.Time `json:"dateAjout"`
}

// ProfilClient représente un segment de clientèle avec des règles spécifiques.
type ProfilClient struct {
	ID          string `json:"id"`
	NomProfil   string `json:"nomProfil"`
	Description string `json:"description"`
}

// RegleControle est une condition métier pour détecter des anomalies.
type RegleControle struct {
	ID               string  `json:"id"`
	NomRegle         string  `json:"nomRegle"`
	Description      string  `json:"description"`
	TypeRegle        string  `json:"typeRegle"` // Ex: "montant", "frequence"
	Criticite        string  `json:"criticite"`
	Seuil            float64 `json:"seuil"`
	PeriodeReference string  `json:"periodeReference"`
	Active           bool    `json:"active"`
	ProfilClientID   string  `json:"profilClientId"`
}

// Alerte est générée lorsqu'une règle est déclenchée.
type Alerte struct {
	ID                string    `json:"id"`
	TransactionID     string    `json:"transactionId"`
	RegleDeclencheeID string    `json:"regleDeclencheeId"`
	NiveauCriticite   string    `json:"niveauCriticite"`
	ScoreRisque       float64   `json:"scoreRisque"`
	RaisonDetection   string    `json:"raisonDetection"`
	Statut            string    `json:"statut"` // "En cours", "Traité", "Faux positif"
	AnalysteAssignéID string    `json:"analysteAssigneId"`
	DateGeneration    time.Time `json:"dateGeneration"`
}

// LogAudit trace les actions critiques effectuées dans le système.
type LogAudit struct {
	ID              int64     `json:"id"`
	DateHeure       time.Time `json:"dateHeure"`
	UtilisateurID   string    `json:"utilisateurId"`
	ActionEffectuee string    `json:"actionEffectuee"`
	Details         string    `json:"details"`
	Resultat        string    `json:"resultat"`
}
