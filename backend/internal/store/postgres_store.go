package store

import (
	// Standard
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	// Externes
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	gormLogger "gorm.io/gorm/logger"

	// Internes
	"anomaly-detector/internal/models"
)

// gormLoggerSilent retourne un logger GORM en mode silent
func gormLoggerSilent() gormLogger.Interface {
	return gormLogger.Default.LogMode(gormLogger.Silent)
}

// Store contient le pool de connexions à la base de données.
type Store struct {
	gormDB *gorm.DB // Connexion GORM
}

// Gorm expose la connexion GORM pour des opérations directes si nécessaire.
func (s *Store) Gorm() *gorm.DB {
	return s.gormDB
}

// NewStore crée et retourne une nouvelle instance de notre Store.
func NewStore(connStr string) (*Store, error) {
	// Initialisation GORM avec logger en mode silent
	gormDB, err := gorm.Open(postgres.Open(connStr), &gorm.Config{
		Logger: gormLoggerSilent(),
	})
	if err != nil {
		return nil, err
	}
	return &Store{gormDB: gormDB}, nil
}

// Close ferme le pool de connexions.
func (s *Store) Close() {
	sqlDB, err := s.gormDB.DB()
	if err == nil {
		sqlDB.Close()
	}
}

// ===================== UTILISATEURS =====================

// CreateUser insère un nouvel utilisateur dans la base de données.
func (s *Store) CreateUser(ctx context.Context, user *models.Utilisateur) (string, error) {
	// --- GORM ---
	// Vérifier si l'email existe déjà
	var existing models.Utilisateur
	err := s.gormDB.WithContext(ctx).Where("email = ?", user.Email).First(&existing).Error
	if err == nil {
		// Utilisateur déjà existant
		return "", fmt.Errorf("un utilisateur avec cet email existe déjà")
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		// Erreur inattendue
		return "", err
	}
	// Créer l'utilisateur
	result := s.gormDB.WithContext(ctx).Create(user)
	if result.Error != nil {
		return "", result.Error
	}
	return user.ID, nil
}

// GetUserByEmail récupère un utilisateur par son adresse email.
func (s *Store) GetUserByEmail(ctx context.Context, email string) (*models.Utilisateur, error) {
	var user models.Utilisateur
	result := s.gormDB.WithContext(ctx).Where("email = ?", email).First(&user)
	if result.Error != nil {
		return nil, result.Error
	}
	return &user, nil
}

// GetUserByID récupère un utilisateur par son ID.
func (s *Store) GetUserByID(ctx context.Context, id string) (*models.Utilisateur, error) {
	var user models.Utilisateur
	result := s.gormDB.WithContext(ctx).First(&user, "id = ?", id)
	if result.Error != nil {
		return nil, result.Error
	}
	return &user, nil
}

// ===================== BLACKLIST =====================

// CreateBlacklistEntry insère une nouvelle entité dans la blacklist.
func (s *Store) CreateBlacklistEntry(ctx context.Context, entry *models.EntiteBlacklist) (*models.EntiteBlacklist, error) {
	if entry.DateAjout.IsZero() {
		entry.DateAjout = time.Now()
	}
	result := s.gormDB.WithContext(ctx).Create(entry)
	if result.Error != nil {
		return nil, result.Error
	}
	return entry, nil
}

// GetBlacklistEntries récupère toutes les entités de la blacklist.
func (s *Store) GetBlacklistEntries(ctx context.Context) ([]models.EntiteBlacklist, error) {
	var entries []models.EntiteBlacklist
	result := s.gormDB.WithContext(ctx).Find(&entries)
	if result.Error != nil {
		return nil, result.Error
	}
	return entries, nil
}

// UpdateBlacklistEntry met à jour une entité existante dans la blacklist.
func (s *Store) UpdateBlacklistEntry(ctx context.Context, id string, entry *models.EntiteBlacklist) (*models.EntiteBlacklist, error) {
	// GORM : mise à jour par ID
	entry.ID = id
	result := s.gormDB.WithContext(ctx).Model(&models.EntiteBlacklist{}).Where("id = ?", id).
		Updates(map[string]interface{}{
			"nom":                          entry.Nom,
			"type_entite":                  entry.TypeEntite,
			"informations_supplementaires": entry.InformationsSupplementaires,
		})
	if result.Error != nil {
		return nil, result.Error
	}
	// Recharger l'entité mise à jour pour obtenir la dateAjout
	var updated models.EntiteBlacklist
	if err := s.gormDB.WithContext(ctx).First(&updated, "id = ?", id).Error; err != nil {
		return nil, err
	}
	*entry = updated
	return entry, nil
}

// DeleteBlacklistEntry supprime une entité de la blacklist par son ID.
func (s *Store) DeleteBlacklistEntry(ctx context.Context, id string) error {
	result := s.gormDB.WithContext(ctx).Delete(&models.EntiteBlacklist{}, "id = ?", id)
	return result.Error
}

// GetBlacklistStatusForCompte vérifie si un compte est dans la blacklist et retourne le type d'entité s'il y est.
func (s *Store) GetBlacklistStatusForCompte(ctx context.Context, compteID string) (string, error) {

	var entite models.EntiteBlacklist
	result := s.gormDB.WithContext(ctx).Where("numero_compte = ?", compteID).First(&entite)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return "", nil
		}
		return "", result.Error
	}
	return entite.TypeEntite, nil
}

// ===================== PROFILS CLIENTS =====================

// GetProfilClientIDByCode retourne l'ID du profil client à partir du code type client.
func (s *Store) GetProfilClientIDByCode(ctx context.Context, code string) (string, error) {
	var nomProfil string
	switch code {
	case "P":
		nomProfil = "Particulier"
	case "M":
		nomProfil = "Entreprise"
	case "C":
		nomProfil = "Commerçant"
	case "B":
		nomProfil = "Business"
	default:
		nomProfil = "Particulier"
	}
	var profil models.ProfilClient
	result := s.gormDB.WithContext(ctx).Where("nom_profil = ?", nomProfil).First(&profil)
	if result.Error != nil {
		return "", result.Error
	}
	return profil.ID, nil
}

// CreateProfilClient insère un nouveau profil client dans la base de données.
func (s *Store) CreateProfilClient(ctx context.Context, profil *models.ProfilClient) (*models.ProfilClient, error) {
	result := s.gormDB.WithContext(ctx).Create(profil)
	if result.Error != nil {
		return nil, result.Error
	}
	return profil, nil
}

// GetProfilsClient récupère tous les profils clients.
func (s *Store) GetProfilsClient(ctx context.Context) ([]models.ProfilClient, error) {
	var profils []models.ProfilClient
	result := s.gormDB.WithContext(ctx).Find(&profils)
	if result.Error != nil {
		return nil, result.Error
	}
	return profils, nil
}

// UpdateProfilClient met à jour un profil client existant.
func (s *Store) UpdateProfilClient(ctx context.Context, id string, profil *models.ProfilClient) (*models.ProfilClient, error) {
	profil.ID = id
	result := s.gormDB.WithContext(ctx).Model(&models.ProfilClient{}).Where("id = ?", id).
		Updates(map[string]interface{}{
			"nom_profil":  profil.NomProfil,
			"description": profil.Description,
		})
	if result.Error != nil {
		return nil, result.Error
	}
	// Recharger le profil mis à jour
	var updated models.ProfilClient
	if err := s.gormDB.WithContext(ctx).First(&updated, "id = ?", id).Error; err != nil {
		return nil, err
	}
	*profil = updated
	return profil, nil
}

// DeleteProfilClient supprime un profil client par son ID.
func (s *Store) DeleteProfilClient(ctx context.Context, id string) error {
	result := s.gormDB.WithContext(ctx).Delete(&models.ProfilClient{}, "id = ?", id)
	return result.Error
}

// ===================== REGLES DE CONTROLE =====================
// --- Méthodes pour Règles de Contrôle ---
// Ces méthodes permettent de gérer les règles de contrôle associées aux profils clients.
// CreateRegleControle insère une nouvelle règle de contrôle dans la base de données.
func (s *Store) CreateRegleControle(ctx context.Context, regle *models.RegleControle) (*models.RegleControle, error) {
	// --- GORM ---
	result := s.gormDB.WithContext(ctx).Create(regle)
	if result.Error != nil {
		return nil, result.Error
	}
	return regle, nil
}

// GetReglesByProfilID récupère toutes les règles associées à un profil client donné.
func (s *Store) GetReglesByProfilID(ctx context.Context, profilID string) ([]models.RegleControle, error) {
	var regles []models.RegleControle
	result := s.gormDB.WithContext(ctx).Where("profil_client_id = ?", profilID).Find(&regles)
	if result.Error != nil {
		return nil, result.Error
	}
	return regles, nil
}

// UpdateRegleControle met à jour une règle de contrôle existante.
func (s *Store) UpdateRegleControle(ctx context.Context, id string, regle *models.RegleControle) (*models.RegleControle, error) {
	regle.ID = id
	result := s.gormDB.WithContext(ctx).Model(&models.RegleControle{}).Where("id = ?", id).
		Updates(map[string]interface{}{
			"nom_regle":         regle.NomRegle,
			"description":       regle.Description,
			"type_regle":        regle.TypeRegle,
			"criticite":         regle.Criticite,
			"seuil":             regle.Seuil,
			"periode_reference": regle.PeriodeReference,
			"active":            regle.Active,
			"profil_client_id":  regle.ProfilClientID,
			"liste_pays":        regle.ListePays,
		})
	if result.Error != nil {
		return nil, result.Error
	}
	// Recharger la règle mise à jour
	var updated models.RegleControle
	if err := s.gormDB.WithContext(ctx).First(&updated, "id = ?", id).Error; err != nil {
		return nil, err
	}
	*regle = updated
	return regle, nil
}

// DeleteRegleControle supprime une règle de contrôle par son ID.
func (s *Store) DeleteRegleControle(ctx context.Context, id string) error {
	result := s.gormDB.WithContext(ctx).Delete(&models.RegleControle{}, "id = ?", id)
	return result.Error
}

// Méthode pour récupérer toutes les règles
func (s *Store) GetAllRegles(ctx context.Context) ([]models.RegleControle, error) {
	var regles []models.RegleControle
	result := s.gormDB.WithContext(ctx).Find(&regles)
	if result.Error != nil {
		return nil, result.Error
	}
	return regles, nil
}

// GetRegleByType retourne la règle métier correspondant à un type donné (ex: PEP, Blacklist, etc.)
func (s *Store) GetRegleByType(ctx context.Context, typeRegle string) (*models.RegleControle, error) {
	var regle models.RegleControle
	result := s.gormDB.WithContext(ctx).Where("type_regle = ? AND active = ?", typeRegle, true).First(&regle)
	if result.Error != nil {
		return nil, result.Error
	}
	return &regle, nil
}

// ===================== COMPTES =====================
// CreateCompte insère un nouveau compte dans la base de données.
func (s *Store) CreateCompte(ctx context.Context, compte *models.Compte) (*models.Compte, error) {
	// Remplissage automatique du profil si vide
	if compte.ProfilClientID == "" && compte.TypeClient != "" {
		profilID, err := s.GetProfilClientIDByCode(ctx, compte.TypeClient)
		if err == nil && profilID != "" {
			compte.ProfilClientID = profilID
		}
	}
	// Initialisation automatique de la date de création si vide
	if compte.DateCreation.IsZero() {
		compte.DateCreation = time.Now()
	}
	// --- GORM ---
	result := s.gormDB.WithContext(ctx).Create(compte)
	if result.Error != nil {
		return nil, result.Error
	}
	return compte, nil
}

// GetComptByNumeroCompte recherche un client par son numéro de compte unique.
func (s *Store) GetCompteByCompte(ctx context.Context, numeroCompte string) (*models.Compte, error) {

	var compte models.Compte
	result := s.gormDB.WithContext(ctx).Where("numero_compte = ?", numeroCompte).First(&compte)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			// Aucun compte trouvé, retourne un compte vide
			return &models.Compte{}, nil
		}
		return nil, result.Error
	}
	return &compte, nil
}

// GetReglesForCompte récupère les règles actives associées au profil client d'un compte donné.
func (s *Store) GetReglesForCompte(ctx context.Context, numeroCompte string) ([]models.RegleControle, error) {
	// On récupère le profil client du compte
	var compte models.Compte
	if err := s.gormDB.WithContext(ctx).Where("numero_compte = ?", numeroCompte).First(&compte).Error; err != nil {
		return nil, err
	}
	var regles []models.RegleControle
	result := s.gormDB.WithContext(ctx).Where("profil_client_id = ? AND active = ?", compte.ProfilClientID, true).Find(&regles)
	if result.Error != nil {
		return nil, result.Error
	}
	return regles, nil
}

// ===================== TRANSACTIONS =====================
// CreateTransaction insère la nouvelle transaction riche dans la base de données.
func (s *Store) CreateTransaction(ctx context.Context, tx *models.Transaction) (*models.Transaction, error) {
	// Initialisation automatique de la date d'opération si vide
	if tx.DateOperation.IsZero() {
		tx.DateOperation = time.Now()
	}
	// --- GORM ---
	result := s.gormDB.WithContext(ctx).Create(tx)
	if result.Error != nil {
		return nil, result.Error
	}
	return tx, nil
}

// GetTransactions récupère toutes les transactions enregistrées dans la base.
func (s *Store) GetTransactions(ctx context.Context) ([]*models.Transaction, error) {
	var transactions []*models.Transaction
	result := s.gormDB.WithContext(ctx).Find(&transactions)
	if result.Error != nil {
		return nil, result.Error
	}
	return transactions, nil
}

// GetTransactionSumForPeriod calcule le montant total des transactions pour un numéro de compte avec GORM.
func (s *Store) GetTransactionSumForPeriod(ctx context.Context, numeroCompte string, jours int) (float64, error) {
	var total float64
	since := time.Now().AddDate(0, 0, -jours)
	result := s.gormDB.WithContext(ctx).
		Model(&models.Transaction{}).
		Where("(numero_compte_payeur = ? OR numero_compte_paye = ?) AND date_operation >= ?", numeroCompte, numeroCompte, since).
		Select("COALESCE(SUM(montant), 0)").
		Row()
	err := result.Scan(&total)
	return total, err
}

// GetTransactionCountForPeriod compte le nombre de transactions pour un numéro de compte avec GORM.
func (s *Store) GetTransactionCountForPeriod(ctx context.Context, numeroCompte string, jours int) (int64, error) {
	var count int64
	since := time.Now().AddDate(0, 0, -jours)
	result := s.gormDB.WithContext(ctx).
		Model(&models.Transaction{}).
		Where("(numero_compte_payeur = ? OR numero_compte_paye = ?) AND date_operation >= ?", numeroCompte, numeroCompte, since).
		Count(&count)
	return count, result.Error
}

// IsAccountOnBlacklist vérifie si un KYC ID est sur la blacklist.
func (s *Store) IsAccountOnBlacklist(ctx context.Context, kycID string) (string, error) {
	var entite models.EntiteBlacklist
	result := s.gormDB.WithContext(ctx).Where("numero_compte = ?", kycID).First(&entite)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return "", nil
		}
		return "", result.Error
	}
	return entite.TypeEntite, nil
}

// CreateSuiviBlacklist enregistre une trace lorsqu'une transaction implique une entité sous surveillance (GORM).
func (s *Store) CreateSuiviBlacklist(ctx context.Context, transactionID, numeroCompte, statutSuivi string) error {
	suivi := &models.SuiviBlacklist{
		TransactionID: transactionID,
		NumeroCompte:  numeroCompte,
		StatutSuivi:   statutSuivi,
		DateDetection: time.Now(),
	}
	result := s.gormDB.WithContext(ctx).Create(suivi)
	return result.Error
}

// ===================== PAYS A RISQUE =====================
// GetReglePaysRisqueID retourne l'UUID d'une règle pays_risque active.
func (s *Store) GetReglePaysRisqueID(ctx context.Context) (string, error) {
	var regle models.RegleControle
	result := s.gormDB.WithContext(ctx).Where("type_regle = ? AND active = ?", "pays_risque", true).First(&regle)
	if result.Error != nil {
		return "", result.Error
	}
	return regle.ID, nil
}

// ===================== SUIVI BLACKLIST =====================
// GetComptesSurveillesCount retourne le nombre de comptes uniques dans la table suivi_blacklist.
func (s *Store) GetComptesSurveillesCount(ctx context.Context) (int64, error) {
	var count int64
	err := s.gormDB.WithContext(ctx).
		Table("suivi_blacklist").
		Select("COUNT(DISTINCT numero_compte)").
		Scan(&count).Error
	return count, err
}

// GetComptesSurveilles retourne la liste des comptes uniques dans suivi_blacklist.
func (s *Store) GetComptesSurveilles(ctx context.Context) ([]string, error) {
	var comptes []string
	err := s.gormDB.WithContext(ctx).
		Table("suivi_blacklist").
		Distinct("numero_compte").
		Pluck("numero_compte", &comptes).Error
	return comptes, err
}

// GetReglePaysRisqueActive retourne la règle pays_risque active avec la liste des pays.
func (s *Store) GetReglePaysRisqueActive(ctx context.Context) (*models.RegleControle, error) {
	var regle models.RegleControle
	result := s.gormDB.WithContext(ctx).Where("type_regle = ? AND active = ?", "pays_risque", true).First(&regle)
	if result.Error != nil {
		return nil, result.Error
	}
	return &regle, nil
}

// AddPaysRisque ajoute un pays à la liste des pays à risque de la règle pays_risque.
func (s *Store) AddPaysRisque(ctx context.Context, pays string) error {
	regle, err := s.GetReglePaysRisqueActive(ctx)
	if err != nil {
		// Si la règle n'existe pas, on la crée
		paysList := []string{pays}
		jsonPays, err := json.Marshal(paysList)
		if err != nil {
			return err
		}
		regle = &models.RegleControle{
			NomRegle:         "Pays à risque",
			Description:      "Liste des pays à haut risque",
			TypeRegle:        "pays_risque",
			Criticite:        "Élevée",
			Seuil:            0,
			PeriodeReference: "",
			Active:           true,
			ProfilClientID:   nil,
			ListePays:        jsonPays,
		}
		return s.gormDB.WithContext(ctx).Create(regle).Error
	}
	// Unmarshal la liste existante
	var paysList []string
	if len(regle.ListePays) > 0 {
		if err := json.Unmarshal(regle.ListePays, &paysList); err != nil {
			return err
		}
	}
	// Vérifie si le pays existe déjà (insensible à la casse)
	for _, p := range paysList {
		if strings.EqualFold(p, pays) {
			return nil
		}
	}
	paysList = append(paysList, pays)
	jsonPays, err := json.Marshal(paysList)
	if err != nil {
		return err
	}
	return s.gormDB.WithContext(ctx).Model(&models.RegleControle{}).
		Where("id = ?", regle.ID).
		Update("liste_pays", jsonPays).Error
}

// RemovePaysRisque retire un pays de la liste des pays à risque de la règle pays_risque.
func (s *Store) RemovePaysRisque(ctx context.Context, pays string) error {
	regle, err := s.GetReglePaysRisqueActive(ctx)
	if err != nil {
		return err
	}
	var paysList []string
	if len(regle.ListePays) > 0 {
		if err := json.Unmarshal(regle.ListePays, &paysList); err != nil {
			return err
		}
	}
	var nouvelleListe []string
	for _, p := range paysList {
		if !strings.EqualFold(p, pays) {
			nouvelleListe = append(nouvelleListe, p)
		}
	}
	jsonPays, err := json.Marshal(nouvelleListe)
	if err != nil {
		return err
	}
	return s.gormDB.WithContext(ctx).Model(&models.RegleControle{}).
		Where("id = ?", regle.ID).
		Update("liste_pays", jsonPays).Error
}

// UpdatePaysRisque met à jour la liste complète des pays à risque de la règle universelle.
func (s *Store) UpdatePaysRisque(ctx context.Context, pays []string) error {
	regle, err := s.GetReglePaysRisqueActive(ctx)
	if err != nil {
		return err
	}
	jsonPays, err := json.Marshal(pays)
	if err != nil {
		return err
	}
	return s.gormDB.WithContext(ctx).Model(&models.RegleControle{}).
		Where("id = ?", regle.ID).
		Update("liste_pays", jsonPays).Error
}

// IsPaysARisque vérifie si un pays est dans la table des pays à risque.
func (s *Store) IsPaysARisque(ctx context.Context, pays string) (bool, error) {
	var paysARisque models.PaysARisque
	result := s.gormDB.WithContext(ctx).Where("LOWER(nom_pays) = ?", strings.ToLower(pays)).First(&paysARisque)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return false, nil
		}
		return false, result.Error
	}
	return true, nil
}

// ===================== ALERTES =====================
// GetAlertes récupère toutes les alertes du système.
func (s *Store) GetAlertes(ctx context.Context) ([]models.Alerte, error) {
	var alertes []models.Alerte
	result := s.gormDB.WithContext(ctx).Order("date_generation DESC").Find(&alertes)
	if result.Error != nil {
		return nil, result.Error
	}
	return alertes, nil
}

// GetAlertCountsByMonthAndType retourne le nombre d'alertes par type pour les derniers 12 mois.
// Retourne une map: type -> [{year: YYYY, month: M, count: N}, ...]
func (s *Store) GetAlertCountsByMonthAndType(ctx context.Context, months int) (map[string][]map[string]int64, error) {
	// On utilise une requête SQL brute pour grouper par type, année et mois
	type Row struct {
		Type  string
		Year  int
		Month int
		Count int64
	}
	rows := []Row{}
	// PostgreSQL: extract year and month from date_generation
	// query variable not needed; we'll use Raw with a since parameter below
	// GORM doesn't support interval with parameter easily, so compute the since time in Go
	since := time.Now().AddDate(0, -months, 0)
	result := s.gormDB.WithContext(ctx).Raw(`SELECT type, EXTRACT(YEAR FROM date_generation)::int AS year, EXTRACT(MONTH FROM date_generation)::int AS month, COUNT(*) as count
			  FROM alertes
			  WHERE date_generation >= ?
			  GROUP BY type, year, month
			  ORDER BY type, year, month`, since).Scan(&rows)
	if result.Error != nil {
		return nil, result.Error
	}
	out := make(map[string][]map[string]int64)
	for _, r := range rows {
		key := r.Type
		out[key] = append(out[key], map[string]int64{"year": int64(r.Year), "month": int64(r.Month), "count": r.Count})
	}
	return out, nil
}

// CreateAlerte insère une nouvelle alerte dans la base de données.
func (s *Store) CreateAlerte(ctx context.Context, alerte *models.Alerte) (*models.Alerte, error) {
	result := s.gormDB.WithContext(ctx).Create(alerte)
	if result.Error != nil {
		return nil, result.Error
	}
	return alerte, nil
}

// UpdateAlerteStatus met à jour le statut d'une alerte et lui assigne un analyste.
func (s *Store) UpdateAlerteStatus(ctx context.Context, alerteID string, analysteID string, newStatus string) (*models.Alerte, error) {
	var alerte models.Alerte
	// Met à jour le statut et l'analyste assigné
	result := s.gormDB.WithContext(ctx).Model(&models.Alerte{}).Where("id = ?", alerteID).
		Updates(map[string]interface{}{
			"statut":              newStatus,
			"analyste_assigne_id": analysteID,
		})
	if result.Error != nil {
		return nil, result.Error
	}
	// Recharge l'alerte mise à jour
	if err := s.gormDB.WithContext(ctx).First(&alerte, "id = ?", alerteID).Error; err != nil {
		return nil, err
	}
	return &alerte, nil
}

// GetAlertMetaByType récupère la meta-donnée d'alerte pour un type donné (ex: Interdits, Blacklist, PEP, Surveillés)
func (s *Store) GetAlertMetaByType(ctx context.Context, alertType string) (*models.AlertMeta, error) {
	normalizedType := strings.TrimSpace(alertType)
	var meta models.AlertMeta
	result := s.gormDB.WithContext(ctx).Table("alertes_meta").Where("type ILIKE ?", normalizedType).Limit(1).First(&meta)
	if result.Error != nil {
		return nil, result.Error
	}
	return &meta, nil
}

// GetAlertesBloquerActivesByCompte retourne les alertes actives (statut 'En cours') de type 'bloquer' pour un compte donné, avec filtrage optionnel par code.
func (s *Store) GetAlertesBloquerActivesByCompte(ctx context.Context, numeroCompte string, code ...string) ([]models.Alerte, error) {
	var alertes []models.Alerte
	// Récupérer les IDs des transactions liées à ce compte
	var transactionIDs []string
	var transactions []models.Transaction
	if err := s.gormDB.WithContext(ctx).
		Where("numero_compte_payeur = ? OR numero_compte_paye = ?", numeroCompte, numeroCompte).
		Find(&transactions).Error; err != nil {
		return nil, err
	}
	for _, tx := range transactions {
		transactionIDs = append(transactionIDs, tx.ID)
	}
	if len(transactionIDs) == 0 {
		return alertes, nil
	}
	query := s.gormDB.WithContext(ctx).
		Where("action = ? AND statut = ? AND transaction_id IN ?", "bloquer", "En cours", transactionIDs)
	if len(code) > 0 && code[0] != "" {
		query = query.Where("code = ?", code[0])
	}
	if err := query.Order("date_generation DESC").Find(&alertes).Error; err != nil {
		return nil, err
	}
	return alertes, nil
}

// ===================== DASHBOARD SUMMARY (Optimisé) =====================
// GetDashboardSummary retourne un résumé complet du dashboard en UNE SEULE requête.
func (s *Store) GetDashboardSummary(ctx context.Context) (map[string]interface{}, error) {
	summary := make(map[string]interface{})

	// 1. Compter les transactions totales
	var transactionCount int64
	if err := s.gormDB.WithContext(ctx).Model(&models.Transaction{}).Count(&transactionCount).Error; err != nil {
		return nil, err
	}
	summary["transactions_count"] = transactionCount

	// 2. Compter les alertes totales
	var alertCount int64
	if err := s.gormDB.WithContext(ctx).Model(&models.Alerte{}).Count(&alertCount).Error; err != nil {
		return nil, err
	}
	summary["anomalies_count"] = alertCount

	// 3. Compter les alertes traitées
	var treatedCount int64
	if err := s.gormDB.WithContext(ctx).Model(&models.Alerte{}).
		Where("statut ILIKE ?", "%trait%").
		Count(&treatedCount).Error; err != nil {
		return nil, err
	}
	summary["anomalies_treated"] = treatedCount

	// 4. Compter les comptes surveillés uniques
	monitoredCount, err := s.GetComptesSurveillesCount(ctx)
	if err != nil {
		return nil, err
	}
	summary["monitored_accounts"] = monitoredCount

	// 5. Métriques par type d'anomalie (derniers 12 mois)
	metrics, err := s.GetAlertCountsByMonthAndType(ctx, 12)
	if err != nil {
		return nil, err
	}
	summary["metrics"] = metrics

	// 6. Dernières 10 transactions (pour le tableau)
	var recentTransactions []models.Transaction
	if err := s.gormDB.WithContext(ctx).
		Order("date_operation DESC").
		Limit(10).
		Find(&recentTransactions).Error; err != nil {
		return nil, err
	}
	summary["recent_transactions"] = recentTransactions

	// 7. Toutes les alertes (pour l'affichage)
	var recentAnomalies []models.Alerte
	if err := s.gormDB.WithContext(ctx).
		Order("date_generation DESC").
		Find(&recentAnomalies).Error; err != nil {
		return nil, err
	}
	summary["recent_anomalies"] = recentAnomalies

	// 8. Comptages par type d'anomalie
	typeCountMap := make(map[string]int64)
	var types []struct {
		Type  string
		Count int64
	}
	if err := s.gormDB.WithContext(ctx).
		Model(&models.Alerte{}).
		Select("type, COUNT(*) as count").
		Group("type").
		Scan(&types).Error; err != nil {
		return nil, err
	}
	for _, t := range types {
		typeCountMap[t.Type] = t.Count
	}
	summary["anomalies_by_type"] = typeCountMap

	return summary, nil
}
