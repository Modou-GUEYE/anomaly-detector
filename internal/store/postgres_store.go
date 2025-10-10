package store

import (
	"anomaly-detector/internal/models"
	"context"
	"database/sql"
	"encoding/json"
	"log"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Store contient le pool de connexions à la base de données.
type Store struct {
	db *pgxpool.Pool
}

// DB expose le pool de connexions pour des opérations directes si nécessaire.
func (s *Store) DB() *pgxpool.Pool {
	return s.db
}

// NewStore crée et retourne une nouvelle instance de notre Store.
func NewStore(connStr string) (*Store, error) {
	pool, err := pgxpool.New(context.Background(), connStr)
	if err != nil {
		return nil, err
	}
	if err := pool.Ping(context.Background()); err != nil {
		return nil, err
	}
	log.Println("✅ Connexion à la base de données via le pool réussie!")
	return &Store{db: pool}, nil
}

// Close ferme le pool de connexions.
func (s *Store) Close() {
	s.db.Close()
}

// CreateUser insère un nouvel utilisateur dans la base de données.
func (s *Store) CreateUser(ctx context.Context, user *models.Utilisateur) (string, error) {
	var id string

	// NOTE : Le mot de passe doit être "hashé". Nous verrons la sécurité plus tard.
	query := `
        INSERT INTO utilisateurs (nom_utilisateur, email, mot_de_passe_hash, role, actif)
        VALUES ($1, $2, $3, $4, $5)
        RETURNING id
    `

	err := s.db.QueryRow(ctx, query, user.NomUtilisateur, user.Email, user.MotDePasseHash, user.Role, user.Actif).Scan(&id)
	if err != nil {
		return "", err
	}

	return id, nil
}

// GetUserByEmail récupère un utilisateur par son adresse email.
// C'est la première étape essentielle du processus de connexion.
func (s *Store) GetUserByEmail(ctx context.Context, email string) (*models.Utilisateur, error) {
	var user models.Utilisateur
	query := `SELECT id, nom_utilisateur, email, mot_de_passe_hash, role, actif, date_creation FROM utilisateurs WHERE email = $1`

	// QueryRow est utilisé car nous nous attendons à au plus une ligne en retour.
	err := s.db.QueryRow(ctx, query, email).Scan(
		&user.ID,
		&user.NomUtilisateur,
		&user.Email,
		&user.MotDePasseHash,
		&user.Role,
		&user.Actif,
		&user.DateCreation,
	)
	if err != nil {
		// Si aucune ligne n'est trouvée, pgx renverra une erreur `pgx.ErrNoRows`.
		// Notre handler d'API interprétera cela comme "utilisateur non trouvé".
		return nil, err
	}

	return &user, nil
}

// GetUserByID récupère un utilisateur par son ID.
func (s *Store) GetUserByID(ctx context.Context, id string) (*models.Utilisateur, error) {
	var user models.Utilisateur
	query := `SELECT id, nom_utilisateur, email, role, actif, date_creation FROM utilisateurs WHERE id = $1`

	err := s.db.QueryRow(ctx, query, id).Scan(
		&user.ID,
		&user.NomUtilisateur,
		&user.Email,
		&user.Role,
		&user.Actif,
		&user.DateCreation,
	)
	if err != nil {
		return nil, err
	}

	return &user, nil
}

// CreateBlacklistEntry insère une nouvelle entité dans la blacklist.
func (s *Store) CreateBlacklistEntry(ctx context.Context, entry *models.EntiteBlacklist) (*models.EntiteBlacklist, error) {
	query := `
			INSERT INTO entites_blacklist (nom, numero_compte, type_entite, informations_supplementaires)
			VALUES ($1, $2, $3, $4)
			RETURNING id, nom, numero_compte, type_entite, informations_supplementaires, date_ajout
		`
	err := s.db.QueryRow(ctx, query, entry.Nom, entry.NumeroCompte, entry.TypeEntite, entry.InformationsSupplementaires).Scan(
		&entry.ID,
		&entry.Nom,
		&entry.NumeroCompte,
		&entry.TypeEntite,
		&entry.InformationsSupplementaires,
		&entry.DateAjout,
	)
	if err != nil {
		return nil, err
	}
	return entry, nil
}

// GetBlacklistEntries récupère toutes les entités de la blacklist.
func (s *Store) GetBlacklistEntries(ctx context.Context) ([]models.EntiteBlacklist, error) {
	var entries []models.EntiteBlacklist
	query := `SELECT id, nom, numero_compte, type_entite, informations_supplementaires, date_ajout FROM entites_blacklist`

	rows, err := s.db.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var entry models.EntiteBlacklist
		if err := rows.Scan(&entry.ID, &entry.Nom, &entry.NumeroCompte, &entry.TypeEntite, &entry.InformationsSupplementaires, &entry.DateAjout); err != nil {
			return nil, err
		}
		entries = append(entries, entry)
	}

	return entries, nil
}

// UpdateBlacklistEntry met à jour une entité existante dans la blacklist.
func (s *Store) UpdateBlacklistEntry(ctx context.Context, id string, entry *models.EntiteBlacklist) (*models.EntiteBlacklist, error) {
	query := `
        UPDATE entites_blacklist
        SET nom = $1, type_entite = $2, informations_supplementaires = $3
        WHERE id = $4
        RETURNING id, nom, type_entite, informations_supplementaires, date_ajout
    `
	err := s.db.QueryRow(ctx, query, entry.Nom, entry.TypeEntite, entry.InformationsSupplementaires, id).Scan(
		&entry.ID,
		&entry.Nom,
		&entry.TypeEntite,
		&entry.InformationsSupplementaires,
		&entry.DateAjout,
	)
	if err != nil {
		return nil, err
	}
	// On s'assure que l'ID retourné est bien celui qu'on a mis à jour.
	entry.ID = id
	return entry, nil
}

// DeleteBlacklistEntry supprime une entité de la blacklist par son ID.
func (s *Store) DeleteBlacklistEntry(ctx context.Context, id string) error {
	query := `DELETE FROM entites_blacklist WHERE id = $1`
	_, err := s.db.Exec(ctx, query, id)
	return err
}

// GetBlacklistStatusForClient vérifie si un client est dans la blacklist et retourne le type d'entité s'il y est.
func (s *Store) GetBlacklistStatusForClient(ctx context.Context, clientID string) (string, error) {
	var typeEntite string
	query := `
		SELECT eb.type_entite FROM entites_blacklist eb
		JOIN clients c ON eb.numero_compte = c.numero_compte
		WHERE c.numero_compte = $1
	`
	err := s.db.QueryRow(ctx, query, clientID).Scan(&typeEntite)
	if err != nil {
		if err == pgx.ErrNoRows {
			return "", nil
		}
		return "", err
	}
	return typeEntite, nil
}

// --- Méthodes pour Profils Clients ---
// Ces méthodes permettent de gérer les profils clients et leurs règles associées.
// CreateProfilClient insère un nouveau profil client dans la base de données.
func (s *Store) CreateProfilClient(ctx context.Context, profil *models.ProfilClient) (*models.ProfilClient, error) {
	query := `INSERT INTO profils_clients (nom_profil, description) VALUES ($1, $2) RETURNING id, nom_profil, description`
	err := s.db.QueryRow(ctx, query, profil.NomProfil, profil.Description).Scan(&profil.ID, &profil.NomProfil, &profil.Description)
	return profil, err
}

// GetProfilsClient récupère tous les profils clients.
func (s *Store) GetProfilsClient(ctx context.Context) ([]models.ProfilClient, error) {
	var profils []models.ProfilClient
	query := `SELECT id, nom_profil, description FROM profils_clients`
	rows, err := s.db.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var profil models.ProfilClient
		if err := rows.Scan(&profil.ID, &profil.NomProfil, &profil.Description); err != nil {
			return nil, err
		}
		profils = append(profils, profil)
	}
	return profils, nil
}

// UpdateProfilClient met à jour un profil client existant.
func (s *Store) UpdateProfilClient(ctx context.Context, id string, profil *models.ProfilClient) (*models.ProfilClient, error) {
	query := `UPDATE profils_clients SET nom_profil = $1, description = $2 WHERE id = $3 RETURNING id, nom_profil, description`
	err := s.db.QueryRow(ctx, query, profil.NomProfil, profil.Description, id).Scan(&profil.ID, &profil.NomProfil, &profil.Description)
	profil.ID = id
	return profil, err
}

// DeleteProfilClient supprime un profil client par son ID.
func (s *Store) DeleteProfilClient(ctx context.Context, id string) error {
	query := `DELETE FROM profils_clients WHERE id = $1`
	_, err := s.db.Exec(ctx, query, id)
	return err
}

// --- Méthodes pour Règles de Contrôle ---
// Ces méthodes permettent de gérer les règles de contrôle associées aux profils clients.
// CreateRegleControle insère une nouvelle règle de contrôle dans la base de données.
func (s *Store) CreateRegleControle(ctx context.Context, regle *models.RegleControle) (*models.RegleControle, error) {
	query := `
        INSERT INTO regles_controle (nom_regle, description, type_regle, criticite, seuil, periode_reference, active, profil_client_id)
        VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
        RETURNING id
    `
	// On ne scanne que l'ID que la base de données nous retourne.
	err := s.db.QueryRow(ctx, query, regle.NomRegle, regle.Description, regle.TypeRegle, regle.Criticite, regle.Seuil, regle.PeriodeReference, regle.Active, regle.ProfilClientID).Scan(&regle.ID)

	// La fonction retourne l'objet 'regle' complet (avec l'ID maintenant rempli) et l'erreur.
	return regle, err
}

// GetReglesByProfilID récupère toutes les règles associées à un profil client donné.
func (s *Store) GetReglesByProfilID(ctx context.Context, profilID string) ([]models.RegleControle, error) {
	var regles []models.RegleControle
	query := `
		SELECT id, nom_regle, description, type_regle, criticite, seuil, periode_reference, active, profil_client_id
		FROM regles_controle
		WHERE profil_client_id = $1
	`
	rows, err := s.db.Query(ctx, query, profilID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var regle models.RegleControle
		if err := rows.Scan(
			&regle.ID,
			&regle.NomRegle,
			&regle.Description,
			&regle.TypeRegle, // <-- AJOUTÉ
			&regle.Criticite,
			&regle.Seuil,
			&regle.PeriodeReference,
			&regle.Active,
			&regle.ProfilClientID,
		); err != nil {
			return nil, err
		}
		regles = append(regles, regle)
	}
	return regles, nil
}

// UpdateRegleControle met à jour une règle de contrôle existante.
func (s *Store) UpdateRegleControle(ctx context.Context, id string, regle *models.RegleControle) (*models.RegleControle, error) {
	query := `
        UPDATE regles_controle
        SET nom_regle = $1, description = $2, type_regle = $3, criticite = $4, seuil = $5, periode_reference = $6, active = $7, profil_client_id = $8
        WHERE id = $9 RETURNING id`

	err := s.db.QueryRow(ctx, query, regle.NomRegle, regle.Description, regle.TypeRegle, regle.Criticite, regle.Seuil, regle.PeriodeReference, regle.Active, regle.ProfilClientID, id).Scan(&regle.ID)

	return regle, err
}

// DeleteRegleControle supprime une règle de contrôle par son ID.
func (s *Store) DeleteRegleControle(ctx context.Context, id string) error {
	query := `DELETE FROM regles_controle WHERE id = $1`
	_, err := s.db.Exec(ctx, query, id)
	return err
}

// CreateClient insère un nouveau client dans la base de données.
func (s *Store) CreateClient(ctx context.Context, client *models.Client) (*models.Client, error) {
	query := `INSERT INTO clients (
			numero_compte, type_client, profil_client_id, date_creation,
			nom_client, type_compte, categorie, pays, ville, adresse
		) VALUES (
			$1, $2, $3, $4,
			$5, $6, $7, $8, $9, $10
		) RETURNING id`
	var profilID interface{}
	if client.ProfilClientID == "" {
		profilID = nil
	} else {
		profilID = client.ProfilClientID
	}
	err := s.db.QueryRow(ctx, query,
		client.NumeroCompte, // numero_compte
		client.TypeClient,   // type_client
		profilID,
		client.DateCreation,
		client.NomClient,
		client.TypeCompte,
		client.Categorie,
		client.Pays,
		client.Ville,
		client.Adresse,
	).Scan(&client.ID)
	return client, err
}

// GetClientByNumeroCompte recherche un client par son numéro de compte unique.
func (s *Store) GetClientByNumeroCompte(ctx context.Context, numeroCompte string) (*models.Client, error) {
	var client models.Client
	query := `SELECT 
			id, numero_compte, type_client, profil_client_id, date_creation,
			nom_client, type_compte, categorie, pays, ville, adresse
			FROM clients WHERE numero_compte = $1`
	err := s.db.QueryRow(ctx, query, numeroCompte).Scan(
		&client.ID,
		&client.NumeroCompte,
		&client.TypeClient,
		&client.ProfilClientID,
		&client.DateCreation,
		&client.NomClient,
		&client.TypeCompte,
		&client.Categorie,
		&client.Pays,
		&client.Ville,
		&client.Adresse,
	)
	if err == pgx.ErrNoRows {
		// Aucun client trouvé, retourne un client vide
		return &models.Client{}, nil
	}
	return &client, err
}

// CreateAlerte insère une nouvelle alerte dans la base de données.
func (s *Store) CreateAlerte(ctx context.Context, alerte *models.Alerte) (*models.Alerte, error) {
	query := `
		INSERT INTO alertes (transaction_id, regle_declenchee_id, niveau_criticite, score_risque, raison_detection, statut)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, date_generation
	`
	err := s.db.QueryRow(ctx, query, alerte.TransactionID, alerte.RegleDeclencheeID, alerte.NiveauCriticite, alerte.ScoreRisque, alerte.RaisonDetection, "En cours").Scan(&alerte.ID, &alerte.DateGeneration)
	return alerte, err
}

// GetReglesForClient récupère les règles actives associées au profil client d'un client donné.
func (s *Store) GetReglesForClient(ctx context.Context, clientID string) ([]models.RegleControle, error) {
	query := `
		SELECT rc.id, rc.nom_regle, rc.description, rc.type_regle,
			   rc.criticite, rc.seuil, rc.periode_reference, rc.active, rc.profil_client_id 
		FROM regles_controle rc
		JOIN clients c ON rc.profil_client_id = c.profil_client_id
		WHERE c.numero_compte = $1 AND rc.active = true
	`
	var regles []models.RegleControle
	rows, err := s.db.Query(ctx, query, clientID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var regle models.RegleControle
		if err := rows.Scan(
			&regle.ID,
			&regle.NomRegle,
			&regle.Description,
			&regle.TypeRegle,
			&regle.Criticite,
			&regle.Seuil,
			&regle.PeriodeReference,
			&regle.Active,
			&regle.ProfilClientID,
		); err != nil {
			return nil, err
		}
		regles = append(regles, regle)
	}
	return regles, nil
}

// CreateTransaction insère la nouvelle transaction riche dans la base de données.
func (s *Store) CreateTransaction(ctx context.Context, tx *models.Transaction) (*models.Transaction, error) {
	rawData, _ := json.Marshal(tx)
	query := `
        INSERT INTO transactions (montant, portee, numero_compte_payeur, nom_payeur, pays_payeur, numero_compte_paye, nom_paye, pays_paye, raw_data) 
        VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9) 
        RETURNING id, date_operation
    `
	err := s.db.QueryRow(ctx, query, tx.Montant, tx.Portee, tx.NumeroComptePayeur, tx.NomPayeur, tx.PaysPayeur, tx.NumeroComptePaye, tx.NomPaye, tx.PaysPaye, rawData).Scan(&tx.ID, &tx.DateOperation)
	return tx, err
}

// GetTransactionSumForPeriod calcule le montant total des transactions pour un numéro de compte.
func (s *Store) GetTransactionSumForPeriod(ctx context.Context, numeroCompte string, jours int) (float64, error) {
	var total float64
	query := `
        SELECT COALESCE(SUM(montant), 0) FROM transactions
        WHERE (numero_compte_payeur = $1 OR numero_compte_paye = $1)
        AND date_operation >= NOW() - INTERVAL '1 day' * $2
    `
	err := s.db.QueryRow(ctx, query, numeroCompte, jours).Scan(&total)
	return total, err
}

// GetTransactionCountForPeriod compte le nombre de transactions pour un numéro de compte.
func (s *Store) GetTransactionCountForPeriod(ctx context.Context, numeroCompte string, jours int) (int64, error) {
	var count int64
	query := `
        SELECT COUNT(*) FROM transactions
        WHERE (numero_compte_payeur = $1 OR numero_compte_paye = $1)
        AND date_operation >= NOW() - INTERVAL '1 day' * $2
    `
	err := s.db.QueryRow(ctx, query, numeroCompte, jours).Scan(&count)
	return count, err
}

// GetReglesForNumeroCompte récupère les règles en se basant sur le numéro de compte (via le KYC ID).
func (s *Store) GetReglesForNumeroCompte(ctx context.Context, numeroCompte string) ([]models.RegleControle, error) {
	var regles []models.RegleControle
	query := `
        SELECT rc.id, rc.nom_regle, rc.description, rc.type_regle,
               rc.criticite, rc.seuil, rc.periode_reference, rc.active, rc.profil_client_id 
        FROM regles_controle rc
        JOIN clients c ON rc.profil_client_id = c.profil_client_id
	WHERE c.numero_compte = $1 AND rc.active = true
    `
	rows, err := s.db.Query(ctx, query, numeroCompte)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var regle models.RegleControle
		if err := rows.Scan(
			&regle.ID,
			&regle.NomRegle,
			&regle.Description,
			&regle.TypeRegle,
			&regle.Criticite,
			&regle.Seuil,
			&regle.PeriodeReference,
			&regle.Active,
			&regle.ProfilClientID,
		); err != nil {
			return nil, err
		}
		regles = append(regles, regle)
	}
	return regles, nil
}

// IsKycOnBlacklist vérifie si un KYC ID est sur la blacklist.
func (s *Store) IsKycOnBlacklist(ctx context.Context, kycID string) (string, error) {
	var typeEntite string
	query := `SELECT type_entite FROM entites_blacklist WHERE numero_compte = $1`
	err := s.db.QueryRow(ctx, query, kycID).Scan(&typeEntite)
	if err != nil {
		if err == pgx.ErrNoRows {
			return "", nil
		}
		return "", err
	}
	return typeEntite, nil
}

// CreateSuiviBlacklist enregistre une trace lorsqu'une transaction implique une entité sous surveillance.
func (s *Store) CreateSuiviBlacklist(ctx context.Context, transactionID, kycID, statut string) error {
	query := `
        INSERT INTO suivi_blacklist (transaction_id, kyc_id, statut_blacklist)
        VALUES ($1, $2, $3)
    `
	_, err := s.db.Exec(ctx, query, transactionID, kycID, statut)
	return err
}

// IsPaysARisque vérifie si un pays est dans la table des pays à risque.
func (s *Store) IsPaysARisque(ctx context.Context, pays string) (bool, error) {
	var exists bool
	query := `SELECT EXISTS(SELECT 1 FROM pays_a_risque WHERE nom_pays = $1)`
	err := s.db.QueryRow(ctx, query, pays).Scan(&exists)
	return exists, err
}

// GetAlertes récupère toutes les alertes du système.
func (s *Store) GetAlertes(ctx context.Context) ([]models.Alerte, error) {
	var alertes []models.Alerte
	query := `SELECT id, transaction_id, regle_declenchee_id, niveau_criticite, score_risque, raison_detection, statut, analyste_assigne_id, date_generation FROM alertes ORDER BY date_generation DESC`
	rows, err := s.db.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var alerte models.Alerte
		var analysteID sql.NullString
		if err := rows.Scan(&alerte.ID, &alerte.TransactionID, &alerte.RegleDeclencheeID, &alerte.NiveauCriticite, &alerte.ScoreRisque, &alerte.RaisonDetection, &alerte.Statut, &analysteID, &alerte.DateGeneration); err != nil {
			return nil, err
		}
		if analysteID.Valid {
			alerte.AnalysteAssignéID = analysteID.String
		}
		alertes = append(alertes, alerte)
	}
	return alertes, nil
}

// UpdateAlerteStatus met à jour le statut d'une alerte et lui assigne un analyste.
func (s *Store) UpdateAlerteStatus(ctx context.Context, alerteID string, analysteID string, newStatus string) (*models.Alerte, error) {
	var alerte models.Alerte
	query := `
        UPDATE alertes 
        SET statut = $1, analyste_assigne_id = $2 
        WHERE id = $3 
        RETURNING id, transaction_id, regle_declenchee_id, niveau_criticite, score_risque, raison_detection, statut, analyste_assigne_id, date_generation
    `
	var returnedAnalysteID sql.NullString
	err := s.db.QueryRow(ctx, query, newStatus, analysteID, alerteID).Scan(
		&alerte.ID, &alerte.TransactionID, &alerte.RegleDeclencheeID, &alerte.NiveauCriticite, &alerte.ScoreRisque, &alerte.RaisonDetection, &alerte.Statut, &returnedAnalysteID, &alerte.DateGeneration,
	)
	if returnedAnalysteID.Valid {
		alerte.AnalysteAssignéID = returnedAnalysteID.String
	}
	return &alerte, err
}
