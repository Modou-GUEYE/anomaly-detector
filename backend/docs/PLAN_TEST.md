# PLAN DE TEST FONCTIONNEL COMPLET
## API MDA - Anomaly Detector

**Date** : 22 Octobre 2025  
**Projet** : Anomaly Detector API  
**Version** : 2.0  
**Auteur** : Architecture Team  

---

## 📋 TABLE DES MATIÈRES

1. Résumé des Routes API
2. Préalables et Configuration
3. Scénarios de Test
4. Guide d'Exécution avec cURL
5. Matrice de Permissions
6. Critères de Succès

---

## 📋 RÉSUMÉ DES ROUTES API

### 🔐 AUTHENTIFICATION (Public)

| Méthode | Route | Description |
|:---:|:---|:---|
| POST | `/api/v1/auth/login` | Connexion utilisateur (email/password) |
| POST | `/api/v1/auth/token` | Générer token système (system_id/role) |

---

### 👤 PROFIL UTILISATEUR (Protégé)

| Méthode | Route | Description |
|:---:|:---|:---|
| GET | `/api/v1/profile` | Récupérer profil utilisateur connecté |

---

### 🌍 CONFIGURATION - PAYS À RISQUE (Admin)

| Méthode | Route | Description |
|:---:|:---|:---|
| GET | `/api/v1/configuration/risk-countries` | Lister tous les pays à risque |
| POST | `/api/v1/configuration/risk-countries` | Ajouter un pays à la liste |
| PUT | `/api/v1/configuration/risk-countries/:country` | Mettre à jour la liste complète |
| DELETE | `/api/v1/configuration/risk-countries/:country` | Retirer un pays de la liste |

---

### 📋 RÈGLES DE CONTRÔLE (Admin/Analyste)

| Méthode | Route | Description |
|:---:|:---|:---|
| GET | `/api/v1/rules` | Lister toutes les règles |
| POST | `/api/v1/rules` | Créer une nouvelle règle (Admin) |
| PUT | `/api/v1/rules/:id` | Modifier une règle (Admin) |
| DELETE | `/api/v1/rules/:id` | Supprimer une règle (Admin) |

---

### 🔍 ANOMALIES / TRANSACTIONS (Analyste/System)

| Méthode | Route | Description |
|:---:|:---|:---|
| POST | `/api/v1/anomalies/analyze` | Analyser une transaction |
| GET | `/api/v1/anomalies` | Lister toutes les anomalies/alertes |
| PUT | `/api/v1/anomalies/:id/status` | Mettre à jour le statut d'une alerte |
| GET | `/api/v1/anomalies/metrics` | Récupérer les métriques d'anomalies |

---

## ⚙️ PRÉALABLES ET CONFIGURATION

### Configuration Requise

- PostgreSQL 14+ en cours d'exécution
- Go 1.24.5+ installé
- curl ou Postman
- Terminal/Command Prompt

### Étapes de Démarrage

```bash
# 1. Peupler la base de données
cd backend
go run ./cmd/seeder

# 2. Lancer le serveur API
cd backend
go run ./cmd/api
```

Le serveur doit démarrer sur : **http://localhost:8080**

### Comptes de Test

| Email | Mot de passe | Rôle |
|:---|:---|:---|
| admin@gsie.sn | admin123 | Admin |
| analyste@gsie.sn | analyste123 | Analyst |

---

## 🧪 SCÉNARIOS DE TEST

### SCÉNARIO 1 : AUTHENTIFICATION ET ACCÈS DE BASE

#### Test 1.1 : Connexion Admin (Succès)

**Endpoint** : `POST /api/v1/auth/login`

**Body JSON** :
```json
{
  "email": "admin@gsie.sn",
  "password": "admin123"
}
```

**Résultat attendu** : 
- Code HTTP : **200 OK**
- Retourne un token JWT (à utiliser pour les requêtes protégées)

---

#### Test 1.2 : Connexion Analyste (Succès)

**Endpoint** : `POST /api/v1/auth/login`

**Body JSON** :
```json
{
  "email": "analyste@gsie.sn",
  "password": "analyste123"
}
```

**Résultat attendu** : 
- Code HTTP : **200 OK**
- Retourne un token JWT

---

#### Test 1.3 : Échec de Connexion (Mot de passe incorrect)

**Endpoint** : `POST /api/v1/auth/login`

**Body JSON** :
```json
{
  "email": "admin@gsie.sn",
  "password": "mauvais_mdp"
}
```

**Résultat attendu** : 
- Code HTTP : **401 Unauthorized**
- Message d'erreur approprié

---

#### Test 1.4 : Accès au Profil (Succès)

**Endpoint** : `GET /api/v1/profile`

**Headers** :
```
Authorization: Bearer <ANALYST_TOKEN>
```

**Résultat attendu** : 
- Code HTTP : **200 OK**
- Retourne les infos de l'utilisateur (id, email, nom, rôle, etc.)

---

#### Test 1.5 : Échec d'Accès au Profil (sans token)

**Endpoint** : `GET /api/v1/profile`

**Headers** : Aucun

**Résultat attendu** : 
- Code HTTP : **401 Unauthorized**
- Message : "Header d'autorisation manquant"

---

### SCÉNARIO 2 : GESTION DES PAYS À RISQUE ET RÈGLES

#### Test 2.1 : Ajouter un Pays à Risque

**Endpoint** : `POST /api/v1/configuration/risk-countries`

**Headers** :
```
Authorization: Bearer <ADMIN_TOKEN>
Content-Type: application/json
```

**Body JSON** :
```json
{
  "pays": "Mali"
}
```

**Résultat attendu** : 
- Code HTTP : **200 OK**
- Message : "Pays ajouté à la liste des pays à risque"

---

#### Test 2.2 : Lister les Pays à Risque

**Endpoint** : `GET /api/v1/configuration/risk-countries`

**Headers** :
```
Authorization: Bearer <ANALYST_TOKEN>
```

**Résultat attendu** : 
- Code HTTP : **200 OK**
- Retourne :
```json
{
  "liste_pays": ["Mali", "Burkina Faso"]
}
```

---

#### Test 2.3 : Modifier la Liste des Pays

**Endpoint** : `PUT /api/v1/configuration/risk-countries/Mali`

**Headers** :
```
Authorization: Bearer <ADMIN_TOKEN>
Content-Type: application/json
```

**Body JSON** :
```json
{
  "liste_pays": ["Burkina Faso", "Niger"]
}
```

**Résultat attendu** : 
- Code HTTP : **200 OK**
- Message : "Liste des pays à risque mise à jour avec succès"

---

#### Test 2.4 : Retirer un Pays de la Liste

**Endpoint** : `DELETE /api/v1/configuration/risk-countries/Mali`

**Headers** :
```
Authorization: Bearer <ADMIN_TOKEN>
Content-Type: application/json
```

**Body JSON** :
```json
{
  "pays": "Mali"
}
```

**Résultat attendu** : 
- Code HTTP : **200 OK**
- Message : "Pays retiré de la liste des pays à risque"

---

#### Test 2.5 : Admin Crée une Règle (Succès)

**Endpoint** : `POST /api/v1/rules`

**Headers** :
```
Authorization: Bearer <ADMIN_TOKEN>
Content-Type: application/json
```

**Body JSON - Exemple 1 (Fréquence)** :
```json
{
  "nomRegle": "Fréquence > 5 en 24h",
  "description": "Déclenche une alerte si plus de 5 transactions en 24h",
  "typeRegle": "frequence",
  "criticite": "Moyenne",
  "seuil": 5,
  "periodeReference": "24h",
  "active": true,
  "profilClientId": "ID_DU_PROFIL_PARTICULIER"
}
```

**Body JSON - Exemple 2 (Montant)** :
```json
{
  "nomRegle": "Cumul Hebdomadaire > 1M FCFA",
  "description": "Alerte si cumul 7 jours > 1,000,000 FCFA",
  "typeRegle": "montant",
  "criticite": "Élevée",
  "seuil": 1000000.0,
  "periodeReference": "7d",
  "active": true,
  "profilClientId": "ID_DU_PROFIL_ENTREPRISE"
}
```

**Résultat attendu** : 
- Code HTTP : **201 Created**
- Retourne la règle créée avec son ID

---

#### Test 2.6 : Analyste ne peut PAS Créer une Règle

**Endpoint** : `POST /api/v1/rules`

**Headers** :
```
Authorization: Bearer <ANALYST_TOKEN>
Content-Type: application/json
```

**Body JSON** : (identique au Test 2.5)

**Résultat attendu** : 
- Code HTTP : **403 Forbidden**
- Message : "Accès refusé pour ce rôle"

---

### SCÉNARIO 3 : DÉTECTION D'ANOMALIES

#### Étape A : Test du Contrôle Blacklist

**Endpoint** : `POST /api/v1/anomalies/analyze`

**Headers** :
```
Authorization: Bearer <ANALYST_TOKEN>
Content-Type: application/json
```

**Body JSON** :
```json
{
  "montant": 50.0,
  "numero_compte_payeur": "SN01100020098765432109876",
  "nom_payeur": "Client Interdit"
}
```

**Résultat attendu** : 
- Code HTTP : **403 Forbidden**
- Message d'erreur indiquant que le payeur est interdit

---

#### Étape B : Test de la Règle de Fréquence

**Préparation** : Utiliser un compte valide : `SN01100020098765432109000`

**Première exécution** : Envoyer 5 fois la même requête

**Endpoint** : `POST /api/v1/anomalies/analyze`

**Headers** :
```
Authorization: Bearer <ANALYST_TOKEN>
Content-Type: application/json
```

**Body JSON** (à répéter 5 fois) :
```json
{
  "montant": 100.0,
  "numero_compte_payeur": "SN01100020098765432109000"
}
```

**Résultat (5 premières)** : 
- Code HTTP : **200 OK**
- Message : "Aucune anomalie détectée"

**6ème exécution** : Même requête

**Résultat (6ème)** : 
- Code HTTP : **201 Created**
- Retourne une alerte de violation de fréquence

---

#### Étape C : Test de la Règle Pays à Risque

**Préparation** : S'assurer que "Mali" est dans la liste des pays à risque

**Endpoint** : `POST /api/v1/anomalies/analyze`

**Headers** :
```
Authorization: Bearer <ANALYST_TOKEN>
Content-Type: application/json
```

**Body JSON** :
```json
{
  "montant": 100.0,
  "numero_compte_payeur": "SN01100020098765432109001",
  "pays_payeur": "Mali"
}
```

**Résultat attendu** : 
- Code HTTP : **201 Created**
- Retourne une alerte pour "Pays à haut risque"
- Champ `scoreRisque` : 1.0
- Message contient : "Transaction depuis un pays à haut risque: Mali"

---

#### Étape D : Test de la Règle de Cumul

**Requête 1** : `POST /api/v1/anomalies/analyze`

**Headers** :
```
Authorization: Bearer <ANALYST_TOKEN>
Content-Type: application/json
```

**Body JSON** :
```json
{
  "montant": 600000.0,
  "numero_compte_payeur": "SN01100040099887766554433"
}
```

**Résultat** : 
- Code HTTP : **200 OK**

**Requête 2** : Même endpoint avec montant différent

**Body JSON** :
```json
{
  "montant": 500000.0,
  "numero_compte_payeur": "SN01100040099887766554433"
}
```

**Résultat attendu** : 
- Code HTTP : **201 Created**
- Alerte de dépassement de cumul générée

---

### SCÉNARIO 4 : TRAITEMENT DES ALERTES

#### Étape A : Consulter les Alertes

**Endpoint** : `GET /api/v1/anomalies`

**Headers** :
```
Authorization: Bearer <ANALYST_TOKEN>
```

**Résultat attendu** : 
- Code HTTP : **200 OK**
- Retourne un tableau d'alertes :

```json
[
  {
    "id": "df8326b4-d29d-47f2-a41c-84a9e07f34db",
    "transactionId": "...",
    "raisonDetection": "Fréquence de 6 transactions dépassant le seuil de 5 sur 24h.",
    "statut": "En cours",
    "analysteAssigneId": "",
    "scoreRisque": 0.8,
    "dateGeneration": "2025-10-22T10:30:00Z"
  }
]
```

---

#### Étape B : Mettre à Jour le Statut d'une Alerte

**Endpoint** : `PUT /api/v1/anomalies/{ALERT_ID}/status`

**Headers** :
```
Authorization: Bearer <ANALYST_TOKEN>
Content-Type: application/json
```

**Body JSON** :
```json
{
  "statut": "En Investigation"
}
```

**Résultat attendu** : 
- Code HTTP : **200 OK**
- Alerte mise à jour avec :
  - Nouveau statut : "En Investigation"
  - `analysteAssigneId` : ID de l'analyste

Réponse :
```json
{
  "id": "df8326b4-d29d-47f2-a41c-84a9e07f34db",
  "statut": "En Investigation",
  "analysteAssigneId": "5bea9b1a-dd0a-490e-927f-a328c87176ed",
  "dateModification": "2025-10-22T10:35:00Z"
}
```

---

## 🧪 GUIDE D'EXÉCUTION AVEC cURL

### Étape 1 : Authentification Admin

```bash
curl -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email": "admin@gsie.sn", "password": "admin123"}'
```

**Réponse** :
```json
{
  "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
  "message": "Connexion réussie"
}
```

**Sauvegarder le token** :
```bash
ADMIN_TOKEN="eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."
```

---

### Étape 2 : Authentification Analyste

```bash
curl -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email": "analyste@gsie.sn", "password": "analyste123"}'
```

**Sauvegarder le token** :
```bash
ANALYST_TOKEN="eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."
```

---

### Étape 3 : Tester les Pays à Risque

**Ajouter un pays** :
```bash
curl -X POST http://localhost:8080/api/v1/configuration/risk-countries \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"pays": "Mali"}'
```

**Lister les pays** :
```bash
curl -X GET http://localhost:8080/api/v1/configuration/risk-countries \
  -H "Authorization: Bearer $ANALYST_TOKEN"
```

---

### Étape 4 : Tester les Règles

**Créer une règle** (Admin seulement) :
```bash
curl -X POST http://localhost:8080/api/v1/rules \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "nomRegle": "Fréquence > 5 en 24h",
    "description": "Alerte si plus de 5 transactions en 24h",
    "typeRegle": "frequence",
    "criticite": "Moyenne",
    "seuil": 5,
    "periodeReference": "24h",
    "active": true
  }'
```

---

### Étape 5 : Analyser une Transaction

```bash
curl -X POST http://localhost:8080/api/v1/anomalies/analyze \
  -H "Authorization: Bearer $ANALYST_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "montant": 500000.0,
    "numero_compte_payeur": "SN01100040099887766554433",
    "nom_payeur": "Client Test"
  }'
```

---

### Étape 6 : Consulter les Alertes

```bash
curl -X GET http://localhost:8080/api/v1/anomalies \
  -H "Authorization: Bearer $ANALYST_TOKEN"
```

---

### Étape 7 : Mettre à Jour le Statut d'une Alerte

```bash
ALERT_ID="df8326b4-d29d-47f2-a41c-84a9e07f34db"

curl -X PUT http://localhost:8080/api/v1/anomalies/$ALERT_ID/status \
  -H "Authorization: Bearer $ANALYST_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"statut": "En Investigation"}'
```

---

## � GUIDE POSTMAN (Recommandé)

### Configuration Initiale

#### 1. Créer un Environnement
- Ouvre Postman
- Clique sur l'icône "Environment" (engrenage) en haut à droite
- Crée un nouvel environment : "MDA Local"
- Ajoute les variables :

| Variable | Valeur |
|:---|:---|
| `base_url` | http://localhost:8080 |
| `admin_token` | (à remplir après login) |
| `analyst_token` | (à remplir après login) |

#### 2. Organiser en Collection
- New Collection : "API MDA Tests"
- Grouper les requests par catégorie

---

### 📝 Requests Postman - Pas à Pas

#### Request 1 : Authentification Admin

| Propriété | Valeur |
|:---|:---|
| **Méthode** | POST |
| **URL** | `{{base_url}}/api/v1/auth/login` |

**Headers** :
```
Content-Type: application/json
```

**Body** (raw JSON) :
```json
{
  "email": "admin@gsie.sn",
  "password": "admin123"
}
```

**Après exécution** : Copie le `token` dans la variable `admin_token`

---

#### Request 2 : Authentification Analyste

| Propriété | Valeur |
|:---|:---|
| **Méthode** | POST |
| **URL** | `{{base_url}}/api/v1/auth/login` |

**Headers** :
```
Content-Type: application/json
```

**Body** (raw JSON) :
```json
    {
    "email": "analyste@gsie.sn",
    "password": "analyste123"
    }
```

**Après exécution** : Copie le `token` dans la variable `analyst_token`

---

#### Request 3 : Générer Token Système ⚠️ IMPORTANT

| Propriété | Valeur |
|:---|:---|
| **Méthode** | POST |
| **URL** | `{{base_url}}/api/v1/auth/token` |

**Headers** :
```
Content-Type: application/json
```

**Body** (raw JSON) :
```json
{
  "system_id": "anomaly-detector",
  "role": "analyst"
}
```

**⚠️ ATTENTION** : Utilise `system_id` et `role`, PAS email/password !

---

#### Request 4 : Récupérer le Profil

| Propriété | Valeur |
|:---|:---|
| **Méthode** | GET |
| **URL** | `{{base_url}}/api/v1/profile` |

**Headers** :
```
Authorization: Bearer {{analyst_token}}
```

---

#### Request 5 : Ajouter un Pays

| Propriété | Valeur |
|:---|:---|
| **Méthode** | POST |
| **URL** | `{{base_url}}/api/v1/configuration/risk-countries` |

**Headers** :
```
Authorization: Bearer {{admin_token}}
Content-Type: application/json
```

**Body** (raw JSON) :
```json
{
  "pays": "Mali"
}
```

---

#### Request 6 : Lister les Pays

| Propriété | Valeur |
|:---|:---|
| **Méthode** | GET |
| **URL** | `{{base_url}}/api/v1/configuration/risk-countries` |

**Headers** :
```
Authorization: Bearer {{analyst_token}}
```

---

#### Request 7 : Créer une Règle

| Propriété | Valeur |
|:---|:---|
| **Méthode** | POST |
| **URL** | `{{base_url}}/api/v1/rules` |

**Headers** :
```
Authorization: Bearer {{admin_token}}
Content-Type: application/json
```

**Body** (raw JSON) :
```json
{
  "nomRegle": "Fréquence > 5 en 24h",
  "description": "Alerte si plus de 5 transactions",
  "typeRegle": "frequence",
  "criticite": "Moyenne",
  "seuil": 5,
  "periodeReference": "24h",
  "active": true
}
```

---

#### Request 8 : Lister les Règles

| Propriété | Valeur |
|:---|:---|
| **Méthode** | GET |
| **URL** | `{{base_url}}/api/v1/rules` |

**Headers** :
```
Authorization: Bearer {{analyst_token}}
```

---

#### Request 9 : Analyser une Transaction

| Propriété | Valeur |
|:---|:---|
| **Méthode** | POST |
| **URL** | `{{base_url}}/api/v1/anomalies/analyze` |

**Headers** :
```
Authorization: Bearer {{analyst_token}}
Content-Type: application/json
```

**Body** (raw JSON) :
```json
{
  "montant": 500000.0,
  "numero_compte_payeur": "SN01100040099887766554433",
  "nom_payeur": "Client Test"
}
```

---

#### Request 10 : Lister les Alertes

| Propriété | Valeur |
|:---|:---|
| **Méthode** | GET |
| **URL** | `{{base_url}}/api/v1/anomalies` |

**Headers** :
```
Authorization: Bearer {{analyst_token}}
```

---

#### Request 11 : Mettre à Jour une Alerte

| Propriété | Valeur |
|:---|:---|
| **Méthode** | PUT |
| **URL** | `{{base_url}}/api/v1/anomalies/{ALERT_ID}/status` |

**Headers** :
```
Authorization: Bearer {{analyst_token}}
Content-Type: application/json
```

**Body** (raw JSON) :
```json
{
  "statut": "En Investigation"
}
```

**Statuts valides** : "En cours", "En Investigation", "Traité", "Résolu", "Faux positif"

---

### 💡 Conseils Postman

1. **Utiliser les Variables**
   - `{{base_url}}` → Remplacé automatiquement
   - `{{admin_token}}` → Remplacé automatiquement
   - Plus besoin de copier-coller partout!

2. **Copier un Token Facilement**
   - Clique sur l'onglet "Tests" dans Postman
   - Ajoute un script :
   ```javascript
   pm.environment.set("admin_token", pm.response.json().token);
   ```
   - Le token sera sauvegardé automatiquement après chaque login

3. **Vérifier les Réponses**
   - Onglet "Body" → Voir la réponse JSON formatée
   - Onglet "Status" → Voir le code HTTP

4. **Organiser les Requests**
   - Grouper par catégorie (Auth, Config, Tests, etc.)
   - Nommer clairement : "1. Login Admin", "2. Get Profile", etc.

---

## �📊 MATRICE DE PERMISSIONS

| Endpoint | Admin | Analyst | System |
|:---|:---:|:---:|:---:|
| `POST /auth/login` | ✅ | ✅ | ❌ |
| `POST /auth/token` | - | - | ✅ |
| `GET /profile` | ✅ | ✅ | ✅ |
| `GET /configuration/risk-countries` | ✅ | ❌ | ❌ |
| `POST /configuration/risk-countries` | ✅ | ❌ | ❌ |
| `PUT /configuration/risk-countries/:country` | ✅ | ❌ | ❌ |
| `DELETE /configuration/risk-countries/:country` | ✅ | ❌ | ❌ |
| `GET /rules` | ✅ | ✅ | ❌ |
| `POST /rules` | ✅ | ❌ | ❌ |
| `PUT /rules/:id` | ✅ | ❌ | ❌ |
| `DELETE /rules/:id` | ✅ | ❌ | ❌ |
| `POST /anomalies/analyze` | ✅ | ✅ | ✅ |
| `GET /anomalies` | ✅ | ✅ | ❌ |
| `PUT /anomalies/:id/status` | ✅ | ✅ | ❌ |
| `GET /anomalies/metrics` | ✅ | ✅ | ❌ |

---

## ✅ CRITÈRES DE SUCCÈS

Le plan de test est considéré comme **réussi** si tous les critères suivants sont validés :

### 1. Codes HTTP Corrects
- ✅ Les réponses retournent le code HTTP attendu (200, 201, 400, 401, 403, etc.)
- ✅ Les cas d'erreur retournent 4xx approprié

### 2. Intégrité des Données
- ✅ Les données créées en base de données sont correctes
- ✅ Les données modifiées sont mises à jour correctement
- ✅ Les données supprimées sont bien supprimées
- ✅ Les relations entre entités sont respectées

### 3. Authentification JWT
- ✅ Les tokens JWT sont générés correctement
- ✅ Les tokens valides permettent d'accéder aux ressources protégées
- ✅ Les tokens invalides retournent 401
- ✅ Les tokens expirés sont rejetés

### 4. Permissions par Rôle
- ✅ Admin peut accéder à tous les endpoints autorisés
- ✅ Analyst peut accéder seulement à ses endpoints
- ✅ System peut accéder seulement à ses endpoints
- ✅ Les tentatives non-autorisées retournent 403

### 5. Messages d'Erreur
- ✅ Les messages d'erreur sont clairs et pertinents
- ✅ Les messages d'erreur indiquent la cause du problème
- ✅ Les messages d'erreur ne révèlent pas d'infos sensibles

### 6. Logique Métier
- ✅ Les anomalies sont détectées correctement
- ✅ Les alertes sont générées selon les règles
- ✅ Les métriques sont calculées correctement
- ✅ Les statuts d'alertes sont mis à jour correctement

---

## 📝 CONCLUSION

Ce plan de test couvre tous les cas d'usage métier et techniques de l'API MDA avec des exemples concrets et exécutables. 

**Durée estimée** : 2-3 heures pour exécuter tous les tests

**Outils recommandés** : Postman, curl, Thunder Client

**Bon testig! 🚀**

---

*Généré le 22 Octobre 2025*  
*Anomaly Detector API - Version 2.0*
