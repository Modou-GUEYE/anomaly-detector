# Structure du fichier server.go

## 📋 Vue d'ensemble

Le fichier `server.go` est organisé en **9 sections logiques** pour une meilleure maintenabilité et une navigation simplifiée.

## 🏗️ Sections du fichier

### 1️⃣ PACKAGE & IMPORTS (Lignes 1-21)
- Déclaration du package `api`
- Imports de dépendances internes et externes
- Documentation Swagger

### 2️⃣ CONSTANTES & VARIABLES GLOBALES (Lignes 23-25)
- `jwtKey` : Clé secrète pour la signature des tokens JWT

### 3️⃣ STRUCTURES PRINCIPALES (Lignes 27-31)
- `Server` : Structure principale du serveur HTTP
  - Champs : `store` (accès base de données), `router` (moteur Gin)

### 4️⃣ MIDDLEWARES D'AUTHENTIFICATION & AUTORISATION (Lignes 33-125)
Trois middlewares de sécurité :

#### `jwtAuthMiddleware()` (Lignes 33-78)
- Valide la présence du header Authorization
- Parse et valide le token JWT
- Extrait les claims (userID, role) dans le contexte Gin

#### `roleMiddleware(allowedRoles ...)` (Lignes 80-100)
- Restreint l'accès selon les rôles
- Utilisable pour contrôler l'accès à un groupe de routes
- Exemple: `roleMiddleware("admin", "analyst")`

#### `checkRole(requiredRole)` (Lignes 102-125)
- Middleware granulaire pour vérifier un rôle spécifique
- Utile pour limiter une route à un seul rôle
- Exemple: `checkRole("admin")`

### 5️⃣ HANDLERS PUBLICS - AUTHENTIFICATION (Lignes 127-175)

#### `handleLogin()` (Lignes 133-161)
- Route: `POST /api/v1/auth/login`
- Authentifie un utilisateur avec email/password
- Retourne un token JWT

#### `handleGetSystemToken()` (Lignes 163-184)
- Route: `POST /api/v1/auth/token`
- Génère un token pour un système externe
- Accepte `system_id` et `role` en paramètres
- Endpoint neutre et générique

### 6️⃣ HANDLERS PROTÉGÉS - DOMAINES PAR DOMAINE

Les handlers sont organisés par domaine fonctionnel :

#### ANOMALIES/TRANSACTIONS (Lignes 186-420)
- `handleAnalyzeTransaction()` : Analyser une transaction
- `handleGetAlertMetrics()` : Métriques d'alertes
- `handleGetAlertes()` : Lister les alertes
- `handleUpdateAlerteStatus()` : Mettre à jour le statut d'une alerte

#### ENTITÉS/BLACKLIST (Lignes 422-530)
- `handleGetBlacklistEntries()` : Lister la blacklist
- `handleCreateBlacklistEntry()` : Créer une entrée
- `handleUpdateBlacklistEntry()` : Modifier une entrée
- `handleDeleteBlacklistEntry()` : Supprimer une entrée
- `handleCheckBlacklist()` : Vérifier si un compte est blacklisté

#### RÈGLES DE CONTRÔLE (Lignes 532-638)
- `handleGetAllRegles()` : Lister toutes les règles
- `handleCreateRegleControle()` : Créer une règle
- `handleUpdateRegleControle()` : Modifier une règle
- `handleDeleteRegleControle()` : Supprimer une règle
- `handleGetReglesByProfil()` : Règles d'un profil
- `handleGetReglesForCompte()` : Règles d'un compte

#### PROFILS CLIENTS (Lignes 640-720)
- `handleGetProfilsClient()` : Lister les profils
- `handleCreateProfilClient()` : Créer un profil
- `handleUpdateProfilClient()` : Modifier un profil
- `handleDeleteProfilClient()` : Supprimer un profil

#### COMPTES CLIENTS (Lignes 722-780)
- `handleGetTransactions()` : Lister les transactions
- `handleCreateCompte()` : Créer un compte
- `handleGetComptesSurveilles()` : Comptes surveillés

#### CONFIGURATION (Lignes 782-860)
- `handleGetPaysRisque()` : Liste des pays à risque
- `handleAddPaysRisque()` : Ajouter un pays
- `handleUpdatePaysRisque()` : Modifier la liste
- `handleRemovePaysRisque()` : Retirer un pays

#### PROFIL UTILISATEUR (Lignes 862-878)
- `handleGetProfile()` : Profil de l'utilisateur connecté

### 7️⃣ INITIALISATION SERVEUR & ROUTES (Lignes 880-980)

#### `NewServer(store)` 
- Crée une instance de serveur Gin
- Configure les 8 groupes de routes :
  1. **Auth** (public)
  2. **Entities** (public + protégé)
  3. **Anomalies** (protégé : admin, analyst, system)
  4. **Rules** (protégé : admin, analyst)
  5. **Profiles** (protégé : admin, analyst)
  6. **Accounts** (protégé : admin, analyst)
  7. **Configuration** (protégé : admin uniquement)
  8. **Profile** (protégé : tous authentifiés)

### 8️⃣ DEMARRAGE SERVEUR (Lignes 982-986)

#### `Start(address)`
- Lance le serveur HTTP sur l'adresse spécifiée
- Exemple: `server.Start(":8080")`

### 9️⃣ FONCTIONS UTILITAIRES (Lignes 988-1003)

#### `HashPassword(password)`
- Hache un mot de passe avec bcrypt
- Retourne le hash sécurisé

#### `checkPasswordHash(password, hash)`
- Compare un mot de passe avec son hash bcrypt
- Retourne `true` si les mots de passe correspondent

---

## 🧭 Navigation rapide

| Section | Type | Lignes | Utilité |
|---------|------|--------|---------|
| PACKAGE | Config | 1-21 | Imports et dépendances |
| CONSTANTES | Config | 23-25 | Clés secrètes |
| STRUCTURES | Data | 27-31 | Structures de données |
| MIDDLEWARES | Auth | 33-125 | Sécurité JWT et rôles |
| HANDLERS AUTH | Handlers | 127-184 | Authentification |
| HANDLERS ANOMALIES | Handlers | 186-420 | Gestion des alertes |
| HANDLERS ENTITIES | Handlers | 422-530 | Gestion de la blacklist |
| HANDLERS RULES | Handlers | 532-638 | Gestion des règles |
| HANDLERS PROFILES | Handlers | 640-720 | Gestion des profils |
| HANDLERS ACCOUNTS | Handlers | 722-780 | Gestion des comptes |
| HANDLERS CONFIG | Handlers | 782-860 | Configuration système |
| HANDLERS PROFILE USER | Handlers | 862-878 | Profil utilisateur |
| INITIALISATION | Setup | 880-980 | Configuration des routes |
| DEMARRAGE | Setup | 982-986 | Lancement serveur |
| UTILS | Helpers | 988-1003 | Utilitaires cryptographiques |

---

## 🎯 Principes de design

### Organisation par domaine
Les handlers sont groupés par **domaine fonctionnel**, pas par CRUD ou statut HTTP.

### Architecture neutre
- Aucune référence à des plateformes spécifiques (BIS, eKYC)
- Noms génériques : "système externe", "partenaire"
- Facilite l'intégration de nouveaux partenaires

### Middleware en chaîne
```go
route.Use(jwtAuthMiddleware(), roleMiddleware("admin", "analyst"))
```
Permet une composition flexible des contrôles de sécurité.

### Handlers orientés ressources
Chaque handler représente une **action sur une ressource** :
- `handleGet*()` : Lecture
- `handleCreate*()` : Création
- `handleUpdate*()` : Modification
- `handleDelete*()` : Suppression
- `handleAnalyze*()` : Traitement métier

---

## 📝 Ajout d'un nouveau handler

1. **Créer le handler** dans la section appropriée
2. **Ajouter la documentation Swagger** (tags `@Summary`, `@Router`, etc.)
3. **Enregistrer la route** dans `NewServer()` avec les middlewares corrects
4. **Tester** l'endpoint

Exemple :
```go
// HANDLERS PROTÉGÉS - MON DOMAINE
func (server *Server) handleMyNewAction(ctx *gin.Context) {
    // Implémentation
}

// Dans NewServer():
myRoutes := apiV1.Group("/my-domain")
myRoutes.Use(jwtAuthMiddleware(), roleMiddleware("admin"))
{
    myRoutes.POST("/action", server.handleMyNewAction)
}
```

---

## 🔒 Sécurité

- **JWT Authentication** : Tous les endpoints protégés utilisent le middleware JWT
- **Role-Based Access Control (RBAC)** : Contrôle des accès par rôle
- **Bcrypt Password Hashing** : Mots de passe sécurisés
- **Granular Permission Control** : Middleware `checkRole()` pour permissions fines
