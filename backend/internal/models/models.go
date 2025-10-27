// ===================== PACKAGE & IMPORTS =====================
package models

import (
	"time"

	"gorm.io/datatypes"
)

// ===================== UTILISATEURS =====================

// Utilisateur représente un acteur du système (admin, analyste).
// @Description Utilisateur du système (admin ou analyste)
type Utilisateur struct {
	ID             string    `json:"id" gorm:"type:uuid;default:gen_random_uuid();primaryKey"`
	NomUtilisateur string    `json:"nomUtilisateur"`
	Email          string    `json:"email"`
	MotDePasseHash string    `json:"password"`
	Role           string    `json:"role"`
	Actif          bool      `json:"actif"`
	DateCreation   time.Time `json:"dateCreation"`
}

// ===================== COMPTES =====================

// Compte représente une personne physique ou morale.
// @Description Client du système (personne physique ou morale) obtenu lors de l'analyse des transactions
type Compte struct {
	ID             string    `json:"id" gorm:"type:uuid;default:gen_random_uuid();primaryKey"`
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

// ===================== TRANSACTIONS =====================

// Transaction représente une opération financière.
// @Description Transaction financière entre deux entités
type Transaction struct {
	ID                 string    `json:"id,omitempty" gorm:"type:uuid;default:gen_random_uuid();primaryKey"`
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
	RawData            []byte    `json:"raw_data"`
}

// ===================== BLACKLIST =====================

// EntiteBlacklist représente une entité sous surveillance.
// @Description Entité sous surveillance (Blacklist, PEP, etc.)
type EntiteBlacklist struct {
	ID                          string    `json:"id" gorm:"type:uuid;default:gen_random_uuid();primaryKey"`
	Nom                         string    `json:"nom"`
	NumeroCompte                string    `json:"numeroCompte"`
	TypeEntite                  string    `json:"typeEntite"`
	InformationsSupplementaires string    `json:"informationsSupplementaires"`
	DateAjout                   time.Time `json:"dateAjout"`
}

// TableName indique à GORM d'utiliser la table 'entites_blacklist'
func (EntiteBlacklist) TableName() string { return "entites_blacklist" }

// ===================== PROFILS CLIENTS =====================

// ProfilClient représente un segment de clientèle avec des règles spécifiques.
// @Description Profil de client avec règles spécifiques
type ProfilClient struct {
	ID          string `json:"id" gorm:"type:uuid;default:gen_random_uuid();primaryKey"`
	NomProfil   string `json:"nomProfil"`
	Description string `json:"description"`
}

// TableName indique à GORM d'utiliser la table 'profils_clients'
func (ProfilClient) TableName() string { return "profils_clients" }

// ===================== REGLES DE CONTROLE =====================

// RegleControle est une condition métier pour détecter des anomalies.
// @Description Règle de contrôle pour la détection d'anomalies
type RegleControle struct {
	ID               string         `json:"id" gorm:"type:uuid;default:gen_random_uuid();primaryKey"`
	NomRegle         string         `json:"nomRegle"`
	Description      string         `json:"description"`
	TypeRegle        string         `json:"typeRegle"`
	Criticite        string         `json:"criticite"`
	Seuil            float64        `json:"seuil"`
	PeriodeReference string         `json:"periodeReference"`
	Active           bool           `json:"active"`
	ProfilClientID   *string        `json:"profilClientId"`
	ListePays        datatypes.JSON `json:"listePays" gorm:"type:jsonb"`
}

// TableName indique à GORM d'utiliser la table 'regles_controle'
func (RegleControle) TableName() string { return "regles_controle" }

// ===================== ALERTES =====================

// Alerte est générée lorsqu'une règle est déclenchée.
// @Description Alerte générée lors du déclenchement d'une règle
type Alerte struct {
	ID                string    `json:"id" gorm:"type:uuid;default:gen_random_uuid();primaryKey"`
	TransactionID     string    `json:"transactionId"`
	Code              string    `json:"code"` // Code d'alerte unique
	Type              string    `json:"type"` // Type d'alerte (ex: "Montant", "Pays à risque", "Surveillé", etc.)
	Statut            string    `json:"statut"`
	Message           string    `json:"message"`
	NiveauCriticite   string    `json:"niveauCriticite"`
	ScoreRisque       float64   `json:"scoreRisque"`
	AnalysteAssigneID *string   `json:"analysteAssigneID"`
	DateGeneration    time.Time `json:"dateGeneration"`
	Action            string    `json:"action"`
	NumeroCompte      string    `json:"numeroCompte"`
	PaysRisque        *string   `json:"paysRisque,omitempty"`
	RegleDeclencheeID *string   `json:"regleDeclencheeID,omitempty"`
	// Documentation des codes d'alerte (exemple)
	// A001 : Montant suspect
	// A002 : Fréquence
	// A003 : Pays à risque
	// A004A : Interdit
	// A004B : PEP
	// A004C : Blacklist
	// A004D : Surveillés
	// ...
}

// ===================== META-ALERTES =====================

// AlertMeta représente la meta-donnée d'une alerte (code, type, statut, message, criticité)
// @Description Meta-donnée d'alerte pour gestion dynamique
type AlertMeta struct {
	Code            string `json:"code"`
	Type            string `json:"type"`
	Statut          string `json:"statut"`
	Message         string `json:"message"`
	NiveauCriticite string `json:"niveauCriticite"`
}

// ===================== SUIVI BLACKLIST =====================

// SuiviBlacklist représente une trace de transaction impliquant une entité sous surveillance.
type SuiviBlacklist struct {
	ID            string    `gorm:"type:uuid;default:gen_random_uuid();primaryKey" json:"id"`
	TransactionID string    `gorm:"type:uuid;not null" json:"transaction_id"`
	NumeroCompte  string    `gorm:"not null" json:"numero_compte"`
	StatutSuivi   string    `gorm:"not null" json:"statut_suivi"`
	DateDetection time.Time `gorm:"column:date_detection;autoCreateTime" json:"date_detection"`
}

func (SuiviBlacklist) TableName() string {
	return "suivi_blacklist"
}

// ===================== PAYS A RISQUE =====================

// PaysARisque représente un pays à risque pour la détection d'anomalies.
type PaysARisque struct {
	ID      string `gorm:"type:uuid;default:gen_random_uuid();primaryKey" json:"id"`
	NomPays string `gorm:"column:nom_pays" json:"nom_pays"`
}

func (PaysARisque) TableName() string {
	return "pays_a_risque"
}

// ===================== LOGS AUDIT =====================

// LogAudit trace les actions critiques effectuées dans le système.
// @Description Log d'audit des actions critiques
type LogAudit struct {
	ID              int64     `json:"id"`
	DateHeure       time.Time `json:"dateHeure"`
	UtilisateurID   string    `json:"utilisateurId"`
	ActionEffectuee string    `json:"actionEffectuee"`
	Details         string    `json:"details"`
	Resultat        string    `json:"resultat"`
}
