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
	Pays           string    `json:"pays"` // Code pays ISO alpha-2 (ex: SN, ML, CI) - Compatible SPI
	Ville          string    `json:"ville"`
	Adresse        string    `json:"adresse"`
	ProfilClientID string    `json:"profilClientId,omitempty"`
	DateCreation   time.Time `json:"dateCreation"`
}

// ===================== TRANSACTIONS =====================

// Transaction représente une transaction financière
// @Description Transaction financière entre deux entités
type Transaction struct {
	ID                 string    `json:"id,omitempty" gorm:"type:uuid;default:gen_random_uuid();primaryKey"`
	DateOperation      time.Time `json:"date_operation"`
	Montant            float64   `json:"montant"`
	Portee             string    `json:"portee"`
	PaysPayeur         string    `json:"pays_payeur"` // Code pays ISO alpha-2 (ex: SN, ML, CI) - Compatible SPI
	VillePayeur        string    `json:"ville_payeur"`
	AdressePayeur      string    `json:"adresse_payeur"`
	TypeComptePayeur   string    `json:"type_compte_payeur"`
	TypeClientPayeur   string    `json:"type_client_payeur"`
	CategoriePayeur    string    `json:"categorie_payeur"`
	NumeroComptePayeur string    `json:"numero_compte_payeur"`
	NomPayeur          string    `json:"nom_payeur"`
	PaysPaye           string    `json:"pays_paye"` // Code pays ISO alpha-2 (ex: SN, ML, CI) - Compatible SPI
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
	Action           string         `json:"action" gorm:"default:'surveiller'"` // Action: "bloquer", "surveiller", "notifier"
	ProfilClientID   *string        `json:"profilClientId"`
	ListePays        datatypes.JSON `json:"listePays" gorm:"type:jsonb"`
	// Pays lie explicitement cette règle aux entrées de la table pays_risque (utile pour la règle de type 'pays_risque')
	Pays []PaysRisque `json:"pays,omitempty" gorm:"foreignKey:RegleControleID;references:ID"`
}

// TableName indique à GORM d'utiliser la table 'regles_controle'
func (RegleControle) TableName() string { return "regles_controle" }

// ===================== PAYS À RISQUE =====================

// PaysRisque représente un pays dans la liste des pays à risque
// @Description Pays à risque pour la détection de transactions
type PaysRisque struct {
	ID string `json:"id" gorm:"type:uuid;default:gen_random_uuid();primaryKey"`
	// RegleControleID établit la liaison explicite avec la règle 'pays_risque'
	RegleControleID *string   `json:"regleControleId" gorm:"type:uuid;index"`
	Code            string    `json:"code" gorm:"unique;size:2;not null"` // Code ISO 3166-1 alpha-2 (ex: SN, ML, CI) - OBLIGATOIRE - Compatible SPI
	NomPays         string    `json:"nomPays" gorm:"unique;not null"`
	Active          bool      `json:"active" gorm:"default:true"`
	CreatedAt       time.Time `json:"createdAt" gorm:"autoCreateTime"`
	UpdatedAt       time.Time `json:"updatedAt" gorm:"autoUpdateTime"`
}

// TableName indique à GORM d'utiliser la table 'pays_risque'
func (PaysRisque) TableName() string { return "pays_risque" }

// ===================== ALERTES =====================

// Alerte est générée lorsqu'une règle est déclenchée.
// @Description Alerte générée lors du déclenchement d'une règle
type Alerte struct {
	ID                string     `json:"id" gorm:"type:uuid;default:gen_random_uuid();primaryKey"`
	TransactionID     string     `json:"transactionId"`
	Code              string     `json:"code"` // Code d'alerte unique
	Type              string     `json:"type"` // Type d'alerte (ex: "Montant", "Pays à risque", "Surveillé", etc.)
	Statut            string     `json:"statut"`
	Message           string     `json:"message"`
	NiveauCriticite   string     `json:"niveauCriticite"`
	ScoreRisque       float64    `json:"scoreRisque"`
	AnalysteAssigneID *string    `json:"analysteAssigneID"`
	DateGeneration    time.Time  `json:"dateGeneration"`
	DateTraitement    *time.Time `json:"dateTraitement,omitempty"` // Date de traitement/résolution
	Action            string     `json:"action"`
	NumeroCompte      string     `json:"numeroCompte"`
	PaysRisque        *string    `json:"paysRisque,omitempty"`
	RegleDeclencheeID *string    `json:"regleDeclencheeID,omitempty"`
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

// SuiviBlacklist trace les transactions liées aux entités blacklistées.
// @Description Suivi des transactions liées aux entités blacklistées
type SuiviBlacklist struct {
	ID            string    `gorm:"type:uuid;default:gen_random_uuid();primaryKey" json:"id"`
	TransactionID string    `gorm:"type:uuid;not null" json:"transaction_id"`
	NumeroCompte  string    `gorm:"not null" json:"numero_compte"`
	StatutSuivi   string    `gorm:"not null" json:"statut_suivi"`
	Raison        string    `gorm:"default:''" json:"raison"` // Source du surveillance (ex: PEP, Blacklist, Montant, etc.)
	DateDetection time.Time `gorm:"column:date_detection;autoCreateTime" json:"date_detection"`
}

func (SuiviBlacklist) TableName() string {
	return "suivi_blacklist"
}

// ===================== LOGS AUDIT =====================

// LogAudit trace les actions critiques effectuées dans le système.
// @Description Log d'audit des actions critiques
type LogAudit struct {
	ID              string    `gorm:"type:uuid;default:gen_random_uuid();primaryKey" json:"id"`
	DateHeure       time.Time `gorm:"column:date_heure;autoCreateTime" json:"date_heure"`
	UtilisateurID   string    `gorm:"column:utilisateur_id" json:"utilisateur_id"`
	ActionEffectuee string    `gorm:"column:action_effectuee" json:"action_effectuee"`
	Details         string    `gorm:"column:details" json:"details"`
	Resultat        string    `gorm:"column:resultat" json:"resultat"`
	TypeAction      string    `gorm:"column:type_action" json:"type_action"` // "alerte", "transaction", "regle", "blacklist"
	EntityID        string    `gorm:"column:entity_id" json:"entity_id"`     // ID de l'entité concernée
	AdresseIP       string    `gorm:"column:adresse_ip" json:"adresse_ip"`
	Role            string    `gorm:"column:role" json:"role"` // Rôle de l'utilisateur
}

func (LogAudit) TableName() string {
	return "logs_audit"
}
