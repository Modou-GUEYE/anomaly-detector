package core

import (
	"anomaly-detector/internal/store"
	"context"
)

// BlacklistChecker est responsable de la vérification des listes de surveillance.
type BlacklistChecker struct {
	store *store.Store
}

func NewBlacklistChecker(store *store.Store) *BlacklistChecker {
	return &BlacklistChecker{store: store}
}

// Check vérifie le statut d'un client en utilisant son identifiant métier (KYC ID / numéro de compte).
// Il renvoie le statut (ex: "Interdit") ou une chaîne vide si le client n'est pas sur une liste.
func (bc *BlacklistChecker) Check(ctx context.Context, kycID string) (string, error) {
	return bc.store.IsKycOnBlacklist(ctx, kycID)
}
