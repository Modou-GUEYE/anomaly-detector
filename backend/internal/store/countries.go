package store

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"gorm.io/gorm"

	"anomaly-detector/internal/models"
)

// ===================== GESTION PAYS À RISQUE =====================

// GetReglePaysRisqueActive retourne la règle pays_risque active avec la liste des pays.
func (s *Store) GetReglePaysRisqueActive(ctx context.Context) (*models.RegleControle, error) {
	var regle models.RegleControle
	result := s.gormDB.WithContext(ctx).Where("type_regle = ? AND active = ?", "pays_risque", true).First(&regle)
	if result.Error != nil {
		return nil, result.Error
	}
	return &regle, nil
}

// IsPaysARisque vérifie si un pays est dans la liste des pays à risque (de la règle "Pays à risque").
func (s *Store) IsPaysARisque(ctx context.Context, pays string) (bool, error) {
	var regle models.RegleControle
	result := s.gormDB.WithContext(ctx).Where("type_regle = ?", "pays_risque").First(&regle)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return false, nil
		}
		return false, result.Error
	}

	// Décoder la liste des pays depuis le JSON - gérer le double-encoding
	var listePays []string
	if len(regle.ListePays) > 0 {
		err := json.Unmarshal(regle.ListePays, &listePays)
		if err != nil {
			// Essayer de récupérer comme string puis re-unmarshal
			var jsonStr string
			if err2 := json.Unmarshal(regle.ListePays, &jsonStr); err2 == nil {
				err = json.Unmarshal([]byte(jsonStr), &listePays)
				if err != nil {
					return false, err
				}
			} else {
				return false, err
			}
		}
	}

	// Vérifier si le pays est dans la liste (case-insensitive)
	paysUpper := strings.ToUpper(pays)
	for _, p := range listePays {
		if strings.ToUpper(p) == paysUpper {
			return true, nil
		}
	}
	return false, nil
}

// GetAllPaysARisque récupère tous les pays à risque de la règle "Pays à risque".
func (s *Store) GetAllPaysARisque(ctx context.Context) ([]string, error) {
	var regle models.RegleControle
	result := s.gormDB.WithContext(ctx).Where("type_regle = ?", "pays_risque").First(&regle)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return []string{}, nil
		}
		return nil, result.Error
	}

	// Décoder la liste des pays depuis le JSON
	var listePays []string
	if len(regle.ListePays) > 0 {
		// Gerer le double-encoding: si c'est une chaîne entre guillemets, l'unmarshal d'abord en string
		err := json.Unmarshal(regle.ListePays, &listePays)
		if err != nil {
			// Essayer de récupérer comme string puis re-unmarshal
			var jsonStr string
			if err2 := json.Unmarshal(regle.ListePays, &jsonStr); err2 == nil {
				err = json.Unmarshal([]byte(jsonStr), &listePays)
				if err != nil {
					return nil, err
				}
			} else {
				return nil, err
			}
		}
	} else {
		listePays = []string{}
	}

	return listePays, nil
}

// AddPaysARisque ajoute un pays à la liste des pays à risque.
func (s *Store) AddPaysARisque(ctx context.Context, nomPays string, userID string) error {
	var regle models.RegleControle
	result := s.gormDB.WithContext(ctx).Where("type_regle = ?", "pays_risque").First(&regle)
	if result.Error != nil {
		return result.Error
	}

	// Décoder la liste actuelle - gérer le double-encoding
	var listePays []string
	if len(regle.ListePays) > 0 {
		err := json.Unmarshal(regle.ListePays, &listePays)
		if err != nil {
			// Essayer de récupérer comme string puis re-unmarshal
			var jsonStr string
			if err2 := json.Unmarshal(regle.ListePays, &jsonStr); err2 == nil {
				err = json.Unmarshal([]byte(jsonStr), &listePays)
				if err != nil {
					return err
				}
			} else {
				return err
			}
		}
	}

	// Vérifier si le pays existe déjà
	paysUpper := strings.ToUpper(nomPays)
	for _, p := range listePays {
		if strings.ToUpper(p) == paysUpper {
			return nil // Déjà présent
		}
	}

	// Ajouter le pays
	listePays = append(listePays, nomPays)

	// Encoder et mettre à jour
	newListeJSON, err := json.Marshal(listePays)
	if err != nil {
		return err
	}

	err = s.gormDB.WithContext(ctx).Model(&regle).Update("liste_pays", string(newListeJSON)).Error
	if err != nil {
		return err
	}

	// Logging d'audit automatique si userID fourni
	if userID != "" {
		actionDescription := fmt.Sprintf("Ajout du pays à risque '%s'", nomPays)
		auditErr := s.LogAction(ctx, userID, actionDescription, "Pays à Risque", regle.ID, fmt.Sprintf("Ajout du pays: %s", nomPays), "succès")
		if auditErr != nil {
			// Ne pas faire échouer l'ajout si le logging échoue, mais logger l'erreur
			fmt.Printf("Erreur lors du logging d'audit pour l'ajout du pays %s: %v\n", nomPays, auditErr)
		}
	}

	return nil
}

// DeletePaysARisque supprime un pays de la liste des pays à risque.
func (s *Store) DeletePaysARisque(ctx context.Context, nomPays string, userID string) error {
	var regle models.RegleControle
	result := s.gormDB.WithContext(ctx).Where("type_regle = ?", "pays_risque").First(&regle)
	if result.Error != nil {
		return result.Error
	}

	// Décoder la liste actuelle - gérer le double-encoding
	var listePays []string
	if len(regle.ListePays) > 0 {
		err := json.Unmarshal(regle.ListePays, &listePays)
		if err != nil {
			// Essayer de récupérer comme string puis re-unmarshal
			var jsonStr string
			if err2 := json.Unmarshal(regle.ListePays, &jsonStr); err2 == nil {
				err = json.Unmarshal([]byte(jsonStr), &listePays)
				if err != nil {
					return err
				}
			} else {
				return err
			}
		}
	}

	// Filtrer le pays à supprimer (case-insensitive)
	paysUpper := strings.ToUpper(nomPays)
	var updatedList []string
	for _, p := range listePays {
		if strings.ToUpper(p) != paysUpper {
			updatedList = append(updatedList, p)
		}
	}

	// Encoder et mettre à jour
	newListeJSON, err := json.Marshal(updatedList)
	if err != nil {
		return err
	}

	err = s.gormDB.WithContext(ctx).Model(&regle).Update("liste_pays", string(newListeJSON)).Error
	if err != nil {
		return err
	}

	// Logging d'audit automatique si userID fourni
	if userID != "" {
		actionDescription := fmt.Sprintf("Suppression du pays à risque '%s'", nomPays)
		auditErr := s.LogAction(ctx, userID, actionDescription, "Pays à Risque", regle.ID, fmt.Sprintf("Suppression du pays: %s", nomPays), "succès")
		if auditErr != nil {
			// Ne pas faire échouer la suppression si le logging échoue, mais logger l'erreur
			fmt.Printf("Erreur lors du logging d'audit pour la suppression du pays %s: %v\n", nomPays, auditErr)
		}
	}

	return nil
}

// UpdatePaysARisque met à jour le nom d'un pays dans la liste (renaming).
func (s *Store) UpdatePaysARisque(ctx context.Context, ancienNom, nouveauNom string, userID string) error {
	var regle models.RegleControle
	result := s.gormDB.WithContext(ctx).Where("type_regle = ?", "pays_risque").First(&regle)
	if result.Error != nil {
		return result.Error
	}

	// Décoder la liste actuelle - gérer le double-encoding
	var listePays []string
	if len(regle.ListePays) > 0 {
		err := json.Unmarshal(regle.ListePays, &listePays)
		if err != nil {
			// Essayer de récupérer comme string puis re-unmarshal
			var jsonStr string
			if err2 := json.Unmarshal(regle.ListePays, &jsonStr); err2 == nil {
				err = json.Unmarshal([]byte(jsonStr), &listePays)
				if err != nil {
					return err
				}
			} else {
				return err
			}
		}
	}

	// Remplacer l'ancien nom par le nouveau (case-insensitive)
	ancienUpper := strings.ToUpper(ancienNom)
	for i, p := range listePays {
		if strings.ToUpper(p) == ancienUpper {
			listePays[i] = nouveauNom
			break
		}
	}

	// Encoder et mettre à jour
	newListeJSON, err := json.Marshal(listePays)
	if err != nil {
		return err
	}

	err = s.gormDB.WithContext(ctx).Model(&regle).Update("liste_pays", string(newListeJSON)).Error
	if err != nil {
		return err
	}

	// Logging d'audit automatique si userID fourni
	if userID != "" {
		actionDescription := fmt.Sprintf("Modification du pays à risque '%s' en '%s'", ancienNom, nouveauNom)
		auditErr := s.LogAction(ctx, userID, actionDescription, "Pays à Risque", regle.ID, fmt.Sprintf("Modification du pays: %s → %s", ancienNom, nouveauNom), "succès")
		if auditErr != nil {
			// Ne pas faire échouer la modification si le logging échoue, mais logger l'erreur
			fmt.Printf("Erreur lors du logging d'audit pour la modification du pays %s: %v\n", ancienNom, auditErr)
		}
	}

	return nil
}
