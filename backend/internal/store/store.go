package store

import (
	"anomaly-detector/internal/models"
	"context"
)

// RuleViolation représente une violation de règle avec son nombre d'occurrences
type RuleViolation struct {
	RuleName   string  `json:"ruleName"`
	Count      int     `json:"count"`
	Percentage float64 `json:"percentage"`
}

// StoreInterface définit toutes les méthodes de notre store
type StoreInterface interface {
	// Configuration et base
	Close()
	Gorm() interface{}

	// Utilisateurs
	CreateUser(ctx context.Context, user *models.Utilisateur) (string, error)
	GetUserByEmail(ctx context.Context, email string) (*models.Utilisateur, error)
	GetUserByID(ctx context.Context, id string) (*models.Utilisateur, error)
	GetAllUsers(ctx context.Context) ([]models.Utilisateur, error)
	UpdateUser(ctx context.Context, id string, updates map[string]interface{}) (*models.Utilisateur, error)
	DeleteUser(ctx context.Context, id string) error

	// Blacklist et surveillance
	CreateBlacklistEntry(ctx context.Context, entry *models.EntiteBlacklist, userID string) (*models.EntiteBlacklist, error)
	GetBlacklistEntries(ctx context.Context) ([]models.EntiteBlacklist, error)
	UpdateBlacklistEntry(ctx context.Context, id string, entry *models.EntiteBlacklist, userID string) (*models.EntiteBlacklist, error)
	DeleteBlacklistEntry(ctx context.Context, id string, userID string) error
	GetBlacklistStatusForCompte(ctx context.Context, compteID string) (string, error)
	IsAccountOnBlacklist(ctx context.Context, kycID string) (string, error)
	CreateSuiviBlacklist(ctx context.Context, transactionID, numeroCompte, statutSuivi, raison string) error
	GetComptesSurveillesCount(ctx context.Context) (int64, error)
	GetComptesSurveilles(ctx context.Context) ([]string, error)
	GetSuiviBlacklist(ctx context.Context) ([]*models.SuiviBlacklist, error)

	// Profils clients
	GetProfilClientIDByCode(ctx context.Context, code string) (*string, error)
	CreateProfilClient(ctx context.Context, profil *models.ProfilClient) (*models.ProfilClient, error)
	GetProfilsClient(ctx context.Context) ([]models.ProfilClient, error)
	UpdateProfilClient(ctx context.Context, id string, profil *models.ProfilClient) (*models.ProfilClient, error)
	DeleteProfilClient(ctx context.Context, id string) error

	// Règles de contrôle et pays à risque
	CreateRegleControle(ctx context.Context, regle *models.RegleControle, userID string) (*models.RegleControle, error)
	GetReglesByProfilID(ctx context.Context, profilID string) ([]models.RegleControle, error)
	UpdateRegleControle(ctx context.Context, id string, regle *models.RegleControle, userID string) (*models.RegleControle, error)
	DeleteRegleControle(ctx context.Context, id string, userID string) error
	UpdateRegleControleStatus(ctx context.Context, id string, status string, userID string) (*models.RegleControle, error)
	GetAllRegles(ctx context.Context) ([]models.RegleControle, error)
	GetRegleByType(ctx context.Context, typeRegle string) (*models.RegleControle, error)
	GetReglePaysRisqueActive(ctx context.Context) (*models.RegleControle, error)
	IsPaysARisque(ctx context.Context, pays string) (bool, error)
	IsPaysARisqueByCode(ctx context.Context, codePays string) (bool, error)
	GetAllPaysARisque(ctx context.Context) ([]models.PaysRisque, error)
	AddPaysARisque(ctx context.Context, nomPays string, userID string) error
	AddPaysARisqueWithCode(ctx context.Context, pays *models.PaysRisque, userID string) error
	DeletePaysARisque(ctx context.Context, nomPays string, userID string) error
	UpdatePaysARisque(ctx context.Context, ancienNom, nouveauNom string, userID string) error
	UpdatePaysARisqueWithCode(ctx context.Context, ancienNom, nouveauNom, code string, userID string) error
	LinkPaysRisqueToRegle(ctx context.Context) error

	// Comptes
	CreateCompte(ctx context.Context, compte *models.Compte, userID string) (*models.Compte, error)
	GetCompteByCompte(ctx context.Context, numeroCompte string) (*models.Compte, error)
	GetReglesForCompte(ctx context.Context, numeroCompte string) ([]models.RegleControle, error)
	GetAllComptes(ctx context.Context) ([]*models.Compte, error)

	// Transactions
	CreateTransaction(ctx context.Context, tx *models.Transaction) (*models.Transaction, error)
	GetTransactions(ctx context.Context) ([]*models.Transaction, error)
	GetTransactionSumForPeriod(ctx context.Context, numeroCompte string, jours int) (float64, error)
	GetTransactionCountForPeriod(ctx context.Context, numeroCompte string, jours int) (int64, error)

	// Alertes
	GetAlertes(ctx context.Context) ([]models.Alerte, error)
	GetAlertCountsByMonthAndType(ctx context.Context, months int) (map[string][]map[string]int64, error)
	CreateAlerte(ctx context.Context, alerte *models.Alerte, userID string) (*models.Alerte, error)
	FixAlertesTaitesesSansDateTraitement(ctx context.Context) error
	UpdateAlerteStatus(ctx context.Context, alerteID string, analysteID string, newStatus string) (*models.Alerte, error)
	GetAlertMetaByType(ctx context.Context, alertType string) (*models.AlertMeta, error)
	GetAlertesBloquerActivesByCompte(ctx context.Context, numeroCompte string, code ...string) ([]models.Alerte, error)

	// Dashboard et métriques
	GetDashboardSummary(ctx context.Context, dateDebut, dateFin string) (map[string]interface{}, error)
	GetEfficiencyMetrics(ctx context.Context, period string) ([]EfficiencyMetric, error)
	GetRuleViolationsByPeriod(ctx context.Context, period string) ([]RuleViolation, error)

	// Audit et conformité
	CreateLogAudit(ctx context.Context, log *models.LogAudit) error
	LogAction(ctx context.Context, utilisateurID, action, typeAction, entityID, details, resultat string) error
	LogActionWithRole(ctx context.Context, utilisateurID, action, typeAction, entityID, details, resultat, role string) error
	GetLogsAudit(ctx context.Context, filters map[string]interface{}) ([]models.LogAudit, error)
	GetAuditStatistics(ctx context.Context, periode int) (map[string]interface{}, error)
	GetComplianceMetrics(ctx context.Context) (map[string]interface{}, error)

	// Systèmes externes
	IsSystemIDAllowed(ctx context.Context, id string, role string) (bool, error)
}
