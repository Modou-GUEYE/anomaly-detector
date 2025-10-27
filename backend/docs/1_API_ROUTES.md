# API Routes - Anomaly Detector

## 📌 Vue d'ensemble

Cette API est **neutre et agnostique**. Elle est conçue pour être utilisée par différentes plateformes et systèmes du domaine LBC/FT. Les partenaires doivent s'adapter à l'API, pas l'inverse.

## 🔐 Rôles et permissions

| Rôle | Description | Permissions |
|------|-------------|------------|
| **admin** | Administrateur système | Accès complet + gestion des utilisateurs, règles, configuration |
| **analyst** | Analyste de risque | Lecture/écriture des anomalies, gestion des statuts |
| **system** | Système externe/API | Analyse de transactions, vérifications basiques |
| **reviewer** | Auditeur/Reviewer | Lecture seule sur toutes les données |

## 📚 Endpoints API

### 🔓 Authentification (Public)

```
POST /api/v1/auth/login
  Description: Authentifier un utilisateur avec email/password
  Body: { "email": "user@example.com", "password": "***" }
  Response: { "token": "eyJ..." }
  Status: 200 OK / 401 Unauthorized
  Roles: all

POST /api/v1/auth/token
  Description: Générer un token JWT pour un système externe (partenaire)
  Body: { "system_id": "external-system-1", "role": "system" }
  Response: { "token": "eyJ..." }
  Status: 200 OK / 400 Bad Request
  Notes: Endpoint générique et neutre pour tous les systèmes externes
  Example:
    curl -X POST http://localhost:8080/api/v1/auth/token \
      -H "Content-Type: application/json" \
      -d '{"system_id":"my-system","role":"system"}'
```

### 🔓 Entités - Vérification publique

```
GET /api/v1/entities/:identifier/status
  Description: Vérifier le statut d'une entité (publique)
  Params: identifier = numéro de compte ou identifiant
  Response: { "status": "Interdit|Blacklist|PEP|Surveillés|Aucun" }
  Status: 200 OK
```

### 🔐 Anomalies (Rôles: admin, analyst, system)

```
POST /api/v1/anomalies/analyze
  Description: Analyser une transaction
  Auth: Bearer token
  Body: { transaction object }
  Response: { "status": "anomalie_detectee|aucune_anomalie_detectee", "alertes": [...] }
  Status: 200 OK / 201 Created / 403 Forbidden

GET /api/v1/anomalies
  Description: Lister toutes les anomalies
  Auth: Bearer token
  Response: [{ alerte object }, ...]
  Status: 200 OK

GET /api/v1/anomalies/metrics?months=12
  Description: Obtenir les métriques d'anomalies (12 derniers mois par défaut)
  Auth: Bearer token
  Query params: months=N (optionnel)
  Response: { "metrics": { "type": [...] } }
  Status: 200 OK

GET /api/v1/anomalies/:id
  Description: Obtenir le détail d'une anomalie
  Auth: Bearer token
  Response: { alerte object }
  Status: 200 OK

PUT /api/v1/anomalies/:id/status
  Description: Mettre à jour le statut d'une anomalie
  Auth: Bearer token
  Body: { "statut": "Traité|Faux positif|..." }
  Response: { alerte object (updated) }
  Status: 200 OK
```

### 🔐 Entités / Blacklist (Rôles: admin, analyst)

```
GET /api/v1/entities
  Description: Lister toutes les entités blacklistées
  Auth: Bearer token
  Response: [{ entity object }, ...]
  Status: 200 OK

POST /api/v1/entities
  Description: Créer une nouvelle entité blacklistée
  Auth: Bearer token
  Body: { "nom": "...", "numeroCompte": "...", "typeEntite": "..." }
  Response: { entity object }
  Status: 201 Created

PUT /api/v1/entities/:id
  Description: Modifier une entité blacklistée
  Auth: Bearer token
  Body: { entity object (partial update) }
  Response: { entity object (updated) }
  Status: 200 OK

DELETE /api/v1/entities/:id
  Description: Supprimer une entité blacklistée
  Auth: Bearer token
  Response: Empty
  Status: 204 No Content

POST /api/v1/entities/:identifier/verify
  Description: Vérifier une entité (usage interne)
  Auth: Bearer token
  Response: { "verified": true|false, "reason": "..." }
  Status: 200 OK
```

### 🔐 Règles de Contrôle (Rôles: admin, analyst - lecture/écriture limitée)

```
GET /api/v1/rules
  Description: Lister toutes les règles
  Auth: Bearer token (admin, analyst)
  Response: [{ rule object }, ...]
  Status: 200 OK

GET /api/v1/rules/:id
  Description: Obtenir le détail d'une règle
  Auth: Bearer token (admin, analyst)
  Response: { rule object }
  Status: 200 OK

GET /api/v1/rules/:id/profiles
  Description: Lister les profils liés à une règle
  Auth: Bearer token (admin, analyst)
  Response: [{ profile object }, ...]
  Status: 200 OK

POST /api/v1/rules
  Description: Créer une nouvelle règle (admin uniquement)
  Auth: Bearer token (admin)
  Body: { rule object }
  Response: { rule object }
  Status: 201 Created

PUT /api/v1/rules/:id
  Description: Modifier une règle (admin uniquement)
  Auth: Bearer token (admin)
  Body: { rule object (partial update) }
  Response: { rule object (updated) }
  Status: 200 OK

DELETE /api/v1/rules/:id
  Description: Supprimer une règle (admin uniquement)
  Auth: Bearer token (admin)
  Response: Empty
  Status: 204 No Content
```

### 🔐 Profils Clients (Rôles: admin, analyst - lecture/écriture limitée)

```
GET /api/v1/profiles
  Description: Lister tous les profils clients
  Auth: Bearer token (admin, analyst)
  Response: [{ profile object }, ...]
  Status: 200 OK

GET /api/v1/profiles/:id
  Description: Obtenir le détail d'un profil
  Auth: Bearer token (admin, analyst)
  Response: { profile object }
  Status: 200 OK

GET /api/v1/profiles/:id/rules
  Description: Lister les règles d'un profil
  Auth: Bearer token (admin, analyst)
  Response: [{ rule object }, ...]
  Status: 200 OK

POST /api/v1/profiles
  Description: Créer un nouveau profil (admin uniquement)
  Auth: Bearer token (admin)
  Body: { profile object }
  Response: { profile object }
  Status: 201 Created

PUT /api/v1/profiles/:id
  Description: Modifier un profil (admin uniquement)
  Auth: Bearer token (admin)
  Body: { profile object (partial update) }
  Response: { profile object (updated) }
  Status: 200 OK

DELETE /api/v1/profiles/:id
  Description: Supprimer un profil (admin uniquement)
  Auth: Bearer token (admin)
  Response: Empty
  Status: 204 No Content
```

### 🔐 Comptes Clients (Rôles: admin, analyst)

```
GET /api/v1/accounts
  Description: Lister tous les comptes clients
  Auth: Bearer token (admin, analyst)
  Response: [{ account object }, ...]
  Status: 200 OK

POST /api/v1/accounts
  Description: Créer un nouveau compte (admin uniquement)
  Auth: Bearer token (admin)
  Body: { account object }
  Response: { account object }
  Status: 201 Created

GET /api/v1/accounts/:number/rules
  Description: Lister les règles actives d'un compte
  Auth: Bearer token (admin, analyst)
  Params: number = numéro de compte
  Response: [{ rule object }, ...]
  Status: 200 OK

GET /api/v1/accounts/monitored
  Description: Lister les comptes en surveillance
  Auth: Bearer token (admin, analyst)
  Response: { "count": N, "comptes": [...] }
  Status: 200 OK
```

### 🔐 Configuration Système (Rôles: admin uniquement)

```
GET /api/v1/configuration/risk-countries
  Description: Lister les pays à risque
  Auth: Bearer token (admin)
  Response: { "liste_pays": ["Pays1", "Pays2", ...] }
  Status: 200 OK

POST /api/v1/configuration/risk-countries
  Description: Ajouter un pays à la liste de risque
  Auth: Bearer token (admin)
  Body: { "pays": "CountryName" }
  Response: { "message": "Pays ajouté..." }
  Status: 200 OK

PUT /api/v1/configuration/risk-countries/:country
  Description: Modifier la liste des pays à risque
  Auth: Bearer token (admin)
  Body: { "liste_pays": ["Pays1", "Pays2", ...] }
  Response: { "message": "Liste mise à jour..." }
  Status: 200 OK

DELETE /api/v1/configuration/risk-countries/:country
  Description: Retirer un pays de la liste de risque
  Auth: Bearer token (admin)
  Body: { "pays": "CountryName" }
  Response: { "message": "Pays retiré..." }
  Status: 200 OK
```

### 🔐 Profil Utilisateur (Tous les rôles authentifiés)

```
GET /api/v1/profile
  Description: Obtenir le profil de l'utilisateur connecté
  Auth: Bearer token
  Response: { user object }
  Status: 200 OK
```

## 🔑 Format des headers

Tous les endpoints protégés (🔐) doivent inclure:

```
Authorization: Bearer <token_jwt>
Content-Type: application/json
```

## 📋 Exemples d'utilisation

### 1. Se connecter
```bash
curl -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"admin@gsie.sn","password":"admin123"}'
```

### 2. Analyser une transaction
```bash
curl -X POST http://localhost:8080/api/v1/anomalies/analyze \
  -H "Authorization: Bearer eyJ..." \
  -H "Content-Type: application/json" \
  -d '{ transaction object }'
```

### 3. Lister les anomalies
```bash
curl -X GET http://localhost:8080/api/v1/anomalies \
  -H "Authorization: Bearer eyJ..."
```

### 4. Obtenir les métriques
```bash
curl -X GET http://localhost:8080/api/v1/anomalies/metrics?months=24 \
  -H "Authorization: Bearer eyJ..."
```

## 🚀 Migration vers Keycloak

Cette API est prête pour la migration vers Keycloak :
- Les rôles sont génériques (admin, analyst, system, reviewer)
- Les tokens JWT sont standards
- Pas de dépendance aux noms de systèmes spécifiques

Pour intégrer Keycloak :
1. Remplacer la génération de JWT local par Keycloak
2. Adapter le middleware `jwtAuthMiddleware()` pour valider les tokens Keycloak
3. Mapper les rôles Keycloak vers les rôles de l'API

## 📝 Notes

- **C'est aux partenaires de s'adapter, pas l'inverse** - L'API expose tous les endpoints, c'est au partenaire de n'utiliser que ceux dont il a besoin
- **Endpoints legacy** conservés pour compatibilité rétroactive
- **Scalable** - Prêt pour Keycloak et multi-applications
