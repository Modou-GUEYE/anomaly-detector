-- =============================================================================
-- Migration Indexation SGBD - Optimisation des performances P50 / Palier 100u
-- Module LAB Anomaly Detector
-- =============================================================================

-- 1. Table ALERTES (Chemin critique d'analyse et tableau de bord)
CREATE INDEX IF NOT EXISTS idx_alertes_transaction_id ON alertes(transaction_id);
CREATE INDEX IF NOT EXISTS idx_alertes_statut ON alertes(statut);
CREATE INDEX IF NOT EXISTS idx_alertes_code ON alertes(code);
CREATE INDEX IF NOT EXISTS idx_alertes_criticite ON alertes(niveau_criticite);
CREATE INDEX IF NOT EXISTS idx_alertes_date_generation ON alertes(date_generation DESC);
CREATE INDEX IF NOT EXISTS idx_alertes_compte_date ON alertes(numero_compte, date_generation DESC);
CREATE INDEX IF NOT EXISTS idx_alertes_type_date ON alertes(type, date_generation DESC);

-- 2. Table COMPTES (Jointures rapides profil / compte)
CREATE INDEX IF NOT EXISTS idx_comptes_profil_client_id ON comptes(profil_client_id);

-- 3. Table SUIVI_BLACKLIST (Accélération des enregistrements et recherches)
CREATE INDEX IF NOT EXISTS idx_suivi_blacklist_transaction_id ON suivi_blacklist(transaction_id);
CREATE INDEX IF NOT EXISTS idx_suivi_blacklist_numero_compte ON suivi_blacklist(numero_compte);

-- 4. Table LOGS_AUDIT (Audit trail non bloquant)
CREATE INDEX IF NOT EXISTS idx_logs_audit_date_heure ON logs_audit(date_heure DESC);
CREATE INDEX IF NOT EXISTS idx_logs_audit_utilisateur ON logs_audit(utilisateur_id);

-- 5. Table TRANSACTIONS (Vérification et renforcement des index temporels)
CREATE INDEX IF NOT EXISTS idx_transactions_date_operation ON transactions(date_operation DESC);
CREATE INDEX IF NOT EXISTS idx_transactions_payeur_date ON transactions(numero_compte_payeur, date_operation DESC);
CREATE INDEX IF NOT EXISTS idx_transactions_paye_date ON transactions(numero_compte_paye, date_operation DESC);

-- Mise à jour des statistiques de l'optimiseur PostgreSQL
ANALYZE alertes;
ANALYZE transactions;
ANALYZE comptes;
ANALYZE suivi_blacklist;
ANALYZE logs_audit;
