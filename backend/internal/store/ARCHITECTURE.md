# Architecture du Store - Documentation Technique

## Vue d'ensemble

Le système de store a été refactorisé d'un fichier monolithique de 1600+ lignes vers une architecture modulaire basée sur les domaines métier. Cette approche améliore la maintenabilité, la lisibilité et la collaboration en équipe.

## Principe d'organisation

### Architecture en couches
```
Store Interface (store.go)
    ↓
Domain Modules (users.go, accounts.go, etc.)
    ↓
Base Infrastructure (base.go)
    ↓
Database (PostgreSQL via GORM)
```

### Séparation des responsabilités
- **1 domaine métier = 1 fichier**
- **Interface centralisée** pour toutes les méthodes
- **Configuration isolée** dans le fichier de base
- **Implémentation distribuée** par expertise fonctionnelle

---

## Description détaillée des fichiers

### 📋 `store.go` - Interface Centrale
**Rôle** : Définit le contrat d'interface pour toutes les opérations du store

**Responsabilités** :
- Centralise les signatures de toutes les méthodes (60+ méthodes)
- Assure la cohérence des types de retour
- Facilite les tests unitaires et mocks
- Documentation des paramètres et comportements attendus

**Sections de l'interface** :
- Gestion des utilisateurs (7 méthodes)
- Gestion des profils clients (4 méthodes)
- Gestion des comptes (8 méthodes)
- Gestion des transactions (6 méthodes)
- Gestion de la liste noire (7 méthodes)
- Gestion des règles de contrôle (6 méthodes)
- Gestion des alertes (6 méthodes)
- Suivi et monitoring (4 méthodes)
- Gestion des pays à risque (4 méthodes)
- Métriques du tableau de bord (2 méthodes)
- Métriques d'efficacité (2 méthodes)
- Audit et conformité (6 méthodes)

---

### 🏗️ `base.go` - Infrastructure de Base
**Rôle** : Gère la configuration et la connexion à la base de données

**Responsabilités** :
- Configuration GORM avec logger silencieux
- Création et gestion de la connexion PostgreSQL
- Structure de base du Store
- Méthodes utilitaires (Close, Gorm)

**Éléments clés** :
```go
type Store struct {
    gormDB *gorm.DB // Connexion GORM principale
}

func NewStore(connStr string) (*Store, error) // Factory
func (s *Store) Close()                       // Cleanup
func (s *Store) Gorm() interface{}           // Accès direct DB
```

---

### 👤 `users.go` - Gestion des Utilisateurs
**Rôle** : Opérations CRUD sur les utilisateurs du système

**Fonctionnalités** :
- **Création** : `CreateUser` - Nouvel utilisateur avec validation
- **Authentification** : `GetUserByEmail` - Login et vérification
- **Recherche** : `GetUserByID` - Récupération par identifiant
- **Listing** : `GetAllUsers` - Liste complète des utilisateurs
- **Mise à jour** : `UpdateUser` - Modification des données
- **Suppression** : `DeleteUser` - Suppression logique/physique
- **Validation** : `ValidateUserCredentials` - Vérification des droits

**Types d'utilisateurs supportés** :
- Administrateurs (admin)
- Analystes (analyst)
- Utilisateurs système

---

### 🏢 `profiles.go` - Profils Clients
**Rôle** : Gestion des profils et types de clients

**Fonctionnalités** :
- **Récupération** : `GetProfilClientIDByCode` - ID par code profil
- **Création** : `CreateProfilClient` - Nouveau type de profil
- **Listing** : `GetAllProfilsClient` - Tous les profils disponibles
- **Recherche** : `GetProfilClientByID` - Profil par identifiant

**Cas d'usage** :
- Classification des clients (particulier, entreprise, association)
- Règles métier spécifiques par type
- Reporting et statistiques par segment

---

### 💳 `accounts.go` - Gestion des Comptes
**Rôle** : Opérations sur les comptes bancaires et entités surveillées

**Fonctionnalités principales** :
- **Création** : `CreateCompte` - Nouveau compte avec audit
- **Consultation** : `GetAllComptes`, `GetCompteByID` - Recherche et listing
- **Mise à jour** : `UpdateCompte` - Modification des informations
- **Suppression** : `DeleteCompte` - Suppression avec traçabilité
- **Surveillance** : `GetComptesSurveillesCount` - Comptage des comptes surveillés
- **Status** : `GetComptesByStatut` - Filtrage par statut
- **Recherche** : `SearchComptes` - Recherche multi-critères
- **Monitoring** : `GetCompteActivity` - Activité récente

**Audit intégré** :
- Traçage automatique des créations avec `LogAction`
- Identification de l'acteur (Admin, Analyst, MDA, App externe)
- Historique complet des modifications

---

### 💰 `transactions.go` - Opérations Financières
**Rôle** : Gestion des transactions et mouvements financiers

**Fonctionnalités** :
- **Enregistrement** : `CreateTransaction` - Nouvelle transaction
- **Agrégation** : `GetTransactionSumForPeriod` - Sommes par période
- **Recherche** : `GetTransactionsByCompte` - Transactions par compte
- **Statistiques** : `GetTransactionStats` - Métriques d'analyse
- **Filtrage** : `GetTransactionsByDateRange` - Recherche temporelle
- **Export** : `GetTransactionsForExport` - Données pour rapports

**Analyses supportées** :
- Détection de patterns suspects
- Calculs de seuils et moyennes
- Identification des pics d'activité
- Corrélations temporelles

---

### 🚫 `blacklist.go` - Liste Noire et Blocage
**Rôle** : Gestion des entités bloquées et surveillance renforcée

**Fonctionnalités** :
- **Ajout** : `CreateBlacklistEntry` - Nouvelle entrée avec motif
- **Vérification** : `IsAccountOnBlacklist` - Contrôle en temps réel
- **Recherche** : `GetBlacklistEntries` - Liste des entrées actives
- **Mise à jour** : `UpdateBlacklistEntry` - Modification des motifs
- **Suppression** : `RemoveFromBlacklist` - Retrait de la liste
- **Historique** : `GetBlacklistHistory` - Traçage des changements
- **Statistiques** : `GetBlacklistStats` - Métriques de blocage

**Types de blocage** :
- Blocage préventif (suspicion)
- Blocage réglementaire (sanction)
- Surveillance renforcée
- Quarantaine temporaire

---

### ⚡ `rules.go` - Règles de Contrôle
**Rôle** : Configuration et gestion des règles métier automatiques

**Fonctionnalités** :
- **Création** : `CreateRegleControle` - Nouvelle règle métier
- **Modification** : `UpdateRegleControle` - Ajustement des paramètres
- **Consultation** : `GetAllReglesControle` - Liste des règles actives
- **Recherche** : `GetRegleControleByID` - Règle spécifique
- **Activation** : `ToggleRegleControle` - Activation/désactivation
- **Suppression** : `DeleteRegleControle` - Retrait d'une règle

**Types de règles supportées** :
- Seuils de montant (quotidien, mensuel, annuel)
- Fréquence des transactions
- Géolocalisation et pays à risque
- Patterns comportementaux
- Règles de conformité réglementaire

---

### 🚨 `alerts.go` - Système d'Alertes
**Rôle** : Génération, gestion et suivi des alertes de sécurité

**Fonctionnalités** :
- **Récupération** : `GetAlertes` - Liste des alertes avec filtres
- **Création** : `CreateAlerte` - Nouvelle alerte automatique
- **Traitement** : `UpdateAlerteStatus` - Mise à jour du statut
- **Recherche** : `GetAlertesByStatut` - Filtrage par statut
- **Statistiques** : `GetAlertesStats` - Métriques d'analyse
- **Actions** : `ProcessAlerteAction` - Exécution des actions

**Statuts d'alertes** :
- `nouvelle` - Alerte générée non traitée
- `en_cours` - En cours d'analyse par un analyst
- `declaree` - Déclarée aux autorités (CENTIF)
- `classee` - Classée sans suite
- `bloquee` - Action de blocage effectuée

**Actions automatiques** :
- Déclaration de soupçon
- Blocage de compte
- Surveillance renforcée
- Notification aux équipes

---

### 📊 `monitoring.go` - Suivi et Surveillance
**Rôle** : Traçage des activités de surveillance et monitoring

**Fonctionnalités** :
- **Création** : `CreateSuiviBlacklist` - Nouveau suivi d'entité
- **Consultation** : `GetSuiviBlacklist` - Historique de surveillance
- **Mise à jour** : `UpdateSuiviBlacklist` - Évolution du suivi
- **Statistiques** : `GetSuiviStats` - Métriques de surveillance

**Cas d'usage** :
- Suivi des comptes sous surveillance
- Traçage des décisions d'analyse
- Historique des actions correctives
- Reporting de conformité

---

### 🌍 `countries.go` - Pays à Risque
**Rôle** : Gestion de la liste des pays à risque géopolitique

**Fonctionnalités** :
- **Consultation** : `GetAllPaysARisque` - Liste complète
- **Ajout** : `AddPaysARisque` - Nouveau pays à risque
- **Mise à jour** : `UpdatePaysARisque` - Modification du niveau de risque
- **Suppression** : `RemovePaysARisque` - Retrait de la liste

**Niveaux de risque** :
- Élevé (blocage automatique)
- Moyen (surveillance renforcée)
- Faible (reporting seul)

**Intégration** :
- Contrôles automatiques sur les transactions
- Génération d'alertes géographiques
- Reporting réglementaire

---

### 📈 `dashboard.go` - Métriques du Tableau de Bord
**Rôle** : Calcul des KPI et métriques pour l'interface de pilotage

**Fonctionnalités** :
- **Synthèse** : `GetDashboardSummary` - Vue d'ensemble des indicateurs
- **KPI temps réel** : Calculs dynamiques des métriques clés

**Métriques calculées** :
- Nombre total de comptes
- Transactions du jour/mois
- Alertes en cours
- Comptes sous surveillance
- Montants traités
- Taux de déclaration

**Performance** :
- Requêtes optimisées avec agrégations SQL
- Cache des calculs coûteux
- Mise à jour en temps réel

---

### 📊 `efficiency.go` - Métriques d'Efficacité
**Rôle** : Calcul des indicateurs de performance opérationnelle

**Fonctionnalités** :
- **Métriques** : `GetEfficiencyMetrics` - Calculs d'efficacité
- **Type** : `EfficiencyMetric` - Structure des données de performance

**Indicateurs calculés** :
- **Efficacité hebdomadaire** : Ratio alertes traitées/générées
- **Efficacité mensuelle** : Performance sur période étendue
- **Temps de traitement** : Délais moyens de résolution
- **Taux de faux positifs** : Qualité de la détection

**Calculs avancés** :
- Moyennes mobiles sur 4 semaines
- Tendances d'amélioration
- Comparaisons périodiques
- Formatage en français des mois

---

### 🔍 `audit.go` - Audit et Conformité
**Rôle** : Système d'audit complet et métriques de conformité réglementaire

**Fonctionnalités d'audit** :
- **Enregistrement** : `CreateLogAudit` - Création d'entrée d'audit
- **Logging automatique** : `LogAction` - Audit avec détection de rôle
- **Logging manuel** : `LogActionWithRole` - Audit avec rôle spécifié
- **Consultation** : `GetLogsAudit` - Recherche dans l'historique
- **Statistiques** : `GetAuditStatistics` - Métriques d'audit
- **Conformité** : `GetComplianceMetrics` - Indicateurs réglementaires

**Système à 4 niveaux d'acteurs** :
1. **Administrateurs** (`admin`) - Gestion système
2. **Analystes** (`analyst`) - Traitement des alertes
3. **MDA** (`system`) - Moteur de Détection d'Anomalie
4. **Applications externes** (`application`) - Apps tierces

**Métriques de conformité LBC/FT** :
- Déclarations de soupçon effectuées
- Entités bloquées
- Comptes sous surveillance
- Transactions avec pays à risque
- Montants à risque traités
- Taux de déclaration global

**Traçabilité complète** :
- Horodatage précis de toutes les actions
- Identification de l'acteur et de son rôle
- Détails des opérations effectuées
- Résultats et codes d'erreur
- Filtrage avancé pour les audits

---

## Avantages de cette architecture

### 🔧 Maintenabilité
- **Responsabilité unique** : Chaque fichier a un domaine clair
- **Modifications isolées** : Impact limité des changements
- **Code lisible** : Fichiers de taille raisonnable (30-80 lignes)

### 👥 Collaboration
- **Travail parallèle** : Équipes peuvent modifier différents domaines
- **Conflits réduits** : Moins de collision sur les merges
- **Expertise métier** : Spécialisation par domaine fonctionnel

### 🧪 Testabilité
- **Tests unitaires** : Isolation des responsabilités
- **Mocks facilités** : Interface centralisée
- **Couverture ciblée** : Tests par domaine métier

### 📚 Documentation
- **Auto-documentation** : Structure explicite
- **Onboarding facilité** : Compréhension rapide
- **Architecture évidente** : Organisation intuitive

### 🚀 Évolutivité
- **Nouveaux domaines** : Ajout simple de nouveaux fichiers
- **Refactoring graduel** : Modification par petits blocs
- **Monitoring** : Métriques et audit intégrés

---

## Statistiques de la refactorisation

### Avant
- **1 fichier** : `postgres_store.go`
- **1600+ lignes** de code
- **Complexité élevée** : Toutes responsabilités mélangées
- **Maintenance difficile** : Navigation complexe

### Après
- **14 fichiers** spécialisés
- **~50 lignes** par fichier en moyenne
- **Réduction de 95%** de la complexité par fichier
- **Architecture scalable** et maintenable

### Impact
- ✅ **Lisibilité** : Navigation facilitée
- ✅ **Performance** : Même performance, meilleure organisation
- ✅ **Robustesse** : Tests et audit intégrés
- ✅ **Conformité** : Audit à 4 niveaux opérationnel

---

## Convention de nommage

### Fichiers
- **Nom au singulier** : `user.go` (pas `users.go` sauf si le domaine est naturellement pluriel)
- **Nom du domaine métier** : Reflet de la responsabilité fonctionnelle
- **Extension `.go`** : Fichiers Go standard

### Méthodes
- **Préfixe d'action** : `Create`, `Get`, `Update`, `Delete`
- **Nom du domaine** : `CreateUser`, `GetAllComptes`
- **Suffixe descriptif** : `GetUserByEmail`, `GetComptesByStatut`

### Structures
- **CamelCase** : `LogAudit`, `EfficiencyMetric`
- **Noms explicites** : Reflet de l'usage métier
- **Documentation** : Commentaires Go standard

---

## Utilisation et bonnes pratiques

### Import
```go
import "anomaly-detector/internal/store"
```

### Instanciation
```go
store, err := store.NewStore(connectionString)
if err != nil {
    log.Fatal(err)
}
defer store.Close()
```

### Usage typique
```go
// Utilisation de l'interface unifiée
user, err := store.GetUserByEmail(ctx, "admin@example.com")
if err != nil {
    return err
}

// Création avec audit automatique
err = store.CreateCompte(ctx, compte, user.ID)
// → Génère automatiquement un log d'audit
```

Cette architecture modulaire assure une base solide pour l'évolution du système de détection d'anomalies tout en maintenant la conformité réglementaire et la traçabilité complète des opérations.