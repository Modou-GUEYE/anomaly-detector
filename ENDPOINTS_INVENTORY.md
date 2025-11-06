# 📋 Inventaire des Endpoints API - Anomaly Detector

## 🔧 Routes Publiques

### Documentation
- `GET /swagger/*any` - Documentation Swagger UI

### Authentification
- `POST /api/v1/auth/login` - Connexion utilisateur
- `POST /api/v1/auth/token` - Génération token système

### Vérification Publique
- `GET /api/v1/entities/:identifier/status` - Statut blacklist d'une entité

---

## 🔒 Routes Protégées (Authentification JWT requise)

### 🚨 Anomalies / Transactions
**Groupe**: `/api/v1/anomalies`  
**Rôles**: admin, analyst, system
- `POST /analyze` - Analyser une transaction
- `GET /` - Lister toutes les alertes
- `GET /metrics` - Métriques d'alertes
- `GET /:id` - Obtenir une alerte spécifique
- `PUT /:id/status` - Mettre à jour le statut d'une alerte

### 📋 Entités / Blacklist  
**Groupe**: `/api/v1/entities`  
**Rôles**: admin, analyst
- `GET /` - Lister entrées blacklist
- `POST /` - Créer entrée blacklist
- `PUT /:id` - Mettre à jour entrée blacklist
- `DELETE /:id` - Supprimer entrée blacklist
- `POST /:identifier/verify` - Vérifier compte blacklist

### ⚙️ Règles de Contrôle
**Groupe**: `/api/v1/rules`  
**Rôles**: admin, analyst (admin requis pour modifications)
- `GET /` - Lister toutes les règles
- `GET /:id/profiles` - Règles d'un profil
- `GET /:id` - Obtenir une règle spécifique
- `POST /` - **[ADMIN ONLY]** Créer règle
- `PUT /:id` - **[ADMIN ONLY]** Mettre à jour règle
- `PATCH /:id/status` - **[ADMIN ONLY]** Activer/Désactiver règle
- `DELETE /:id` - **[ADMIN ONLY]** Supprimer règle

### 📊 Dashboard
**Groupe**: `/api/v1/dashboard`  
**Rôles**: admin, analyst, system, reviewer
- `GET /summary` - **Résumé complet dashboard (inclut anomalies_by_type)**

### 👥 Profils Clients
**Groupe**: `/api/v1/profiles`  
**Rôles**: admin, analyst (admin requis pour modifications)
- `GET /` - Lister profils clients
- `GET /:id` - Obtenir un profil spécifique
- `GET /:id/rules` - Règles d'un profil
- `POST /` - **[ADMIN ONLY]** Créer profil
- `PUT /:id` - **[ADMIN ONLY]** Mettre à jour profil
- `DELETE /:id` - **[ADMIN ONLY]** Supprimer profil

### 🏦 Comptes Clients
**Groupe**: `/api/v1/accounts`  
**Rôles**: admin, analyst (admin requis pour modifications)
- `GET /` - Lister tous les comptes
- `GET /:number/rules` - Règles d'un compte
- `GET /monitored` - Comptes surveillés
- `POST /` - **[ADMIN ONLY]** Créer compte

### 👁️ Surveillance
**Groupe**: `/api/v1/suivi-blacklist`  
**Rôles**: admin, analyst
- `GET /` - Lister comptes surveillés

### 🌍 Pays à Risque
**Groupe**: `/api/v1/pays-risque`  
**Rôles**: admin, analyst (admin requis pour modifications)
- `GET /` - Lister pays à risque
- `POST /` - **[ADMIN ONLY]** Ajouter pays
- `POST /import` - **[ADMIN ONLY]** Importer pays (CSV/JSON)
- `PUT /:nomPays` - **[ADMIN ONLY]** Mettre à jour pays
- `DELETE /:nomPays` - **[ADMIN ONLY]** Supprimer pays

### 💳 Transactions
**Groupe**: `/api/v1/transactions`  
**Rôles**: admin, analyst
- `GET /` - Lister toutes les transactions

### 👤 Profil Utilisateur
**Groupe**: `/api/v1/profile`  
**Rôles**: tous les utilisateurs authentifiés
- `GET /` - Obtenir profil utilisateur connecté

### 📈 Rapports & Statistiques
**Groupe**: `/api/v1/rapports`  
**Rôles**: admin, analyst
- `GET /efficiency` - **Données d'efficacité RÉELLES** (délai moyen, taux résolution)
- `GET /performance` - KPIs de performance (simulés)
- `GET /audit` - Données d'audit (simulés)

---

## 🔑 Notes Importantes

### Authentification
- **JWT Bearer Token** requis pour toutes les routes protégées
- Token obtenu via `/api/v1/auth/login` ou `/api/v1/auth/token`

### Autorisations
- **admin**: Accès complet + modifications
- **analyst**: Lecture + certaines modifications
- **system**: Accès pour systèmes externes
- **reviewer**: Lecture seule dashboard

### ⚠️ APIs par Type de Données
- **🟢 DONNÉES RÉELLES** : `/api/v1/dashboard/summary`, `/api/v1/rapports/efficiency`
- **🟡 DONNÉES SIMULÉES** : `/api/v1/rapports/performance`, `/api/v1/rapports/audit`

### ⚠️ API Supprimée (Architecture Unifiée)
- ~~`GET /api/v1/rapports/rule-violations`~~ - **SUPPRIMÉE**
- ✅ **Remplacée par**: `/api/v1/dashboard/summary` (section `anomalies_by_type`)

### Bonnes Pratiques
1. ✅ **Toujours vérifier** cet inventaire avant de créer de nouveaux endpoints
2. ✅ **Éviter la duplication** d'APIs (comme l'exemple rule-violations)
3. ✅ **Utiliser l'API centralisée** dashboard pour les données communes
4. ✅ **Privilégier les données réelles** pour les rapports métier
5. ✅ **Respecter les rôles** et permissions définis